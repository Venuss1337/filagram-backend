package hooks
import (

"bytes"
"encoding/base64"
"encoding/hex"
"filachat/internal/core"
"github.com/joho/godotenv"
mqtt "github.com/mochi-mqtt/server/v2"
"github.com/mochi-mqtt/server/v2/packets"
"log"
"os"
)

type JWTHook struct {
	mqtt.HookBase
}

func (h *JWTHook) ID() string {
	return "jwt-hook"
}

// Provides indicates which hook methods this hook provides.
func (h *JWTHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnectAuthenticate,
		mqtt.OnACLCheck,
	}, []byte{b})
}

func (h *JWTHook) OnConnectAuthenticate(client *mqtt.Client, pk packets.Packet) bool {
	godotenv.Load()
	log.Println("[INFO] OnConnectAuthenticate")
	token := string(pk.Connect.Password)
	if token == "" {
		log.Println("[WARN] no token found in connect packet")
		return false
	}

	decodedToken, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	key, err := hex.DecodeString(os.Getenv("JWT_ACCESS_SECRET"))
	if err != nil {
		return false
	}

	decryptedToken, err := core.JWTEncrypter.Decrypt(decodedToken, key)
	if err != nil {
		return false
	}
	claims, err := core.JWTFactory.ParseToken(string(decryptedToken), true)
	if err != nil {
		return false
	}

	if err := core.JWTFactory.VerifyClaims(claims, true); err != nil {
		return false
	}
	log.Println("[INFO] connect packet authenticated", client.ID)
	return true
}