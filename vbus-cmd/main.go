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
	"flag"

	//"io/ioutil"
	"log"
	//	"net/http"
	"os"
	"time"

	"bitbucket.org/vbus/vbus.go"
)

//flag for address to serve content

var cmdPTR = flag.String("c", "publish", "command")
var attPTR = flag.String("a", "", "attribute")
var dataPTR = flag.String("d", "", "data")

var closer chan os.Signal

func main() {

	//parse flag commands
	flag.Parse()
	cmd := *cmdPTR
	att := *attPTR
	data := *dataPTR

	// new session
	veeabus, err := vbus.Open("vbus-test")
	if err != nil {
		log.Fatalf("Can't connect to vbus server: %v\n", err)
	}

	veeabus.Permission_Subscribe("system.>")
	veeabus.Permission_Publish("system.>")

	if cmd == "publish" {
		veeabus.Publish(att, []byte(data))
		log.Printf("Publish done\n")
	} else if cmd == "subscribe" {
		veeabus.Subscribe(att, "none", func(subject string, reply string, msg []byte) {
			log.Println(string(msg))
		})
		log.Printf("Subscribe done\n")
		for true {
			time.Sleep(time.Second)
		}
	} else if cmd == "request" {
		elementlist := veeabus.Request(att, []byte(data))
		log.Printf("Request done\n")
		log.Println(string(elementlist))
	} else if cmd == "list" {
		elementlist := veeabus.List([]byte(data))
		log.Printf("List done\n")
		log.Println(string(elementlist))
	} else {
		log.Fatalf("bad argument\n")
	}

}
