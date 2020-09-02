package main

import (
	"encoding/json"
	"log"
	"strings"

	vBus "bitbucket.org/vbus/vbus.go"
)

// Get a new vBus connection.
func getConnection() *vBus.Client {
	conn := vBus.NewClient("system", "vbus-cmd")
	if err := conn.Connect(); err != nil {
		log.Fatal(err.Error())
	}
	return conn
}

// Replace `local` keyword by vBus hostname.
func sanitizePath(path string, conn *vBus.Client) string {
	return strings.Replace(path, ".local.", conn.GetHostname(), 1)
}

// Get a remote attribute.
func getAttribute(path string) *vBus.AttributeProxy {
	conn := getConnection()
	path = sanitizePath(path, conn)
	autoAskPermission(path, conn)
	if attr, err := conn.GetRemoteAttr(path); err != nil {
		log.Fatal(err.Error())
	} else {
		return attr
	}
	return nil
}

// Get a remote attribute.
func askPermission(path string, conn *vBus.Client) {
	if success, err := conn.AskPermission(path); err != nil {
		log.Fatal(err.Error())
	} else {
		if !success {
			log.Fatal("cannot get permission: ", path)
		}
	}
}

// Auto ask permission on the two first segments.
func autoAskPermission(path string, conn *vBus.Client) {
	parts := strings.Split(path, ".")
	permPath := strings.Join(parts[:2], ".")
	askPermission(permPath + ".>", conn)
}

// Get a remote method.
func getMethod(path string) *vBus.MethodProxy {
	conn := getConnection()
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

func goToPrettyJson(val interface{}) string {
	if b, err := json.MarshalIndent(val, "", "    "); err != nil {
		log.Fatal(err)
	} else {
		return string(b)
	}
	return ""
}
