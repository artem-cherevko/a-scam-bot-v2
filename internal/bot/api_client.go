package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// apiRequest выполняет запрос к API.
func apiRequest(
	ctx context.Context,
	method string,
	url string,
	requesterID int64,
	body []byte,
) (*http.Response, error) {
	var reader io.Reader

	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		url,
		reader,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set(
		"X-User-ID",
		strconv.FormatInt(requesterID, 10),
	)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}

// getUser получает пользователя по Telegram ID или username.
func getUser(
	ctx context.Context,
	identifier string,
	requesterID int64,
	isCheck bool,
) (*UserResponse, error) {
	identifier = strings.TrimSpace(identifier)

	requestURL := "http://localhost:8080/api/user/" +
		url.PathEscape(identifier)

	if isCheck {
		requestURL += "?isCheck=true"
	}

	resp, err := apiRequest(
		ctx,
		http.MethodGet,
		requestURL,
		requesterID,
		nil,
	)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrUserNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"api returned status: %s",
			resp.Status,
		)
	}

	var user UserResponse

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}
