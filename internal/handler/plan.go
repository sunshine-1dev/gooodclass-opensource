package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/jwgl"
)

// GetTokenHandler handles GET /api/getToken?username=&password=
// Proxies jwgl login and returns the education system token.
func GetTokenHandler(client *jwgl.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Query("username")
		password := c.Query("password")
		if username == "" || password == "" {
			c.String(http.StatusBadRequest, "missing username or password")
			return
		}

		token, err := client.LoginAndGetToken(username, password)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"error": fmt.Sprintf("获取token失败: %v", err)})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}

// PlanHandler handles GET /api/getPlan?token=
func PlanHandler(client *jwgl.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.String(http.StatusBadRequest, "missing token")
			return
		}

		form := url.Values{
			"biz_type_id": {"1"},
			"random":      {fmt.Sprintf("%d", rand.Intn(90000)+10000)},
			"token":       {token},
		}

		raw, err := client.PostAPI("/student/course/plan/my-plan", form, token)
		if err != nil {
			c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(fmt.Sprintf("获取培养计划错误: %v", err)))
			return
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", raw)
	}
}

// PlanCompletionHandler handles GET /api/getPlanCompletion?token=
func PlanCompletionHandler(client *jwgl.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.String(http.StatusBadRequest, "missing token")
			return
		}

		form := url.Values{
			"biz_type_id": {"1"},
			"random":      {fmt.Sprintf("%d", rand.Intn(90000)+10000)},
			"token":       {token},
		}

		raw, err := client.PostAPI("/student/course/plan/completion/my-plan-completion", form, token)
		if err != nil {
			c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(fmt.Sprintf("获取培养计划完成情况错误: %v", err)))
			return
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", raw)
	}
}
