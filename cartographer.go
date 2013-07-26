package main

import (
	"bufio"
	"crypto/tls"
	"femebe"
	"femebe/pgproto"
	"fmt"
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

func NewMITMProxySession(errch chan error,
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

	c := femebe.NewClientMessageStream(
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

	s := femebe.NewServerMessageStream("Server", newBufWriteCon(sConn))
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
	session := NewMITMProxySession(done,
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

func oidToTypeName() {
	
}

type SessionWatcher struct {
	lastQuery *pgproto.Query
	lastMetadata *pgproto.RowDescription
	lastData []*pgproto.DataRow
	lastError *pgproto.ErrorResponse

	activityCh chan string
}

func NewSessionWatcher(activity chan string) {
	return &SessionWatcher{activityCh: activity}
}

type Column struct {
	colName string `json:"name"`
	colType string `json:"type"`
}

type SessionEvent struct {
	id int
	query string
	runtime float64
	columns []*Column
	data [][]interface{}
	error string `json:omitempty`
}

// gross global state
nextEventId int = 0

func newSessionEvent(query *pgproto.Query, metadata *pgproto.RowDescription,
	data []*pgproto.DataRow, error *pgproto.ErrorResponse) {
	return &SessionEvent{
		id: nextEventId++,
		query: query.Query,
		runtime
		
	}
}


func (sw *SessionWatcher) onRequest(m *femebe.Message) {
	msgJSON := toJSON(m)
	if msgJSON != "" {
		fmt.Printf("> %v\n", msgJSON)
	}
	switch t := m.MsgType(); t {
	case 'Q':
		q, err := pgproto.ReadQuery(m)
		if err != nil {
			sw.lastQuery = q
		}
		// TODO: support extended query protocol
	default:
		// just ignore the message; it's not interesting for now
	}
}



func (sw *SessionWatcher) onResponse(m *femebe.Message) {
	msgJSON := toJSON(m)
	if msgJSON != "" {
		fmt.Printf("< %v\n", msgJSON)
	}
	switch t := m.MsgType(); t {
	case 'C':
		// command complete; encode and send off the current query
		eventData = newSessionEvent(query, metadata, data, error)
		sw.activityCh <- eventData

		// and flush state
		sw.lastQuery = nil
		sw.lastMetada = nil
		sw.lastData = nil
		sw.lastError = nil
	case 'B':
		// error response
		eresp, err := pgproto.ReadErrorResponse(m)
		if err != nil {
			sw.lastError = eresp
		} else {
			panic("Oh snap")
		}
	case 'T':
		desc, err := pgproto.ReadRowDescription(m)
		if err != nil {
			sw.metadata = desc
		} else {
			panic("Oh snap")
		}
	case 'D':
		datarow, err := pgproto.ReadDataRow(m)
		if err != nil {
			sw.data = append(sw.data, datarow)
		} else {
			panic("Oh snap")
		}
	default:
		// just ignore the message; it's not interesting for now
	}
}

func messageListener(parser *ActivityParser, frontend chan *femebe.Message,
	backend chan *femebe.Message) {
	for {
		select {
		case m := <- frontend:
			parser.onRequest(m)
		case m := <- backend:
			parser.onResponse(m)
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

	var p ActivityParser
	frontend := make(chan *femebe.Message)
	backend := make(chan *femebe.Message)

	go messageListener(&p, frontend, backend);

	for {
		conn, err := ln.Accept()

		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		go handleConnection(conn, targetaddr, frontend, backend)
	}

	log.Println("cartographer exits successfully")
	return
}

func toJSON(msg *femebe.Message) string {
	msgFuncs := make(map[byte](func(*femebe.Message)string))

	msgFuncs[pgproto.MSG_QUERY_Q] = func(msg *femebe.Message) string {
		query, err := pgproto.ReadQuery(msg)
		if err != nil {
			panic("Oh snap!")
		}
		return "{ query: \"" + escape(query.Query) + "\"}"
	}
	msgFuncs[pgproto.MSG_ROW_DESCRIPTION_T] = func(msg *femebe.Message) string {
		rowDescription, err := pgproto.ReadRowDescription(msg)
		if err != nil {
			panic("Oh snap!")
		}
		descr := "{ description: ["
		for i, field := range rowDescription.Fields {
			if i != 0 {
				descr += ","
			}
			descr += "\"" +  escape(field.Name) + "\""
		}

		descr += "] }"

		return descr
	}
	msgFuncs[pgproto.MSG_DATA_ROW_D] = func(msg *femebe.Message) string {
		dat, err := pgproto.ReadDataRow(msg)
		if err != nil {
			panic("Oh snap!")
		}
		return fmt.Sprintf("{ data: %v }", dat.Values)
	}
	msgFuncs[pgproto.MSG_COMMAND_COMPLETE_C] = func(msg *femebe.Message) string {
		cc, err := pgproto.ReadCommandComplete(msg)
		if err != nil {
			panic("Oh snap!")
		}
		return fmt.Sprintf("%v %v %v", cc.Tag, cc.AffectedCount, cc.Oid)
	}




	readFunc := msgFuncs[msg.MsgType()]
	if readFunc != nil {
		return fmt.Sprintf("%c %v", msg.MsgType(), readFunc(msg))
	} else {
		return fmt.Sprintf("%c", msg.MsgType())
	}

	panic("Oh snap")
	//return fmt.Sprintf("%c", msg.MsgType())
}

func escape(str string) string {
	return strings.Replace(str, "\"", "\\\"", -1)
}