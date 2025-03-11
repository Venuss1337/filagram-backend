package imiddleware

import (
	"filachat/internal/models"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/v2/bson"
    "net/http"
)

func UserAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !c.IsTLS() { return c.JSON(http.StatusBadRequest, "connection not secured") }

		if c.Request().Body == nil { return c.JSON(http.StatusBadRequest, "missing request body") }

		var user models.User = models.User{Id: bson.NewObjectID() }
		if err := c.Bind(&user); err != nil { return c.JSON(http.StatusBadRequest, "invalid json body") }

		if user.Username == "" || user.Password == "" || user.Email == "" { return c.JSON(http.StatusBadRequest, "missing username or password") }
		c.Set("user", user)
		return next(c)
	}
}