package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/jwgl"
)

// ScheduleHandler handles GET /api/base?username=&password=
// Returns the course schedule JSON matching the Python output format.
func ScheduleHandler(client *jwgl.Client) gin.HandlerFunc {
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

		output, err := fetchSchedule(client, token)
		if err != nil {
			c.String(http.StatusInternalServerError, "fetch schedule failed: %v", err)
			return
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", output)
	}
}

func fetchSchedule(client *jwgl.Client, token string) ([]byte, error) {
	// Match Python: iterate week by week from start_date to end_date
	startDate, _ := time.Parse("2006-01-02", "2026-03-02")
	endDate, _ := time.Parse("2006-01-02", "2026-07-11")

	type courseEntry struct {
		CourseName string `json:"course_name"`
		LessonID   string `json:"lesson_id"`
		Address    string `json:"address"`
		StartWeek  int    `json:"start_week"`
		EndWeek    int    `json:"end_week"`
		Week       int    `json:"week"`
		StartJie   int    `json:"start_jie"`
		EndJie     int    `json:"end_jie"`
		Teacher    string `json:"teacher"`
		Type       int    `json:"type"`
	}

	courses := make([]courseEntry, 0)
	seen := make(map[string]bool)

	current := startDate
	for current.Before(endDate) {
		currentEnd := current.AddDate(0, 0, 6)
		if currentEnd.After(endDate) {
			currentEnd = endDate
		}

		form := url.Values{
			"biz_type_id": {"1"},
			"random":      {fmt.Sprintf("%d", rand.Intn(90000)+10000)},
			"start_date":  {current.Format("2006-01-02")},
			"end_date":    {currentEnd.Format("2006-01-02")},
			"semester_id": {"320"},
			"token":       {token},
		}

		raw, err := client.PostAPI("/student/course/schedule/get-course-tables", form, token)
		if err != nil {
			current = currentEnd.AddDate(0, 0, 1)
			continue
		}

		var items []struct {
			CourseName string `json:"course_name"`
			LessonID   int    `json:"lesson_id"`
			Rooms      []struct {
				Name string `json:"name"`
			} `json:"rooms"`
			StartUnit int `json:"start_unit"`
			EndUnit   int `json:"end_unit"`
			Teachers  []struct {
				Name string `json:"name"`
			} `json:"teachers"`
			Date      string `json:"date"`
			Weekstate string `json:"weekstate"`
			Weeks     string `json:"weeks"`
		}

		if err := json.Unmarshal(raw, &items); err != nil {
			current = currentEnd.AddDate(0, 0, 1)
			continue
		}

		for _, item := range items {
			address := ""
			if len(item.Rooms) > 0 {
				address = item.Rooms[0].Name
			}

			teacherNames := make([]string, len(item.Teachers))
			for i, t := range item.Teachers {
				teacherNames[i] = t.Name
			}
			teacher := strings.Join(teacherNames, ", ")

			lessonID := strconv.Itoa(item.LessonID)

			dateTime, err := time.Parse("2006-01-02", item.Date)
			if err != nil {
				continue
			}
			weekday := int(dateTime.Weekday())
			if weekday == 0 {
				weekday = 7
			}

			periods := parseWeekstateAndWeeks(item.Weekstate, item.Weeks)
			for _, p := range periods {
				entry := courseEntry{
					CourseName: item.CourseName,
					LessonID:   lessonID,
					Address:    address,
					StartWeek:  p.StartWeek,
					EndWeek:    p.EndWeek,
					Week:       weekday,
					StartJie:   item.StartUnit,
					EndJie:     item.EndUnit,
					Teacher:    teacher,
					Type:       p.Type,
				}

				key := fmt.Sprintf("%s_%s_%d_%d_%d_%d_%d_%d",
					entry.CourseName, entry.LessonID,
					entry.StartWeek, entry.EndWeek, entry.Week,
					entry.StartJie, entry.EndJie, entry.Type)

				if !seen[key] {
					seen[key] = true
					courses = append(courses, entry)
				}
			}
		}

		current = currentEnd.AddDate(0, 0, 1)
	}

	result := map[string]interface{}{
		"data":      courses,
		"startDate": "2026-03-02",
	}

	return json.Marshal(result)
}

// weekPeriod represents a parsed week range with odd/even/all type.
type weekPeriod struct {
	StartWeek   int   `json:"start_week"`
	EndWeek     int   `json:"end_week"`
	Type        int   `json:"type"`
	ActualWeeks []int `json:"actual_weeks"`
}

// parseWeekstateAndWeeks mirrors the Python parse_weekstate_and_weeks function.
func parseWeekstateAndWeeks(weekstate, weeksStr string) []weekPeriod {
	weeksList := parseIntList(weeksStr)
	if len(weeksList) == 0 {
		return nil
	}

	// Simple case: no comma
	if !strings.Contains(weekstate, ",") {
		typeVal := 3
		if strings.Contains(weekstate, "单") {
			typeVal = 1
		} else if strings.Contains(weekstate, "双") {
			typeVal = 2
		}
		return []weekPeriod{{
			StartWeek:   minInt(weeksList),
			EndWeek:     maxInt(weeksList),
			Type:        typeVal,
			ActualWeeks: weeksList,
		}}
	}

	// Complex: split by comma
	parts := strings.Split(weekstate, ",")
	var result []weekPeriod
	rangeRe := regexp.MustCompile(`(\d+)-(\d+)`)
	singleRe := regexp.MustCompile(`(\d+)`)

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "-") {
			matches := rangeRe.FindStringSubmatch(part)
			if len(matches) < 3 {
				continue
			}
			startRange, _ := strconv.Atoi(matches[1])
			endRange, _ := strconv.Atoi(matches[2])

			typeVal := 3
			var actual []int

			if strings.Contains(part, "单") {
				typeVal = 1
				for _, w := range weeksList {
					if w >= startRange && w <= endRange && w%2 == 1 {
						actual = append(actual, w)
					}
				}
			} else if strings.Contains(part, "双") {
				typeVal = 2
				for _, w := range weeksList {
					if w >= startRange && w <= endRange && w%2 == 0 {
						actual = append(actual, w)
					}
				}
			} else {
				for _, w := range weeksList {
					if w >= startRange && w <= endRange {
						actual = append(actual, w)
					}
				}
			}

			if len(actual) > 0 {
				result = append(result, weekPeriod{
					StartWeek:   minInt(actual),
					EndWeek:     maxInt(actual),
					Type:        typeVal,
					ActualWeeks: actual,
				})
			}
		} else {
			matches := singleRe.FindStringSubmatch(part)
			if len(matches) < 2 {
				continue
			}
			weekNum, _ := strconv.Atoi(matches[1])
			found := false
			for _, w := range weeksList {
				if w == weekNum {
					found = true
					break
				}
			}
			if found {
				typeVal := 3
				if strings.Contains(part, "单") {
					typeVal = 1
				} else if strings.Contains(part, "双") {
					typeVal = 2
				}
				result = append(result, weekPeriod{
					StartWeek:   weekNum,
					EndWeek:     weekNum,
					Type:        typeVal,
					ActualWeeks: []int{weekNum},
				})
			}
		}
	}

	return result
}

func parseIntList(s string) []int {
	parts := strings.Split(s, ",")
	var result []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if v, err := strconv.Atoi(p); err == nil {
			result = append(result, v)
		}
	}
	sort.Ints(result)
	return result
}

func minInt(s []int) int {
	m := s[0]
	for _, v := range s[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxInt(s []int) int {
	m := s[0]
	for _, v := range s[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
