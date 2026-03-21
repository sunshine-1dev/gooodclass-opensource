package handler

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// OverrideScriptHandler handles GET /overrideScript
// Serves the holiday.json file from the configured data directory.
func OverrideScriptHandler(filePath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := os.ReadFile(filePath)
		if err != nil {
			c.String(http.StatusNotFound, "File not found")
			return
		}
		c.Data(http.StatusOK, "text/plain; charset=utf-8", data)
	}
}
