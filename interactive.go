package main

import (
	"bitbucket.org/veeafr/utils.go/types"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	vBus "bitbucket.org/vbus/vbus.go"
	"github.com/c-bata/go-prompt"
	gocache "github.com/patrickmn/go-cache"
)

var cache *gocache.Cache // cache used to store vBus element
var writer = NewAdvWriter()
var hubIpAddress string
var hubSerial string
var vbusConn *vBus.Client

func init() {
	cache = gocache.New(20*time.Second, 1*time.Minute)
}

// Get interactive shell vBus connection.
func getInteractiveConnection() (*vBus.Client, error) {
	if vbusConn != nil {
		return vbusConn, nil
	}

	writer.WriteLog("Connecting to vBus, please wait...")
	if hubIpAddress != "" {
		_ = os.Setenv("VBUS_URL", "nats://"+hubIpAddress+":21400")
	}

	conn := vBus.NewClient(domain, appName)

	if hubSerial != "" {
		if err := conn.Connect(vBus.HubId(hubSerial)); err != nil {
			return nil, err
		}
	} else {
		if err := conn.Connect(); err != nil {
			return nil, err
		}
	}
	vbusConn = conn

	if conf, err := conn.GetConfig(); err == nil && conf != nil {
		writer.WriteSuccess("Connected to " + conf.Vbus.Hostname + " on " + conf.Vbus.Url)
	} else {
		writer.WriteSuccess("Connected !")
	}

	return vbusConn, nil
}

const (
	shortcutColor = prompt.Purple
	nodeColor     = prompt.Blue
	attrColor     = prompt.Green
	methColor     = prompt.Yellow
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Writer
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type AdvWriter struct {
	writer prompt.ConsoleWriter
}

func NewAdvWriter() *AdvWriter {
	return &AdvWriter{
		writer: prompt.NewStdoutWriter(),
	}
}

func (a *AdvWriter) Write(str string) {
	a.writer.WriteRawStr(str)
}

func (a *AdvWriter) WriteBold(str string) {
	a.WriteColorBold(str, prompt.DefaultColor)
}

func (a *AdvWriter) WriteSecondary(str string) {
	a.WriteColor(str, prompt.DefaultColor)
}

func (a *AdvWriter) WriteLn(str string) {
	a.writer.WriteRawStr(str + "\n")
}

func (a *AdvWriter) WriteColorBold(str string, fg prompt.Color) {
	a.writer.SetColor(fg, prompt.DefaultColor, true)
	a.writer.WriteRawStr(str)
	a.writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
}

func (a *AdvWriter) WriteColor(str string, fg prompt.Color) {
	a.writer.SetColor(fg, prompt.DefaultColor, false)
	a.writer.WriteRawStr(str)
	a.writer.SetColor(prompt.DefaultColor, prompt.DefaultColor, false)
}

// Print a note message
func (a *AdvWriter) WriteNote(msg string) {
	a.Write("Note: ")
	a.WriteColor(msg+"\n", prompt.Yellow)
	a.Flush()
}

func (a *AdvWriter) WriteBanner() {
	a.WriteBold("Welcome to ")
	a.WriteColorBold("vBus-Cmd ", prompt.Cyan)
	a.WriteBold("interactive ")
	a.WriteBold("shell")
	a.WriteBold(" | Powered by ")
	a.WriteColorBold("Veea\n", prompt.DarkRed)

	// table
	a.Write("-------------------------------------------------------\n")

	// Exit memo
	a.Write("-   ")
	a.WriteSecondary("Press ")
	a.WriteColorBold("Ctrl+C", shortcutColor)
	a.WriteSecondary(" to exit")
	a.Write("                              -\n")

	// return memo
	a.Write("-   ")
	a.WriteSecondary("Press ")
	a.WriteColorBold("Ctrl+D", shortcutColor)
	a.WriteSecondary(" to go back")
	a.Write("                           -\n")

	// Navigate history
	a.Write("-   ")
	a.WriteSecondary("Navigate command history with ")
	a.WriteColorBold("Up", shortcutColor)
	a.WriteSecondary(" or ")
	a.WriteColorBold("Down", shortcutColor)
	a.WriteSecondary(" arrow")
	a.Write("    -\n")

	// Build path instruction
	a.Write("-   ")
	a.WriteSecondary("Build vBus path using ")
	a.WriteColorBold("Shift", shortcutColor)
	a.WriteSecondary(" and ")
	a.WriteColorBold("'.'", shortcutColor)
	a.WriteSecondary("")
	a.Write("               -\n")
	a.Write("-------------------------------------------------------\n")

	a.Flush()
}

func (a *AdvWriter) WriteLog(msg string) {
	a.WriteColor(msg+"\n", prompt.DefaultColor)
	a.Flush()
}

// Print an error
func (a *AdvWriter) WriteError(err error) {
	a.Write("\nError: ")
	a.WriteColor(err.Error()+"\n", prompt.Red)
	a.Flush()
}

func (a *AdvWriter) WriteSuccess(msg string) {
	a.WriteColor(msg+"\n", prompt.DarkGreen)
	a.Flush()
}

func (a *AdvWriter) Flush() {
	_ = a.writer.Flush()
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Create a simple text completer
func simpleCompleter(suggest []prompt.Suggest) prompt.Completer {
	return func(d prompt.Document) []prompt.Suggest {
		return prompt.FilterHasPrefix(suggest, d.GetWordBeforeCursor(), true)
	}
}

// Default input prompt options
func getCommonOptions(opt ...prompt.Option) []prompt.Option {
	opt = append(opt, prompt.OptionDescriptionBGColor(prompt.DarkGray),
		prompt.OptionDescriptionTextColor(prompt.White),
		prompt.OptionSelectedDescriptionTextColor(prompt.White),
		prompt.OptionSelectedDescriptionBGColor(prompt.LightGray),
		prompt.OptionSuggestionBGColor(prompt.DarkGray),
		prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
		prompt.OptionSelectedSuggestionTextColor(prompt.Black),
		prompt.OptionInputTextColor(prompt.Blue),
		prompt.OptionPreviewSuggestionTextColor(prompt.DarkBlue),
		prompt.OptionSelectedDescriptionBGColor(prompt.DarkGray),
		prompt.OptionSuggestionTextColor(prompt.White),
		prompt.OptionPrefix(">>> "),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn:  exit,
		}))
	return opt
}

