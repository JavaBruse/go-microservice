package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type IntegrationService struct {
	httpClient *http.Client
}

func NewIntegrationService() *IntegrationService {
	return &IntegrationService{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *IntegrationService) CallExternalAPI(url, method string, data map[string]string) (map[string]interface{}, error) {
	var req *http.Request
	var err error

	switch method {
	case "GET":
		req, err = http.NewRequest("GET", url, nil)
	case "POST":
		var body io.Reader
		if data != nil {
			jsonData, _ := json.Marshal(data)
			body = &byteReader{data: jsonData}
		}
		req, err = http.NewRequest("POST", url, body)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}

	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"data":        result,
	}, nil
}

// Вспомогательная структура для преобразования данных в io.Reader
type byteReader struct {
	data []byte
	pos  int
}

func (b *byteReader) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}
