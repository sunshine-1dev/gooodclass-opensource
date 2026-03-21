package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RedirectHandler handles GET /redirect
func RedirectHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, "https://whoisluckydog.pwxiao.top/404.html")
	}
}
