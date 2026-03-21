package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/jwgl"
)

// EmptyRoomHandler handles GET /api/getEmptyRooms?token=&building_id=&start_date=
// Proxies the jwgl room occupancy search endpoint.
func EmptyRoomHandler(client *jwgl.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		buildingID := c.Query("building_id")
		startDate := c.Query("start_date")

		if token == "" || buildingID == "" || startDate == "" {
			c.String(http.StatusBadRequest, "missing token, building_id, or start_date")
			return
		}

		form := url.Values{
			"building_id": {buildingID},
			"end_date":    {""},
			"random":      {fmt.Sprintf("%d", rand.Intn(90000)+10000)},
			"start_date":  {startDate},
			"timestamp":   {fmt.Sprintf("%d", timeNowMillis())},
		}

		raw, err := client.PostAPI("/room/borrow/occupancy/search", form, token)
		if err != nil {
			c.String(http.StatusInternalServerError, "fetch empty rooms failed: %v", err)
			return
		}

		// raw is already decoded JSON from business_data, pass through directly
		var parsed interface{}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			// If it doesn't parse as JSON, return raw
			c.Data(http.StatusOK, "text/plain; charset=utf-8", raw)
			return
		}

		c.JSON(http.StatusOK, parsed)
	}
}
