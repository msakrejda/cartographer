package main

import (
	"fmt"
	"github.com/deafbybeheading/femebe"
	"github.com/deafbybeheading/femebe/core"
	"github.com/deafbybeheading/femebe/proto"
	"github.com/deafbybeheading/femebe/util"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
)

type proxy struct {
	resolver femebe.Resolver
	manager  femebe.SessionManager
	frontend chan *core.Message
	backend  chan *core.Message
}

type fixedResolver struct {
	targetAddr string
}

func (pr *fixedResolver) Resolve(params map[string]string) femebe.Connector {
	return femebe.NewSimpleConnector(pr.targetAddr, params)
}

// Generic connection handler
//
// This redelegates to more specific proxy handlers that contain the
// main proxy loop logic.
func (p *proxy) handleConnection(cConn net.Conn) {
	defer cConn.Close()
	var err error
	// Log disconnections
	defer func() {
		if pn := recover(); pn != nil {
			fmt.Printf("error in handling connection: %v", pn)
			cConn.Close()
		} else {
			if err != nil && err != io.EOF {
				log.Print("Session exits with error: ", err)
			} else {
				log.Print("Session exits cleanly")
			}
		}
	}()

	feStream := core.NewFrontendStream(util.NewBufferedReadWriteCloser(cConn))
	var m core.Message
	err = feStream.Next(&m)
	if err != nil {
		panic(fmt.Errorf("could not read client startup message: %v", err))
	}
	if proto.IsStartupMessage(&m) {
		startup, err := proto.ReadStartupMessage(&m)
		if err != nil {
			panic(fmt.Errorf("could not parse client startup message: %v", err))
		}
		log.Print("Accept connection with startup message:")
		connector := p.resolver.Resolve(startup.Params)
		beStream, err := connector.Startup()
		if err != nil {
			panic(fmt.Errorf("could not connect to backend: %v", err))
		}
		router := NewSniffingRouter(feStream, beStream, p.frontend, p.backend)
		session := femebe.NewSimpleSession(router, connector)
		err = p.manager.RunSession(session)
	} else if proto.IsCancelRequest(&m) {
		cancel, err := proto.ReadCancelRequest(&m)
		if err != nil {
			panic(fmt.Errorf("could not parse cancel message: %v", err))
		}
		err = p.manager.Cancel(cancel.BackendPid, cancel.SecretKey)
		if err != nil {
			panic(fmt.Errorf("could not process cancellation: %v", err))
		}
		err = cConn.Close()
		if err != nil {
			fmt.Println(err)
		}
	} else {
		panic(fmt.Errorf("could not understand client"))
	}
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

	ln, err := util.AutoListen(os.Args[1])
	if err != nil {
		log.Printf("Could not listen on address: %v", err)
		os.Exit(1)
	}

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

		target := os.Args[2]
		resolver := &fixedResolver{target}
		manager := femebe.NewSimpleSessionManager()

		frontend := make(chan *core.Message)
		backend := make(chan *core.Message)

		p := &proxy{resolver, manager, frontend, backend}

		sw := NewSessionWatcher(frontend, backend, activity)
		go sw.listen()
		go p.handleConnection(conn)
	}

	log.Println("cartographer finished")
}
