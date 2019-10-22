package main

import (
	"github.com/gorilla/websocket"
	"sync"
)

type connection struct {
	id   string
	lock sync.Mutex
	c    *websocket.Conn
}
