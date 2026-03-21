package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/middleware"
	"gooodclass/internal/store"
)

type submitReviewRequest struct {
	UserID      string `json:"user_id"`
	Nickname    string `json:"nickname"`
	LessonID    string `json:"lesson_id"`
	CourseName  string `json:"course_name"`
	Teacher     string `json:"teacher"`
	VibeLevel   int    `json:"vibe_level"`
	Comment     string `json:"comment"`
	Semester    string `json:"semester"`
	IsAnonymous bool   `json:"is_anonymous"`
}

func GetCourseReviewsHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		lessonID := c.Query("lesson_id")
		if lessonID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing lesson_id"})
			return
		}
		currentUserID := middleware.GetUserID(c)
		if currentUserID == "" {
			currentUserID = c.Query("user_id")
		}

		summary, err := s.GetRatingSummary(lessonID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		reviews, err := s.GetReviewsByLesson(lessonID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if currentUserID != "" {
			isAdmin := middleware.GetIsAdmin(c)
			ids := make([]int64, len(reviews))
			for i := range reviews {
				reviews[i].IsOwn = reviews[i].UserID == currentUserID || isAdmin
				ids[i] = reviews[i].ID
			}
			if votes, err := s.GetUserVotes(currentUserID, "review", ids); err == nil {
				for i := range reviews {
					reviews[i].UserVote = votes[reviews[i].ID]
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"summary": summary, "reviews": reviews})
	}
}

func SubmitCourseReviewHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req submitReviewRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		req.UserID = middleware.GetUserID(c)
		req.Nickname = middleware.GetNickname(c)
		if req.LessonID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lesson_id required"})
			return
		}
		if req.CourseName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "course_name required"})
			return
		}
		if req.VibeLevel < 1 || req.VibeLevel > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vibe_level must be 1-5"})
			return
		}
		if len(req.Comment) > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "comment max 500 chars"})
			return
		}

		review := store.CourseReview{
			UserID:      req.UserID,
			Nickname:    req.Nickname,
			LessonID:    req.LessonID,
			CourseName:  req.CourseName,
			Teacher:     req.Teacher,
			VibeLevel:   req.VibeLevel,
			Comment:     req.Comment,
			Semester:    req.Semester,
			IsAnonymous: req.IsAnonymous,
			CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		}

		if err := s.SubmitReview(review); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

func TopCourseReviewsHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		summaries, err := s.GetTopCourses()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, summaries)
	}
}

func MyReviewsHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetUserID(c)

		reviews, err := s.GetUserReviews(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for i := range reviews {
			reviews[i].IsOwn = true
		}

		c.JSON(http.StatusOK, reviews)
	}
}

func DeleteCourseReviewHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		isAdmin := middleware.GetIsAdmin(c)
		lessonID := c.Query("lesson_id")

		if lessonID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing lesson_id"})
			return
		}

		var err error
		if isAdmin {
			targetUserID := c.Query("user_id")
			if targetUserID == "" {
				targetUserID = userID
			}
			err = s.AdminDeleteReview(lessonID, targetUserID)
		} else {
			err = s.DeleteReview(userID, lessonID)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

type ensureCourseRequest struct {
	LessonID   string `json:"lesson_id"`
	CourseName string `json:"course_name"`
	Teacher    string `json:"teacher"`
}

func EnsureCourseHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ensureCourseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
		if req.LessonID == "" || req.CourseName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lesson_id and course_name required"})
			return
		}

		if err := s.EnsureCourse(req.LessonID, req.CourseName, req.Teacher); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}
