package main

import (
	"log"
	"net"
	"net/url"

	toml "github.com/pelletier/go-toml"
)

type Setting struct {
	socksAddr net.TCPAddr
	remote    string
	origin    string
	portname  string
	sec       *Security
}

func loadSetting() *Setting {
	t, err := toml.LoadFile("setting.toml")
	if err != nil {
		log.Fatal("Load setting.toml failure:", err)
	}

	port := t.Get("local.port").(int64)
	socksAddr := net.TCPAddr{Port: int(port)}

	u, err := url.Parse(t.Get("local.remote").(string))
	if err != nil {
		log.Fatal("Parse local.remote failure", err)
	}
	origin := u.String()
	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}
	remote := scheme + "://" + u.Host + "/_a"

	portname := t.Get("remote.portname").(string)

	passkey := t.Get("security.passkey").(string)
	sec, err := NewSecurity(passkey)
	if err != nil {
		log.Fatal("Cipher init failure:", err)
	}

	s := &Setting{socksAddr, remote, origin, portname, sec}
	return s

}
