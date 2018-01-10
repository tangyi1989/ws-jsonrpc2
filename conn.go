package jsonrpc2

import (
	"sync"

	"errors"
	"net/http"
)

type notifyEvent struct {
	method string
	params interface{}
}

type ConnCloseHandler func()

type Conn struct {
	notifyChan chan *notifyEvent
	stopChan   chan interface{}

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
	conn := &Conn{
		Request:    req,
		sending:    sending,
		codec:      codec,
		notifyChan: make(chan *notifyEvent, 1024),
		stopChan:   make(chan interface{}),
		extraData:  make(map[string]interface{}),
	}

	go conn.notifyLoop()
	return conn
}

func (c *Conn) notifyLoop() {
	for {
		var isStop bool
		select {
		case notify := <-c.notifyChan:
			c.sending.Lock()
			err := c.codec.WriteNotification(notify.method, notify.params)
			c.sending.Unlock()

			if err != nil {
				c.Close()
				isStop = true
			}
		case <-c.stopChan:
			isStop = true
		}

		if isStop {
			break
		}
	}
}

func (c *Conn) ternimating() {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.stopChan)
	c.closed = true
	for _, handler := range c.closeHandlers {
		handler()
	}

	c.closeHandlers = []ConnCloseHandler{}
}

func (c *Conn) AsyncNotify(method string, params interface{}) error {
	if c.closed {
		return errors.New("conn is closed.")
	}

	select {
	case c.notifyChan <- &notifyEvent{
		method: method,
		params: params,
	}:
	default:
		return errors.New("Notify chan is full.")
	}

	return nil
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

func (c *Conn) DelData(key string) {
	c.mu.Lock()
	c.mu.Unlock()

	delete(c.extraData, key)
}
