package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/auth"
	"gooodclass/internal/jwgl"
	"gooodclass/internal/middleware"
	"gooodclass/internal/store"
)

func LoginHandler(client *jwgl.Client, s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Query("username")
		password := c.Query("password")
		if username == "" || password == "" {
			c.String(http.StatusBadRequest, "missing username or password")
			return
		}

		result, err := auth.Login(client, username, password)
		if err != nil {
			c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(result))
			return
		}

		if !strings.HasPrefix(result, "SUCCESS,") {
			c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(result))
			return
		}

		nickname := strings.TrimPrefix(result, "SUCCESS,")
		isAdmin := s.IsAdmin(username)

		token, tokenErr := middleware.GenerateToken(username, nickname, isAdmin)
		if tokenErr == nil {
			c.Header("X-Auth-Token", token)
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(result))
	}
}
