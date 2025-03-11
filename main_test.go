package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebSocketEchos(t *testing.T) {
	// Setup a mux router and register the /ws endpoint.
	router := mux.NewRouter()
	router.HandleFunc("/ws", wsHandler)

	// Create a test HTTP server with the mux router.
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Convert the test server URL from http:// to ws://.
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	// Connect to the WebSocket server.
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer ws.Close()

	// Send a test message.
	testMsg := "hello, websocket"
	jsonData, _ := json.Marshal(testMsg)
	if err := ws.WriteMessage(websocket.BinaryMessage, jsonData); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Read echoed message.
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	if string(msg) != string(jsonData) {
		t.Errorf("Expected message %q, got %q", testMsg, string(msg))
	}
}
