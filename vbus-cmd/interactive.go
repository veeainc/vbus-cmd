package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	vBus "bitbucket.org/vbus/vbus.go"
	"github.com/c-bata/go-prompt"
	gocache "github.com/patrickmn/go-cache"
)

var cache *gocache.Cache

func init() {
	cache = gocache.New(40*time.Second, 1*time.Minute)
}

func printBanner() {
	writer := prompt.NewStdoutWriter()
	writer.WriteRawStr("Welcome to ")
	writer.SetColor(prompt.Cyan, prompt.DefaultColor, true)
	writer.WriteRawStr("vBus-Cmd ")
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("interactive ")
	writer.SetColor(prompt.Yellow, prompt.DefaultColor, true)
	writer.WriteRawStr("shell")
	writer.SetColor(prompt.Blue, prompt.DefaultColor, true)
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr(" | Powered by ")
	writer.SetColor(prompt.DarkRed, prompt.DefaultColor, false)
	writer.WriteRawStr("Veea")
	writer.WriteRawStr("\n")

	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("-------------------------------------------------------\n")

	shortcutColor := prompt.Purple
	// Exit memo
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("-   ")
	writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
	writer.WriteRawStr("Press ")
	writer.SetColor(shortcutColor, prompt.DefaultColor, true)
	writer.WriteRawStr("Ctrl+C")
	writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
	writer.WriteRawStr(" or ")
	writer.SetColor(shortcutColor, prompt.DefaultColor, true)
	writer.WriteRawStr("Ctrl+D")
	writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
	writer.WriteRawStr(" to exit")
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("                    -\n")

	// Navigate history
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("-   ")
	writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
	writer.WriteRawStr("Navigate command history with ")
	writer.SetColor(shortcutColor, prompt.DefaultColor, true)
	writer.WriteRawStr("Up")
	writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
	writer.WriteRawStr(" or ")
	writer.SetColor(shortcutColor, prompt.DefaultColor, true)
	writer.WriteRawStr("Down")
	writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
	writer.WriteRawStr(" arrow")
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("    -\n")

	// Build path instruction
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("-   ")
	writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
	writer.WriteRawStr("Build vBus path using ")
	writer.SetColor(shortcutColor, prompt.DefaultColor, true)
	writer.WriteRawStr("Shift")
	writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
	writer.WriteRawStr(" and ")
	writer.SetColor(shortcutColor, prompt.DefaultColor, true)
	writer.WriteRawStr("'.'")
	writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
	writer.WriteRawStr("")
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("               -\n")
	writer.WriteRawStr("-------------------------------------------------------\n")

	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("\n")
	_ = writer.Flush()
}

// Print a note message
func printNote(msg string) {
	writer := prompt.NewStdoutWriter()
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("Note: ")
	writer.SetColor(prompt.Yellow, prompt.DefaultColor, false)
	writer.WriteRawStr(msg)
	writer.WriteRawStr("\n")
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	_ = writer.Flush()
}

func printLog(msg string) {
	writer := prompt.NewStdoutWriter()
	writer.SetColor(prompt.DarkGray, prompt.DefaultColor, false)
	writer.WriteRawStr(msg)
	writer.WriteRawStr("\n")
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	_ = writer.Flush()
}

// Print an error
func printError(err error) {
	writer := prompt.NewStdoutWriter()
	writer.WriteRawStr("\n")
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	writer.WriteRawStr("Error: ")
	writer.SetColor(prompt.Red, prompt.DefaultColor, false)
	writer.WriteRawStr(err.Error())
	writer.WriteRawStr("\n")
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	_ = writer.Flush()
}

func printSuccess(msg string) {
	writer := prompt.NewStdoutWriter()
	writer.SetColor(prompt.Green, prompt.DefaultColor, false)
	writer.WriteRawStr(msg)
	writer.WriteRawStr("\n")
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	_ = writer.Flush()
}

// Create a simple text completer
func simpleCompleter(suggest []prompt.Suggest) prompt.Completer {
	return func(d prompt.Document) []prompt.Suggest {
		return prompt.FilterHasPrefix(suggest, d.GetWordBeforeCursor(), true)
	}
}

