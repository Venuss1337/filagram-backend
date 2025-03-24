package database

import (
	"context"
	"errors"
	"filachat/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"time"
)

type DB struct {
	Db *mongo.Database
}

func (DB *DB) GetUser(id bson.ObjectID) (models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := DB.Db.Collection("users").FindOne(ctx, bson.D{{"_id", id}})
	if err := result.Err(); err != nil { return models.NilUser, err }

	var user models.User
	if err := result.Decode(&user); err != nil { return models.NilUser, err }

	return user, nil
}
func (DB *DB) GetUserByName(username string) (models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := DB.Db.Collection("users").FindOne(ctx, bson.D{{"username", username}})
	if err := result.Err(); err != nil { return models.NilUser, err }
	var user models.User
	if err := result.Decode(&user); err != nil { return models.NilUser, err }
	return user, nil
}
func (DB *DB) Exists(username string, email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    filter := bson.M{
        "$or": []bson.M{
            {"email": email},
            {"username": username},
        },
    }

	var result models.User
	err := DB.Db.Collection("users").FindOne(ctx, filter).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
func (DB *DB) NewUser(id bson.ObjectID, username string, email string, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user := models.User{Id: id, Username: username, Email: email, Password: password}

	_, err := DB.Db.Collection("users").InsertOne(ctx, user)
	if err != nil { return err }

	return nil
}