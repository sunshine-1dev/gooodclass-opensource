package store

import "fmt"

type CampusCategory struct {
	ID        int64  `json:"id"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	SortOrder int    `json:"sort_order"`
}

type CampusItem struct {
	ID        int64  `json:"id"`
	Category  string `json:"category"`
	Name      string `json:"name"`
	Location  string `json:"location"`
	ImageURL  string `json:"image_url"`
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at"`
}

type CampusReview struct {
	ID          int64  `json:"id"`
	ItemID      int64  `json:"item_id"`
	UserID      string `json:"-"`
	Nickname    string `json:"nickname"`
	Rating      int    `json:"rating"`
	Comment     string `json:"comment"`
	ImageURL    string `json:"image_url"`
	IsAnonymous bool   `json:"is_anonymous"`
	CreatedAt   string `json:"created_at"`
	IsOwn       bool   `json:"is_own"`
	Likes       int    `json:"likes"`
	Dislikes    int    `json:"dislikes"`
	UserVote    int    `json:"user_vote"`
}

type CampusItemSummary struct {
	ID              int64   `json:"id"`
	Category        string  `json:"category"`
	Name            string  `json:"name"`
	Location        string  `json:"location"`
	ImageURL        string  `json:"image_url"`
	CreatedBy       string  `json:"-"`
	IsOwn           bool    `json:"is_own"`
	AvgRating       float64 `json:"avg_rating"`
	TotalReviews    int     `json:"total_reviews"`
	RatingHistogram [5]int  `json:"rating_histogram"`
}

func (s *Store) MigrateCampus() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS campus_item (
			id           INT AUTO_INCREMENT PRIMARY KEY,
			category     VARCHAR(32)  NOT NULL,
			name         VARCHAR(255) NOT NULL,
			location     VARCHAR(255) NOT NULL DEFAULT '',
			image_url    VARCHAR(512) NOT NULL DEFAULT '',
			created_at   VARCHAR(32)  NOT NULL,
			UNIQUE KEY uq_category_name (category, name),
			INDEX idx_campus_item_category (category)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return fmt.Errorf("create campus_item: %w", err)
	}

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS campus_review (
			id           INT AUTO_INCREMENT PRIMARY KEY,
			item_id      INT          NOT NULL,
			user_id      VARCHAR(64)  NOT NULL,
			nickname     VARCHAR(64)  NOT NULL DEFAULT '',
			rating       INT          NOT NULL,
			comment      TEXT         NOT NULL,
			image_url    VARCHAR(512) NOT NULL DEFAULT '',
			is_anonymous TINYINT(1)   NOT NULL DEFAULT 1,
			created_at   VARCHAR(32)  NOT NULL,
			INDEX idx_campus_review_item (item_id),
			INDEX idx_campus_review_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return fmt.Errorf("create campus_review: %w", err)
	}

	s.db.Exec("ALTER TABLE campus_item ADD COLUMN created_by VARCHAR(64) NOT NULL DEFAULT ''")

	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS campus_category (
			id         INT AUTO_INCREMENT PRIMARY KEY,
			cat_key    VARCHAR(32)  NOT NULL,
			label      VARCHAR(64)  NOT NULL,
			sort_order INT          NOT NULL DEFAULT 0,
			UNIQUE KEY uq_cat_key (cat_key)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return fmt.Errorf("create campus_category: %w", err)
	}

	// Seed default categories if table is empty
	var catCount int
	s.db.QueryRow("SELECT COUNT(*) FROM campus_category").Scan(&catCount)
	if catCount == 0 {
		defaults := []struct {
			Key   string
			Label string
			Order int
		}{
			{"canteen", "食堂", 1},
			{"food", "美食", 2},
			{"facility", "设施", 3},
			{"cat", "猫猫", 4},
			{"scenery", "风景", 5},
		}
		for _, d := range defaults {
			s.db.Exec("INSERT IGNORE INTO campus_category (cat_key, label, sort_order) VALUES (?, ?, ?)", d.Key, d.Label, d.Order)
		}
	}

	return nil
}

