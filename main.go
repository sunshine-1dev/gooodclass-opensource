package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"gooodclass/internal/handler"
	"gooodclass/internal/jwgl"
	"gooodclass/internal/middleware"
	"gooodclass/internal/store"
)

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	s, err := store.New()
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer s.Close()

	if err := handler.InitMinio(); err != nil {
		log.Printf("warning: minio init failed (uploads disabled): %v", err)
	}

	if err := s.MigrateCampus(); err != nil {
		log.Fatalf("migrate campus: %v", err)
	}

	client := jwgl.NewClient()
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	}))

	cleanupStats := RegisterStatsRoutes(r, client)
	defer cleanupStats()

	r.GET("/isGood", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "isGood"})
	})

	api := r.Group("/api")
	{
		// --- Public routes (no auth required) ---
		api.GET("/login", StatsLoginMiddleware(client), handler.LoginHandler(client, s))
		api.GET("/image/*path", handler.ImageProxyHandler())
		api.GET("/overrideScript", handler.OverrideScriptHandler(filepath.Join(dataDir, "holiday.json")))
		api.GET("/redirect", handler.RedirectHandler())

		// Public read-only data (optional auth for is_own detection)
		optAuth := middleware.AuthOptional()
		api.GET("/courseReviewsTop", handler.TopCourseReviewsHandler(s))
		api.GET("/campusCategories", handler.GetCampusCategoriesHandler(s))
		api.GET("/campusItems", optAuth, handler.GetCampusItemsHandler(s))
		api.GET("/campusItemDetail", optAuth, handler.GetCampusItemDetailHandler(s))
		api.GET("/courseReviews", optAuth, handler.GetCourseReviewsHandler(s))
		api.GET("/questions", optAuth, handler.GetQuestionsHandler(s))
		api.GET("/questionDetail", optAuth, handler.GetQuestionDetailHandler(s))
		api.GET("/checkInRank", handler.CheckInRankHandler(s))
		api.GET("/check_in", handler.CheckInHandler(s))

		// Jwgl proxy routes — self-authenticated via username/password or token
		api.GET("/base", handler.ScheduleHandler(client))
		api.GET("/getExam", handler.ExamHandler(client))
		api.GET("/getGPA", handler.GPAHandler(client))
		api.GET("/getRank", handler.RankHandler(client))
		api.GET("/getPlan", handler.PlanHandler(client))
		api.GET("/getPlanCompletion", handler.PlanCompletionHandler(client))
		api.GET("/getUnscheduledCourses", handler.UnscheduledHandler(client))
		api.GET("/getStudents", handler.StudentsHandler(client))
		api.GET("/getEmptyRooms", handler.EmptyRoomHandler(client))
		api.GET("/getToken", handler.GetTokenHandler(client))

		// --- Protected routes (JWT auth required) ---
		auth := api.Group("", middleware.AuthRequired())
		{
			// User actions
			auth.POST("/upload", handler.UploadHandler())

			// Course review write operations
			auth.POST("/courseReview", handler.SubmitCourseReviewHandler(s))
			auth.GET("/myReviews", handler.MyReviewsHandler(s))
			auth.DELETE("/courseReview", handler.DeleteCourseReviewHandler(s))
			auth.POST("/ensureCourse", handler.EnsureCourseHandler(s))

			// Q&A write operations
			auth.POST("/question", handler.CreateQuestionHandler(s))
			auth.DELETE("/question", handler.DeleteQuestionHandler(s))
			auth.POST("/answer", handler.CreateAnswerHandler(s))
			auth.DELETE("/answer", handler.DeleteAnswerHandler(s))
			auth.POST("/vote", handler.VoteHandler(s))

			// Campus review write operations
			auth.POST("/campusItem", handler.CreateCampusItemHandler(s))
			auth.POST("/campusReview", handler.SubmitCampusReviewHandler(s))
			auth.DELETE("/campusReview", handler.DeleteCampusReviewHandler(s))
			auth.DELETE("/campusItem", handler.DeleteCampusItemHandler(s))
		}
	}

	log.Println("Starting GoodClass API server on :8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
