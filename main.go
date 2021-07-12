package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/veeainc/utils.go/system"
	vBus "github.com/veeainc/vbus.go"
)

// default module name, can be overrided with option
var domain = "cmd"
var appName = "new"
var creds = ""
var jwt = ""
var wait = false
var loop = false
var deleteConfigFile = false
var logR = logrus.New()

type lf = logrus.Fields // alias

func removeConfig() {
	if deleteConfigFile == true {
		vbusPath := os.Getenv("VBUS_PATH")
		if vbusPath == "" {
			vbusPath = path.Join(os.Getenv("HOME"), "vbus")
		}
		os.Remove(path.Join(vbusPath, domain+"."+appName+".conf"))
	}
}

func printMsg(m *nats.Msg) {
	logR.WithFields(lf{
		"subject": m.Subject,
		"data":    string(m.Data),
		"reply":   m.Reply,
	}).Info("vBus Message")
}

func main() {
	var vbusConn *vBus.Client
	var emptyPermission []string

	logR.SetFormatter(&logrus.TextFormatter{})

	// get vBus connection instance
	getConn := func(permission []string) *vBus.Client {
		if vbusConn == nil {
			if jwt != "" {
				vbusConn = getConnection("", "", jwt, permission, wait)
			} else if creds == "" {
				vbusConn = getConnection(domain, appName, "", permission, wait)
			} else {
				vbusConn = getConnection(creds, "", "", permission, wait)
			}
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
			"\n   vbus-cmd -p \"system.foobar.>\" attribute get system.foobar.local.config.service_ip" +
			"\n   vbus-cmd --wait --domain=mydomain --app=myapp expose --name=redis --protocol=redis --port=6379",
		Description: "This command line tool allow you to run vBus commands. When running for the first time, a configuration\n" +
			"   file will be created in $HOME or $VBUS_PATH env. variable. So you need to have write access to this folder.\n" +
			"\nENV. VARIABLES:" +
			"\n   VBUS_PATH: the config path used to store the config file (optional)" +
			"\n   VBUS_URL: direct nats server url (optional)",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "debug", Aliases: []string{"d"}, Value: false, Usage: "Show vBus library logs"},
			&cli.BoolFlag{Name: "wait", Aliases: []string{"w"}, Value: false, Destination: &wait, Usage: "Wait for vBus connection"},
			&cli.BoolFlag{Name: "loop", Aliases: []string{"l"}, Value: false, Destination: &loop, Usage: "Loop until is successful"},
			&cli.BoolFlag{Name: "interactive", Aliases: []string{"i"}, Value: false, Usage: "Start an interactive prompt"},
			&cli.StringSliceFlag{Name: "permission", Aliases: []string{"p"}, Usage: "Ask a permission before running the command"},
			&cli.StringFlag{Name: "domain", Usage: "Change domain name", Value: domain, Destination: &domain},
			&cli.StringFlag{Name: "app", Usage: "Change app name", Value: appName, Destination: &appName},
			&cli.StringFlag{Name: "creds", Aliases: []string{"c"}, Usage: "Provide Credentials file (domain.name.creds)", Destination: &creds},
			&cli.StringFlag{Name: "jwt", Aliases: []string{"j"}, Usage: "Provide JWT seed", Destination: &jwt},
		},
		Before: func(c *cli.Context) error {
			// debug mode
			if c.Bool("debug") {
				vBus.SetLogLevel(logrus.DebugLevel)
			} else {
				vBus.SetLogLevel(logrus.FatalLevel)
			}

			if appName == "new" && creds == "" {
				// create a random app name in case no credentials nor name are been provided
				randomValue := rand.Intn(99999999)
				appName = strconv.Itoa(randomValue)
				deleteConfigFile = true
			}

			if c.Bool("interactive") {
				if vbusConn != nil {
					vbusConn.Close()
				}
				startInteractivePrompt()

				os.Exit(0)
			} else {
				getConn(c.StringSlice("permission"))
			}
			return nil
		},
		After: func(c *cli.Context) error {
			if vbusConn != nil {
				vbusConn.Close()
			}
			removeConfig()
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

					conn := getConn([]string{c.Args().Get(0)})
					if conn == nil {
						return errors.New("no vBus connection")
					}
					if elem, err := conn.Discover(c.Args().Get(0), 2*time.Second); err != nil {
						return err
					} else {
						if c.Bool("flatten") {
							dumpElementFlattened(elem)
						} else if c.Bool("list") {
							dumpElement(elem)
						} else {
							dumpElementToColoredJson(elem)
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
						Flags: []cli.Flag{
							&cli.BoolFlag{Name: "json", Aliases: []string{"j"}, Usage: "Display output as a simplified json (no method, no json-schema)"},
						},
						ArgsUsage: "PATH",
						Action: func(c *cli.Context) error {
							if c.Args().Len() != 1 {
								return errors.New("'get' expect exactly one PATH argument")
							}

							conn := getConn(emptyPermission)
							if conn == nil {
								return errors.New("no vBus connection")
							}

							node := getNode(c.Args().Get(0), conn)
							if node == nil {
								return errors.New("Node not available")
							}

							if c.Bool("json") {
								fmt.Println(goToPrettyColoredJson(node.AsNode().Json()))
							} else {
								dumpElementToColoredJson(node)
							}

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
									log.Print(err.Error())
									return err
								}
								input = string(buf)
							} else {
								input = strings.Join(c.Args().Slice()[1:], "")
							}

							// validate uuid
							if strings.Contains(uuid, ".") {
								log.Print("Not a valid node uuid: " + uuid)
								return errors.New("Not a valid node uuid")
							}

							// validate tree
							tree := jsonToGo(input)
							if tree == nil {
								return errors.New("json not valid")
							}

							// create vBus raw node
							rawNode := jsonObjToRawDef(tree)
							if rawNode == nil {
								return errors.New("raw node not valid")
							}

							conn := getConn(emptyPermission)
							if conn == nil {
								return errors.New("no vBus connection")
							}
							_, err := conn.AddNode(uuid, rawNode)
							if err != nil {
								log.Print(err.Error())
								return err
							}

							log.Println("node successfully created, do not close this app (exit with Ctrl+C)")

							system.WaitForCtrlC()
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
							conn := getConn(emptyPermission)
							if conn == nil {
								return errors.New("no vBus connection")
							}
							attr := getAttribute(c.Args().Get(0), conn)
							if attr == nil {
								return errors.New("attribute not available")
							}
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
							conn := getConn(emptyPermission)
							if conn == nil {
								return errors.New("no vBus connection")
							}
							attr := getAttribute(c.Args().Get(0), conn)
							if attr == nil {
								return errors.New("attribute not available")
							}
							if val, err := attr.ReadValueWithTimeout(time.Duration(c.Int("timeout")) * time.Second); err != nil {
								return err
							} else {
								fmt.Println(goToJson(val))
								return nil
							}
						},
					},
					{
						Name:  "sub",
						Usage: "Subscribe `ATTR` value",
						Flags: []cli.Flag{
							&cli.IntFlag{Name: "timeout", Aliases: []string{"t"}, Value: 1},
						},
						Action: func(c *cli.Context) error {
							conn := getConn(emptyPermission)
							attr := getAttribute(c.Args().Get(0), conn)
							attr.SubscribeSet(func(node *vBus.UnknownProxy, segment ...string) {
								fmt.Println(jsonToGo(node.String()))
							})
							log.Println("subscribe started (exit with Ctrl+C)")
							system.WaitForCtrlC()
							return nil
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
							conn := getConn(emptyPermission)
							if conn == nil {
								return errors.New("no vBus connection")
							}
							attr := getMethod(c.Args().Get(0), conn)
							if attr == nil {
								return errors.New("method not available")
							}
							args := jsonToGo(c.Args().Get(1))

							if _, ok := args.([]interface{}); !ok {
								// try to wrap args as a json array
								args = jsonToGo("[" + c.Args().Get(1) + "]")
							}
							if casted, ok := args.([]interface{}); !ok {
								return errors.New("method args must be passed as a json array")
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
					conn := getConn(emptyPermission)
					if conn == nil {
						return errors.New("no vBus connection")
					}
					if err := conn.Expose(c.String("name"), c.String("protocol"), c.Int("port"), c.String("path")); err != nil {
						return err
					}

					log.Println("exposing service, do not close this app (exit with Ctrl+C)")

					system.WaitForCtrlC()
					return nil
				},
			},
			{
				Name:    "spy",
				Aliases: []string{"s"},
				Usage:   "spy pub/sub messages",
				Action: func(c *cli.Context) error {
					// request full permission then close regular vBus connection
					if vbusConn != nil {
						vbusConn.Close()
						vbusConn = nil
					}
					conn := getConn([]string{">"})
					if conn == nil {
						return errors.New("no vBus connection")
					}
					conn.Close()

					// re-open the same connection but with direct nats access
					vbusPath := os.Getenv("VBUS_PATH")
					if vbusPath == "" {
						vbusPath = path.Join(os.Getenv("HOME"), "vbus")
					}
					confFile := path.Join(vbusPath, domain+"."+appName+".conf")
					file, _ := ioutil.ReadFile(confFile)
					clientConfig, _ := gabs.ParseJSON([]byte(file))

					client, err := nats.Connect(clientConfig.Search("vbus", "url").Data().(string), nats.UserInfo(clientConfig.Search("client", "user").Data().(string), clientConfig.Search("key", "private").Data().(string)))
					if err != nil {
						return err
					}
					defer client.Close()

					// Subscribe to everything
					if _, err := client.Subscribe(">", func(m *nats.Msg) {
						printMsg(m)
					}); err != nil {
						logR.WithFields(lf{
							"error": err.Error(),
						}).Error("cannot subscribe spi")
						return err
					}

					log.Println("spi started (exit with Ctrl+C)")
					system.WaitForCtrlC()
					return nil
				},
			},
			{
				Name:  "info",
				Usage: "Get vBus information",
				Subcommands: []*cli.Command{
					{
						Name:    "address",
						Aliases: []string{"a"},
						Usage:   "get the IP address of your service",
						Action: func(c *cli.Context) error {
							conn := getConn(emptyPermission)
							if conn == nil {
								return errors.New("no vBus connection")
							}
							IPaddress, err := conn.GetNetworkIP()

							if err != nil {
								logR.WithFields(lf{
									"error": err.Error(),
								}).Error("cannot get network IP")
								return err
							}

							fmt.Println(IPaddress)
							return nil
						},
					},
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
	err := errors.New("fake")
	for err != nil {
		err = app.Run(os.Args)
		if err != nil {
			if loop == true {
				log.Print(err)
				log.Print("loop until success ....")
			} else {
				log.Fatal(err)
			}
		}
	}

	if vbusConn != nil {
		vbusConn.Close()
	}
}