func (s *Store) GetCampusCategories() ([]CampusCategory, error) {
	rows, err := s.db.Query("SELECT id, cat_key, label, sort_order FROM campus_category ORDER BY sort_order")
	if err != nil {
		return nil, fmt.Errorf("query campus categories: %w", err)
	}
	defer rows.Close()

	cats := make([]CampusCategory, 0)
	for rows.Next() {
		var c CampusCategory
		if err := rows.Scan(&c.ID, &c.Key, &c.Label, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("scan campus category: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (s *Store) GetCampusItemsByCategory(category string) ([]CampusItemSummary, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.category, c.name, c.location, c.image_url, c.created_by,
			COALESCE((SELECT AVG(r.rating) FROM campus_review r WHERE r.item_id = c.id), 0) AS avg_r,
			COALESCE((SELECT COUNT(*) FROM campus_review r WHERE r.item_id = c.id), 0) AS cnt,
			COALESCE((SELECT SUM(r.rating) FROM campus_review r WHERE r.item_id = c.id), 0) AS sum_r
		FROM campus_item c WHERE c.category = ?
		ORDER BY
			cnt DESC,
			avg_r DESC,
			c.name`, category,
	)
	if err != nil {
		return nil, fmt.Errorf("query campus items: %w", err)
	}
	defer rows.Close()

	items := make([]CampusItemSummary, 0)
	for rows.Next() {
		var item CampusItemSummary
		var sumR float64
		if err := rows.Scan(&item.ID, &item.Category, &item.Name, &item.Location, &item.ImageURL, &item.CreatedBy, &item.AvgRating, &item.TotalReviews, &sumR); err != nil {
			return nil, fmt.Errorf("scan campus item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetAllCampusItems() ([]CampusItemSummary, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.category, c.name, c.location, c.image_url, c.created_by,
			COALESCE((SELECT AVG(r.rating) FROM campus_review r WHERE r.item_id = c.id), 0) AS avg_r,
			COALESCE((SELECT COUNT(*) FROM campus_review r WHERE r.item_id = c.id), 0) AS cnt,
			COALESCE((SELECT SUM(r.rating) FROM campus_review r WHERE r.item_id = c.id), 0) AS sum_r
		FROM campus_item c
		ORDER BY
			cnt DESC,
			avg_r DESC,
			c.name`,
	)
	if err != nil {
		return nil, fmt.Errorf("query all campus items: %w", err)
	}
	defer rows.Close()

	items := make([]CampusItemSummary, 0)
	for rows.Next() {
		var item CampusItemSummary
		var sumR float64
		if err := rows.Scan(&item.ID, &item.Category, &item.Name, &item.Location, &item.ImageURL, &item.CreatedBy, &item.AvgRating, &item.TotalReviews, &sumR); err != nil {
			return nil, fmt.Errorf("scan campus item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetCampusItemSummary(itemID int64) (*CampusItemSummary, error) {
	var item CampusItemSummary
	err := s.db.QueryRow(`
		SELECT c.id, c.category, c.name, c.location, c.image_url,
			COALESCE((SELECT AVG(r.rating) FROM campus_review r WHERE r.item_id = c.id), 0),
			COALESCE((SELECT COUNT(*) FROM campus_review r WHERE r.item_id = c.id), 0)
		FROM campus_item c WHERE c.id = ?`, itemID,
	).Scan(&item.ID, &item.Category, &item.Name, &item.Location, &item.ImageURL, &item.AvgRating, &item.TotalReviews)
	if err != nil {
		return nil, fmt.Errorf("get campus item summary: %w", err)
	}

	err = s.db.QueryRow(`
		SELECT
			COUNT(CASE WHEN rating = 1 THEN 1 END),
			COUNT(CASE WHEN rating = 2 THEN 1 END),
			COUNT(CASE WHEN rating = 3 THEN 1 END),
			COUNT(CASE WHEN rating = 4 THEN 1 END),
			COUNT(CASE WHEN rating = 5 THEN 1 END)
		FROM campus_review WHERE item_id = ?`, itemID,
	).Scan(&item.RatingHistogram[0], &item.RatingHistogram[1], &item.RatingHistogram[2], &item.RatingHistogram[3], &item.RatingHistogram[4])
	if err != nil {
		return nil, fmt.Errorf("query campus histogram: %w", err)
	}

	return &item, nil
}

func (s *Store) GetCampusReviews(itemID int64) ([]CampusReview, error) {
	rows, err := s.db.Query(`
		SELECT r.id, r.item_id, r.user_id, r.nickname, r.rating, r.comment, r.image_url, r.is_anonymous, r.created_at,
			COALESCE((SELECT SUM(vote_value = 1) FROM vote WHERE target_type = 'campus_review' AND target_id = r.id), 0),
			COALESCE((SELECT SUM(vote_value = -1) FROM vote WHERE target_type = 'campus_review' AND target_id = r.id), 0)
		FROM campus_review r WHERE r.item_id = ?
		ORDER BY r.created_at DESC`, itemID,
	)
	if err != nil {
		return nil, fmt.Errorf("query campus reviews: %w", err)
	}
	defer rows.Close()

	reviews := make([]CampusReview, 0)
	for rows.Next() {
		var r CampusReview
		if err := rows.Scan(&r.ID, &r.ItemID, &r.UserID, &r.Nickname, &r.Rating, &r.Comment, &r.ImageURL, &r.IsAnonymous, &r.CreatedAt, &r.Likes, &r.Dislikes); err != nil {
			return nil, fmt.Errorf("scan campus review: %w", err)
		}
		if r.IsAnonymous {
			r.Nickname = ""
		}
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}

func (s *Store) SubmitCampusReview(r CampusReview) error {
	_, err := s.db.Exec(`
		INSERT INTO campus_review (item_id, user_id, nickname, rating, comment, image_url, is_anonymous, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ItemID, r.UserID, r.Nickname, r.Rating, r.Comment, r.ImageURL, r.IsAnonymous, r.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("submit campus review: %w", err)
	}
	return nil
}

func (s *Store) CreateCampusItem(item CampusItem) (int64, error) {
	result, err := s.db.Exec(`
		INSERT IGNORE INTO campus_item (category, name, location, image_url, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		item.Category, item.Name, item.Location, item.ImageURL, item.CreatedBy, item.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("create campus item: %w", err)
	}
	return result.LastInsertId()
}

func (s *Store) DeleteCampusReview(userID string, reviewID int64) error {
	_, err := s.db.Exec("DELETE FROM campus_review WHERE id = ? AND user_id = ?", reviewID, userID)
	if err != nil {
		return fmt.Errorf("delete campus review: %w", err)
	}
	return nil
}

func (s *Store) AdminDeleteCampusReview(reviewID int64) error {
	_, err := s.db.Exec("DELETE FROM campus_review WHERE id = ?", reviewID)
	if err != nil {
		return fmt.Errorf("admin delete campus review: %w", err)
	}
	return nil
}

func (s *Store) DeleteCampusItem(userID string, itemID int64) error {
	var createdBy string
	err := s.db.QueryRow("SELECT created_by FROM campus_item WHERE id = ?", itemID).Scan(&createdBy)
	if err != nil {
		return fmt.Errorf("get campus item owner: %w", err)
	}
	if createdBy != userID {
		return fmt.Errorf("forbidden: not the creator")
	}

	_, err = s.db.Exec("DELETE FROM campus_review WHERE item_id = ?", itemID)
	if err != nil {
		return fmt.Errorf("delete campus item reviews: %w", err)
	}
	_, err = s.db.Exec("DELETE FROM campus_item WHERE id = ?", itemID)
	if err != nil {
		return fmt.Errorf("delete campus item: %w", err)
	}
	return nil
}

func (s *Store) AdminDeleteCampusItem(itemID int64) error {
	_, err := s.db.Exec("DELETE FROM campus_review WHERE item_id = ?", itemID)
	if err != nil {
		return fmt.Errorf("delete campus item reviews: %w", err)
	}
	_, err = s.db.Exec("DELETE FROM campus_item WHERE id = ?", itemID)
	if err != nil {
		return fmt.Errorf("admin delete campus item: %w", err)
	}
	return nil
}
