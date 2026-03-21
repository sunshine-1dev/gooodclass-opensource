// Package store provides MySQL-backed persistence for check-in and course review records.
package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type CheckInRecord struct {
	UserID      string `json:"user_id"`
	CheckInDate string `json:"check_in_date"`
	CheckInTime string `json:"check_in_time"`
}

type CourseReview struct {
	ID          int64  `json:"id"`
	UserID      string `json:"-"`
	Nickname    string `json:"nickname"`
	LessonID    string `json:"lesson_id"`
	CourseName  string `json:"course_name"`
	Teacher     string `json:"teacher"`
	VibeLevel   int    `json:"vibe_level"`
	Comment     string `json:"comment"`
	Semester    string `json:"semester"`
	IsAnonymous bool   `json:"is_anonymous"`
	CreatedAt   string `json:"created_at"`
	IsOwn       bool   `json:"is_own"`
	Likes       int    `json:"likes"`
	Dislikes    int    `json:"dislikes"`
	UserVote    int    `json:"user_vote"`
}

type CourseRatingSummary struct {
	LessonID        string  `json:"lesson_id"`
	CourseName      string  `json:"course_name"`
	Teacher         string  `json:"teacher"`
	AvgVibe         float64 `json:"avg_vibe"`
	TotalReviews    int     `json:"total_reviews"`
	TotalQuestions  int     `json:"total_questions"`
	RatingHistogram [5]int  `json:"rating_histogram"`
}

