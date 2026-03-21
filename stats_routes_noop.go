//go:build !stats

package main

import (
	"github.com/gin-gonic/gin"

	"gooodclass/internal/jwgl"
)

// RegisterStatsRoutes is a no-op when built without the stats tag.
func RegisterStatsRoutes(r *gin.Engine, client *jwgl.Client) func() {
	return func() {}
}

// StatsLoginMiddleware is a no-op when built without the stats tag.
func StatsLoginMiddleware(client *jwgl.Client) gin.HandlerFunc {
	return func(c *gin.Context) {}
}
