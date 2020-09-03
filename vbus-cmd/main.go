package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	vBus "bitbucket.org/vbus/vbus.go"
	"github.com/jeremywohl/flatten"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "vbus-cmd",
		Usage: "send vbus commands",
		UsageText: "vbus-cmd [global options] command [command options] [arguments...]" +
			"\n\n   Examples:" +
			"\n   vbus-cmd discover system.zigbee" +
			"\n   vbus-cmd discover -j system.zigbee (json output)" +
			"\n   vbus-cmd discover -f system.zigbee (flattened output)" +
			"\n   vbus-cmd attribute get -t 10 system.zigbee.[...].1026.attributes.0" +
			"\n   vbus-cmd method call -t 120 system.zigbee.boolangery-ThinkPad-P1-Gen-2.controller.scan 120",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "debug", Aliases: []string{"d"}, Value: false, Usage: "Show vBus library logs"},
		},
		Before: func(c *cli.Context) error {
			if c.Bool("debug") {
				vBus.SetLogLevel(logrus.DebugLevel)
			} else {
				vBus.SetLogLevel(logrus.FatalLevel)
			}
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "discover",
				Aliases: []string{"d"},
				Usage:   "Discover elements on `PATH`",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "flatten", Aliases: []string{"f"}, Usage: "Display output as a flattened list"},
					&cli.BoolFlag{Name: "json", Aliases: []string{"j"}, Usage: "Display output as Json"},
				},
				ArgsUsage: "PATH",
				Action: func(c *cli.Context) error {
					conn := getConnection()
					askPermission(c.Args().Get(0), conn)
					if elem, err := conn.Discover(c.Args().Get(0), 2*time.Second); err != nil {
						return err
					} else {
						if c.Bool("flatten") {
							if casted, ok := elem.Tree().(map[string]interface{}); ok {
								flat, err := flatten.Flatten(casted, "", flatten.DotStyle)
								if err != nil {
									log.Fatal(err)
								}
								for k, v := range flat {
									fmt.Printf("%s.%s %v\n", c.Args().Get(0), k, v)
								}
							}
						} else if c.Bool("json") {
							fmt.Println(goToPrettyJson(elem.Tree()), "", flatten.DotStyle)
						} else {
							if elem.IsNode() {
								traverseNode(elem.AsNode(), 0)
							}
						}

						return nil
					}
				},
			},
			{
				Name:    "attribute",
				Aliases: []string{"a"},
				Usage:   "Send a command on a remote attribute ",
				Subcommands: []*cli.Command{
					{
						Name:    "set",
						Aliases: []string{"s"},
						Usage:   "Set `ATTR` `VALUE` (value is a Json string)",
						Description: "PATH is a dot style vBus path"+
							"\n	 VALUE is a Json value",
						ArgsUsage: "PATH VALUE",
						Action: func(c *cli.Context) error {
							attr := getAttribute(c.Args().Get(0))
							return attr.SetValue(jsonToGo(c.Args().Get(1)))
						},
					},
					{
						Name:    "get",
						Aliases: []string{"g"},
						Usage:   "Get `ATTR` value",
						Flags: []cli.Flag{
							&cli.IntFlag{Name: "timeout", Aliases: []string{"t"}, Value: 1},
						},
						Action: func(c *cli.Context) error {
							attr := getAttribute(c.Args().Get(0))
							if val, err := attr.ReadValueWithTimeout(time.Duration(c.Int("timeout")) * time.Second); err != nil {
								return err
							} else {
								fmt.Println(goToJson(val))
								return nil
							}
						},
					},
				},
			},
			{
				Name:    "method",
				Aliases: []string{"a"},
				Usage:   "Send a command on a remote method",
				Subcommands: []*cli.Command{
					{
						Name:    "call",
						Aliases: []string{"g"},
						Usage:   "Call `METHOD` (args must be passed as a Json string)",
						Flags: []cli.Flag{
							&cli.IntFlag{Name: "timeout", Aliases: []string{"t"}, Value: 1},
						},
						Action: func(c *cli.Context) error {
							attr := getMethod(c.Args().Get(0))
							args := jsonToGo(c.Args().Get(1))

							if _, ok := args.([]interface{}); !ok {
								// try to wrap args as a json array
								args = jsonToGo("[" + c.Args().Get(1) + "]")
							}
							if casted, ok := args.([]interface{}); !ok {
								log.Fatal("method args must be passed as a json array")
							} else {
								if val, err := attr.CallWithTimeout(time.Duration(c.Int("timeout"))*time.Second, casted...); err != nil {
									return err
								} else {
									fmt.Println(goToJson(val))
									return nil
								}
							}
							return nil
						},
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func traverseNode(node *vBus.NodeProxy, level int) {
	for name, elem := range node.Elements() {
		if elem.IsNode() {
			n := elem.AsNode()
			fmt.Printf("%s%s:\n", strings.Repeat(" ", level*2), name)
			traverseNode(n, level+1)
		} else if elem.IsAttribute() {
			attr := elem.AsAttribute()
			fmt.Printf("%s%s = %v\n", strings.Repeat(" ", level*2), name, attr.Value())
		} else if elem.IsMethod() {
			fmt.Printf("%s%s\n", strings.Repeat(" ", level*2), name)
			fmt.Printf("  %sParams: %s\n", strings.Repeat(" ", level*2), goToJson(elem.Tree().(map[string]interface{})["params"].(map[string]interface{})["schema"]))
		}
	}
}
