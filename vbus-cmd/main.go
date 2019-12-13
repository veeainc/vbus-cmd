/*
Test program for interacting with audiomanager from go and playing audio
Can be run like so:
./discotheque -am 10.0.14.169:21403 -b 15 -media /Users/lucas/workspace/go/src/github.com/Max2Inc/SimpleAudio/media/201500.wav -z test
or to see help options
./discotheque -h
*/

package main

import (

	//	"bytes"

	"fmt"
	"io/ioutil"
	"strconv"

	//"io/ioutil"
	"log"
	//	"net/http"
	"os"
	"time"

	//"bitbucket.org/vbus/vbus.go"
	"github.com/Jeffail/gabs"

	"github.com/spf13/cobra"
)

//var closer chan os.Signal
var serviceName string
var maintain bool
var timeout int
var vpath string
var value string
var vtype string

func Init() *Node {
	// new session
	veeabus, err := Open(serviceName)
	if err != nil {
		log.Fatalf("Can't connect to vbus server: %v\n", err)
	}

	//veeabus.Permission_Subscribe("system.>")
	//veeabus.Permission_Publish("system.>")

	time.Sleep(2 * time.Second)

	_, err = os.Stat(serviceName + ".db")
	if err == nil {
		file, _ := ioutil.ReadFile(serviceName + ".db")
		localConfig, _ := gabs.ParseJSON(file)
		log.Printf("tree already existing:")
		log.Printf(localConfig.StringIndent("", " "))
		log.Printf("////////////////////////////////////////")
		veeabus.Node(string(file))
	}
	return veeabus
}

func Close(veeabus *Node) {
	time.Sleep(time.Second)

	ioutil.WriteFile(serviceName+".db", []byte(veeabus.Tree()), 0666)

	if maintain == true {
		log.Printf("loop forever")
		for {
			time.Sleep(time.Second)
		}
	}
	veeabus.Close()
}

func ConvertAttribute(value string, t string) (interface{}, error) {
	var val interface{}
	var err error
	switch t {
	default:
		log.Printf("type not supported")
	case "boolean":
		val, err = strconv.ParseBool(value)
	case "integer":
		val, err = strconv.ParseInt(value, 10, 32)
	case "string":
		val = value
	case "number":
		val, err = strconv.ParseFloat(value, 32)
	}

	return val, err
}

func PrintAttribute(value interface{}, t string) {
	switch t {
	default:
		log.Printf("Error in get attribute: type not supported\n")
	case "boolean":
		log.Printf(strconv.FormatBool(value.(bool)))
	case "integer":
		log.Printf(strconv.Itoa(value.(int)))
	case "string":
		log.Printf(value.(string))
	case "number":
		log.Printf(strconv.FormatFloat(float64(value.(float32)), 'b', -1, 32))
	}
}