func getCommonOptions(opt ...prompt.Option) []prompt.Option {
	opt = append(opt, prompt.OptionDescriptionBGColor(prompt.DarkGray),
		prompt.OptionDescriptionTextColor(prompt.DefaultColor),
		prompt.OptionDescriptionTextColor(prompt.Black),
		prompt.OptionSelectedDescriptionTextColor(prompt.Black),
		prompt.OptionSelectedDescriptionBGColor(prompt.LightGray),
		prompt.OptionSuggestionBGColor(prompt.DarkGray),
		prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
		prompt.OptionSelectedDescriptionBGColor(prompt.DarkGray),
		prompt.OptionSuggestionTextColor(prompt.Black),
		prompt.OptionPrefix(">>> "),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn: func(buffer *prompt.Buffer) {
				os.Exit(0)
			},
		}, prompt.KeyBind{
			Key: prompt.ControlD,
			Fn: func(buffer *prompt.Buffer) {
				os.Exit(0)
			},
		}))
	return opt
}

func promptInput(completer prompt.Completer, opt ...prompt.Option) string {
	options := getCommonOptions(opt...)
	return prompt.Input(">>> ", completer, options...)
}

func suggestsContains(suggests []prompt.Suggest, suggest string) bool {
	for _, a := range suggests {
		if a.Text == suggest {
			return true
		}
	}
	return false
}

func startInteractiveDiscover(conn *vBus.Client) {
	//moduleSuggests := []prompt.Suggest{
	//	{Text: "back"},
	//}
	//defaultLen := len(suggests)

	printLog("Searching running modules...")
	modules, err := conn.DiscoverModules(1 * time.Second)
	if err != nil {
		printError(err)
		return
	}

	var currentElement *vBus.UnknownProxy

	completer := func(d prompt.Document) []prompt.Suggest {
		path := strings.Trim(d.TextBeforeCursor(), " ")
		parts := strings.Split(path, ".")

		var suggests []prompt.Suggest

		// empty path, we retrieve module list
		if len(parts) == 0 || path == "" {
			for _, mod := range modules {
				suggests = append(suggests, prompt.Suggest{Text: mod.Id})
			}
		}

		// module id level
		if len(parts) == 2 || len(parts) == 3 {
			currentElement = nil // reset
			id := strings.Join(parts[:2], ".")

			for _, mod := range modules {
				if mod.Id == id {
					suggests = append(suggests, prompt.Suggest{Text: mod.Hostname})
				}
			}
		}

		// full module level
		if len(parts) > 3 {
			parts := parts[:len(parts)-1]
			var elem *vBus.UnknownProxy

			if e, ok := cache.Get(strings.Join(parts, ".")); ok {
				elem = e.(*vBus.UnknownProxy)
			} else {
				if e, err := conn.GetRemoteElement(parts...); err != nil {
					printError(err)
				} else {
					elem = e
					cache.Set(strings.Join(parts, "."), e, gocache.DefaultExpiration)
				}
			}

			if elem != nil {
				currentElement = elem
				if elem.IsNode() {
					for name, _ := range elem.AsNode().Elements() {
						suggests = append(suggests, prompt.Suggest{Text: name})
					}
				}
			}
		} /*else {
			if currentElement.IsNode() {
				elemParts := parts[3:]

				elem, err := currentElement.AsNode().GetElement(elemParts...)
				if err != nil {
					printError(err)
				} else {
					if elem.IsNode() {
						for name, _ := range elem.AsNode().Elements() {
							suggests = append(suggests, prompt.Suggest{Text: name})
						}
					}
				}
			}
		}*/

		/*if strings.HasSuffix(d.GetWordBeforeCursor(), ".") {
			if len(parts) == 3 { // module level

			}
		}*/

		// sort suggest to always return same result
		sort.SliceStable(suggests, func(i, j int) bool {
			return strings.Compare(suggests[i].Text, suggests[j].Text) < 0
		})

		return prompt.FilterHasPrefix(suggests, d.GetWordBeforeCursorUntilSeparator("."), true)
	}
	executor := func(s string) {
		switch s {
		case "back":
			return
		default: // vBus path
			discoverEnterLevel(conn, s)
		}
	}

	rootPrompt := prompt.New(executor, completer, getCommonOptions(prompt.OptionCompletionWordSeparator("."))...)

	for {
		rootPrompt.Run()
	}
}

func getElementDescription(elem *vBus.UnknownProxy) string {
	if elem.IsNode() {
		return "Node"
	} else if elem.IsMethod() {
		return "Method"
	} else {
		return "Attribute"
	}
}

func countNodeElements(node *vBus.NodeProxy) (nodeCount int, attrCount int, methCount int) {
	for _, elem := range node.Elements() {
		if elem.IsNode() {
			nodeCount++
		} else if elem.IsMethod() {
			methCount++
		} else {
			attrCount++
		}
	}
	return
}

