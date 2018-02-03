package main

import (
	"html/template"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/websocket"
)

func remote(sts Settings) {
	srv := &http.Server{
		Addr: sts.rAddr.String(),
	}

	fs := http.FileServer(http.Dir("asset"))
	http.Handle("/static/", fs)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("asset/index.html"))
		tmpl.Execute(w, nil)
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
