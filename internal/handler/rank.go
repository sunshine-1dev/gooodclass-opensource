package handler

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"gooodclass/internal/jwgl"
)

// RankHandler handles GET /api/getRank?username=&password=
// This endpoint hits a DIFFERENT system: xsgl.aust.edu.cn (学生管理系统).
func RankHandler(client *jwgl.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Query("username")
		password := c.Query("password")
		if username == "" || password == "" {
			c.String(http.StatusBadRequest, "missing username or password")
			return
		}

		result, err := fetchRank(client, username, password)
		if err != nil {
			c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("null"))
			return
		}
		c.Data(http.StatusOK, "text/plain; charset=utf-8", result)
	}
}

func fetchRank(client *jwgl.Client, username, password string) ([]byte, error) {
	// Custom MD5 password encoding matching the Python code
	hash := md5.Sum([]byte(password))
	md5Str := fmt.Sprintf("%x", hash)
	// Remove last 2 chars, then insert 'a' after index 5 and 'b' after index 9
	md5Pass := md5Str[:len(md5Str)-2]
	customPass := md5Pass[:5] + "a" + md5Pass[5:9] + "b" + md5Pass[9:]

	form := url.Values{
		"uname": {username},
		"pd_mm": {customPass},
	}

	// Use a cookie jar-aware client for this flow
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{
		Jar:     jar,
		Timeout: client.HTTP.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // follow redirects
		},
	}

	// Step 1: Login to xsgl
	loginURL := "https://xsgl.aust.edu.cn/student/website/login"
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xsgl login HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var loginData struct {
		Goto2 string `json:"goto2"`
	}
	if err := json.Unmarshal(body, &loginData); err != nil {
		return nil, fmt.Errorf("parse xsgl login response: %w", err)
	}

	// Step 2: Fetch rank data
	rankURL := "http://xsgl.aust.edu.cn/student/content/student/zhcp/stu/myinfo/2024"
	rankResp, err := httpClient.Get(rankURL)
	if err != nil {
		return nil, err
	}
	defer rankResp.Body.Close()

	rankBody, err := io.ReadAll(rankResp.Body)
	if err != nil {
		return nil, err
	}

	var rankData map[string]interface{}
	if err := json.Unmarshal(rankBody, &rankData); err != nil {
		return nil, fmt.Errorf("parse rank data: %w", err)
	}

	output := map[string]interface{}{
		"测评专业排名": rankData["ZYPM"],
		"测评班级排名": rankData["BJPM"],
		"智育专业排名": rankData["ZY_ZYPM"],
		"智育班级排名": rankData["ZY_BJPM"],
	}

	return json.Marshal(output)
}
