package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"time"
)

const (
	TypeMessage MessageType = "message"
	TypeTyping  MessageType = "typing"
	TypeStatus  MessageType = "status"
	StatusRead StatusType = "read"
	StatusDelivered StatusType = "delivered"
)

type (
	User struct {
		Id           bson.ObjectID `json:"id" bson:"_id"`
		Username     string        `json:"username,omitempty" bson:"username,omitempty"`
		Email        string        `json:"email,omitempty" bson:"email,omitempty"`
		Password     string        `json:"password,omitempty" bson:"password,omitempty"`
		AccessToken  string        `json:"access_token,omitempty" bson:"-"`
		RefreshToken string        `json:"refresh_token,omitempty" bson:"-"`
	}
	Message struct {
		Id          bson.ObjectID `json:"id" bson:"_id"`
		SenderId    bson.ObjectID `json:"sender_id" bson:"sender_id"`
		RecipientId bson.ObjectID `json:"recipient_id" bson:"recipient_id"`
		Content     string        `json:"content,omitempty" bson:"content,omitempty"`
		AesSecret   string        `json:"aes_secret,omitempty" bson:"aes_secret,omitempty"`
		SharedSecretSalt []byte   `json:"shared_secret_salt,omitempty" bson:"shared_secret_salt,omitempty"`
		Read        bool          `json:"read,omitempty" bson:"read,omitempty"`
		Timestamp   time.Time     `json:"timestamp,omitempty" bson:"timestamp,omitempty"`
	}
	TypingNotification struct {
    	Sender    bson.ObjectID   `json:"sender"`
    	Receiver  string    `json:"receiver"`
    	IsTyping  bool      `json:"is_typing"`
    	Timestamp time.Time `json:"timestamp"`
    }
	StatusUpdate struct {
    	Sender    bson.ObjectID    `json:"sender"`
    	Receiver  bson.ObjectID     `json:"receiver"`
    	MessageID bson.ObjectID    `json:"message_id"`
    	Status    StatusType `json:"status"`
    	Timestamp time.Time  `json:"timestamp"`
    }
	UserStatus struct {
    	UserID    bson.ObjectID    `json:"user_id"`
    	IsOnline  bool      `json:"is_online"`
    	LastSeen  time.Time `json:"last_seen"`
    	Timestamp time.Time `json:"timestamp"`
    }
	MessageType string
	StatusType string
)

var (
	NilUser     = User{}
	NilMessage  = Message{}
	NilMessages []Message
)
