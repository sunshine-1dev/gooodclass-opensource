package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/jwgl"
)

// StudentsHandler handles GET /api/getStudents?token=&lesson_id=
func StudentsHandler(client *jwgl.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		lessonID := c.Query("lesson_id")
		if token == "" || lessonID == "" {
			c.String(http.StatusBadRequest, "missing token or lesson_id")
			return
		}

		form := url.Values{
			"lesson_id":   {lessonID},
			"random":      {fmt.Sprintf("%d", rand.Intn(90000)+10000)},
			"semester_id": {"260"},
			"token":       {token},
		}

		raw, err := client.PostAPI("/course/schedule/lesson/get-students", form, token)
		if err != nil {
			c.String(http.StatusInternalServerError, "fetch students failed: %v", err)
			return
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", raw)
	}
}
