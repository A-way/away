package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/websocket"
)

func remote(sts Settings) {
	srv := &http.Server{
		Addr: sts.rAddr.String(),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "A way to Away!")
	})
	http.Handle("/_a", websocket.Handler(secureHandler(sts.sec)))

	log.Println("Remote start on", srv.Addr)
	log.Fatal("Remote start failure:", srv.ListenAndServe())
}

func secureHandler(sec *Security) func(*websocket.Conn) {
	return func(ws *websocket.Conn) {
		wss := sec.secure(ws)
		defer wss.Close()

		addr, err := ReadAddr(wss, "tcp")
		if err != nil {
			log.Println("Addr read failure:", err)
			return
		}

		// Relay to target
		tc, err := net.Dial(addr.Network(), addr.String())
		if err != nil {
			log.Println("Target dial failure:", err)
			return
		}
		defer tc.Close()

		tc.(*net.TCPConn).SetKeepAlive(true)
		if nout, nin, err := relay(tc, wss); err != nil {
			log.Println("Relay target failure:", err)
			return
		} else {
			log.Printf("Away: %s ~ %s <%d %d>", wss.RemoteAddr(), addr.String(), nin, nout)
		}
	}
}
