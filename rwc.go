package jsonrpc2

import (
	"io"
	"net/http"

	"github.com/gorilla/websocket"
)

type RWCCloseHandler func()

type ReadWriteCloser struct {
	WS           *websocket.Conn
	r            io.Reader
	w            io.WriteCloser
	closeHandler RWCCloseHandler
}

func (rwc *ReadWriteCloser) OnClose(handler RWCCloseHandler) {
	rwc.closeHandler = handler
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
			if rwc.closeHandler != nil {
				go rwc.closeHandler()
			}
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
	if rwc.w == nil {
		rwc.w, err = rwc.WS.NextWriter(websocket.TextMessage)
		if err != nil {
			return 0, err
		}
	}
	for n = 0; n < len(p); {
		var m int
		m, err = rwc.w.Write(p)
		n += m
		if err != nil {
			break
		}
	}
	if err != nil || n == len(p) {
		err = rwc.Close()
	}
	return
}

func (rwc *ReadWriteCloser) Close() (err error) {
	if rwc.w != nil {
		err = rwc.w.Close()
		rwc.w = nil
	}
	return err
}

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
