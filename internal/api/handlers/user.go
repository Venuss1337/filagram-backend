package handlers

import (
	"encoding/base64"
	"encoding/hex"
	"filachat/internal/core"
	"filachat/internal/models"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
)

func (h *Handler) SignUp(c echo.Context) error {
	user := c.Get("user").(models.User)

	if userExists, err := h.DB.Exists(user.Username, user.Email); err != nil || userExists {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: "user already exists"}
	}

	hash, err := core.Hashing.Hash([]byte(user.Password))
	if err != nil { return &echo.HTTPError{Code:http.StatusInternalServerError, Message: "hashing failed"} }

	if err := h.DB.NewUser(user.Id, user.Username, user.Email, hash); err != nil {
		return &echo.HTTPError{Code:http.StatusInternalServerError, Message: "user not created"}
	}
	user.Password = ""
	return c.JSON(http.StatusCreated, user)
}
func (h *Handler) SignIn(c echo.Context) error {
	user := c.Get("user").(models.User)

	if userExists, err := h.DB.Exists(user.Username, user.Email); err != nil || !userExists {
		return &echo.HTTPError{Code: http.StatusNotFound, Message: "invalid user or password"}
	}

	dbUser, err := h.DB.GetUser(user.Id)
	if err != nil {
		return &echo.HTTPError{Code: http.StatusNotFound, Message: "invalid user or password"}
	}

	if core.Hashing.Verify([]byte(user.Password), dbUser.Password) != nil {
		return &echo.HTTPError{Code: http.StatusUnauthorized, Message: "invalid user or password"}
	}
	user.Password = ""
	user.Email = ""

	rawAccessToken, err := core.JWTFactory.NewAccess(user.Id, "https://auth.filagram.pl/signin")
	rawRefreshToken, err := core.JWTFactory.NewRefresh(user.Id, "https://auth.filagram.pl/signin")
	if err != nil {
		return &echo.HTTPError{Code: http.StatusUnauthorized, Message: "error signing token" }
	}
	godotenv.Load()

	encAccessKey, err := hex.DecodeString(os.Getenv("JWT_ACCESS_SECRET"))
	encRefreshKey, err := hex.DecodeString(os.Getenv("JWT_REFRESH_SECRET"))
	if err != nil {
		return &echo.HTTPError{Code: http.StatusUnauthorized, Message: "error creating token" }
	}
	encAccessToken, err := core.JWTEncrypter.Encrypt([]byte(rawAccessToken), encAccessKey)
	encRefreshToken, err := core.JWTEncrypter.Encrypt([]byte(rawRefreshToken), encRefreshKey)
	if err != nil {
		return &echo.HTTPError{Code: http.StatusUnauthorized, Message: "error creating token" }
	}

	user.AccessToken = base64.StdEncoding.EncodeToString(encAccessToken)
	user.RefreshToken = base64.StdEncoding.EncodeToString(encRefreshToken)

	return c.JSON(http.StatusCreated, user)
}

func (h *Handler) Hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}