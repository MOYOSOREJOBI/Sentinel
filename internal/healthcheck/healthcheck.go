package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

func Run(port int, path string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
	defer cancel()

	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return 0
	}
	return 1
}

func MustPort(env string, fallback int) int {
	if v := os.Getenv(env); v != "" {
		var p int
		if _, err := fmt.Sscanf(v, "%d", &p); err == nil && p > 0 {
			return p
		}
	}
	return fallback
}
