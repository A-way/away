// +build !lib

package main

import (
	"flag"

	log "github.com/sirupsen/logrus"
)

func main() {

	rp := flag.String("rp", "", "Remote Port. eg: -rp 8080")
	lp := flag.String("lp", "", "Local Port. eg: -lp 1080")
	pk := flag.String("pk", "AwayPasskey", "Passkey to do crypto. eg: -pk \"Away Passkey\"")
	ru := flag.String("ru", "", "Remote Url to connect. eg: -ru http://away.remote")
	rf := flag.String("rf", "", "Rules File use to initilize rules. eg: /path/rules")
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: "2006/01/02 15:04:05.000"})

	if *rp != "" && *lp != "" {
		go startSocks(*lp, *pk, *rp, *ru, *rf)
		startRemote(*rp, *pk)
	} else if *rp != "" {
		startRemote(*rp, *pk)
	} else {
		startSocks(*lp, *pk, *rp, *ru, *rf)
	}

}

func defaultVal(origin, value string) string {
	if origin != "" {
		return origin
	}
	return value
}

func startRemote(rp, pk string) {
	s := &Settings{
		Passkey: pk,
		Port:    rp,
	}
	Remote(s)
}

func startSocks(lp, pk, rp, ru, rf string) {
	s := &Settings{
		Remote:  defaultVal(ru, "http://localhost:"+rp),
		Passkey: pk,
		Port:    lp,
	}

	away := NewAway(ModeAway, rf)
	if rf != "" {
		n, err := away.LoadRules()
		if err != nil {
			log.Error(err)
		} else {
			log.Infof("Initilize [%d] rules.", n)
		}
		if n > 0 {
			away.ChangeMode(ModeRule)
		}
	}

	srv, err := NewSocksSrv(s, away)
	if err != nil {
		log.Fatal(err)
	}
	srv.Start()
}
