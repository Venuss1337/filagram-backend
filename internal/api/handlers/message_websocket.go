package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/http"
	"time"
)

// User represents a chat user
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// ChatMessage represents a chat message with all metadata
type ChatMessage struct {
	ID        string    `json:"id"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

// UserStatus represents the online/offline status of a user
type UserStatus struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	IsOnline bool   `json:"isOnline"`
}

// APIServer represents the REST API server that integrates with MQTT
type APIServer struct {
	e           *echo.Echo
	mqttClient  mqtt.Client
	userService *UserService
}

// UserService handles user-related operations
type UserService struct {
	users      map[string]*User
	userStatus map[string]bool
}

// NewUserService creates a new user service
func NewUserService() *UserService {
	return &UserService{
		users:      make(map[string]*User),
		userStatus: make(map[string]bool),
	}
}

// AddUser adds a new user
func (us *UserService) AddUser(username string) *User {
	userID := uuid.New().String()
	user := &User{
		ID:       userID,
		Username: username,
	}
	us.users[userID] = user
	return user
}

// GetUser gets a user by ID
func (us *UserService) GetUser(userID string) (*User, bool) {
	user, exists := us.users[userID]
	return user, exists
}

// SetUserStatus sets a user's online status
func (us *UserService) SetUserStatus(userID string, isOnline bool) {
	us.userStatus[userID] = isOnline
}

// GetUserStatus gets a user's online status
func (us *UserService) GetUserStatus(userID string) bool {
	online, exists := us.userStatus[userID]
	if !exists {
		return false
	}
	return online
}

// NewAPIServer creates a new API server
func NewAPIServer(brokerURL string) *APIServer {
	// Set up MQTT client
	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(fmt.Sprintf("api-server-%d", time.Now().UnixNano())).
		SetKeepAlive(60 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetCleanSession(false).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(5 * time.Minute)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	// Create Echo instance
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Create user service
	userService := NewUserService()

	// Create API server
	server := &APIServer{
		e:           e,
		mqttClient:  client,
		userService: userService,
	}

	// Set up routes
	server.setupRoutes()

	// Subscribe to status updates
	token := client.Subscribe("chat/+/+/status", 1, server.handleMQTTStatus)
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Failed to subscribe to status updates: %v", token.Error())
	}

	// Subscribe to online status
	token = client.Subscribe("chat/+/online", 1, server.handleMQTTOnline)
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("Failed to subscribe to online status: %v", token.Error())
	}

	return server
}

// Set up API routes
func (s *APIServer) setupRoutes() {
	// API version group
	api := s.e.Group("/api/v1")

	// User routes
	api.POST("/users", s.registerUser)
	api.GET("/users", s.listUsers)
	api.GET("/users/:id", s.getUser)
	api.GET("/users/:id/status", s.getUserStatus)

	// Message routes
	api.POST("/messages", s.sendMessage)
	api.GET("/messages", s.getMessages)
	api.PUT("/messages/:id/status", s.updateMessageStatus)

	// Typing indicator routes
	api.POST("/typing", s.sendTypingIndicator)
}

// Start the API server
func (s *APIServer) Start(address string) error {
	return s.e.Start(address)
}

// Shutdown the API server
func (s *APIServer) Shutdown(ctx context.Context) error {
	s.mqttClient.Disconnect(250)
	return s.e.Shutdown(ctx)
}

// Handler for MQTT status updates
func (s *APIServer) handleMQTTStatus(client mqtt.Client, msg mqtt.Message) {
	var status MessageStatus
	if err := json.Unmarshal(msg.Payload(), &status); err != nil {
		log.Printf("Failed to unmarshal status update: %v", err)
		return
	}

	// Process status update (could store in database, etc.)
	log.Printf("Received status update for message %s: %s", status.MessageID, status.Status)
}

// Handler for MQTT online status
func (s *APIServer) handleMQTTOnline(client mqtt.Client, msg mqtt.Message) {
	var status OnlineStatus
	if err := json.Unmarshal(msg.Payload(), &status); err != nil {
		log.Printf("Failed to unmarshal online status: %v", err)
		return
	}

	// Update user status
	s.userService.SetUserStatus(status.UserID, status.IsOnline)
	log.Printf("User %s is now %s", status.UserID, map[bool]string{true: "online", false: "offline"}[status.IsOnline])
}

// API Handlers

// Register a new user
func (s *APIServer) registerUser(c echo.Context) error {
	type RegisterRequest struct {
		Username string `json:"username"`
	}

	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	user := s.userService.AddUser(req.Username)

	// Publish user's online status
	status := OnlineStatus{
		UserID:    user.ID,
		IsOnline:  true,
		LastSeen:  time.Now(),
		Timestamp: time.Now(),
	}

	payload, err := json.Marshal(status)
	if err != nil {
		log.Printf("Failed to marshal online status: %v", err)
	} else {
		topic := fmt.Sprintf("chat/%s/online", user.ID)
		token := s.mqttClient.Publish(topic, 1, true, payload)
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to publish online status: %v", token.Error())
		}
	}

	return c.JSON(http.StatusCreated, user)
}

// List all users
func (s *APIServer) listUsers(c echo.Context) error {
	var users []*User
	for _, user := range s.userService.users {
		users = append(users, user)
	}
	return c.JSON(http.StatusOK, users)
}

// Get a user by ID
func (s *APIServer) getUser(c echo.Context) error {
	userID := c.Param("id")
	user, exists := s.userService.GetUser(userID)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}
	return c.JSON(http.StatusOK, user)
}

// Get a user's status
func (s *APIServer) getUserStatus(c echo.Context) error {
	userID := c.Param("id")
	user, exists := s.userService.GetUser(userID)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	status := UserStatus{
		UserID:   user.ID,
		Username: user.Username,
		IsOnline: s.userService.GetUserStatus(userID),
	}

	return c.JSON(http.StatusOK, status)
}

// Send a message
func (s *APIServer) sendMessage(c echo.Context) error {
	type MessageRequest struct {
		Sender   string `json:"sender"`
		Receiver string `json:"receiver"`
		Content  string `json:"content"`
	}

	var req MessageRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Validate users
	_, senderExists := s.userService.GetUser(req.Sender)
	_, receiverExists := s.userService.GetUser(req.Receiver)
	if !senderExists || !receiverExists {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Sender or receiver not found"})
	}

	// Create message
	message := Message{
		ID:        uuid.New().String(),
		Sender:    req.Sender,
		Receiver:  req.Receiver,
		Content:   req.Content,
		Timestamp: time.Now(),
	}

	// Marshal message
	payload, err := json.Marshal(message)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to marshal message"})
	}

	// Publish message to MQTT
	topic := fmt.Sprintf("chat/%s/%s/message", req.Sender, req.Receiver)
	token := s.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to publish message"})
	}

	// Update sender's online status
	s.userService.SetUserStatus(req.Sender, true)

	return c.JSON(http.StatusCreated, message)
}

// Get messages (placeholder - would require a database in a real implementation)
func (s *APIServer) getMessages(c echo.Context) error {
	return c.JSON(http.StatusOK, []ChatMessage{})
}

// Update message status
func (s *APIServer) updateMessageStatus(c echo.Context) error {
	type StatusRequest struct {
		Sender   string `json:"sender"`
		Receiver string `json:"receiver"`
		Status   string `json:"status"`
	}

	messageID := c.Param("id")
	var req StatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Create status update
	status := MessageStatus{
		MessageID: messageID,
		Sender:    req.Sender,
		Receiver:  req.Receiver,
		Status:    req.Status,
		Timestamp: time.Now(),
	}

	// Marshal status
	payload, err := json.Marshal(status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to marshal status"})
	}

	// Publish status to MQTT
	topic := fmt.Sprintf("chat/%s/%s/status", req.Sender, req.Receiver)
	token := s.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to publish status"})
	}

	return c.JSON(http.StatusOK, status)
}

// Send typing indicator
func (s *APIServer) sendTypingIndicator(c echo.Context) error {
	type TypingRequest struct {
		Sender   string `json:"sender"`
		Receiver string `json:"receiver"`
		IsTyping bool   `json:"isTyping"`
	}

	var req TypingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Create typing indicator
	typing := TypingIndicator{
		Sender:    req.Sender,
		Receiver:  req.Receiver,
		IsTyping:  req.IsTyping,
		Timestamp: time.Now(),
	}

	// Marshal typing indicator
	payload, err := json.Marshal(typing)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to marshal typing indicator"})
	}

	// Publish typing indicator to MQTT
	topic := fmt.Sprintf("chat/%s/%s/typing", req.Sender, req.Receiver)
	token := s.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to publish typing indicator"})
	}

	return c.JSON(http.StatusOK, typing)
}

// Command-line entry point
/*func main() {
	server := NewAPIServer("tcp://localhost:1883")

	// Start server
	go func() {
		if err := server.Start(":8080"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	// Gracefully shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to gracefully shutdown server: %v", err)
	}
}*/
