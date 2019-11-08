package proxy

import (
	"net"
	"sync"
	"time"
)

const maxUDPSize = 512

func (p Proxy) serveUDP(l net.PacketConn) error {
	bpool := sync.Pool{
		New: func() interface{} {
			b := make([]byte, maxUDPSize)
			return &b
		},
	}

	for {
		buf := *bpool.Get().(*[]byte)
		qsize, addr, err := l.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				bpool.Put(&buf)
				continue
			}
			return err
		}
		if qsize <= 14 {
			bpool.Put(&buf)
			continue
		}
		start := time.Now()
		go func() {
			var err error
			var rsize int
			ip := addrIP(addr)
			q, err := newQuery(buf[:qsize], "udp", ip)
			if err != nil {
				p.logErr(err)
			}
			defer func() {
				bpool.Put(&buf)
				p.logQuery(QueryInfo{
					Query:        q,
					ResponseSize: rsize,
					Duration:     time.Since(start),
				})
				p.logErr(err)
			}()
			if rsize, err = p.Upstream.Resolve(q, buf); err != nil {
				return
			}
			_, err = l.WriteTo(buf[:rsize], addr)
		}()
	}
}