func promptInput(completer prompt.Completer, opt ...prompt.Option) string {
	options := getCommonOptions(opt...)
	return prompt.Input(">>> ", completer, options...)
}

func startInteractiveDiscover() {
	conn, err := getInteractiveConnection()
	if err != nil {
		writer.WriteError(err)
		return
	}

	writer.WriteLog("Searching running modules...")
	writer.WriteLog("Ctrl+D to go back")
	modules, err := conn.DiscoverModules(1 * time.Second)
	if err != nil {
		writer.WriteError(err)
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
					writer.WriteError(err)
				} else {
					elem = e
					cache.Set(strings.Join(parts, "."), e, gocache.DefaultExpiration)
				}
			}

			if elem != nil {
				currentElement = elem
				if elem.IsNode() {
					for name := range elem.AsNode().Elements() {
						suggests = append(suggests, prompt.Suggest{Text: name})
					}
				}
			}
		}

		// sort suggest to always return same result
		sort.SliceStable(suggests, func(i, j int) bool {
			return strings.Compare(suggests[i].Text, suggests[j].Text) < 0
		})

		return prompt.FilterHasPrefix(suggests, d.GetWordBeforeCursorUntilSeparator("."), true)
	}

	executor := func(s string) {
		switch s {
		default: // vBus path
			discoverEnterLevel(conn, s)
		}
	}

	rootPrompt := prompt.New(executor, completer, getCommonOptions(prompt.OptionCompletionWordSeparator("."))...)
	rootPrompt.Run()
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

func printPathType(elem *vBus.UnknownProxy) {
	if elem.IsNode() {
		writer.WriteColorBold(elem.GetPath(), nodeColor)
		writer.WriteBold(" [node]\n")
	} else if elem.IsMethod() {
		writer.WriteColorBold(elem.GetPath(), methColor)
		writer.WriteBold(" [method]\n")
	} else {
		writer.WriteColorBold(elem.GetPath(), attrColor)
		writer.WriteBold(" [attribute]\n")
	}
	writer.Flush()
}

