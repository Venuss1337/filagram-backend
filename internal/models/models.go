package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"time"
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
		Read        bool          `json:"read,omitempty" bson:"read,omitempty"`
		Timestamp   time.Time     `json:"timestamp,omitempty" bson:"timestamp,omitempty"`
	}
)

var (
	NilUser     = User{}
	NilMessage  = Message{}
	NilMessages []Message
)
