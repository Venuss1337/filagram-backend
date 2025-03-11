package database

import (
	"context"
	"filachat/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"time"
)

func (DB *DB) SaveMessage(message *models.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

	_, err := DB.Db.Collection("messages").InsertOne(ctx, *message)
	if err != nil { return err }
	return nil
}
func (DB *DB) ReadMessage(id bson.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := DB.Db.Collection("messages").UpdateByID(ctx, id, bson.D{{"read", true}})
	return err
}
func (DB *DB) GetUnreadMessages(id bson.ObjectID) ([]models.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.D{{"recipient_id", id}}
	result, err := DB.Db.Collection("messages").Find(ctx, filter)
	if err != nil { return models.NilMessages, err }

	var messages []models.Message
	if err := result.All(ctx, &messages); err != nil { return models.NilMessages, err }

	return messages, nil
}