package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/middleware"
	"gooodclass/internal/store"
)

type voteRequest struct {
	UserID     string `json:"user_id"`
	TargetType string `json:"target_type"`
	TargetID   int64  `json:"target_id"`
	Value      int    `json:"value"`
}

func VoteHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req voteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		req.UserID = middleware.GetUserID(c)
		if req.TargetType != "review" && req.TargetType != "question" && req.TargetType != "answer" && req.TargetType != "campus_review" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target_type"})
			return
		}
		if req.TargetID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target_id required"})
			return
		}
		if req.Value != -1 && req.Value != 0 && req.Value != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "value must be -1, 0, or 1"})
			return
		}

		if err := s.Vote(req.UserID, req.TargetType, req.TargetID, req.Value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}
