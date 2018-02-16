package main

import (
	"errors"
	"net"
	"net/url"
	"strconv"
)

type Settings struct {
	rAddr  net.TCPAddr
	lAddr  net.TCPAddr
	remote string
	origin string
	sec    *Security
}

func NewSettingsWith(rp, lp, pk, ru string) (s Settings, err error) {
	rp = defaultVal(rp, "8080")
	lp = defaultVal(lp, "1080")
	pk = defaultVal(pk, "AwayPasskey")
	ru = defaultVal(ru, "http://localhost:"+rp)

	var srp, slp int
	srp, err = strconv.Atoi(rp)
	if err != nil {
		err = errors.New("wrong -rp " + err.Error())
		return
	}
	ra := net.TCPAddr{Port: srp}
	slp, err = strconv.Atoi(lp)
	if err != nil {
		err = errors.New("wrong -lp " + err.Error())
		return
	}
	la := net.TCPAddr{Port: slp}

	var u *url.URL
	u, err = url.Parse(ru)
	scheme := "ws"
	ori := u.String()
	if u.Scheme == "https" {
		scheme = "wss"
	}
	rmo := scheme + "://" + u.Host + "/_a"

	var sec *Security
	sec, err = NewSecurity(pk)
	if err != nil {
		return
	}

	s = Settings{ra, la, rmo, ori, sec}
	return s, nil
}

func defaultVal(origin, value string) string {
	if origin != "" {
		return origin
	}
	return value
}
