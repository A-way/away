package main

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"io"
	"net"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/net/websocket"
)

type SecConn struct {
	net.Conn
	sec *Security
	buf *bytes.Buffer
}

func (c *SecConn) Read(b []byte) (n int, err error) {
	nrm := c.buf.Len()
	ntr := len(b)
	if nrm >= ntr {
		return c.buf.Read(b)
	} else {
		buf := new(bytes.Buffer)
		zrd, err := zlib.NewReader(c.Conn)
		if err != nil {
			if nrm > 0 {
				return c.buf.Read(b)
			}
			if err == io.ErrUnexpectedEOF {
				return 0, io.EOF
			}
			return 0, err
		}
		if _, err := buf.ReadFrom(zrd); err != nil {
			if nrm > 0 {
				return c.buf.Read(b)
			}
			return 0, err
		}
		ptx, err := c.sec.aead.Open(nil, c.sec.nonce, buf.Bytes(), nil)
		if err != nil {
			return 0, err
		}
		if _, err := c.buf.Write(ptx); err != nil {
			return 0, err
		}
		return c.buf.Read(b)
	}

}

func (c *SecConn) Write(b []byte) (n int, err error) {
	ctx := c.sec.aead.Seal(nil, c.sec.nonce, b, nil)
	buf := new(bytes.Buffer)
	zwt := zlib.NewWriter(buf)
	if n, err = zwt.Write(ctx); err != nil {
		return
	}
	zwt.Close()
	if n, err = c.Conn.Write(buf.Bytes()); err != nil {
		return
	}
	n = len(b)
	return
}

func (c *SecConn) RemoteAddr() net.Addr {
	if ws, ok := c.Conn.(*websocket.Conn); ok && ws.IsServerConn() {
		req := ws.Request()
		fwd := req.Header.Get("x-forwarded-for")
		if fwd == "" {
			return c.Conn.RemoteAddr()
		}
		ip := net.ParseIP(fwd)
		return &net.TCPAddr{IP: ip}
	}
	return c.Conn.RemoteAddr()
}

type Security struct {
	aead  cipher.AEAD
	nonce []byte
}

func NewSecurity(passkey string) (*Security, error) {
	key := pbkdf2.Key([]byte(passkey), []byte("away&salt"), 4096, 16, sha256.New)
	bl, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(bl)
	if err != nil {
		return nil, err
	}
	nonce := pbkdf2.Key(key, []byte("away&nonce"), 4096, 12, sha256.New)
	return &Security{aesgcm, nonce}, nil
}

func (sec *Security) secure(c net.Conn) (sc *SecConn) {
	sc = &SecConn{c, sec, new(bytes.Buffer)}
	return sc
}
