package main

import (
	vBus "bitbucket.org/vbus/vbus.go"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

// default module name, can be overrided with option
var domain = "system"
var appName = "vbus-cmd"

func main() {
	var vbusConn *vBus.Client

	// get vBus connection instance
	getConn := func() *vBus.Client {
		if vbusConn == nil {
			vbusConn = getConnection(domain, appName)
		}
		return vbusConn
	}

	app := &cli.App{
		Name:  "vbus-cmd",
		Usage: "send vbus commands (" + version + ")",
		UsageText: "vbus-cmd [global options] command [command options] [arguments...]" +
			"\n\n   Examples:" +
			"\n   vbus-cmd discover system.zigbee" +
			"\n   vbus-cmd discover -j system.zigbee (json output)" +
			"\n   vbus-cmd discover -f system.zigbee (flattened output)" +
			"\n   vbus-cmd attribute get -t 10 system.zigbee.[...].1026.attributes.0" +
			"\n   vbus-cmd method call -t 120 system.zigbee.boolangery-ThinkPad-P1-Gen-2.controller.scan 120" +
			"\n   vbus-cmd --app=foobar node add config \"{\\\"service_ip\\\":\\\"192.168.1.88\\\"}\"" +
			"\n   vbus-cmd -p \"system.foobar.>\" attribute get system.foobar.local.config.service_ip",
		Description: "This command line tool allow you to run vBus commands. When running for the first time, a configuration" +
			" file will be created in $HOME or $VBUS_PATH env. variable. So you need to have write access to this folder.\n" +
			"\nENV. VARIABLES:" +
			"\n   VBUS_PATH: the config path used to store the config file (optional)" +
			"\n   VBUS_URL: direct nats server url (optional)",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "debug", Aliases: []string{"d"}, Value: false, Usage: "Show vBus library logs"},
			&cli.BoolFlag{Name: "interactive", Aliases: []string{"i"}, Value: false, Usage: "Start an interactive prompt"},
			&cli.StringSliceFlag{Name: "permission", Aliases: []string{"p"}, Usage: "Ask a permission before running the command"},
			&cli.StringFlag{Name: "domain", Usage: "Change domain name", Value: domain, Destination: &domain},
			&cli.StringFlag{Name: "app", Usage: "Change app name", Value: appName, Destination: &appName},
		},
		Before: func(c *cli.Context) error {
			// debug mode
			if c.Bool("debug") {
				vBus.SetLogLevel(logrus.DebugLevel)
			} else {
				vBus.SetLogLevel(logrus.FatalLevel)
			}

			for _, perm := range c.StringSlice("permission") {
				conn := getConn()
				askPermission(perm, conn)
			}

			if c.Bool("interactive") {
				if vbusConn != nil {
					vbusConn.Close()
				}
				startInteractivePrompt()
				os.Exit(0)
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
					&cli.BoolFlag{Name: "list", Aliases: []string{"l"}, Usage: "Display output as a key value list"},
				},
				ArgsUsage: "PATH",
				Action: func(c *cli.Context) error {
					if c.Args().Len() != 1 {
						return errors.New("'discover' exactly one PATH argument")
					}

					conn := getConn()
					askPermission(c.Args().Get(0), conn)
					if elem, err := conn.Discover(c.Args().Get(0), 2*time.Second); err != nil {
						return err
					} else {
						if c.Bool("flatten") {
							dumpElementFlattened(elem)
						} else if c.Bool("list") {
							dumpElement(elem)
						} else {
							dumpElementJson(elem)
						}

						return nil
					}
				},
			},
			{
				Name:    "node",
				Aliases: []string{"n"},
				Usage:   "Send a command on a remote node ",
				Subcommands: []*cli.Command{
					{
						Name:        "get",
						Aliases:     []string{"s"},
						Usage:       "Get node on `PATH`",
						Description: "PATH is a dot style vBus path",
						ArgsUsage:   "PATH",
						Action: func(c *cli.Context) error {
							if c.Args().Len() != 1 {
								return errors.New("'get' expect exactly one PATH argument")
							}

							conn := getConn()
							node := getNode(c.Args().Get(0), conn)
							dumpElementJson(node)
							return nil
						},
					}, {
						Name:        "add",
						Aliases:     []string{"s"},
						Usage:       "Add a node with UUID`",
						Description: "UUID is a vBus path segment, it will be appended to <domain>.<app>.local",
						ArgsUsage:   "UUID",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "Get input from a file"},
						},
						Action: func(c *cli.Context) error {
							// validate args
							if c.String("file") != "" {
								if c.Args().Len() < 1 {
									return errors.New("'add' expect an UUID")
								}
							} else {
								if c.Args().Len() < 2 {
									return errors.New("'add' expect an UUID and a Json value for the node")
								}
							}

							// get args
							uuid := c.Args().Get(0)
							input := ""
							if c.String("file") != "" {
								buf, err := ioutil.ReadFile(c.String("file"))
								if err != nil {
									log.Fatal(err.Error())
								}
								input = string(buf)
							} else {
								input = strings.Join(c.Args().Slice()[1:], "")
							}

							// validate uuid
							if strings.Contains(uuid, ".") {
								log.Fatal("Not a valid node uuid: " + uuid)
							}

							// validate tree
							tree := jsonToGo(input)

							// create vBus raw node
							rawNode := jsonObjToRawDef(tree)

							conn := getConn()
							_, err := conn.AddNode(uuid, rawNode)
							if err != nil {
								log.Fatal(err.Error())
							}

							log.Println("node successfully created, do not close this app (exit with Ctrl+C)")

							waitForCtrlC()
							return nil
						},
					},
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
						Description: "PATH is a dot style vBus path" +
							"\n	 VALUE is a Json value",
						ArgsUsage: "PATH VALUE",
						Action: func(c *cli.Context) error {
							conn := getConn()
							attr := getAttribute(c.Args().Get(0), conn)
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
							conn := getConn()
							attr := getAttribute(c.Args().Get(0), conn)
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
				Aliases: []string{"m"},
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
							conn := getConn()
							attr := getMethod(c.Args().Get(0), conn)
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
			{
				Name:    "expose",
				Aliases: []string{"e"},
				Usage:   "Expose a service URI",
				Description: "It will expose an URI constructed with values from options.\n" +
					"   Public Ip address is retrieved automatically.\n\n" +
					"   Generated URI will look like: <protocol>://<ip>:<port>/<path>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "Service name", Required: true},
					&cli.StringFlag{Name: "protocol", Aliases: []string{"p"}, Usage: "Protocol scheme (http, tcp, mqtt...)", Required: true},
					&cli.IntFlag{Name: "port", Aliases: []string{"o"}, Usage: "Port number", Required: true},
					&cli.StringFlag{Name: "path", Aliases: []string{"a"}, Usage: "Optional path appended to service uri", Value: ""},
				},
				Action: func(c *cli.Context) error {
					conn := getConn()
					if err := conn.Expose(c.String("name"), c.String("protocol"), c.Int("port"), c.String("path")); err != nil {
						return err
					}

					log.Println("exposing service, do not close this app (exit with Ctrl+C)")

					waitForCtrlC()
					return nil
				},
			},
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Display version number",
				Action: func(context *cli.Context) error {
					fmt.Println(version)
					return nil
				},
			},
		},
	}

	app.EnableBashCompletion = true
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
