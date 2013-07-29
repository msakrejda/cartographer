package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"femebe"
	"femebe/pgproto"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
)

// Automatically chooses between unix sockets and tcp sockets for
// listening
func autoListen(place string) (net.Listener, error) {
	if strings.Contains(place, "/") {
		return net.Listen("unix", place)
	}

	return net.Listen("tcp", place)
}

// Automatically chooses between unix sockets and tcp sockets for
// dialing.
func autoDial(place string) (net.Conn, error) {
	if strings.Contains(place, "/") {
		return net.Dial("unix", place)
	}

	return net.Dial("tcp", place)
}

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

// Generic connection handler
//
// This redelegates to more specific proxy handlers that contain the
// main proxy loop logic.
func handleConnection(cConn net.Conn, destaddr string, frontend, backend chan *femebe.Message) {
	var err error

	// Log disconnections
	defer func() {
		if err != nil && err != io.EOF {
			log.Printf("Session exits with error: %v\n", err)
		} else {
			log.Printf("Session exits cleanly\n")
		}
	}()

	defer cConn.Close()

	c := femebe.NewFrontendMessageStream(
		"Client", newBufWriteCon(cConn))

	// Must interpret Startup and Cancel requests.
	//
	// SSL Negotiation requests not handled for now.
	var firstPacket femebe.Message
	c.Next(&firstPacket)

	// Handle Startup packets
	var sup *pgproto.Startup
	if sup, err = pgproto.ReadStartupMessage(&firstPacket); err != nil {
		log.Print(err)
		return
	}
	log.Print("Accept connection with startup message:")
	for key, value := range sup.Params {
		log.Printf("\t%s: %s\n", key, value)
	}

	unencryptServerConn, err := autoDial(destaddr)
	if err != nil {
		log.Printf("Could not connect to server: %v\n", err)
		return
	}

	tlsConf := tls.Config{}
	tlsConf.InsecureSkipVerify = true

	sConn, err := femebe.NegotiateTLS(
		unencryptServerConn, "prefer", &tlsConf)
	if err != nil {
		log.Printf("Could not negotiate TLS: %v\n", err)
		return
	}

	s := femebe.NewBackendMessageStream("Server", newBufWriteCon(sConn))
	if err != nil {
		log.Printf("Could not initialize connection to server: %v\n", err)
		return
	}

	err = s.Send(&firstPacket)
	if err != nil {
		return
	}

	err = s.Flush()
	if err != nil {
		return
	}

	done := make(chan error)
	session := NewSniffingProxySession(done,
		&ProxyPair{c, cConn},
		&ProxyPair{s, sConn},
		frontend, backend)
	session.start()
	// Both sides must exit to finish
	_ = <-done
	_ = <-done
}

func installSignalHandlers() {
	sigch := make(chan os.Signal)
	signal.Notify(sigch, os.Interrupt, os.Kill)
	go func() {
		for sig := range sigch {
			log.Printf("Got signal %v", sig)
			if sig == os.Kill {
				os.Exit(2)
			} else if sig == os.Interrupt {
				os.Exit(0)
			}
		}
	}()
}

type SessionWatcher struct {
	lastQuery *pgproto.Query
	lastMetadata *pgproto.RowDescription
	lastData []*pgproto.DataRow
	lastError *pgproto.ErrorResponse

	activityCh chan string
	nextEventId int
}

func NewSessionWatcher(activity chan string) *SessionWatcher {
	return &SessionWatcher{activityCh: activity}
}

type Column struct {
	ColName string `json:"name"`
	ColType string `json:"type"`
}

type SessionEvent struct {
	Id int
	Query string
	Columns []*Column `json:omitempty`
	Data [][]interface{} `json:omitempty`
	Error map[string]string `json:omitempty`
}

