package main

import (
	"flag"
	"saturncloud/proxy-server/proxy"
)

func main() {
	settingsFile := flag.String("f", "/etc/saturn/settings.yaml", "Settings YAML file path")
	proxy.Run(*settingsFile)
}