func main() {

	//parse flag commands

	var cmdDiscover = &cobra.Command{
		Use:   "discover",
		Short: "return tree of the path discovered",
		//Long: `print is for printing anything back to the screen. For many years people have printed back to the screen.`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("Discover: " + vpath)
			veeabus := Init()
			tmpNode, err := veeabus.Discover(vpath, "", time.Duration(timeout)*time.Second)
			if err != nil {
				log.Fatalf("Error: %v\n", err)
			}
			tree := tmpNode.Tree()
			log.Printf(tree)
			Close(veeabus)
		},
	}
	cmdDiscover.Flags().StringVarP(&vpath, "path", "p", "", "path to node")
	cmdDiscover.Flags().IntVarP(&timeout, "timeout", "o", 4, "time out (in second)")

	var cmdAddAttribute = &cobra.Command{
		Use:   "add",
		Short: "add an attribute in vBus tree",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("add attribute " + vpath + " , value: " + value + " (" + vtype + ")")
			veeabus := Init()
			val, err := ConvertAttribute(value, vtype)
			if err != nil {
				log.Printf(err.Error())
			} else {
				veeabus.AddAttribute(vpath, val)
			}
			Close(veeabus)
		},
	}
	cmdAddAttribute.Flags().StringVarP(&vpath, "path", "p", "", "path to attribute")
	cmdAddAttribute.Flags().StringVarP(&value, "value", "v", "nil", "attribute value")
	cmdAddAttribute.Flags().StringVarP(&vtype, "type", "t", "string", "attribute value type")

	var cmdAddNode = &cobra.Command{
		Use:   "add",
		Short: "add a node in vBus tree",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("add node " + vpath + " : " + value)
			veeabus := Init()
			err := veeabus.AddNode(vpath, value)
			if err != nil {
				log.Printf(err.Error())
			}
			Close(veeabus)
		},
	}
	cmdAddNode.Flags().StringVarP(&vpath, "path", "p", "", "path to node")
	cmdAddNode.Flags().StringVarP(&value, "node", "n", "nil", "node (string)")

	var cmdAddMethod = &cobra.Command{
		Use:   "add",
		Short: "add a method in vBus tree",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("add method " + vpath + " : " + value)
			veeabus := Init()
			_, err := veeabus.AddMethod(vpath, func(data []byte) []byte {
				fmt.Printf("Received a message: %s\n", string(data))
				return nil
			})
			if err != nil {
				log.Printf(err.Error())
			}
			Close(veeabus)
		},
	}
	cmdAddMethod.Flags().StringVarP(&vpath, "path", "p", "", "path to node")
	cmdAddMethod.Flags().StringVarP(&value, "node", "n", "nil", "node (string)")

	var cmdSetNode = &cobra.Command{
		Use:   "set",
		Short: "set a node in vBus tree",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("set node " + vpath + " : " + value)
			veeabus := Init()
			node, err := veeabus.Node(vpath)
			if err != nil {
				log.Printf(err.Error())
			} else {
				err = node.Set(value)
				if err != nil {
					log.Printf(err.Error())
				}
			}

			Close(veeabus)
		},
	}
	cmdSetNode.Flags().StringVarP(&vpath, "path", "p", "", "path to node")
	cmdSetNode.Flags().StringVarP(&value, "node", "n", "nil", "node (string)")

	var cmdSetAttribute = &cobra.Command{
		Use:   "set",
		Short: "set an attribute in vBus tree",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("set attribute " + vpath + " , value: " + value + " (" + vtype + ")")
			veeabus := Init()
			val, err := ConvertAttribute(value, vtype)
			if err != nil {
				log.Printf(err.Error())
			} else {
				attribute, err := veeabus.Attribute(vpath)
				if err != nil {
					log.Printf(err.Error())
				} else {
					err = attribute.Set(val)
					if err != nil {
						log.Printf(err.Error())
					}
				}
			}
			Close(veeabus)
		},
	}
	cmdSetAttribute.Flags().StringVarP(&vpath, "path", "p", "", "path to attribute")
	cmdSetAttribute.Flags().StringVarP(&value, "value", "v", "nil", "attribute value")
	cmdSetAttribute.Flags().StringVarP(&vtype, "type", "t", "string", "attribute value type")

	var cmdGetNode = &cobra.Command{
		Use:   "get",
		Short: "get a node in vBus tree",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("get node " + vpath)
			veeabus := Init()
			node, err := veeabus.Node(vpath)
			if err != nil {
				log.Printf(err.Error())
			} else {
				err = node.Get()
				if err != nil {
					log.Printf(err.Error())
				} else {
					log.Printf(node.Tree())
				}
			}

			Close(veeabus)
		},
	}
	cmdGetNode.Flags().StringVarP(&vpath, "path", "p", "", "path to node")

	var cmdGetAttribute = &cobra.Command{
		Use:   "get",
		Short: "get an attribute in vBus tree",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("get attribute " + vpath)
			veeabus := Init()
			att, err := veeabus.Attribute(vpath)
			if err != nil {
				log.Printf(err.Error())
			} else {
				_, err := att.Get()
				if err != nil {
					log.Printf(err.Error())
				} else {
					PrintAttribute(att.value, att.atype)
				}
			}

			Close(veeabus)
		},
	}
	cmdGetAttribute.Flags().StringVarP(&vpath, "path", "p", "", "path to attribute")

	var cmdAddNodeSub = &cobra.Command{
		Use:   "sub",
		Short: "subscribe to a node add",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("subscribe to " + vpath)
			veeabus := Init()
			node, err := veeabus.Node(vpath)
			if err != nil {
				log.Printf(err.Error())
			} else {
				err := node.SubscribeAdd(func(data string) string {
					fmt.Printf("Received a message: %s\n", data)
					return ""
				})
				if err != nil {
					log.Printf(err.Error())
				}
			}

			Close(veeabus)
		},
	}
	cmdAddNodeSub.Flags().StringVarP(&vpath, "path", "p", "", "path to node")

	var cmdGetNodeSub = &cobra.Command{
		Use:   "sub",
		Short: "subscribe to a node get",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("subscribe to " + vpath)
			veeabus := Init()
			node, err := veeabus.Node(vpath)
			if err != nil {
				log.Printf(err.Error())
			} else {
				err := node.SubscribeGet(func(data string) string {
					fmt.Printf("Received a message: %s\n", data)
					return ""
				})
				if err != nil {
					log.Printf(err.Error())
				}
			}

			Close(veeabus)
		},
	}
	cmdGetNodeSub.Flags().StringVarP(&vpath, "path", "p", "", "path to node")

	var cmdSetNodeSub = &cobra.Command{
		Use:   "sub",
		Short: "subscribe to a node set",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("subscribe to " + vpath)
			veeabus := Init()
			node, err := veeabus.Node(vpath)
			if err != nil {
				log.Printf(err.Error())
			} else {
				err := node.SubscribeSet(func(data string) string {
					fmt.Printf("Received a message: %s\n", data)
					return ""
				})
				if err != nil {
					log.Printf(err.Error())
				}
			}

			Close(veeabus)
		},
	}
	cmdSetNodeSub.Flags().StringVarP(&vpath, "path", "p", "", "path to node")

	var cmdGetAttSub = &cobra.Command{
		Use:   "sub",
		Short: "subscribe to a attribute get",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("subscribe to " + vpath)
			veeabus := Init()
			att, err := veeabus.Attribute(vpath)
			if err != nil {
				log.Printf(err.Error())
			} else {
				err := att.SubscribeGet(func(data interface{}) interface{} {
					fmt.Printf("Received a message: %s\n", data)
					return nil
				})
				if err != nil {
					log.Printf(err.Error())
				}
			}

			Close(veeabus)
		},
	}
	cmdGetAttSub.Flags().StringVarP(&vpath, "path", "p", "", "path to node")

	var cmdSetAttSub = &cobra.Command{
		Use:   "sub",
		Short: "subscribe to a attribute set",
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("subscribe to " + vpath)
			veeabus := Init()
			att, err := veeabus.Attribute(vpath)
			if err != nil {
				log.Printf(err.Error())
			} else {
				err := att.SubscribeSet(func(data interface{}) interface{} {
					fmt.Printf("Received a message: %s\n", data)
					PrintAttribute(att.value, att.atype)
					return ""
				})
				if err != nil {
					log.Printf(err.Error())
				}
			}

			Close(veeabus)
		},
	}
	cmdSetAttSub.Flags().StringVarP(&vpath, "path", "p", "", "path to node")

	/////////////////////////////////////////////////
	var rootCmd = &cobra.Command{Use: "vbus-cmd"}
	rootCmd.PersistentFlags().StringVarP(&serviceName, "user", "u", "system.vbus-cmd", "service name")
	rootCmd.PersistentFlags().BoolVarP(&maintain, "maintain", "m", false, "loop forever")

	var nodeCmd = &cobra.Command{Use: "node"}
	cmdAddNode.AddCommand(cmdAddNodeSub)
	cmdGetNode.AddCommand(cmdGetNodeSub)
	cmdSetNode.AddCommand(cmdSetNodeSub)
	nodeCmd.AddCommand(cmdAddNode, cmdSetNode, cmdGetNode)

	var attCmd = &cobra.Command{Use: "attribute"}
	cmdGetAttribute.AddCommand(cmdGetAttSub)
	cmdSetAttribute.AddCommand(cmdSetAttSub)
	attCmd.AddCommand(cmdAddAttribute, cmdGetAttribute, cmdSetAttribute)

	var methodCmd = &cobra.Command{Use: "method"}
	methodCmd.AddCommand(cmdAddMethod)

	rootCmd.AddCommand(cmdDiscover, nodeCmd, attCmd, methodCmd)
	rootCmd.Execute()

}