func printLocation(elem *vBus.UnknownProxy) {
	printPathType(elem)
	if elem.IsNode() {
		nodeCount, attrCount, methCount := countNodeElements(elem.AsNode())
		writer.WriteSecondary("Contains ")
		writer.WriteColorBold(fmt.Sprintf("%d ", nodeCount), nodeColor)
		writer.WriteSecondary("nodes, ")

		writer.WriteColorBold(fmt.Sprintf("%d ", attrCount), attrColor)
		writer.WriteSecondary("attributes, ")

		writer.WriteColorBold(fmt.Sprintf("%d ", methCount), methColor)
		writer.WriteSecondary("methods")
	} else if elem.IsMethod() {
		method := elem.AsMethod()
		writer.WriteSecondary("Input params: \n")
		printJsonSchema(method.ParamsSchema(), "    ")

		writer.WriteSecondary("Returns: \n")
		printJsonSchema(method.ReturnsSchema(), "    ")
	} else {
		attr := elem.AsAttribute()
		writer.WriteSecondary("Current value: ")
		writer.WriteColorBold(fmt.Sprintf("%v", attr.Value()), prompt.DarkGreen)
		writer.Write(" ")
		printJsonSchema(attr.Schema(), "")
	}
	writer.Write("\n")
	writer.Flush()
}

func printJsonSchema(schema vBus.JsonObj, prefix string) {
	if types.HasKey(schema, "type") && types.HasKey(schema, "items") {
		if types.GetKey(schema, "type") == "array" {
			s := types.GetKey(schema, "items")
			if s == nil {
				writer.Write(prefix)
				writer.WriteBold("[null]\n")
				return
			}

			items := s.(JsonArray)
			for _, item := range items {
				if types.HasKey(item, "title") {
					writer.WriteColor(prefix+types.GetKey(item, "title").(string)+" ", prompt.DarkGreen)
				} else {
					writer.Write(prefix)
				}

				if types.HasKey(item, "type") {
					writer.WriteBold("[" + types.GetKey(item, "type").(string) + "]")
				}

				if types.HasKey(item, "description") {
					writer.Write(" (" + types.GetKey(item, "description").(string) + ")")
				}

				writer.Write("\n")
			}

			return
		}
	} else if types.HasKey(schema, "type") {
		writer.Write(prefix)
		writer.WriteBold("[" + types.GetKey(schema, "type").(string) + "]")
		return
	}

	// fallback
	writer.WriteBold(goToJson(schema) + "\n")
}

func globalSubscribeAddReceiver(proxy *vBus.UnknownProxy, segments ...string) {
	writer.WriteBold("[Notification][add]: ")
	printPathType(proxy)
	writer.WriteSecondary("Received value: ")
	writer.WriteSuccess(goToJson(proxy.Tree()))
	writer.Write("\n")
}

func globalSubscribeDelReceiver(proxy *vBus.UnknownProxy, segments ...string) {
	writer.WriteBold("[Notification][sel]: ")
	printPathType(proxy)
	writer.WriteSecondary("Received value: ")
	writer.WriteSuccess(goToJson(proxy.Tree()))
	writer.Write("\n")
}

func globalSubscribeSetReceiver(proxy *vBus.UnknownProxy, segments ...string) {
	writer.WriteBold("[Notification][set]: ")
	printPathType(proxy)
	writer.WriteSecondary("Received value: ")
	writer.WriteSuccess(goToJson(proxy.Tree()))
	writer.Write("\n")
}

