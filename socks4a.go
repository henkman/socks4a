package socks4a

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type Server struct {
	ReadTimeout     time.Duration
	MaxUserIdLength uint
	MaxNameLength   uint
}

func (srv *Server) serve(c net.Conn) {
	var head [8]byte
	n, err := c.Read(head[:])
	if err != nil || n != len(head) || head[0] != 4 || head[1] != 1 {
		c.Write([]byte{0, 0x5b, 0, 0, 0, 0, 0, 0})
		c.Close()
		return
	}
	port := binary.BigEndian.Uint16(head[2:])
	ip := net.IP([]byte(head[4:8]))
	until := srv.MaxUserIdLength
	for {
		n, err = c.Read(head[0:1])
		if err != nil || n <= 0 || until == 0 {
			c.Write([]byte{0, 0x5b, 0, 0, 0, 0, 0, 0})
			c.Close()
			return
		}
		if head[0] == 0 {
			break
		}
		until--
	}
	var addr string
	if ip[0] == 0 && ip[1] == 0 && ip[2] == 0 && ip[3] != 0 {
		until = srv.MaxNameLength
		name := make([]byte, 0, 16)
		for {
			n, err = c.Read(head[0:1])
			if err != nil || n <= 0 || until == 0 {
				c.Write([]byte{0, 0x5b, 0, 0, 0, 0, 0, 0})
				c.Close()
				return
			}
			if head[0] == 0 {
				break
			} else {
				name = append(name, head[0])
			}
			until--
		}
		addr = fmt.Sprintf("%s:%d", name, port)
	} else {
		addr = fmt.Sprintf("%s:%d", ip, port)
	}
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		c.Write([]byte{0, 0x5b, 0, 0, 0, 0, 0, 0})
		c.Close()
		return
	}
	r, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		c.Write([]byte{0, 0x5b, 0, 0, 0, 0, 0, 0})
		c.Close()
		return
	}
	c.Write([]byte{0, 0x5a, 0, 0, 0, 0, 0, 0})
	remoteclosed := false
	clientclosed := false
	go func() {
		var buf [8 * 1024]byte
		for !clientclosed {
			r.SetReadDeadline(time.Now().Add(srv.ReadTimeout))
			n, err := r.Read(buf[:])
			if n > 0 {
				c.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		remoteclosed = true
		r.Close()
	}()
	var buf [8 * 1024]byte
	for !remoteclosed {
		c.SetReadDeadline(time.Now().Add(srv.ReadTimeout))
		n, err := c.Read(buf[:])
		if n > 0 {
			r.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	clientclosed = true
	c.Close()
}

func (srv *Server) ListenAndServe(laddr string) error {
	ln, err := net.Listen("tcp", laddr)
	if err != nil {
		return err
	}
	for {
		rw, err := ln.Accept()
		if err != nil {
			return err
		}
		go srv.serve(rw)
	}
}
