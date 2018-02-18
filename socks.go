package main

import (
	"io"
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
)

const (
	atypIPv4       = 1
	atypDomainName = 3
	atypIPv6       = 4
)

const (
	cmdConnect = 1
	cmdBind    = 2
	cmdUdp     = 3
)

type Addr struct {
	net.Addr
	network string
	addr    []byte
}

func ReadAddr(r io.Reader, network string) (addr *Addr, err error) {
	buf := make([]byte, 256)
	if _, e := io.ReadFull(r, buf[:1]); e != nil {
		return nil, e
	}
	atyp := buf[0]

	var n int
	switch atyp {
	case atypDomainName:
		if _, e := io.ReadFull(r, buf[1:2]); e != nil {
			return nil, e
		}
		if _, e := io.ReadFull(r, buf[2:2+int(buf[1])+2]); e != nil {
			return nil, e
		}
		n = 2 + int(buf[1]) + 2
	case atypIPv4:
		if _, e := io.ReadFull(r, buf[1:1+net.IPv4len+2]); e != nil {
			return nil, e
		}
		n = 1 + net.IPv4len + 2
	case atypIPv6:
		if _, e := io.ReadFull(r, buf[1:1+net.IPv6len+2]); e != nil {
			return nil, e
		}
		n = 1 + net.IPv6len + 2
	}

	a := make([]byte, n)
	copy(a, buf[:n])
	addr = &Addr{network: network, addr: a}
	return addr, nil
}

func (a *Addr) Network() string {
	return a.network
}

func (a *Addr) String() string {
	buf := a.addr
	atyp := buf[0]
	var host string
	switch atyp {
	case atypDomainName:
		host = string(buf[2 : 2+int(buf[1])])
	case atypIPv4:
		host = net.IP(buf[1 : 1+net.IPv4len]).String()
	case atypIPv6:
		host = net.IP(buf[1 : 1+net.IPv6len]).String()
	}
	port := strconv.Itoa((int(buf[len(buf)-2]) << 8) | int(buf[len(buf)-1]))
	return net.JoinHostPort(host, port)
}

func socks(sts Settings) {

	addr := sts.lAddr
	l, err := net.Listen(addr.Network(), addr.String())
	if err != nil {
		log.Fatal("Away start failure: ", err)
	}
	defer l.Close()
	log.Infof("Away %s ~ %s", l.Addr(), sts.remote)

	for {
		oc, err := l.Accept()
		if err != nil {
			log.Warn("Accepting connection failure: ", err)
			continue
		}

		go func(oc net.Conn) {
			defer func() {
				oc.Close()
			}()

			oc.(*net.TCPConn).SetKeepAlive(true)

			buf := make([]byte, 300)

			// Method selection  https://tools.ietf.org/html/rfc1928
			// +----+----------+----------+
			// |VER | NMETHODS | METHODS  |
			// +----+----------+----------+
			// | 1  |    1     | 1 to 255 |
			// +----+----------+----------+

			if _, err := io.ReadFull(oc, buf[:2]); err != nil {
				return
			}
			ver := buf[0]
			if ver != 5 {
				return
			}

			nmethods := buf[1]
			if _, err := io.ReadFull(oc, buf[:nmethods]); err != nil {
				return
			}

			// +----+--------+
			// |VER | METHOD |
			// +----+--------+
			// | 1  |   1    |
			// +----+--------+

			if _, err := oc.Write([]byte{5, 0}); err != nil {
				return
			}

			// Requests
			// +----+-----+-------+------+----------+----------+
			// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
			// +----+-----+-------+------+----------+----------+
			// | 1  |  1  | X'00' |  1   | Variable |    2     |
			// +----+-----+-------+------+----------+----------+

			if _, err := io.ReadFull(oc, buf[:3]); err != nil {
				return
			}
			cmd := buf[1]

			addr, err := ReadAddr(oc, "tcp")
			if err != nil {
				log.Warn("Read addr failure: ", err)
				return
			}
			log.Infof("Away~ %s->%s", oc.RemoteAddr().String(), addr.String())

			// Replies
			// +----+-----+-------+------+----------+----------+
			// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
			// +----+-----+-------+------+----------+----------+
			// | 1  |  1  | X'00' |  1   | Variable |    2     |
			// +----+-----+-------+------+----------+----------+

			switch cmd {
			case cmdConnect:
				oc.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
			default:
				return
			}

			// Relay to remote
			ws, err := websocket.Dial(sts.remote, "", sts.origin)
			if err != nil {
				log.Warn("Remote dial failure: ", err)
				return
			}
			wss := sts.sec.secure(ws)
			defer wss.Close()

			if _, err := wss.Write(addr.addr); err != nil {
				log.Warn("Write addr failure: ", err)
				return
			}
			if nout, nin, err := relay(wss, oc); err != nil {
				log.Warn("Relay remote failure: ", err)
				return
			} else {
				log.Infof("Relay: %s <%d %d>", addr.String(), nin, nout)
			}
		}(oc)
	}
}

type signal struct {
	n int64
	e error
}

func relay(wf, rf net.Conn) (nout, nin int64, err error) {
	timeout := 30 * time.Second
	s := make(chan signal)
	go func() {
		nin, err = timeoutCopy(rf, wf, timeout)
		if e, ok := err.(net.Error); ok && e.Timeout() {
			err = nil
		}
		s <- signal{nin, err}
	}()
	nout, err = timeoutCopy(wf, rf, timeout)
	if e, ok := err.(net.Error); ok && e.Timeout() {
		err = nil
	}
	r := <-s

	if err == nil {
		err = r.e
	}
	return nout, r.n, err
}

func timeoutCopy(dst, src net.Conn, timeout time.Duration) (written int64, err error) {
	buf := make([]byte, 10*1024)
	for {
		src.SetDeadline(time.Now().Add(timeout))
		dst.SetDeadline(time.Now().Add(timeout))
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