func (sw *SessionWatcher) generateSessionEvent() *SessionEvent {
	if sw.lastQuery == nil {
		log.Printf("%#v\n", sw)
	}
	var cols []*Column
	var data [][]interface{}
	var errors map[string]string
	if sw.lastMetadata != nil {
		cols = make([]*Column, len(sw.lastMetadata.Fields))
		for i, field := range sw.lastMetadata.Fields {
			typeDescr := pgproto.DescribeType(field.TypeOid)
			cols[i] = &Column{field.Name, typeDescr}
		}
		if sw.lastData != nil {
			data = make([][]interface{}, len(sw.lastData))
			for i, dataRow := range sw.lastData {
				data[i] = make([]interface{}, len(cols))
				for col, val := range dataRow.Values {
					if val != nil {
						data[i][col] = pgproto.Decode(val,
							sw.lastMetadata.Fields[col].TypeOid)
					}
				}
			}
		}
	}
	if sw.lastError != nil {
		errors = make(map[string]string)
		for key, value := range sw.lastError.Details {
			keyDescr := pgproto.DescribeStatusCode(key)
			errors[keyDescr] = value
		}
	}

	sw.nextEventId++
	// create a new event
	event := &SessionEvent{
		Id: sw.nextEventId,
		Query: sw.lastQuery.Query,
		Columns: cols,
		Data: data,
		Error: errors,
	}
	// and flush state
	sw.lastQuery = nil
	sw.lastMetadata = nil
	sw.lastData = make([]*pgproto.DataRow, 0)
	sw.lastError = nil

	return event
}

func (sw *SessionWatcher) onRequest(m *femebe.Message) {
	log.Printf("< %c", m.MsgType())
	switch t := m.MsgType(); t {
	case 'Q':
		q, err := pgproto.ReadQuery(m)
		if err != nil {
			panic("Oh snap")
		}
		sw.lastQuery = q
		// TODO: support extended query protocol
	default:
		// just ignore the message; it's not interesting for now
	}
}

func (sw *SessionWatcher) onResponse(m *femebe.Message) {
	log.Printf("< %c", m.MsgType())
	switch t := m.MsgType(); t {
	case 'C':
		// command complete; encode and send off the current query
		eventData := sw.generateSessionEvent()
		eventStr, err := json.Marshal(eventData)
		if err != nil {
			panic("Oh snap")
		}
		sw.activityCh <- string(eventStr)
	case 'B':
		// error response
		eresp, err := pgproto.ReadErrorResponse(m)
		if err != nil {
			panic("Oh snap")
		}
		sw.lastError = eresp
	case 'T':
		desc, err := pgproto.ReadRowDescription(m)
		if err != nil {
			panic("Oh snap")
		}
		sw.lastMetadata = desc
	case 'D':
		datarow, err := pgproto.ReadDataRow(m)
		if err != nil {
			panic("Oh snap")
		}
		sw.lastData = append(sw.lastData, datarow)
	default:
		// just ignore the message; it's not interesting for now
	}
}

func messageListener(sw *SessionWatcher, frontend chan *femebe.Message,
	backend chan *femebe.Message) {
	for {
		select {
		case m := <- frontend:
			sw.onRequest(m)
		case m := <- backend:
			sw.onResponse(m)
		}
	}
}

// Startup and main client acceptance loop
func main() {
	installSignalHandlers()

	if len(os.Args) < 2 {
		log.Printf("Usage: cartographer LISTENADDR TARGETADDR")
		os.Exit(1)
	}

	ln, err := autoListen(os.Args[1])
	if err != nil {
		log.Printf("Could not listen on address: %v", err)
		os.Exit(1)
	}
	targetaddr := os.Args[2]

	for {
		conn, err := ln.Accept()

		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		activityCh := make(chan string)
		go func() {
			for event := range activityCh {
				log.Println(event)
			}
		}()
		sw := NewSessionWatcher(activityCh)

		frontend := make(chan *femebe.Message)
		backend := make(chan *femebe.Message)

		go messageListener(sw, frontend, backend);
		go handleConnection(conn, targetaddr, frontend, backend)
	}

	log.Println("cartographer finished")
}
