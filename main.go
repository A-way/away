package main

import (
	"flag"
	"log"
)

func main() {

	rp := flag.String("rp", "", "Remote Port. eg: -rp 8080")
	lp := flag.String("lp", "", "Local Port. eg: -lp 1080")
	pk := flag.String("pk", "", "Passkey to do crypto. eg: -pk \"Away Passkey\"")
	ru := flag.String("ru", "", "Remote Url to connect. eg: -ru http://away.remote")
	flag.Parse()

	setting, err := NewSettingWith(*rp, *lp, *pk, *ru)
	if err != nil {
		log.Fatal(err)
	}
	if *rp != "" && *lp != "" {
		go socks(setting)
		remote(setting)
	} else if *rp != "" {
		remote(setting)
	} else {
		socks(setting)
	}

}
