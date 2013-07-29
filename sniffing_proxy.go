package main

import (
	"bufio"
	"femebe"
	"io"
	"net"
)

type session struct {
	ingress func()
	egress  func()
}

func (s *session) start() {
	go s.ingress()
	go s.egress()
}

type ProxyPair struct {
	*femebe.MessageStream
	net.Conn
}

func NewSniffingProxySession(errch chan error,
	client *ProxyPair, server *ProxyPair,
	frontend, backend chan *femebe.Message) *session {
	mover := func(from, to *ProxyPair, msgStream chan *femebe.Message) func() {
		return func() {
			var err error

			defer func() {
				from.Close()
				to.Close()
				errch <- err
			}()

			var m femebe.Message

			for {
				err = from.Next(&m)
				if err != nil {
					return
				}

				// N.B.: the clone must be
				// instantiated in the inner loop,
				// since the msgStream side channel
				// may not be done with the previous
				// clone yet by the time we get around
				// to cloning the next message (if we
				// instantiate the clone outside the
				// loop, we scribble all over the
				// previous clone)
				var clone femebe.Message
				clone.InitFromMessage(&m)
				msgStream <- &clone

				err = to.Send(&m)
				if err != nil {
					return
				}

				if !from.HasNext() {
					err = to.Flush()
					if err != nil {
						return
					}
				}
			}
		}
	}

	return &session{
		ingress: mover(client, server, frontend),
		egress:  mover(server, client, backend),
	}
}

type bufWriteCon struct {
	io.ReadCloser
	femebe.Flusher
	io.Writer
}

func newBufWriteCon(c net.Conn) *bufWriteCon {
	bw := bufio.NewWriter(c)
	return &bufWriteCon{c, bw, bw}
}
