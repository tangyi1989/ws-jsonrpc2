package jsonrpc2

import (
	"io"
	"net/http"

	"github.com/gorilla/websocket"
)

// ReadWriteCloser ...
type ReadWriteCloser struct {
	WS *websocket.Conn
	r  io.Reader
}

func (rwc *ReadWriteCloser) Read(p []byte) (n int, err error) {
	if rwc.r == nil {
		_, rwc.r, err = rwc.WS.NextReader()
		if err != nil {
			return 0, err
		}
	}
	for n = 0; n < len(p); {
		var m int
		m, err = rwc.r.Read(p[n:])
		n += m
		if err == io.EOF {
			rwc.r = nil
			break
		}
		if err != nil {
			break
		}
	}

	return
}

func (rwc *ReadWriteCloser) Write(p []byte) (n int, err error) {
	var w io.WriteCloser
	w, err = rwc.WS.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}

	for n = 0; n < len(p); {
		var m int
		m, err = w.Write(p)
		n += m
		if err != nil {
			break
		}
	}

	if err != nil {
		err = rwc.Close()
		return
	}

	w.Close()
	return
}

// Close ...
func (rwc *ReadWriteCloser) Close() (err error) {
	err = rwc.WS.Close()
	return
}

// ServeRPC ...
func ServeRPC(r *http.Request, ws *websocket.Conn, server ...*Server) {
	var s *Server

	rwc := &ReadWriteCloser{WS: ws}
	codec := NewServerCodec(rwc)
	if len(server) == 0 {
		s = DefaultServer
	} else {
		s = server[0]
	}

	s.ServeCodec(r, codec, nil)
}

// ServeRPCwithInit ...
func ServeRPCwithInit(r *http.Request, ws *websocket.Conn, onInit ConnHandler,
	server ...*Server) {

	var s *Server

	rwc := &ReadWriteCloser{WS: ws}
	codec := NewServerCodec(rwc)
	if len(server) == 0 {
		s = DefaultServer
	} else {
		s = server[0]
	}

	s.ServeCodec(r, codec, onInit)
}
