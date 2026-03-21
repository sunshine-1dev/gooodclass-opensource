package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/jwgl"
)

// GPAHandler handles GET /api/getGPA?username=&password=
func GPAHandler(client *jwgl.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Query("username")
		password := c.Query("password")
		if username == "" || password == "" {
			c.String(http.StatusBadRequest, "missing username or password")
			return
		}

		token, err := client.LoginAndGetToken(username, password)
		if err != nil {
			c.String(http.StatusOK, "信息错误")
			return
		}

		form := url.Values{
			"biz_type_id": {"1"},
			"kind":        {"all"},
			"random":      {"30828"},
			"token":       {token},
			"timestamp":   {fmt.Sprintf("%d", timeNowMillis())},
		}

		raw, err := client.PostAPI("/student/exam/grade/get-grades", form, token)
		if err != nil {
			c.String(http.StatusOK, "连接错误")
			return
		}

		// Parse the grade data structure
		var data struct {
			TotalCredits    interface{} `json:"total_credits"`
			TotalGP         interface{} `json:"total_gp"`
			SemesterLessons []struct {
				Code            string      `json:"code"`
				ID              int         `json:"id"`
				SemesterCredits interface{} `json:"semester_credits"`
				SemesterGP      interface{} `json:"semester_gp"`
				Lessons         []struct {
					CourseCode   string      `json:"course_code"`
					CourseName   string      `json:"course_name"`
					CourseCredit interface{} `json:"course_credit"`
					CourseGP     interface{} `json:"course_gp"`
					ScoreText    string      `json:"score_text"`
				} `json:"lessons"`
			} `json:"semester_lessons"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			c.String(http.StatusInternalServerError, "parse gpa data: %v", err)
			return
		}

		type summaryItem struct {
			Year    string      `json:"学年度"`
			Term    string      `json:"学期"`
			Count   string      `json:"门数"`
			Credits interface{} `json:"总学分"`
			GPA     interface{} `json:"平均绩点"`
			CMP     int         `json:"cmp"`
		}
		type detailItem struct {
			YearTerm   string      `json:"学年学期"`
			CourseCode string      `json:"课程代码"`
			CourseName string      `json:"课程名称"`
			Category   string      `json:"课程类别"`
			Credits    interface{} `json:"学分"`
			Score      string      `json:"总评成绩"`
			Final      string      `json:"最终"`
			GPA        interface{} `json:"绩点"`
		}

		var summary []summaryItem
		var detail []detailItem
		totalCourses := 0

		for _, sem := range data.SemesterLessons {
			year := sem.Code[:4]
			term := string(sem.Code[len(sem.Code)-1])
			numCourses := len(sem.Lessons)
			totalCourses += numCourses

			nextYear, _ := strconv.Atoi(year)
			yearStr := fmt.Sprintf("%s-%d", year, nextYear+1)

			summary = append(summary, summaryItem{
				Year:    yearStr,
				Term:    term,
				Count:   strconv.Itoa(numCourses),
				Credits: sem.SemesterCredits,
				GPA:     sem.SemesterGP,
				CMP:     sem.ID,
			})

			for _, lesson := range sem.Lessons {
				scoreText := lesson.ScoreText
				if scoreText == "--" {
					scoreText = "0"
				}
				detail = append(detail, detailItem{
					YearTerm:   fmt.Sprintf("%s %s", yearStr, term),
					CourseCode: lesson.CourseCode,
					CourseName: lesson.CourseName,
					Category:   "未知",
					Credits:    lesson.CourseCredit,
					Score:      scoreText,
					Final:      scoreText,
					GPA:        lesson.CourseGP,
				})
			}
		}

		summaryTotal := []map[string]interface{}{
			{
				"门数":   strconv.Itoa(totalCourses),
				"总学分":  data.TotalCredits,
				"平均绩点": data.TotalGP,
			},
		}

		result := map[string]interface{}{
			"summary":       summary,
			"summary_total": summaryTotal,
			"detail":        detail,
		}

		out, _ := json.Marshal(result)
		c.Data(http.StatusOK, "text/plain; charset=utf-8", out)
	}
}
