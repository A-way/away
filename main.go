package main

import (
	"flag"

	log "github.com/sirupsen/logrus"
)

func main() {

	rp := flag.String("rp", "", "Remote Port. eg: -rp 8080")
	lp := flag.String("lp", "", "Local Port. eg: -lp 1080")
	pk := flag.String("pk", "", "Passkey to do crypto. eg: -pk \"Away Passkey\"")
	ru := flag.String("ru", "", "Remote Url to connect. eg: -ru http://away.remote")
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: "2006/01/02 15:04:05.000"})

	settings, err := NewSettingsWith(*rp, *lp, *pk, *ru)
	if err != nil {
		log.Fatal(err)
	}
	if *rp != "" && *lp != "" {
		go socks(settings)
		remote(settings)
	} else if *rp != "" {
		remote(settings)
	} else {
		socks(settings)
	}

}