func printLocation(elem *vBus.UnknownProxy) {
	writer := prompt.NewStdoutWriter()
	if elem.IsNode() {
		writer.SetColor(prompt.Blue, prompt.DefaultColor, true)
		writer.WriteRawStr(elem.GetPath())
		writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, true)
		writer.WriteRawStr(" [node]\n")

		nodeCount, attrCount, methCount := countNodeElements(elem.AsNode())
		writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
		writer.WriteRawStr("Contains ")
		writer.SetColor(prompt.Blue, prompt.DefaultColor, true)
		writer.WriteRawStr(fmt.Sprintf("%d ", nodeCount))
		writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
		writer.WriteRawStr("nodes, ")

		writer.SetColor(prompt.Green, prompt.DefaultColor, true)
		writer.WriteRawStr(fmt.Sprintf("%d ", attrCount))
		writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
		writer.WriteRawStr("attributes, ")

		writer.SetColor(prompt.Yellow, prompt.DefaultColor, true)
		writer.WriteRawStr(fmt.Sprintf("%d ", methCount))
		writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
		writer.WriteRawStr("methods")
	} else if elem.IsMethod() {
		writer.SetColor(prompt.Yellow, prompt.DefaultColor, true)
		writer.WriteRawStr(elem.GetPath())
		writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, true)
		writer.WriteRawStr(" [method]\n")

		method := elem.AsMethod()

		writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
		writer.WriteRawStr("Input params: \n")
		printJsonSchema(method.ParamsSchema(), writer, "    ")

		writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
		writer.WriteRawStr("Returns: \n")
		printJsonSchema(method.ReturnsSchema(), writer, "    ")
	} else {
		writer.SetColor(prompt.Green, prompt.DefaultColor, true)
		writer.WriteRawStr(elem.GetPath())
		writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, true)
		writer.WriteRawStr(" [attribute]\n")

		attr := elem.AsAttribute()
		writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
		writer.WriteRawStr("Current value: ")
		writer.SetColor(prompt.Green, prompt.DefaultColor, true)
		writer.WriteRawStr(fmt.Sprintf("%v", attr.Value()))
		writer.WriteRawStr(" ")
		printJsonSchema(attr.Schema(), writer, "")
	}
	writer.WriteRawStr("\n")
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
	_ = writer.Flush()
}

func printJsonSchema(schema vBus.JsonObj, writer prompt.ConsoleWriter, prefix string) {
	if hasKey(schema, "type") && hasKey(schema, "items") {
		if getKey(schema, "type") == "array" {
			items := getKey(schema, "items").(JsonArray)
			for _, item := range items {
				if hasKey(item, "title") {
					writer.SetColor(prompt.DarkGreen, prompt.DefaultColor, false)
					writer.WriteRawStr(prefix + getKey(item, "title").(string) + " ")
				} else {
					writer.WriteRawStr(prefix)
				}

				if hasKey(item, "type") {
					writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, true)
					writer.WriteRawStr("[" + getKey(item, "type").(string) + "]")
				}

				if hasKey(item, "description") {
					writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
					writer.WriteRawStr(" (" + getKey(item, "description").(string) + ")")
				}

				writer.WriteRawStr("\n")
			}

			return
		}
	} else if hasKey(schema, "type") {
		writer.WriteRawStr(prefix)
		writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, true)
		writer.WriteRawStr("[" + getKey(schema, "type").(string) + "]")
		return
	}

	// fallback
	writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, true)
	writer.WriteRawStr(goToJson(schema) + "\n")
}

func navigateNode(conn *vBus.Client, node *vBus.NodeProxy) {
	for {
		elements := node.Elements()
		var suggests []prompt.Suggest

		suggests = append(suggests, prompt.Suggest{Text: "list", Description: "List first level elements"})
		suggests = append(suggests, prompt.Suggest{Text: "dump", Description: "Dump node content"})
		suggests = append(suggests, prompt.Suggest{Text: "back", Description: "Go back"})

		for name, elem := range elements {
			suggests = append(suggests, prompt.Suggest{Text: name, Description: getElementDescription(elem)})
		}

		fmt.Print("\n")
		i := promptInput(simpleCompleter(suggests))

		switch i {
		case "back":
			return
		case "dump":
			printSuccess(goToPrettyJson(node.Tree()))
		case "list":
			writer := prompt.NewStdoutWriter()
			nodes := node.Nodes()
			attributes := node.Attributes()
			methods := node.Methods()

			if len(nodes) > 0 {
				writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
				writer.WriteRawStr("Nodes: \n")

				for _, n := range nodes {
					writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
					writer.WriteRawStr("    - ")
					writer.SetColor(prompt.Blue, prompt.DefaultColor, true)
					writer.WriteRawStr(n.GetName() + "\n")
				}

				writer.WriteRawStr("\n")
			}

			if len(attributes) > 0 {
				writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
				writer.WriteRawStr("Attributes: \n")

				for _, a := range attributes {
					writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
					writer.WriteRawStr("    - ")
					writer.SetColor(prompt.Green, prompt.DefaultColor, true)
					writer.WriteRawStr(a.GetName() + "\n")
				}

				writer.WriteRawStr("\n")
			}

			if len(methods) > 0 {
				writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
				writer.WriteRawStr("Methods: \n")

				for _, m := range methods {
					writer.SetColor(prompt.LightGray, prompt.DefaultColor, false)
					writer.WriteRawStr("    - ")
					writer.SetColor(prompt.Yellow, prompt.DefaultColor, true)
					writer.WriteRawStr(m.GetName() + "\n")
				}

				writer.WriteRawStr("\n")
			}

			_ = writer.Flush()
		default:
			navigateElement(conn, elements[i])
		}
	}
}

