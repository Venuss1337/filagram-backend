package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha3"
	"filachat/internal/api/handlers"
	imiddleware "filachat/internal/api/middleware"
	"filachat/internal/core"
	database "filachat/internal/data"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/curve25519"
	"io"
	"log"
	"net/http"
)

type User struct {
	name       string
	publicKey  []byte
	privateKey []byte
}

type Message struct {
	content          []byte
	aesSecret        []byte
	sharedSecretSalt []byte
}

func generateKeyPair() ([]byte, []byte) {
	var privateKey = make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, privateKey); err != nil {
		panic(err)
	}

	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		panic(err)
	}

	return publicKey, privateKey
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all connections by returning true (in production, you should validate the origin).
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		log.Printf("Received: %s", message)

		// Echo the received message back to the client.
		if err := conn.WriteMessage(messageType, message); err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func main() {
	err := core.LoadKeys()
	if err != nil {
		panic(err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:      "1; mode=block",
		XFrameOptions:      "SAMEORIGIN",
		ContentTypeNosniff: "nosniff",
		HSTSMaxAge:         3600,
	}))
	e.Use(middleware.BodyLimit("1M"))

	client, err := database.Connect()
	if err != nil {
		panic(err)
	}
	db := database.DB{Db: client.Database("filagram")}
	h := &handlers.Handler{DB: &db}
	e.POST("/signup", imiddleware.UserAuth(h.SignUp))
	e.POST("/signin", imiddleware.UserAuth(h.SignIn))
	e.POST("/refresh-token", imiddleware.JWTRefreshAuth(h.RefreshToken))

	e.File("/", "./public/index.html")

	e.Logger.Fatal(e.StartTLS(":8080", "./secrets/cert.pem", "./secrets/key.pem"))
}

func EncryptMessage(content, publicKey, privateKey []byte) (Message, error) {
	messageAes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, messageAes); err != nil {
		return Message{}, err
	}

	messageBlock, err := aes.NewCipher(messageAes)
	if err != nil {
		panic(err)
	}

	messageNonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, messageNonce); err != nil {
		return Message{}, err
	}
	aesgcm, err := cipher.NewGCM(messageBlock)
	if err != nil {
		return Message{}, err
	}
	messageCipherText := aesgcm.Seal(messageNonce, messageNonce, content, nil)

	rawShared, err := curve25519.X25519(privateKey, publicKey)
	if err != nil {
		return Message{}, err
	}

	hash := sha3.New256
	sharedSecretSalt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, sharedSecretSalt); err != nil {
		return Message{}, err
	}

	sharedSecret, err := hkdf.Key(hash, rawShared, sharedSecretSalt, "", hash().Size())
	if err != nil {
		return Message{}, err
	}

	encKeyBlock, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return Message{}, err
	}

	encKeyCipher, err := cipher.NewGCM(encKeyBlock)
	if err != nil {
		return Message{}, err
	}

	encKeyNonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, encKeyNonce); err != nil {
		return Message{}, err
	}

	encKeyEncrypted := encKeyCipher.Seal(encKeyNonce, encKeyNonce, messageAes, nil)

	message := Message{
		content:          messageCipherText,
		aesSecret:        encKeyEncrypted,
		sharedSecretSalt: sharedSecretSalt,
	}
	return message, nil
}

func DecryptMessage(message Message, publicKey, privateKey []byte) (string, error) {
	// natalia creates a raw shared secret
	rawShared, err := curve25519.X25519(privateKey, publicKey)
	if err != nil {
		return "", err
	}

	// hash function for secret shared
	hash := sha3.New256
	sharedSecret, err := hkdf.Key(hash, rawShared, message.sharedSecretSalt, "", hash().Size())
	if err != nil {
		return "", err
	}

	encKeyBlock, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return "", err
	}

	encKeyCipher, err := cipher.NewGCM(encKeyBlock)
	if err != nil {
		return "", err
	}

	messageAes, err := encKeyCipher.Open(nil,
		message.aesSecret[:encKeyCipher.NonceSize()],
		message.aesSecret[encKeyCipher.NonceSize():],
		nil)
	if err != nil {
		return "", err
	}

	textBlock, err := aes.NewCipher(messageAes)
	if err != nil {
		return "", err
	}

	textCipher, err := cipher.NewGCM(textBlock)
	if err != nil {
		return "", err
	}

	plainText, err := textCipher.Open(nil, message.content[:textCipher.NonceSize()], message.content[textCipher.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}
