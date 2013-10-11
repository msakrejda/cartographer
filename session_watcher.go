package main

import (
	"encoding/json"
	"github.com/deafbybeheading/femebe/codec"
	"github.com/deafbybeheading/femebe/core"
	"github.com/deafbybeheading/femebe/proto"
	"log"
)

var (
	nextSessionId int
)

type SessionWatcher struct {
	sessionId    int
	lastQuery    *proto.Query
	lastMetadata *proto.RowDescription
	lastData     []*proto.DataRow
	lastError    *proto.ErrorResponse

	frontend chan *core.Message
	backend  chan *core.Message

	activity    chan string
	nextEventId int
}

// Relay the protocol message flow activity from the two frontend and
// backend channels onto a single channel providing a simplified JSON
// view of the activity
func NewSessionWatcher(fe, be chan *core.Message, activity chan string) *SessionWatcher {
	nextSessionId++
	return &SessionWatcher{sessionId: nextSessionId,
		frontend: fe,
		backend:  be,
		activity: activity,
	}
}

type Column struct {
	ColName string `json:"name"`
	ColType string `json:"type"`
}

type SessionEvent struct {
	SessionId int               `json:"session_id"`
	Id        int               `json:"id"`
	Query     string            `json:"query"`
	Columns   []*Column         `json:"columns,omitempty"`
	Data      [][]interface{}   `json:"data,omitempty"`
	Error     map[string]string `json:"error,omitempty"`
}

func (sw *SessionWatcher) generateSessionEvent() *SessionEvent {
	var cols []*Column
	var data [][]interface{}
	var errors map[string]string
	if sw.lastMetadata != nil {
		cols = make([]*Column, len(sw.lastMetadata.Fields))
		for i, field := range sw.lastMetadata.Fields {
			typeDescr := codec.DescribeType(field.TypeOid)
			cols[i] = &Column{field.Name, typeDescr}
		}
		if sw.lastData != nil {
			data = make([][]interface{}, len(sw.lastData))
			for i, dataRow := range sw.lastData {
				data[i] = make([]interface{}, len(cols))
				for col, val := range dataRow.Values {
					if val != nil {
						data[i][col] = codec.Decode(val,
							sw.lastMetadata.Fields[col].TypeOid)
					}
				}
			}
		}
	}
	if sw.lastError != nil {
		errors = make(map[string]string)
		for key, value := range sw.lastError.Details {
			keyDescr := proto.DescribeStatusCode(key)
			errors[keyDescr] = value
		}
	}

	sw.nextEventId++
	// create a new event
	event := &SessionEvent{
		SessionId: sw.sessionId,
		Id:        sw.nextEventId,
		Query:     sw.lastQuery.Query,
		Columns:   cols,
		Data:      data,
		Error:     errors,
	}
	// and flush state
	sw.lastQuery = nil
	sw.lastMetadata = nil
	sw.lastData = make([]*proto.DataRow, 0)
	sw.lastError = nil

	return event
}

func (sw *SessionWatcher) onRequest(m *core.Message) {
	log.Printf("< %c", m.MsgType())
	switch t := m.MsgType(); t {
	case 'Q':
		q, err := proto.ReadQuery(m)
		if err != nil {
			panic("Oh snap")
		}
		sw.lastQuery = q
		// TODO: support extended query protocol
	default:
		// just ignore the message; it's not interesting for now
	}
}

func (sw *SessionWatcher) onResponse(m *core.Message) {
	log.Printf("< %c", m.MsgType())
	switch t := m.MsgType(); t {
	case 'C':
		// command complete; encode and send off the current query
		eventData := sw.generateSessionEvent()
		eventStr, err := json.Marshal(eventData)
		if err != nil {
			panic("Oh snap")
		}
		sw.activity <- string(eventStr)
	case 'B':
		// error response
		eresp, err := proto.ReadErrorResponse(m)
		if err != nil {
			panic("Oh snap")
		}
		sw.lastError = eresp
	case 'T':
		desc, err := proto.ReadRowDescription(m)
		if err != nil {
			panic("Oh snap")
		}
		sw.lastMetadata = desc
	case 'D':
		datarow, err := proto.ReadDataRow(m)
		if err != nil {
			panic("Oh snap")
		}
		sw.lastData = append(sw.lastData, datarow)
	default:
		// just ignore the message; it's not interesting for now
	}
}

func (sw *SessionWatcher) listen() {
	for {
		select {
		case m := <-sw.frontend:
			sw.onRequest(m)
		case m := <-sw.backend:
			sw.onResponse(m)
		}
	}
}
