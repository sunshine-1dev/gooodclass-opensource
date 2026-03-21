package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/middleware"
	"gooodclass/internal/store"
)

type createQuestionRequest struct {
	UserID      string `json:"user_id"`
	Nickname    string `json:"nickname"`
	LessonID    string `json:"lesson_id"`
	CourseName  string `json:"course_name"`
	Teacher     string `json:"teacher"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	IsAnonymous bool   `json:"is_anonymous"`
}

type createAnswerRequest struct {
	UserID      string `json:"user_id"`
	Nickname    string `json:"nickname"`
	QuestionID  int64  `json:"question_id"`
	Content     string `json:"content"`
	IsAnonymous bool   `json:"is_anonymous"`
}

func GetQuestionsHandler(s *store.Store) gin.HandlerFunc {
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

		questions, err := s.GetQuestionsByLesson(lessonID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if currentUserID != "" {
			isAdmin := middleware.GetIsAdmin(c)
			ids := make([]int64, len(questions))
			for i := range questions {
				questions[i].IsOwn = questions[i].UserID == currentUserID || isAdmin
				ids[i] = questions[i].ID
			}
			if votes, err := s.GetUserVotes(currentUserID, "question", ids); err == nil {
				for i := range questions {
					questions[i].UserVote = votes[questions[i].ID]
				}
			}
		}

		c.JSON(http.StatusOK, questions)
	}
}

func GetQuestionDetailHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Query("id")
		questionID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || questionID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
			return
		}
		currentUserID := middleware.GetUserID(c)
		if currentUserID == "" {
			currentUserID = c.Query("user_id")
		}

		question, err := s.GetQuestionByID(questionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if question == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
			return
		}

		if currentUserID != "" {
			isAdmin := middleware.GetIsAdmin(c)
			question.IsOwn = question.UserID == currentUserID || isAdmin
			if votes, err := s.GetUserVotes(currentUserID, "question", []int64{questionID}); err == nil {
				question.UserVote = votes[questionID]
			}
		}

		answers, err := s.GetAnswersByQuestion(questionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if currentUserID != "" {
			isAdmin := middleware.GetIsAdmin(c)
			ids := make([]int64, len(answers))
			for i := range answers {
				answers[i].IsOwn = answers[i].UserID == currentUserID || isAdmin
				ids[i] = answers[i].ID
			}
			if votes, err := s.GetUserVotes(currentUserID, "answer", ids); err == nil {
				for i := range answers {
					answers[i].UserVote = votes[answers[i].ID]
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"question": question, "answers": answers})
	}
}

func CreateQuestionHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createQuestionRequest
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
		if req.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
			return
		}
		if len(req.Title) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title max 100 chars"})
			return
		}
		if len(req.Content) > 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content max 2000 chars"})
			return
		}

		q := store.CourseQuestion{
			UserID:      req.UserID,
			Nickname:    req.Nickname,
			LessonID:    req.LessonID,
			CourseName:  req.CourseName,
			Teacher:     req.Teacher,
			Title:       req.Title,
			Content:     req.Content,
			IsAnonymous: req.IsAnonymous,
			CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		}

		id, err := s.CreateQuestion(q)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success", "id": id})
	}
}

func DeleteQuestionHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		isAdmin := middleware.GetIsAdmin(c)
		idStr := c.Query("id")
		questionID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		if isAdmin {
			err = s.AdminDeleteQuestion(questionID)
		} else {
			err = s.DeleteQuestion(userID, questionID)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

func CreateAnswerHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createAnswerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		req.UserID = middleware.GetUserID(c)
		req.Nickname = middleware.GetNickname(c)
		if req.QuestionID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "question_id required"})
			return
		}
		if req.Content == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content required"})
			return
		}
		if len(req.Content) > 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content max 2000 chars"})
			return
		}

		a := store.CourseAnswer{
			QuestionID:  req.QuestionID,
			UserID:      req.UserID,
			Nickname:    req.Nickname,
			Content:     req.Content,
			IsAnonymous: req.IsAnonymous,
			CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		}

		id, err := s.CreateAnswer(a)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success", "id": id})
	}
}

func DeleteAnswerHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		isAdmin := middleware.GetIsAdmin(c)
		idStr := c.Query("id")
		answerID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		if isAdmin {
			err = s.AdminDeleteAnswer(answerID)
		} else {
			err = s.DeleteAnswer(userID, answerID)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}
