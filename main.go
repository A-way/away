package main

import "os"

func main() {

	setting := loadSetting()
	if len(os.Args) > 1 && os.Args[1] == "dev" {
		go socks(setting)
		remote(setting)
	} else if os.Getenv(setting.portname) != "" {
		remote(setting)
	} else {
		socks(setting)
	}

}
