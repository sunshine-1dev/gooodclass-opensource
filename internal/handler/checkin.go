package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/store"
)

func CheckInHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Query("user_id")
		if userID == "" {
			c.String(http.StatusBadRequest, "missing user_id")
			return
		}

		rank, alreadyDone, err := s.CheckIn(userID)
		if err != nil {
			c.String(http.StatusInternalServerError, "打卡失败: %v", err)
			return
		}

		if alreadyDone {
			c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("你今天已经打过卡了"))
			return
		}

		msg := fmt.Sprintf("打卡成功，你是今天第%d个打卡的人", rank)
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(msg))
	}
}

// CheckInRankHandler handles GET /checkInRank
func CheckInRankHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		records, err := s.TodayRankings()
		if err != nil {
			c.String(http.StatusInternalServerError, "获取排行榜失败: %v", err)
			return
		}

		out, _ := json.Marshal(records)
		c.Data(http.StatusOK, "text/plain; charset=utf-8", out)
	}
}
