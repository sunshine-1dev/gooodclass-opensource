package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/middleware"
	"gooodclass/internal/store"
)

func GetCampusCategoriesHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		cats, err := s.GetCampusCategories()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cats)
	}
}

func GetCampusItemsHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.Query("category")

		var items []store.CampusItemSummary
		var err error

		if category == "" || category == "all" {
			items, err = s.GetAllCampusItems()
		} else {
			items, err = s.GetCampusItemsByCategory(category)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		currentUserID := middleware.GetUserID(c)
		if currentUserID == "" {
			currentUserID = c.Query("user_id")
		}
		if currentUserID != "" {
			isAdmin := middleware.GetIsAdmin(c)
			for i := range items {
				items[i].IsOwn = items[i].CreatedBy == currentUserID || isAdmin
			}
		}

		c.JSON(http.StatusOK, items)
	}
}

func GetCampusItemDetailHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Query("id")
		itemID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || itemID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		currentUserID := middleware.GetUserID(c)
		if currentUserID == "" {
			currentUserID = c.Query("user_id")
		}

		summary, err := s.GetCampusItemSummary(itemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		reviews, err := s.GetCampusReviews(itemID)
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
			if votes, err := s.GetUserVotes(currentUserID, "campus_review", ids); err == nil {
				for i := range reviews {
					reviews[i].UserVote = votes[reviews[i].ID]
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"summary": summary, "reviews": reviews})
	}
}

type submitCampusReviewRequest struct {
	ItemID      int64  `json:"item_id"`
	UserID      string `json:"user_id"`
	Nickname    string `json:"nickname"`
	Rating      int    `json:"rating"`
	Comment     string `json:"comment"`
	ImageURL    string `json:"image_url"`
	IsAnonymous bool   `json:"is_anonymous"`
}

func SubmitCampusReviewHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req submitCampusReviewRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid JSON: %v", err)})
			return
		}
		req.UserID = middleware.GetUserID(c)
		req.Nickname = middleware.GetNickname(c)
		if req.ItemID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "item_id required"})
			return
		}
		if req.Rating < 1 || req.Rating > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "rating must be 1-5"})
			return
		}

		review := store.CampusReview{
			ItemID:      req.ItemID,
			UserID:      req.UserID,
			Nickname:    req.Nickname,
			Rating:      req.Rating,
			Comment:     req.Comment,
			ImageURL:    req.ImageURL,
			IsAnonymous: req.IsAnonymous,
			CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		}

		if err := s.SubmitCampusReview(review); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

type createCampusItemRequest struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Location string `json:"location"`
	ImageURL string `json:"image_url"`
	UserID   string `json:"user_id"`
}

func CreateCampusItemHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createCampusItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
		req.UserID = middleware.GetUserID(c)
		if req.Category == "" || req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category and name required"})
			return
		}

		item := store.CampusItem{
			Category:  req.Category,
			Name:      req.Name,
			Location:  req.Location,
			ImageURL:  req.ImageURL,
			CreatedBy: req.UserID,
			CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		}

		id, err := s.CreateCampusItem(item)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": id})
	}
}

func DeleteCampusReviewHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		isAdmin := middleware.GetIsAdmin(c)
		idStr := c.Query("id")
		reviewID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		if isAdmin {
			err = s.AdminDeleteCampusReview(reviewID)
		} else {
			err = s.DeleteCampusReview(userID, reviewID)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

func DeleteCampusItemHandler(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		isAdmin := middleware.GetIsAdmin(c)
		idStr := c.Query("id")
		itemID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || itemID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		if isAdmin {
			err = s.AdminDeleteCampusItem(itemID)
		} else {
			err = s.DeleteCampusItem(userID, itemID)
		}

		if err != nil {
			if err.Error() == "forbidden: not the creator" {
				c.JSON(http.StatusForbidden, gin.H{"error": "只能删除自己创建的条目"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}
