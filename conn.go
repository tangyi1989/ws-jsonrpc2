package jsonrpc2

import (
	"sync"

	"net/http"
)

type ConnCloseHandler func()

type Conn struct {
	Request       *http.Request
	rwc           *ReadWriteCloser
	codec         ServerCodec
	sending       *sync.Mutex
	closed        bool
	mu            sync.RWMutex
	closeHandlers []ConnCloseHandler
	extraData     map[string]interface{}
}

func NewConn(req *http.Request, sending *sync.Mutex, codec ServerCodec) *Conn {
	return &Conn{
		Request:   req,
		sending:   sending,
		codec:     codec,
		extraData: make(map[string]interface{}),
	}
}

func (c *Conn) ternimated() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	for _, handler := range c.closeHandlers {
		go handler()
	}

	c.closeHandlers = []ConnCloseHandler{}
}

func (c *Conn) Notify(method string, params interface{}) error {
	c.sending.Lock()
	defer c.sending.Unlock()

	return c.codec.WriteNotification(method, params)
}

func (c *Conn) Close() error {
	return c.codec.Close()
}

func (c *Conn) OnClose(f ConnCloseHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		go f()
	}
	c.closeHandlers = append(c.closeHandlers, f)
}

func (c *Conn) GetData(key string) interface{} {
	c.mu.RLock()
	c.mu.RUnlock()

	return c.extraData[key]
}

func (c *Conn) SetData(key string, value interface{}) {
	c.mu.Lock()
	c.mu.Unlock()

	c.extraData[key] = value
}
