package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type WebRelay struct {
	activity <-chan string
	clients  []*websocket.Conn
	connLock sync.Mutex
}

func NewWebRelay(activity <-chan string) *WebRelay {
	return &WebRelay{activity: activity}
}

// Relay activity messages to all connected clients
func (w *WebRelay) Relay() {
	for proto := range w.activity {
		w.relayMessage(proto)
	}
}

func (w *WebRelay) relayMessage(proto string) {
	log.Println(proto)
	w.connLock.Lock()
	defer w.connLock.Unlock()
	for i, client := range w.clients {
		err := websocket.Message.Send(client, proto)
		if err != nil {
			client.Close()
			copy(w.clients[i:], w.clients[i+1:])
			w.clients[len(w.clients)-1] = nil
			w.clients = w.clients[:len(w.clients)-1]
		}
	}
}

func (w *WebRelay) handleNewClient(ws *websocket.Conn) {
	log.Printf("Got new client %v\n", ws)
	w.connLock.Lock()
	w.clients = append(w.clients, ws)
	w.connLock.Unlock()
	// TODO: some cleaner way to block until clients disconnect
	select {}
}

func (w *WebRelay) listenHttp(port int) {
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		http.Redirect(res, req, "cartographer", http.StatusFound)
	})
	http.Handle("/cartographer/connect", websocket.Handler(w.handleNewClient))
	http.Handle("/cartographer/", http.StripPrefix("/cartographer",
		http.FileServer(http.Dir("public/"))))

	log.Printf("listening for http requests on %v...\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		panic(err)
	}
}