func navigateNode(conn *vBus.Client, node *vBus.NodeProxy) {
	for {
		elements := node.Elements()
		var suggests []prompt.Suggest

		suggests = append(suggests, prompt.Suggest{Text: "list", Description: "List first level elements"})
		suggests = append(suggests, prompt.Suggest{Text: "subscribe", Description: "Subscribe to node change"})
		suggests = append(suggests, prompt.Suggest{Text: "dump", Description: "Dump node content"})
		suggests = append(suggests, prompt.Suggest{Text: "back", Description: "Go back"})

		for name, elem := range elements {
			suggests = append(suggests, prompt.Suggest{Text: name, Description: getElementDescription(elem)})
		}

		fmt.Print("\n")
		i := promptInput(func(d prompt.Document) []prompt.Suggest {
			parts := strings.Split(strings.Trim(d.Text, " "), " ")
			wordBefore := parts[len(parts)-1]

			if wordBefore == "subscribe" {
				return prompt.FilterHasPrefix([]prompt.Suggest{
					{Text: "all", Description: "Receive all notifications"},
					{Text: "add", Description: "Receive 'add' notifications"},
					{Text: "del", Description: "Receive 'del' notifications"},
				}, d.GetWordBeforeCursor(), true)
			}

			return prompt.FilterHasPrefix(suggests, d.GetWordBeforeCursor(), true)
		})

		switch i {
		case "":
			return
		case "back":
			return
		case "dump":
			writer.WriteSuccess(string(goToPrettyColoredJson(node.Tree())))
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
			if strings.HasPrefix(i, "subscribe") {
				parts := strings.Split(i, " ")
				if len(parts) < 2 {
					writer.WriteError(errors.New("missing notification type"))
				} else {
					doAdd := func() {
						err := node.SubscribeAdd(globalSubscribeAddReceiver)
						if err != nil {
							writer.WriteError(err)
						} else {
							writer.WriteSuccess("Listening 'add' notifications")
						}
					}

					doDel := func() {
						err := node.SubscribeDel(globalSubscribeDelReceiver)
						if err != nil {
							writer.WriteError(err)
						} else {
							writer.WriteSuccess("Listening 'del' notifications")
						}
					}

					switch parts[1] {
					case "add":
						doAdd()
					case "del":
						doDel()
					case "all":
						doAdd()
						doDel()
					}
				}
			} else {
				if elem, ok := elements[i]; ok {
					navigateElement(conn, elem)
				}
			}
		}
	}
}

