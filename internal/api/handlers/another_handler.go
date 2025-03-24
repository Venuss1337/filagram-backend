package handlers

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
)

// Configuration for the MQTT broker
const (
	DefaultBrokerURL = "tcp://localhost:1883"
	ClientIDPrefix   = "backend-service-"
	QosLevel         = 1
)

// Message types
const (
	MessageTypeText   = "message"
	MessageTypeTyping = "typing"
	MessageTypeStatus = "status"
	MessageTypeOnline = "online"
)

// Status types
const (
	StatusDelivered = "delivered"
	StatusRead      = "read"
	StatusUnread    = "unread"
)

// Message represents a chat message
type Message struct {
	ID        string    `json:"id"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// TypingIndicator represents a typing notification
type TypingIndicator struct {
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	IsTyping  bool      `json:"isTyping"`
	Timestamp time.Time `json:"timestamp"`
}

// MessageStatus represents a message status update
type MessageStatus struct {
	MessageID string    `json:"messageId"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// OnlineStatus represents user's online status
type OnlineStatus struct {
	UserID    string    `json:"userId"`
	IsOnline  bool      `json:"isOnline"`
	LastSeen  time.Time `json:"lastSeen"`
	Timestamp time.Time `json:"timestamp"`
}

// UserPresence tracks user online status
type UserPresence struct {
	mutex       sync.RWMutex
	onlineUsers map[string]time.Time
}

// NewUserPresence creates a new UserPresence tracker
func NewUserPresence() *UserPresence {
	return &UserPresence{
		onlineUsers: make(map[string]time.Time),
	}
}

// SetUserOnline marks a user as online
func (up *UserPresence) SetUserOnline(userID string) {
	up.mutex.Lock()
	defer up.mutex.Unlock()
	up.onlineUsers[userID] = time.Now()
}

// SetUserOffline marks a user as offline
func (up *UserPresence) SetUserOffline(userID string) {
	up.mutex.Lock()
	defer up.mutex.Unlock()
	delete(up.onlineUsers, userID)
}

// IsUserOnline checks if a user is online
func (up *UserPresence) IsUserOnline(userID string) (bool, time.Time) {
	up.mutex.RLock()
	defer up.mutex.RUnlock()
	lastSeen, exists := up.onlineUsers[userID]
	return exists, lastSeen
}

// ActiveUsers returns all online users
func (up *UserPresence) ActiveUsers() map[string]time.Time {
	up.mutex.RLock()
	defer up.mutex.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]time.Time, len(up.onlineUsers))
	for user, stime := range up.onlineUsers {
		result[user] = stime
	}
	return result
}

// Server represents the chat server
type Server struct {
	client      mqtt.Client
	presence    *UserPresence
	messageChan chan []byte
	statusChan  chan []byte
	typingChan  chan []byte
	onlineChan  chan []byte
	doneChan    chan struct{}
	msgCounter  uint64
	msgMutex    sync.Mutex
}

// NewServer creates a new chat server
func NewServer(brokerURL string) *Server {
	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(fmt.Sprintf("%s%d", ClientIDPrefix, time.Now().UnixNano())).
		SetKeepAlive(60 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetCleanSession(false).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(5 * time.Minute).
		SetConnectionLostHandler(func(client mqtt.Client, err error) {
			log.Println("Connection lost:", err)
		}).
		SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
			log.Println("Attempting to reconnect...")
		}).
		SetOnConnectHandler(func(client mqtt.Client) {
			log.Println("Connected to MQTT broker")
		})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	server := &Server{
		client:      client,
		presence:    NewUserPresence(),
		messageChan: make(chan []byte, 1000),
		statusChan:  make(chan []byte, 1000),
		typingChan:  make(chan []byte, 1000),
		onlineChan:  make(chan []byte, 1000),
		doneChan:    make(chan struct{}),
	}

	return server
}

// Start begins the server operation
func (s *Server) Start() {
	// Subscribe to chat messages
	token := s.client.Subscribe("chat/+/+/message", QosLevel, s.handleMessage)
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Failed to subscribe to messages: %v", token.Error())
	}

	// Subscribe to typing indicators
	token = s.client.Subscribe("chat/+/+/typing", QosLevel, s.handleTyping)
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Failed to subscribe to typing indicators: %v", token.Error())
	}

	// Subscribe to message status updates
	token = s.client.Subscribe("chat/+/+/status", QosLevel, s.handleStatus)
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Failed to subscribe to status updates: %v", token.Error())
	}

	// Subscribe to online status
	token = s.client.Subscribe("chat/+/online", QosLevel, s.handleOnline)
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Failed to subscribe to online status: %v", token.Error())
	}

	// Start the worker pools
	go s.processMessages()
	go s.processTyping()
	go s.processStatus()
	go s.processOnline()

	// Start the user presence cleanup routine
	go s.cleanupInactiveUsers()

	log.Println("Chat server started successfully")
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown() {
	close(s.doneChan)
	s.client.Disconnect(250) // Wait 250ms for the client to disconnect
	log.Println("Chat server shut down")
}

// Handler for chat messages
func (s *Server) handleMessage(client mqtt.Client, msg mqtt.Message) {
	// Queue the message for processing
	select {
	case s.messageChan <- msg.Payload():
	default:
		log.Println("Message queue full, dropping message")
	}
}

