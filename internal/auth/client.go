// Package auth provides the login handler that authenticates a user
// against the jwglyd system and returns "SUCCESS,<real_name>" or an error.
package auth

import (
	"encoding/json"
	"strings"

	"gooodclass/internal/jwgl"
)

// Result mirrors what the Python login returns: "SUCCESS,<name>" or "WRONG INFO" / "SERVER ERROR".
func Login(client *jwgl.Client, username, password string) (string, error) {
	raw, errMsg, err := client.Login(username, password)
	if err != nil {
		return "SERVER ERROR", err
	}

	var data struct {
		UserInfo struct {
			UserName string `json:"user_name"`
		} `json:"user_info"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return "SERVER ERROR", err
	}

	if strings.Contains(errMsg, "用户名或密码不正确") {
		return "WRONG INFO", nil
	}

	return "SUCCESS," + data.UserInfo.UserName, nil
}
