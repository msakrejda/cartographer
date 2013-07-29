package main

import (
	"encoding/json"
	"femebe"
	"femebe/pgproto"
	"log"
)

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