// Handler for typing indicators
func (s *Server) handleTyping(client mqtt.Client, msg mqtt.Message) {
	// Queue the typing notification for processing
	select {
	case s.typingChan <- msg.Payload():
	default:
		// For typing indicators, it's okay to drop if queue is full
		// as new typing indicators will come in
	}
}

// Handler for status updates
func (s *Server) handleStatus(client mqtt.Client, msg mqtt.Message) {
	// Queue the status update for processing
	select {
	case s.statusChan <- msg.Payload():
	default:
		log.Println("Status queue full, dropping status update")
	}
}

// Handler for online status
func (s *Server) handleOnline(client mqtt.Client, msg mqtt.Message) {
	// Queue the online status for processing
	select {
	case s.onlineChan <- msg.Payload():
	default:
		log.Println("Online status queue full, dropping update")
	}
}

// Process messages from the message queue
func (s *Server) processMessages() {
	for {
		select {
		case payload := <-s.messageChan:
			var message Message
			if err := json.Unmarshal(payload, &message); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			// Process the message
			log.Printf("Message from %s to %s: %s", message.Sender, message.Receiver, message.Content)

			// Auto-send delivery receipt
			status := MessageStatus{
				MessageID: message.ID,
				Sender:    message.Receiver, // Reverse for status updates
				Receiver:  message.Sender,   // Reverse for status updates
				Status:    StatusDelivered,
				Timestamp: time.Now(),
			}

			statusPayload, err := json.Marshal(status)
			if err != nil {
				log.Printf("Failed to marshal status update: %v", err)
				continue
			}

			topic := fmt.Sprintf("chat/%s/%s/status", message.Receiver, message.Sender)
			token := s.client.Publish(topic, QosLevel, false, statusPayload)
			token.Wait()
			if token.Error() != nil {
				log.Printf("Failed to publish status update: %v", token.Error())
			}

			// Update presence
			s.presence.SetUserOnline(message.Sender)

		case <-s.doneChan:
			return
		}
	}
}

// Process typing indicators
func (s *Server) processTyping() {
	for {
		select {
		case payload := <-s.typingChan:
			var typing TypingIndicator
			if err := json.Unmarshal(payload, &typing); err != nil {
				log.Printf("Failed to unmarshal typing indicator: %v", err)
				continue
			}

			// Process the typing indicator
			log.Printf("Typing indicator from %s to %s: %v",
				typing.Sender, typing.Receiver, typing.IsTyping)

			// Update presence
			s.presence.SetUserOnline(typing.Sender)

		case <-s.doneChan:
			return
		}
	}
}

// Process status updates
func (s *Server) processStatus() {
	for {
		select {
		case payload := <-s.statusChan:
			var status MessageStatus
			if err := json.Unmarshal(payload, &status); err != nil {
				log.Printf("Failed to unmarshal status update: %v", err)
				continue
			}

			// Process the status update
			log.Printf("Status update for message %s from %s to %s: %s",
				status.MessageID, status.Sender, status.Receiver, status.Status)

			// Update presence
			s.presence.SetUserOnline(status.Sender)

		case <-s.doneChan:
			return
		}
	}
}

// Process online status updates
func (s *Server) processOnline() {
	for {
		select {
		case payload := <-s.onlineChan:
			var online OnlineStatus
			if err := json.Unmarshal(payload, &online); err != nil {
				log.Printf("Failed to unmarshal online status: %v", err)
				continue
			}

			// Process the online status
			if online.IsOnline {
				s.presence.SetUserOnline(online.UserID)
				log.Printf("User %s is now online", online.UserID)
			} else {
				s.presence.SetUserOffline(online.UserID)
				log.Printf("User %s is now offline", online.UserID)
			}

		case <-s.doneChan:
			return
		}
	}
}

// Periodically check for inactive users and mark them as offline
func (s *Server) cleanupInactiveUsers() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			activeUsers := s.presence.ActiveUsers()

			for userID, lastSeen := range activeUsers {
				// If user hasn't been seen for more than 5 minutes, mark as offline
				if now.Sub(lastSeen) > 5*time.Minute {
					s.presence.SetUserOffline(userID)

					// Publish offline status
					status := OnlineStatus{
						UserID:    userID,
						IsOnline:  false,
						LastSeen:  lastSeen,
						Timestamp: now,
					}

					payload, err := json.Marshal(status)
					if err != nil {
						log.Printf("Failed to marshal offline status: %v", err)
						continue
					}

					topic := fmt.Sprintf("chat/%s/online", userID)
					token := s.client.Publish(topic, QosLevel, true, payload)
					token.Wait()
					if token.Error() != nil {
						log.Printf("Failed to publish offline status: %v", token.Error())
					}

					log.Printf("User %s marked as offline due to inactivity", userID)
				}
			}

		case <-s.doneChan:
			return
		}
	}
}

func main() {
	// Parse command line flags
	brokerURL := flag.String("broker", DefaultBrokerURL, "MQTT broker URL")
	flag.Parse()

	// Create a context for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up server
	server := NewServer(*brokerURL)
	server.Start()

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until we receive a signal
	<-c
	log.Println("Shutting down server...")
	server.Shutdown()
}
