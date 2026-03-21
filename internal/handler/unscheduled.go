package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/jwgl"
)

// UnscheduledHandler handles GET /api/getUnscheduledCourses?token=
func UnscheduledHandler(client *jwgl.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.String(http.StatusBadRequest, "missing token")
			return
		}

		form := url.Values{
			"biz_type_id": {"1"},
			"random":      {fmt.Sprintf("%d", rand.Intn(90000)+10000)},
			"semester_id": {"320"},
			"token":       {token},
		}

		raw, err := client.PostAPI("/student/course/schedule/get-unscheduled-lessons", form, token)
		if err != nil {
			c.String(http.StatusInternalServerError, "fetch unscheduled courses failed: %v", err)
			return
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", raw)
	}
}
