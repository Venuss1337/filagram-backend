package core

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"time"
)

type JWTTokens struct{}

func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}

func (j *JWTTokens) NewToken(id bson.ObjectID, iss string, access bool) (string, error) {
	rawToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"sub": id,
		"iss": iss,
		"exp": If(access, time.Now().Add(time.Hour*2).Unix(), time.Now().Add(time.Hour*24*7).Unix()),
		"iat": time.Now().Unix(),
		"typ": If(access, "access_token", "refresh_token"),
	})

	return rawToken.SignedString(If(access, Ed25519Keys.AccessPrivateKey, Ed25519Keys.RefreshPrivateKey))
}
func (j *JWTTokens) ParseToken(token string, access bool) (*jwt.MapClaims, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return If(access, Ed25519Keys.AccessPublicKey, Ed25519Keys.RefreshPublicKey), nil
	})
	if err != nil {
		return nil, err
	}
	var (
		claims jwt.MapClaims
		ok     bool
	)
	if claims, ok = parsedToken.Claims.(jwt.MapClaims); !ok || !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}
	return &claims, nil
}
func (j *JWTTokens) VerifyClaims(claims *jwt.MapClaims, access bool) error {
	if iss, err := claims.GetIssuer(); err != nil || If(access, iss != "https://auth.filagram.pl/refresh-token" && iss != "https://auth.filagram.pl/signin", iss != "https://auth.filagram.pl/signin") {
		return errors.New("invalid token")
	}
	if iat, err := claims.GetIssuedAt(); err != nil || (iat.Unix() >= time.Now().Unix()) {
		return errors.New("invalid token")
	}
	if exp, err := claims.GetExpirationTime(); err != nil || exp.Unix() < time.Now().Unix() {
		return errors.New("token expired")
	}
	if _, err := claims.GetSubject(); err != nil {
		return errors.New("invalid token")
	}
	return nil
}

var JWTFactory = &JWTTokens{}
