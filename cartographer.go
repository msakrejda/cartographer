package main

import (
	"crypto/tls"
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

	activity := make(chan string)
	web := NewWebRelay(activity)
	go web.Relay()
	go web.listenHttp(8080)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		frontend := make(chan *femebe.Message)
		backend := make(chan *femebe.Message)

		sw := NewSessionWatcher(frontend, backend, activity)
		go sw.listen()
		go handleConnection(conn, targetaddr, frontend, backend)
	}

	log.Println("cartographer finished")
}