type CourseQuestion struct {
	ID          int64  `json:"id"`
	UserID      string `json:"-"`
	Nickname    string `json:"nickname"`
	LessonID    string `json:"lesson_id"`
	CourseName  string `json:"course_name"`
	Teacher     string `json:"teacher"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	IsAnonymous bool   `json:"is_anonymous"`
	CreatedAt   string `json:"created_at"`
	AnswerCount int    `json:"answer_count"`
	IsOwn       bool   `json:"is_own"`
	Likes       int    `json:"likes"`
	Dislikes    int    `json:"dislikes"`
	UserVote    int    `json:"user_vote"`
}

type CourseAnswer struct {
	ID          int64  `json:"id"`
	QuestionID  int64  `json:"question_id"`
	UserID      string `json:"-"`
	Nickname    string `json:"nickname"`
	Content     string `json:"content"`
	IsAnonymous bool   `json:"is_anonymous"`
	CreatedAt   string `json:"created_at"`
	IsOwn       bool   `json:"is_own"`
	Likes       int    `json:"likes"`
	Dislikes    int    `json:"dislikes"`
	UserVote    int    `json:"user_vote"`
}

type Store struct {
	db *sql.DB
}

// New opens a MySQL connection pool and initializes the schema.
func New() (*Store, error) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "root:dadada123@tcp(47.96.80.84:3306)/gooodclass?charset=utf8mb4&parseTime=true&loc=Local&timeout=10s&readTimeout=10s&writeTimeout=10s"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	log.Println("[store] MySQL connection pool initialized")

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS checkin (
			id           INT AUTO_INCREMENT PRIMARY KEY,
			user_id      VARCHAR(64)  NOT NULL,
			checkin_date VARCHAR(16)  NOT NULL,
			checkin_time VARCHAR(32)  NOT NULL,
			UNIQUE KEY uq_user_date (user_id, checkin_date),
			INDEX idx_checkin_date (checkin_date)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS course_review (
			id           INT AUTO_INCREMENT PRIMARY KEY,
			user_id      VARCHAR(64)  NOT NULL,
			lesson_id    VARCHAR(64)  NOT NULL,
			course_name  VARCHAR(255) NOT NULL,
			teacher      VARCHAR(255) NOT NULL DEFAULT '',
			vibe_level   INT          NOT NULL,
			comment      TEXT         NOT NULL,
			semester     VARCHAR(32)  NOT NULL DEFAULT '',
			is_anonymous TINYINT(1)   NOT NULL DEFAULT 1,
			created_at   VARCHAR(32)  NOT NULL,
			UNIQUE KEY uq_user_lesson (user_id, lesson_id),
			INDEX idx_review_lesson (lesson_id),
			INDEX idx_review_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS course_question (
			id           INT AUTO_INCREMENT PRIMARY KEY,
			user_id      VARCHAR(64)  NOT NULL,
			lesson_id    VARCHAR(255) NOT NULL,
			course_name  VARCHAR(255) NOT NULL,
			teacher      VARCHAR(255) NOT NULL DEFAULT '',
			title        VARCHAR(255) NOT NULL,
			content      TEXT         NOT NULL,
			is_anonymous TINYINT(1)   NOT NULL DEFAULT 1,
			created_at   VARCHAR(32)  NOT NULL,
			INDEX idx_question_lesson (lesson_id),
			INDEX idx_question_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS course_answer (
			id           INT AUTO_INCREMENT PRIMARY KEY,
			question_id  INT          NOT NULL,
			user_id      VARCHAR(64)  NOT NULL,
			content      TEXT         NOT NULL,
			is_anonymous TINYINT(1)   NOT NULL DEFAULT 1,
			created_at   VARCHAR(32)  NOT NULL,
			INDEX idx_answer_question (question_id),
			INDEX idx_answer_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vote (
			id           INT AUTO_INCREMENT PRIMARY KEY,
			user_id      VARCHAR(64)  NOT NULL,
			target_type  VARCHAR(16)  NOT NULL,
			target_id    INT          NOT NULL,
			vote_value   TINYINT      NOT NULL,
			UNIQUE KEY uq_user_target (user_id, target_type, target_id),
			INDEX idx_vote_target (target_type, target_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	addNicknameColumns := []string{
		"ALTER TABLE course_review ADD COLUMN nickname VARCHAR(64) NOT NULL DEFAULT ''",
		"ALTER TABLE course_question ADD COLUMN nickname VARCHAR(64) NOT NULL DEFAULT ''",
		"ALTER TABLE course_answer ADD COLUMN nickname VARCHAR(64) NOT NULL DEFAULT ''",
	}
	for _, stmt := range addNicknameColumns {
		db.Exec(stmt) // ignore "duplicate column" errors
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS course_catalog (
			lesson_id    VARCHAR(255) NOT NULL,
			course_name  VARCHAR(255) NOT NULL,
			teacher      VARCHAR(255) NOT NULL DEFAULT '',
			created_at   VARCHAR(32)  NOT NULL,
			PRIMARY KEY (lesson_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS admin (
			user_id VARCHAR(64) NOT NULL PRIMARY KEY
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) IsAdmin(userID string) bool {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM admin WHERE user_id = ?", userID).Scan(&count)
	return err == nil && count > 0
}

// CheckIn records a check-in for the user today.
// Returns (rank, alreadyCheckedIn, error).
func (s *Store) CheckIn(userID string) (int, bool, error) {
	today := time.Now().Format("2006-01-02")
	now := time.Now().Format("2006-01-02 15:04:05")

	// INSERT IGNORE: if duplicate (user_id, checkin_date), silently skip
	result, err := s.db.Exec(
		"INSERT IGNORE INTO checkin (user_id, checkin_date, checkin_time) VALUES (?, ?, ?)",
		userID, today, now,
	)
	if err != nil {
		return 0, false, fmt.Errorf("insert checkin: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	alreadyExisted := rowsAffected == 0

	var count int
	err = s.db.QueryRow(
		"SELECT COUNT(*) FROM checkin WHERE checkin_date = ?", today,
	).Scan(&count)
	if err != nil {
		return 0, false, fmt.Errorf("count checkin: %w", err)
	}

	return count, alreadyExisted, nil
}

// TodayRankings returns all check-in records for today, sorted by check-in time.
func (s *Store) TodayRankings() ([]CheckInRecord, error) {
	today := time.Now().Format("2006-01-02")

	rows, err := s.db.Query(
		"SELECT user_id, checkin_date, checkin_time FROM checkin WHERE checkin_date = ? ORDER BY checkin_time ASC",
		today,
	)
	if err != nil {
		return nil, fmt.Errorf("query rankings: %w", err)
	}
	defer rows.Close()

	records := make([]CheckInRecord, 0)
	for rows.Next() {
		var r CheckInRecord
		if err := rows.Scan(&r.UserID, &r.CheckInDate, &r.CheckInTime); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// EnsureCourse inserts a course into the catalog if it doesn't already exist.
func (s *Store) EnsureCourse(lessonID, courseName, teacher string) error {
	_, err := s.db.Exec(`
		INSERT IGNORE INTO course_catalog (lesson_id, course_name, teacher, created_at)
		VALUES (?, ?, ?, NOW())`,
		lessonID, courseName, teacher,
	)
	if err != nil {
		return fmt.Errorf("ensure course: %w", err)
	}
	return nil
}

// SubmitReview inserts or replaces a course review.
func (s *Store) SubmitReview(r CourseReview) error {
	_, err := s.db.Exec(`
		REPLACE INTO course_review
		(user_id, nickname, lesson_id, course_name, teacher, vibe_level, comment, semester, is_anonymous, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.UserID, r.Nickname, r.LessonID, r.CourseName, r.Teacher, r.VibeLevel, r.Comment, r.Semester, r.IsAnonymous, r.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("submit review: %w", err)
	}
	return nil
}

