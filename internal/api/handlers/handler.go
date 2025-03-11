package handlers

import database "filachat/internal/data"

type (
	Handler struct {
		DB *database.DB
	}
)