func navigateAttribute(conn *vBus.Client, attr *vBus.AttributeProxy) {
	for {
		fmt.Print("\n")
		i := promptInput(simpleCompleter([]prompt.Suggest{
			{Text: "get", Description: "Get attribute value"},
			{Text: "set", Description: "Set attribute value"},
			{Text: "back", Description: "Go back"},
		}))

		switch i {
		case "back":
			return
		case "get":
			if val, err := attr.ReadValue(); err != nil {
				printError(err)
			} else {
				printSuccess(fmt.Sprintf("%v", val))
			}
		case "set":
			return
		}
	}
}

func jsonToGoErr(arg string) (interface{}, error) {
	b := []byte(arg)
	var m interface{}
	err := json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func navigateMethod(conn *vBus.Client, method *vBus.MethodProxy) {
	for {
		fmt.Print("\n")
		i := promptInput(func(d prompt.Document) []prompt.Suggest {
			if d.Text == "" {
				return prompt.FilterHasPrefix([]prompt.Suggest{
					{Text: "call", Description: "Followed by method params (Json)"},
					{Text: "back", Description: "Go back"},
				}, d.GetWordBeforeCursor(), true)
			}

			return []prompt.Suggest{}
		})

		switch i {
		case "back":
			return
		default:
			paramsStr := strings.TrimPrefix(i, "call")

			if strings.Trim(paramsStr, " ") == "" {
				if val, err := method.Call(); err != nil {
					printError(err)
				} else {
					printSuccess(goToJson(val))
				}
			}

			args, err := jsonToGoErr(paramsStr)
			if err != nil {
				printError(err)
				continue
			}

			if _, ok := args.([]interface{}); !ok {
				// try to wrap args as a json array
				args, err = jsonToGoErr("[" + paramsStr + "]")
				if err != nil {
					printError(err)
					continue
				}
			}
			if casted, ok := args.([]interface{}); !ok {
				printError(errors.New("method args must be passed as a json array"))
			} else {
				if val, err := method.Call(casted...); err != nil {
					printError(err)
				} else {
					printSuccess(goToJson(val))
				}
			}
		}
	}
}

// navigate a vBus element with autocomplete
func navigateElement(conn *vBus.Client, elem *vBus.UnknownProxy) {
	printLocation(elem)

	if elem.IsNode() {
		navigateNode(conn, elem.AsNode())
	} else if elem.IsMethod() {
		navigateMethod(conn, elem.AsMethod())
	} else {
		navigateAttribute(conn, elem.AsAttribute())
	}
}

func discoverEnterLevel(conn *vBus.Client, path string) {
	if elem, err := conn.GetRemoteElement(strings.Split(path, ".")...); err != nil {
		printError(err)
	} else {
		// navigate element
		navigateElement(conn, elem)
	}
}

func startCommandSession(conn *vBus.Client) {
	/*for {
		i := promptInput(simpleCompleter([]prompt.Suggest{
			{Text: "discover", Description: "Discover vBus elements"},
			{Text: "permission", Description: "Ask permission on vBus"},
			{Text: "attribute", Description: "Manage a remote attribute"},
			{Text: "method", Description: "Call a remote method"},
			{Text: "debug", Description: "Enable vBus library log"},
			{Text: "exit", Description: "Exit this shell"},
		}))

		switch i {
		case "discover":
			startInteractiveDiscover(conn)
		case "debug":
			printNote("You can also enable log when starting the shell with -d")
			vBus.SetLogLevel(logrus.TraceLevel)
			printSuccess("Debug logs enabled")
		case "exit":
			return
		}
	}*/
	startInteractiveDiscover(conn)
}

func startInteractivePrompt() {
	printBanner()
	fmt.Println("Connecting to vBus, please wait...")
	conn := getConnection()

	startCommandSession(conn)
}
