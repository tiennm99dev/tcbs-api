package tcbs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"nhooyr.io/websocket"
)

// WSEndpoint represents a known WebSocket endpoint.
type WSEndpoint string

const (
	// WSStockMatch is the WebSocket endpoint for stock match information.
	WSStockMatch WSEndpoint = "/ws/aither"
	// WSDerivativeMatch is the WebSocket endpoint for derivative match information.
	WSDerivativeMatch WSEndpoint = "/ws/nesoi"
	// WSCenter is the general WebSocket center endpoint.
	WSCenter WSEndpoint = "/ws/ouranos/v1/stream"
	// WSStockPrice is the WebSocket endpoint for normal stock prices.
	WSStockPrice WSEndpoint = "/ws/thesis/v1/stream/normal"
	// WSDerivativePrice is the WebSocket endpoint for derivative prices.
	WSDerivativePrice WSEndpoint = "/ws/thesis/v1/stream/derivative"
)

// MessageHandler is a callback invoked for each received WebSocket message.
type MessageHandler func(msgType websocket.MessageType, data []byte)

// WSConn represents a managed WebSocket connection.
type WSConn struct {
	conn    *websocket.Conn
	cancel  context.CancelFunc
	done    chan struct{}
	mu      sync.Mutex
	closed  bool
}

// Close gracefully closes the WebSocket connection.
func (ws *WSConn) Close() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if ws.closed {
		return nil
	}
	ws.closed = true
	ws.cancel()
	<-ws.done
	return ws.conn.Close(websocket.StatusNormalClosure, "client closed")
}

// Send sends a text message over the WebSocket connection.
func (ws *WSConn) Send(ctx context.Context, msg []byte) error {
	return ws.conn.Write(ctx, websocket.MessageText, msg)
}

// SendJSON marshals v to JSON and sends it as a text message.
func (ws *WSConn) SendJSON(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("tcbs: marshal ws message: %w", err)
	}
	return ws.conn.Write(ctx, websocket.MessageText, data)
}

// ConnectWS establishes a WebSocket connection to the given endpoint.
// The handler is called for each message received. The connection reads
// messages in a background goroutine until the context is cancelled or
// Close is called.
func (c *Client) ConnectWS(ctx context.Context, endpoint WSEndpoint, handler MessageHandler) (*WSConn, error) {
	wsURL := c.baseURL + string(endpoint)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)

	header := http.Header{}
	if token := c.currentToken(); token != "" {
		header.Set("Authorization", "Bearer "+token)
	}

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: header,
	})
	if err != nil {
		return nil, fmt.Errorf("tcbs: ws dial %s: %w", endpoint, err)
	}

	readCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	ws := &WSConn{
		conn:   conn,
		cancel: cancel,
		done:   done,
	}

	go func() {
		defer close(done)
		for {
			msgType, data, err := conn.Read(readCtx)
			if err != nil {
				return
			}
			handler(msgType, data)
		}
	}()

	return ws, nil
}

// ConnectStockMatch connects to the stock match information WebSocket.
func (c *Client) ConnectStockMatch(ctx context.Context, handler MessageHandler) (*WSConn, error) {
	return c.ConnectWS(ctx, WSStockMatch, handler)
}

// ConnectDerivativeMatch connects to the derivative match information WebSocket.
func (c *Client) ConnectDerivativeMatch(ctx context.Context, handler MessageHandler) (*WSConn, error) {
	return c.ConnectWS(ctx, WSDerivativeMatch, handler)
}

// ConnectCenter connects to the general WebSocket center.
func (c *Client) ConnectCenter(ctx context.Context, handler MessageHandler) (*WSConn, error) {
	return c.ConnectWS(ctx, WSCenter, handler)
}

// ConnectStockPrice connects to the normal stock price WebSocket.
func (c *Client) ConnectStockPrice(ctx context.Context, handler MessageHandler) (*WSConn, error) {
	return c.ConnectWS(ctx, WSStockPrice, handler)
}

// ConnectDerivativePrice connects to the derivative price WebSocket.
func (c *Client) ConnectDerivativePrice(ctx context.Context, handler MessageHandler) (*WSConn, error) {
	return c.ConnectWS(ctx, WSDerivativePrice, handler)
}
