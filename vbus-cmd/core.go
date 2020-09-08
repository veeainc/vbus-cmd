package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/pretty"
	"log"
	"reflect"
	"strings"

	vBus "bitbucket.org/vbus/vbus.go"
	"github.com/jeremywohl/flatten"
)

// Get a new vBus connection.
func getConnection(domain, appName string) *vBus.Client {
	conn := vBus.NewClient(domain, appName)
	if err := conn.Connect(); err != nil {
		log.Fatal(err.Error())
	}
	return conn
}

// Replace `local` keyword by vBus hostname.
func sanitizePath(path string, conn *vBus.Client) string {
	return strings.Replace(path, ".local.", "."+conn.GetHostname()+".", 1)
}

// Get a remote attribute.
func getAttribute(path string, conn *vBus.Client) *vBus.AttributeProxy {
	path = sanitizePath(path, conn)
	if attr, err := conn.GetRemoteAttr(path); err != nil {
		log.Fatal(err.Error())
	} else {
		return attr
	}
	return nil
}

// Get a remote node.
func getNode(path string, conn *vBus.Client) *vBus.UnknownProxy {
	path = sanitizePath(path, conn)
	if attr, err := conn.GetRemoteElement(path); err != nil {
		log.Fatal(err.Error())
	} else {
		return attr
	}
	return nil
}

// Get a remote attribute.
func askPermission(path string, conn *vBus.Client) {
	if badSubject(path) {
		log.Fatal(errors.New("invalid vBus path: " + path))
	}
	if success, err := conn.AskPermission(path); err != nil {
		log.Fatal(err.Error())
	} else {
		if !success {
			log.Fatal("cannot get permission: ", path)
		}
	}
}

// Get a remote method.
func getMethod(path string, conn *vBus.Client) *vBus.MethodProxy {
	path = sanitizePath(path, conn)
	if meth, err := conn.GetRemoteMethod(path); err != nil {
		log.Fatal(err.Error())
	} else {
		return meth
	}
	return nil
}

func jsonToGo(arg string) interface{} {
	b := []byte(arg)
	var m interface{}
	err := json.Unmarshal(b, &m)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

func goToJson(val interface{}) string {
	if b, err := json.Marshal(val); err != nil {
		log.Fatal(err)
	} else {
		return string(b)
	}
	return ""
}

// return a colored json string
func goToPrettyJson(val interface{}) string {
	if b, err := json.MarshalIndent(val, "", "    "); err != nil {
		log.Fatal(err)
	} else {
		return string(pretty.Color(b, nil))
	}
	return ""
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

func dumpElement(elem *vBus.UnknownProxy) {
	if elem.IsNode() {
		traverseNode(elem.AsNode(), 0)
	}
}

func dumpElementJson(elem *vBus.UnknownProxy) {
	fmt.Println(goToPrettyJson(elem.Tree()))
}

func dumpElementFlattened(elem *vBus.UnknownProxy) {
	if casted, ok := elem.Tree().(map[string]interface{}); ok {
		flat, err := flatten.Flatten(casted, "", flatten.DotStyle)
		if err != nil {
			log.Fatal(err)
		}
		for k, v := range flat {
			fmt.Printf("%s %v\n", k, v)
		}
	}
}

func isMap(v interface{}) bool {
	return reflect.TypeOf(v).Kind() == reflect.Map
}

func jsonObjToRawDef(tree vBus.JsonAny) vBus.RawNode {
	if _, ok := tree.(vBus.JsonObj); !ok {
		log.Fatal("Not a valid Json object")
	}
	obj := tree.(vBus.JsonObj)

	if !vBus.IsNode(obj) {
		log.Fatal("Your root object must be a vBus node")
	}

	rawNode := vBus.RawNode{}
	for k, v := range obj {
		if isMap(v) {
			rawNode[k] = vBus.NewNodeDef(jsonObjToRawDef(v))
		} else if vBus.IsNode(v) {
			rawNode[k] = vBus.NewAttributeDef(k, v)
		} else {
			log.Fatal("Only attribute and node are supported")
		}
	}

	return rawNode
}
