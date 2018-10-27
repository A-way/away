package main

import (
	"io"
	"net"
	"net/url"
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

const (
	repSucceeded       = 0
	repNotAllowed      = 2
	repCmdNotSupported = 7
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

func (a *Addr) Host() string {
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
	return host
}

func (a *Addr) String() string {
	buf := a.addr
	port := strconv.Itoa((int(buf[len(buf)-2]) << 8) | int(buf[len(buf)-1]))
	return net.JoinHostPort(a.Host(), port)
}

type SocksSrv struct {
	listener net.Listener
	away     *Away

	settings *Settings
	remote   string
	origin   string
	security *Security

	stop    chan struct{}
	stopped chan struct{}
}

func NewSocksSrv(s *Settings, a *Away) (*SocksSrv, error) {
	u, err := url.Parse(s.Remote)
	if err != nil {
		return nil, err
	}
	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}
	remote := scheme + "://" + u.Host + "/_a"
	origin := u.String()

	security, err := NewSecurity(s.Passkey)
	if err != nil {
		return nil, err
	}

	l, err := net.Listen("tcp", ":"+s.Port)
	if err != nil {
		return nil, err
	}

	srv := &SocksSrv{
		listener: l,
		away:     a,
		settings: s,
		remote:   remote,
		origin:   origin,
		security: security,
		stop:     make(chan struct{}),
		stopped:  make(chan struct{})}

	return srv, nil
}

func (s *SocksSrv) Stop() {
	go func() {
		close(s.stop)
		s.listener.Close()
	}()
	<-s.stopped
}

func (s *SocksSrv) Start() {
	l := s.listener

	log.Infof("Away %s %c %s", l.Addr(), s.away.Mode(), s.remote)

	for {
		oc, err := l.Accept()
		if err != nil {
			select {
			case <-s.stop:
				close(s.stopped)
				return
			default:
				log.Warn("Accepting connection failure: ", err)
				continue
			}
		}

		go func(oc net.Conn) {
			defer func() {
				oc.Close()
			}()

			keepAlive(oc)

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

			// Replies
			m := s.away.ResloveMode(addr)
			switch cmd {
			case cmdConnect:
				var rep byte = repSucceeded
				if m == ModeDrop {
					rep = repNotAllowed
				}
				reply(oc, rep)
			default:
				reply(oc, repCmdNotSupported)
				return
			}

			// Relay to remote
			s.route(oc, addr, m)
		}(oc)
	}
}

func reply(conn net.Conn, rep byte) (int, error) {
	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	return conn.Write([]byte{5, rep, 0, atypIPv4, 0, 0, 0, 0, 0, 0})
}

func (s *SocksSrv) route(conn net.Conn, addr *Addr, mode rune) {
	log.Infof("%c %s->%s", mode, conn.RemoteAddr().String(), addr.String())

	if mode == ModeDrop {
		conn.Close()
		log.Infof("%c %s", mode, addr.String())
		return
	}

	var ac net.Conn
	var err error
	if mode == ModeRule {
		timeout := 5 * time.Second
		ac, err = net.DialTimeout(addr.Network(), addr.String(), timeout)
		if e, ok := err.(net.Error); ok && e.Timeout() { // we choose to fall through to away mode
			ac, err = s.dialRemote(addr)
			mode = ModeAway
		}
	} else if mode == ModeAway {
		ac, err = s.dialRemote(addr)
	} else if mode == ModePass {
		ac, err = net.Dial(addr.Network(), addr.String())
	}
	if err != nil {
		log.Warnf("Dial %c %s failure: %s", mode, addr.String(), err)
		return
	}
	defer ac.Close()

	nout, nin, err := relay(ac, conn)
	if err != nil {
		log.Warn("Relay remote failure: ", err)
	}
	log.Infof("%c %s->%s <%d %d>", mode, conn.RemoteAddr().String(), addr.String(), nin, nout)
}

func (s *SocksSrv) dialRemote(addr *Addr) (net.Conn, error) {
	ws, err := websocket.Dial(s.remote, "", s.origin)
	if err != nil {
		return nil, err
	}
	ac := s.security.secure(ws)

	if _, err := ac.Write(addr.addr); err != nil {
		ac.Close()
		return nil, err
	}
	return ac, nil
}

type relayResult struct {
	n int64
	e error
}

func relay(wf, rf net.Conn) (nout, nin int64, err error) {
	timeout := 30 * time.Second
	res := make(chan relayResult)
	go func() {
		nin, err = timeoutCopy(rf, wf, timeout)
		if e, ok := err.(net.Error); ok && e.Timeout() {
			err = nil
		}
		res <- relayResult{nin, err}
	}()
	nout, err = timeoutCopy(wf, rf, timeout)
	if e, ok := err.(net.Error); ok && e.Timeout() {
		err = nil
	}
	r := <-res

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

func keepAlive(conn net.Conn) {
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetKeepAlive(true)
		tcp.SetKeepAlivePeriod(60 * time.Second)
	}
}
