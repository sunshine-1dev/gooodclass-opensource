package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/jwgl"
)

// ExamHandler handles GET /api/getExam?username=&password=
func ExamHandler(client *jwgl.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Query("username")
		password := c.Query("password")
		if username == "" || password == "" {
			c.String(http.StatusBadRequest, "missing username or password")
			return
		}

		token, err := client.LoginAndGetToken(username, password)
		if err != nil {
			c.String(http.StatusInternalServerError, "login failed: %v", err)
			return
		}

		form := url.Values{
			"biz_type_id": {"1"},
			"semester_id": {"260"},
			"random":      {"30828"},
			"token":       {token},
		}

		raw, err := client.PostAPI("/student/exam/schedule/lesson/get-exam-tasks", form, token)
		if err != nil {
			c.String(http.StatusInternalServerError, "fetch exam failed: %v", err)
			return
		}

		var items []struct {
			CourseName   string `json:"course_name"`
			ExamTypeName string `json:"exam_type_name"`
			Date         string `json:"date"`
			TimeStart    string `json:"time_start"`
			TimeEnd      string `json:"time_end"`
			PlaceName    string `json:"place_name"`
		}

		println(string(raw)) // Debug: print raw response
		if err := json.Unmarshal(raw, &items); err != nil {
			c.String(http.StatusInternalServerError, "parse exam data: %v", err)
			return
		}

		type examOutput struct {
			CourseName string `json:"课程名称"`
			ExamType   string `json:"考试类型"`
			Date       string `json:"日期"`
			Time       string `json:"时间"`
			Place      string `json:"地点"`
			SeatNo     string `json:"座位号"`
			Status     string `json:"状态"`
		}

		result := make([]examOutput, 0)
		for _, item := range items {
			// Extract last part of place_name (split by whitespace, take last)
			place := lastWord(item.PlaceName)
			result = append(result, examOutput{
				CourseName: item.CourseName,
				ExamType:   item.ExamTypeName,
				Date:       item.Date,
				Time:       fmt.Sprintf("%s~%s", item.TimeStart, item.TimeEnd),
				Place:      place,
				SeatNo:     "未知",
				Status:     "正常",
			})
		}

		out, _ := json.Marshal(result)
		c.Data(http.StatusOK, "text/plain; charset=utf-8", out)
	}
}
