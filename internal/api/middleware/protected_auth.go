package imiddleware

import (
	"encoding/base64"
	"encoding/hex"
	"filachat/internal/core"
"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"strings"
)

func JWTRefreshAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !c.IsTLS() { return &echo.HTTPError{Code: http.StatusForbidden, Message: "connection not secured"}}

		var (
			after string = ""
			found bool = false
		)
		if after, found = strings.CutPrefix(c.Request().Header.Get("Authorization"), "Bearer "); after=="" || !found {
			return &echo.HTTPError{Code: http.StatusForbidden, Message: "invalid token"}
		}

		decodedToken, err := base64.StdEncoding.DecodeString(after)
		if err != nil {
			return &echo.HTTPError{Code: http.StatusForbidden, Message: "invalid token"}
		}

		key, err := hex.DecodeString(os.Getenv("JWT_REFRESH_SECRET"))
		if err != nil {
			return &echo.HTTPError{Code: http.StatusInternalServerError, Message: "encryption failed"}
		}

		decryptedToken, err := core.JWTEncrypter.Decrypt(decodedToken, key)
		if err != nil {
			return &echo.HTTPError{Code: http.StatusForbidden, Message: "encryption failed"}
		}
		claims, err := core.JWTFactory.VerifyRefresh(string(decryptedToken))
		if err != nil {
			return &echo.HTTPError{Code: http.StatusForbidden, Message: err.Error()}
		}
		c.Set("claims", claims)
		return next(c)
	}
}
func JWTAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}