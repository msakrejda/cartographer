package main

import (
	"github.com/uhoh-itsmaciek/femebe"
	"github.com/uhoh-itsmaciek/femebe/core"
	"github.com/uhoh-itsmaciek/femebe/proto"
)

type sniffingRouter struct {
	backendPid uint32
	secretKey  uint32
	fe         core.Stream
	be         core.Stream
	feBuf      core.Message
	beBuf      core.Message
	feCh       chan<- *core.Message
	beCh       chan<- *core.Message
}

// Make a new Router that copies all messages on the given streams to
// the provided channels
func NewSniffingRouter(fe, be core.Stream, feCh, beCh chan<- *core.Message) femebe.Router {
	return &sniffingRouter{
		backendPid: 0,
		secretKey:  0,
		fe:         fe,
		be:         be,
		feCh:       feCh,
		beCh:       beCh,
	}
}

func (s *sniffingRouter) BackendKeyData() (uint32, uint32) {
	return s.backendPid, s.secretKey
}

func (s *sniffingRouter) RouteFrontend() (err error) {
	// route the next message from frontend to backend,
	// blocking and flushing if necessary
	err = s.fe.Next(&s.feBuf)
	if err != nil {
		return
	}

	var clone core.Message
	clone.InitFromMessage(&s.feBuf)
	s.feCh <- &clone

	err = s.be.Send(&s.feBuf)
	if err != nil {
		return
	}
	if !s.fe.HasNext() {
		return s.be.Flush()
	}
	return
}

func (s *sniffingRouter) RouteBackend() error {
	// route the next message from backend to frotnend,
	// blocking and flushing if necessary
	err := s.be.Next(&s.beBuf)
	if err != nil {
		return err
	}
	if proto.IsBackendKeyData(&s.beBuf) {
		beInfo, err := proto.ReadBackendKeyData(&s.beBuf)
		if err != nil {
			return err
		}
		s.backendPid = beInfo.BackendPid
		s.secretKey = beInfo.SecretKey
	}

	var clone core.Message
	clone.InitFromMessage(&s.beBuf)
	s.beCh <- &clone

	err = s.fe.Send(&s.beBuf)
	if !s.be.HasNext() {
		return s.fe.Flush()
	}
	return nil
}