func (s *Store) GetReviewsByLesson(lessonID string) ([]CourseReview, error) {
	rows, err := s.db.Query(`
		SELECT r.id, r.user_id, r.nickname, r.lesson_id, r.course_name, r.teacher, r.vibe_level, r.comment, r.semester, r.is_anonymous, r.created_at,
			COALESCE((SELECT SUM(vote_value = 1) FROM vote WHERE target_type = 'review' AND target_id = r.id), 0),
			COALESCE((SELECT SUM(vote_value = -1) FROM vote WHERE target_type = 'review' AND target_id = r.id), 0)
		FROM course_review r WHERE r.lesson_id = ?
		ORDER BY r.created_at DESC`, lessonID,
	)
	if err != nil {
		return nil, fmt.Errorf("query reviews by lesson: %w", err)
	}
	defer rows.Close()

	reviews := make([]CourseReview, 0)
	for rows.Next() {
		var r CourseReview
		if err := rows.Scan(&r.ID, &r.UserID, &r.Nickname, &r.LessonID, &r.CourseName, &r.Teacher, &r.VibeLevel, &r.Comment, &r.Semester, &r.IsAnonymous, &r.CreatedAt, &r.Likes, &r.Dislikes); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		if r.IsAnonymous {
			r.Nickname = ""
		}
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}

func (s *Store) GetRatingSummary(lessonID string) (*CourseRatingSummary, error) {
	var summary CourseRatingSummary

	err := s.db.QueryRow(`
		SELECT lesson_id, ANY_VALUE(course_name), ANY_VALUE(teacher), AVG(vibe_level), COUNT(*),
			COALESCE((SELECT COUNT(*) FROM course_question q WHERE q.lesson_id = ?), 0)
		FROM course_review WHERE lesson_id = ? GROUP BY lesson_id`, lessonID, lessonID,
	).Scan(&summary.LessonID, &summary.CourseName, &summary.Teacher, &summary.AvgVibe, &summary.TotalReviews, &summary.TotalQuestions)

	if err != nil {
		if err == sql.ErrNoRows {
			// No reviews yet — try catalog fallback
			var cat CourseRatingSummary
			catErr := s.db.QueryRow(`
				SELECT c.lesson_id, c.course_name, c.teacher,
					COALESCE((SELECT COUNT(*) FROM course_question q WHERE q.lesson_id = c.lesson_id), 0)
				FROM course_catalog c WHERE c.lesson_id = ?`, lessonID,
			).Scan(&cat.LessonID, &cat.CourseName, &cat.Teacher, &cat.TotalQuestions)
			if catErr != nil {
				return nil, nil
			}
			return &cat, nil
		}
		return nil, fmt.Errorf("query rating summary: %w", err)
	}

	err = s.db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN vibe_level = 1 THEN 1 END),
			COUNT(CASE WHEN vibe_level = 2 THEN 1 END),
			COUNT(CASE WHEN vibe_level = 3 THEN 1 END),
			COUNT(CASE WHEN vibe_level = 4 THEN 1 END),
			COUNT(CASE WHEN vibe_level = 5 THEN 1 END)
		FROM course_review WHERE lesson_id = ?`, lessonID,
	).Scan(&summary.RatingHistogram[0], &summary.RatingHistogram[1], &summary.RatingHistogram[2], &summary.RatingHistogram[3], &summary.RatingHistogram[4])
	if err != nil {
		return nil, fmt.Errorf("query histogram: %w", err)
	}

	return &summary, nil
}

func (s *Store) GetTopCourses() ([]CourseRatingSummary, error) {
	// Bayesian average: (C * M + SUM) / (C + N)
	// C = global avg review count (weight), M = global avg rating
	// Courses with few reviews get pulled toward the global mean.
	rows, err := s.db.Query(`
		SELECT lesson_id, course_name, teacher, avg_vibe, total_reviews, total_questions FROM (
			SELECT r.lesson_id, ANY_VALUE(r.course_name) AS course_name, ANY_VALUE(r.teacher) AS teacher,
				AVG(r.vibe_level) AS avg_vibe, COUNT(*) AS total_reviews,
				SUM(r.vibe_level) AS sum_vibe,
				COALESCE((SELECT COUNT(*) FROM course_question q WHERE q.lesson_id = r.lesson_id), 0) AS total_questions
			FROM course_review r GROUP BY r.lesson_id
			UNION ALL
			SELECT c.lesson_id, c.course_name, c.teacher, 0, 0, 0,
				COALESCE((SELECT COUNT(*) FROM course_question q WHERE q.lesson_id = c.lesson_id), 0)
			FROM course_catalog c WHERE c.lesson_id NOT IN (SELECT DISTINCT lesson_id FROM course_review)
		) combined
		ORDER BY
			total_reviews DESC,
			avg_vibe DESC,
			total_questions DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query top courses: %w", err)
	}
	defer rows.Close()

	summaries := make([]CourseRatingSummary, 0)
	for rows.Next() {
		var s CourseRatingSummary
		if err := rows.Scan(&s.LessonID, &s.CourseName, &s.Teacher, &s.AvgVibe, &s.TotalReviews, &s.TotalQuestions); err != nil {
			return nil, fmt.Errorf("scan top course: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

func (s *Store) GetUserReviews(userID string) ([]CourseReview, error) {
	rows, err := s.db.Query(`
		SELECT r.id, r.user_id, r.nickname, r.lesson_id, r.course_name, r.teacher, r.vibe_level, r.comment, r.semester, r.is_anonymous, r.created_at,
			COALESCE((SELECT SUM(vote_value = 1) FROM vote WHERE target_type = 'review' AND target_id = r.id), 0),
			COALESCE((SELECT SUM(vote_value = -1) FROM vote WHERE target_type = 'review' AND target_id = r.id), 0)
		FROM course_review r WHERE r.user_id = ? ORDER BY r.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query user reviews: %w", err)
	}
	defer rows.Close()

	reviews := make([]CourseReview, 0)
	for rows.Next() {
		var r CourseReview
		if err := rows.Scan(&r.ID, &r.UserID, &r.Nickname, &r.LessonID, &r.CourseName, &r.Teacher, &r.VibeLevel, &r.Comment, &r.Semester, &r.IsAnonymous, &r.CreatedAt, &r.Likes, &r.Dislikes); err != nil {
			return nil, fmt.Errorf("scan user review: %w", err)
		}
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}

func (s *Store) DeleteReview(userID, lessonID string) error {
	_, err := s.db.Exec("DELETE FROM course_review WHERE user_id = ? AND lesson_id = ?", userID, lessonID)
	if err != nil {
		return fmt.Errorf("delete review: %w", err)
	}
	return nil
}

func (s *Store) AdminDeleteReview(lessonID, targetUserID string) error {
	_, err := s.db.Exec("DELETE FROM course_review WHERE lesson_id = ? AND user_id = ?", lessonID, targetUserID)
	if err != nil {
		return fmt.Errorf("admin delete review: %w", err)
	}
	return nil
}

// === Q&A ===

func (s *Store) CreateQuestion(q CourseQuestion) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO course_question (user_id, nickname, lesson_id, course_name, teacher, title, content, is_anonymous, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		q.UserID, q.Nickname, q.LessonID, q.CourseName, q.Teacher, q.Title, q.Content, q.IsAnonymous, q.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("create question: %w", err)
	}
	return result.LastInsertId()
}

func (s *Store) GetQuestionsByLesson(lessonID string) ([]CourseQuestion, error) {
	rows, err := s.db.Query(`
		SELECT q.id, q.user_id, q.nickname, q.lesson_id, q.course_name, q.teacher, q.title, q.content, q.is_anonymous, q.created_at,
			(SELECT COUNT(*) FROM course_answer a WHERE a.question_id = q.id),
			COALESCE((SELECT SUM(vote_value = 1) FROM vote WHERE target_type = 'question' AND target_id = q.id), 0),
			COALESCE((SELECT SUM(vote_value = -1) FROM vote WHERE target_type = 'question' AND target_id = q.id), 0)
		FROM course_question q WHERE q.lesson_id = ? ORDER BY q.created_at DESC`, lessonID,
	)
	if err != nil {
		return nil, fmt.Errorf("query questions: %w", err)
	}
	defer rows.Close()

	questions := make([]CourseQuestion, 0)
	for rows.Next() {
		var q CourseQuestion
		if err := rows.Scan(&q.ID, &q.UserID, &q.Nickname, &q.LessonID, &q.CourseName, &q.Teacher, &q.Title, &q.Content, &q.IsAnonymous, &q.CreatedAt, &q.AnswerCount, &q.Likes, &q.Dislikes); err != nil {
			return nil, fmt.Errorf("scan question: %w", err)
		}
		if q.IsAnonymous {
			q.Nickname = ""
		}
		questions = append(questions, q)
	}
	return questions, rows.Err()
}

func (s *Store) GetQuestionByID(questionID int64) (*CourseQuestion, error) {
	var q CourseQuestion
	err := s.db.QueryRow(`
		SELECT q.id, q.user_id, q.nickname, q.lesson_id, q.course_name, q.teacher, q.title, q.content, q.is_anonymous, q.created_at,
			(SELECT COUNT(*) FROM course_answer a WHERE a.question_id = q.id),
			COALESCE((SELECT SUM(vote_value = 1) FROM vote WHERE target_type = 'question' AND target_id = q.id), 0),
			COALESCE((SELECT SUM(vote_value = -1) FROM vote WHERE target_type = 'question' AND target_id = q.id), 0)
		FROM course_question q WHERE q.id = ?`, questionID,
	).Scan(&q.ID, &q.UserID, &q.Nickname, &q.LessonID, &q.CourseName, &q.Teacher, &q.Title, &q.Content, &q.IsAnonymous, &q.CreatedAt, &q.AnswerCount, &q.Likes, &q.Dislikes)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get question: %w", err)
	}
	if q.IsAnonymous {
		q.Nickname = ""
	}
	return &q, nil
}

func (s *Store) DeleteQuestion(userID string, questionID int64) error {
	_, err := s.db.Exec("DELETE FROM course_answer WHERE question_id = ?", questionID)
	if err != nil {
		return fmt.Errorf("delete answers for question: %w", err)
	}
	_, err = s.db.Exec("DELETE FROM course_question WHERE id = ? AND user_id = ?", questionID, userID)
	if err != nil {
		return fmt.Errorf("delete question: %w", err)
	}
	return nil
}

func (s *Store) AdminDeleteQuestion(questionID int64) error {
	_, err := s.db.Exec("DELETE FROM course_answer WHERE question_id = ?", questionID)
	if err != nil {
		return fmt.Errorf("delete answers for question: %w", err)
	}
	_, err = s.db.Exec("DELETE FROM course_question WHERE id = ?", questionID)
	if err != nil {
		return fmt.Errorf("admin delete question: %w", err)
	}
	return nil
}

func (s *Store) CreateAnswer(a CourseAnswer) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO course_answer (question_id, user_id, nickname, content, is_anonymous, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		a.QuestionID, a.UserID, a.Nickname, a.Content, a.IsAnonymous, a.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("create answer: %w", err)
	}
	return result.LastInsertId()
}

func (s *Store) GetAnswersByQuestion(questionID int64) ([]CourseAnswer, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.question_id, a.user_id, a.nickname, a.content, a.is_anonymous, a.created_at,
			COALESCE((SELECT SUM(vote_value = 1) FROM vote WHERE target_type = 'answer' AND target_id = a.id), 0),
			COALESCE((SELECT SUM(vote_value = -1) FROM vote WHERE target_type = 'answer' AND target_id = a.id), 0)
		FROM course_answer a WHERE a.question_id = ? ORDER BY a.created_at ASC`, questionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query answers: %w", err)
	}
	defer rows.Close()

	answers := make([]CourseAnswer, 0)
	for rows.Next() {
		var a CourseAnswer
		if err := rows.Scan(&a.ID, &a.QuestionID, &a.UserID, &a.Nickname, &a.Content, &a.IsAnonymous, &a.CreatedAt, &a.Likes, &a.Dislikes); err != nil {
			return nil, fmt.Errorf("scan answer: %w", err)
		}
		if a.IsAnonymous {
			a.Nickname = ""
		}
		answers = append(answers, a)
	}
	return answers, rows.Err()
}

func (s *Store) DeleteAnswer(userID string, answerID int64) error {
	_, err := s.db.Exec("DELETE FROM course_answer WHERE id = ? AND user_id = ?", answerID, userID)
	if err != nil {
		return fmt.Errorf("delete answer: %w", err)
	}
	return nil
}

func (s *Store) AdminDeleteAnswer(answerID int64) error {
	_, err := s.db.Exec("DELETE FROM course_answer WHERE id = ?", answerID)
	if err != nil {
		return fmt.Errorf("admin delete answer: %w", err)
	}
	return nil
}

// === Votes ===

func (s *Store) Vote(userID, targetType string, targetID int64, value int) error {
	if value == 0 {
		_, err := s.db.Exec("DELETE FROM vote WHERE user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID)
		return err
	}
	_, err := s.db.Exec(`
		INSERT INTO vote (user_id, target_type, target_id, vote_value) VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE vote_value = ?`,
		userID, targetType, targetID, value, value,
	)
	return err
}

func (s *Store) GetUserVotes(userID, targetType string, targetIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int)
	if len(targetIDs) == 0 || userID == "" {
		return result, nil
	}

	query := "SELECT target_id, vote_value FROM vote WHERE user_id = ? AND target_type = ? AND target_id IN ("
	args := []interface{}{userID, targetType}
	for i, id := range targetIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var targetID int64
		var value int
		if err := rows.Scan(&targetID, &value); err != nil {
			return nil, err
		}
		result[targetID] = value
	}
	return result, rows.Err()
}