func navigateAttribute(conn *vBus.Client, attr *vBus.AttributeProxy) {
	for {
		fmt.Print("\n")
		i := promptInput(func(d prompt.Document) []prompt.Suggest {
			parts := strings.Split(strings.Trim(d.Text, " "), " ")
			wordBefore := parts[len(parts)-1]

			if wordBefore == "subscribe" {
				return prompt.FilterHasPrefix([]prompt.Suggest{
					{Text: "set", Description: "Receive 'set' notification"},
				}, d.GetWordBeforeCursor(), true)
			}

			return prompt.FilterHasPrefix([]prompt.Suggest{
				{Text: "back", Description: "Go back"},
				{Text: "get", Description: "Get attribute value"},
				{Text: "set", Description: "Set attribute value"},
				{Text: "subscribe", Description: "Subscribe to"},
			}, d.GetWordBeforeCursor(), true)
		})

		switch i {
		case "":
			return
		case "back":
			return
		case "get":
			if val, err := attr.ReadValue(); err != nil {
				writer.WriteError(err)
			} else {
				writer.WriteSuccess(fmt.Sprintf("%v", val))
			}
		default:
			if strings.HasPrefix(i, "subscribe") {
				parts := strings.Split(i, " ")
				if len(parts) < 2 {
					writer.WriteError(errors.New("missing notification type"))
				} else {
					switch parts[1] {
					case "set":
						err := attr.SubscribeSet(globalSubscribeSetReceiver)
						if err != nil {
							writer.WriteError(err)
						} else {
							writer.WriteSuccess("Listening 'set' notifications")
						}
					}
				}
			} else if strings.HasPrefix(i, "set") {
				parts := strings.Split(i, " ")
				if len(parts) < 2 {
					writer.WriteError(errors.New("missing attribute value"))
				}
				valueStr := strings.Join(parts[1:], " ")
				value, err := jsonToGoErr(valueStr)
				if err != nil {
					writer.WriteError(err)
					continue
				}
				err = attr.SetValue(value)
				if err != nil {
					writer.WriteError(err)
				}
			}
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
			parts := strings.Split(strings.Trim(d.Text, " "), " ")
			wordBefore := parts[len(parts)-1]

			if wordBefore == "" {
				return prompt.FilterHasPrefix([]prompt.Suggest{
					{Text: "call", Description: "Followed by method params (Json)"},
					{Text: "back", Description: "Go back"},
				}, d.GetWordBeforeCursor(), true)
			}

			if wordBefore == "call" {
				return []prompt.Suggest{
					{Text: "-t", Description: "Timeout in seconds"},
				}
			}

			if wordBefore == "-t" {
				return []prompt.Suggest{
					{Text: "5"},
					{Text: "10"},
					{Text: "60"},
				}
			}

			return []prompt.Suggest{}
		}, prompt.OptionCompletionWordSeparator(" "))

		switch i {
		case "":
			return
		case "back":
			return
		default:
			parts := strings.Split(i, " ")
			if parts[0] == "call" {
				timeout := 1 * time.Second
				argOffset := 1

				if len(parts) > 1 && parts[1] == "-t" {
					t, err := strconv.ParseInt(parts[2], 10, 0)
					if err != nil {
						writer.WriteError(err)
						continue
					} else {
						timeout = time.Duration(t) * time.Second
					}
					argOffset = 3
				}

				paramsStr := strings.Join(parts[argOffset:], " ")

				if strings.Trim(paramsStr, " ") == "" {
					if val, err := method.CallWithTimeout(timeout); err != nil {
						writer.WriteError(err)
					} else {
						writer.WriteSuccess("Return value: " + goToJson(val))
					}
					continue
				}

				args, err := jsonToGoErr(paramsStr)
				if err != nil {
					writer.WriteError(err)
					continue
				}

				if _, ok := args.([]interface{}); !ok {
					// try to wrap args as a json array
					args, err = jsonToGoErr("[" + paramsStr + "]")
					if err != nil {
						writer.WriteError(err)
						continue
					}
				}
				if casted, ok := args.([]interface{}); !ok {
					writer.WriteError(errors.New("method args must be passed as a json array"))
				} else {
					if val, err := method.CallWithTimeout(timeout, casted...); err != nil {
						writer.WriteError(err)
					} else {
						writer.WriteSuccess(goToJson(val))
					}
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
		writer.WriteError(err)
	} else {
		// navigate element
		navigateElement(conn, elem)
	}
}

type Exit int

func exit(_ *prompt.Buffer) {
	panic(Exit(0))
}

func handleExit() {
	switch v := recover().(type) {
	case nil:
		return
	case Exit:
		os.Exit(int(v))
	default:
		fmt.Println(v)
		fmt.Println(string(debug.Stack()))
	}
}

func promptConnectionParams() {
	writer.Write("Enter hub")
	writer.WriteColor(" ip address", prompt.Yellow)
	writer.WriteLn(":")
	writer.Flush()
	hubIpAddress = promptInput(simpleCompleter([]prompt.Suggest{}))
	if hubIpAddress == "" {
		return
	}

	writer.Write("Enter hub")
	writer.WriteColor(" serial number ", prompt.Yellow)
	writer.WriteLn("(this is needed by the permission system):")
	writer.Flush()
	hubSerial = promptInput(simpleCompleter([]prompt.Suggest{}))
	if hubSerial == "" {
		return
	}

	_, err := getInteractiveConnection()
	if err != nil {
		writer.WriteError(err)
	}
}

func promptPermission() {
	conn, err := getInteractiveConnection()
	if err != nil {
		writer.WriteError(err)
	}

	writer.WriteLn("Enter permission string:")
	writer.Flush()
	permission := promptInput(simpleCompleter([]prompt.Suggest{}))
	if permission == "" {
		return
	}

	ok, err := conn.AskPermission(permission)
	if err != nil {
		writer.WriteError(err)
	}
	writer.WriteSuccess(fmt.Sprintf("%v", ok))
}

func promptMainActions() {
	for {
		i := promptInput(simpleCompleter([]prompt.Suggest{
			{Text: "introspect", Description: "Introspect vBus tree"},
			{Text: "connect", Description: "Connect to remote Hub"},
			{Text: "permission", Description: "Ask a permission"},
			{Text: "back", Description: "Go back"},
		}))

		switch i {
		case "":
		case "back":
			return
		case "introspect":
			startInteractiveDiscover()
		case "permission":
			promptPermission()
		case "connect":
			promptConnectionParams()
		}
	}
}

func startInteractivePrompt() {
	defer handleExit()
	writer.WriteBanner()
	promptMainActions()
}
