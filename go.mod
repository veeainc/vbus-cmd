module github.com/veeainc/vbus-cmd

go 1.13

require (
	github.com/Jeffail/gabs v1.4.0
	github.com/c-bata/go-prompt v0.2.3
	github.com/jeremywohl/flatten v1.0.1
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mattn/go-tty v0.0.3 // indirect; ib    ndirect
	github.com/nats-io/nats.go v1.9.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/pkg/term v0.0.0-20200520122047-c3ffed290a03 // indirect
	github.com/sirupsen/logrus v1.5.0
	github.com/tidwall/pretty v1.0.2
	github.com/urfave/cli/v2 v2.2.0
	github.com/veeainc/utils.go v1.3.3
	github.com/veeainc/vbus.go v1.5.1
)

replace github.com/veeainc/vbus.go => ../vbus.go