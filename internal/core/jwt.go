package core

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"time"
)

type JWTTokens struct{}

func (j *JWTTokens) NewAccess(id bson.ObjectID, iss string) (string, error) {
	rawToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"sub": id,
		"iss": iss,
		"exp": time.Now().Add(time.Hour * 2).Unix(),
		"iat": time.Now().Unix(),
		"typ": "access_token",
	})

	return rawToken.SignedString(Ed25519Keys.AccessPrivateKey)
}
func (j *JWTTokens) NewRefresh(id bson.ObjectID, iss string) (string, error) {
	rawToken := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"sub": id,
		"iss": iss,
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(),
		"iat": time.Now().Unix(),
		"typ": "refresh_token",
	})

	return rawToken.SignedString(Ed25519Keys.RefreshPrivateKey)
}
func (j *JWTTokens) VerifyAccess(token string) (*jwt.MapClaims, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return Ed25519Keys.AccessPublicKey, nil
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
	if iss, err := claims.GetIssuer(); err != nil || (iss != "https://auth.filagram.pl/refresh-token" && iss != "https://auth.filagram.pl/signin") {
		return nil, errors.New("invalid token")
	}
	if iat, err := claims.GetIssuedAt(); err != nil || (iat.Unix() >= time.Now().Unix()) {
		return nil, errors.New("invalid token")
	}
	if exp, err := claims.GetExpirationTime(); err != nil || exp.Unix() < time.Now().Unix() {
		return nil, errors.New("token expired")
	}
	if typ, ok := claims["typ"].(string); !ok || typ != "access_token" {
		return nil, errors.New("invalid token")
	}

	return &claims, nil
}
func (j *JWTTokens) VerifyRefresh(token string) (*jwt.MapClaims, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return Ed25519Keys.RefreshPublicKey, nil
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
	if iss, err := claims.GetIssuer(); err != nil || iss != "https://auth.filagram.pl/signin" {
		return nil, errors.New("invalid token")
	}
	if iat, err := claims.GetIssuedAt(); err != nil || (iat.Unix() >= time.Now().Unix()) {
		return nil, errors.New("invalid token")
	}
	if exp, err := claims.GetExpirationTime(); err != nil || exp.Unix() < time.Now().Unix() {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}

var JWTFactory = &JWTTokens{}
