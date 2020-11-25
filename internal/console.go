package gateway

import (
	"sync"
	"sync/atomic"

	websocket "github.com/fasthttp/websocket"
)

// ConsolePump handles serving the current logs on the dashboard through Websocket
type ConsolePump struct {
	ConnsMu sync.RWMutex              `json:"-"`
	Conns   map[int64]*websocket.Conn `json:"-"`

	DeadMu sync.RWMutex        `json:"-"`
	Dead   map[int64]chan bool `json:"-"`

	iter *int64
}

// NewConsolePump creates a new console pump
func NewConsolePump() *ConsolePump {
	return &ConsolePump{
		ConnsMu: sync.RWMutex{},
		Conns:   make(map[int64]*websocket.Conn),
		DeadMu:  sync.RWMutex{},
		Dead:    make(map[int64]chan bool),
		iter:    new(int64),
	}
}

// Write implements io.Writer
func (cp *ConsolePump) Write(p []byte) (n int, err error) {

	message, err := websocket.NewPreparedMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}

	cp.ConnsMu.RLock()
	for id, conn := range cp.Conns {
		err = conn.WritePreparedMessage(message)
		if err != nil {
			close(cp.Dead[id])
		}
	}
	cp.ConnsMu.RUnlock()

	return len(p), err
}

// RegisterConnection will take the already established websocket and add it to the pump
func (cp *ConsolePump) RegisterConnection(conn *websocket.Conn) (id int64) {
	id = atomic.AddInt64(cp.iter, 1)

	cp.ConnsMu.Lock()
	defer cp.ConnsMu.Unlock()

	cp.DeadMu.Lock()
	defer cp.DeadMu.Unlock()

	cp.Conns[id] = conn
	cp.Dead[id] = make(chan bool, 1)
	return
}

// DeregisterConnection removes a connection from the pump. Returns boolean if it was removed
func (cp *ConsolePump) DeregisterConnection(id int64) (ok bool) {
	cp.ConnsMu.Lock()
	defer cp.ConnsMu.Unlock()

	cp.DeadMu.Lock()
	defer cp.DeadMu.Unlock()

	_, ok = cp.Conns[id]
	if ok {
		delete(cp.Conns, id)
	}

	delete(cp.Dead, id)
	return
}
