package listen

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

func NotifyBestEffort(configPath string, e Event) {
	b, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return
	}
	notifyURL := strings.TrimSuffix(cfg.URL, "/events") + "/notify"
	if e.At == "" {
		e.At = time.Now().UTC().Format(time.RFC3339)
	}
	body, err := json.Marshal(e)
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, notifyURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}
