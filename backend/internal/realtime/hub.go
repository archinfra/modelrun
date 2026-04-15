package realtime

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type Message struct {
	Type         string `json:"type"`
	DeploymentID string `json:"deploymentId,omitempty"`
	Data         any    `json:"data"`
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

type client struct {
	conn          net.Conn
	reader        *bufio.Reader
	send          chan []byte
	subscriptions map[string]struct{}
	hub           *Hub
	mu            sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: map[*client]struct{}{},
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "websocket upgrade required", http.StatusUpgradeRequired)
		return
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accept := websocketAccept(key)
	_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	_, _ = rw.WriteString("Upgrade: websocket\r\n")
	_, _ = rw.WriteString("Connection: Upgrade\r\n")
	_, _ = rw.WriteString("Sec-WebSocket-Accept: " + accept + "\r\n\r\n")
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return
	}

	c := &client{
		conn:          conn,
		reader:        rw.Reader,
		send:          make(chan []byte, 32),
		subscriptions: map[string]struct{}{},
		hub:           h,
	}

	h.add(c)
	go c.writeLoop()
	go c.readLoop()
}

func (h *Hub) Broadcast(msg Message) {
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if !c.accepts(msg.DeploymentID) {
			continue
		}
		select {
		case c.send <- raw:
		default:
		}
	}
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
}

func (c *client) accepts(deploymentID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.subscriptions) == 0 || deploymentID == "" {
		return true
	}
	_, ok := c.subscriptions[deploymentID]
	return ok
}

func (c *client) readLoop() {
	defer func() {
		c.hub.remove(c)
		_ = c.conn.Close()
	}()

	for {
		opcode, payload, err := readFrame(c.reader)
		if err != nil {
			return
		}

		switch opcode {
		case 0x1:
			c.handleText(payload)
		case 0x8:
			return
		case 0x9:
			c.send <- makeFrame(0xA, payload)
		}
	}
}

func (c *client) writeLoop() {
	for raw := range c.send {
		frame := raw
		if len(raw) == 0 || raw[0]>>4 != 0x8 {
			frame = makeFrame(0x1, raw)
		}
		if _, err := c.conn.Write(frame); err != nil {
			return
		}
	}
}

func (c *client) handleText(payload []byte) {
	var req struct {
		Type         string `json:"type"`
		DeploymentID string `json:"deploymentId"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return
	}

	switch req.Type {
	case "subscribe":
		if req.DeploymentID == "" {
			return
		}
		c.mu.Lock()
		c.subscriptions[req.DeploymentID] = struct{}{}
		c.mu.Unlock()
	case "unsubscribe":
		c.mu.Lock()
		delete(c.subscriptions, req.DeploymentID)
		c.mu.Unlock()
	}
}

func websocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func readFrame(r *bufio.Reader) (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}

	opcode := header[0] & 0x0F
	masked := header[1]&0x80 != 0
	length := uint64(header[1] & 0x7F)

	switch length {
	case 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(r, ext); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(r, ext); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext)
	}

	if length > 1<<20 {
		return 0, nil, errors.New("websocket frame too large")
	}

	mask := make([]byte, 4)
	if masked {
		if _, err := io.ReadFull(r, mask); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}

	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return opcode, payload, nil
}

func makeFrame(opcode byte, payload []byte) []byte {
	size := len(payload)
	header := []byte{0x80 | opcode}

	switch {
	case size < 126:
		header = append(header, byte(size))
	case size <= 65535:
		header = append(header, 126, byte(size>>8), byte(size))
	default:
		header = append(header, 127)
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(size))
		header = append(header, ext[:]...)
	}

	return append(header, payload...)
}
