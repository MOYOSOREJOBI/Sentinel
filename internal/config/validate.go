package config

import (
	"fmt"
	"net/url"
	"strings"
)

func Validate(c Config) error {
	if strings.TrimSpace(c.ServiceName) == "" {
		return fmt.Errorf("service name is required")
	}
	if c.HTTPAddr == "" {
		return fmt.Errorf("http addr is required")
	}
	if c.PostgresURL != "" {
		u, err := url.Parse(c.PostgresURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("invalid postgres url")
		}
	}
	if len(c.KafkaBrokers) == 0 || strings.TrimSpace(c.KafkaBrokers[0]) == "" {
		return fmt.Errorf("kafka broker is required")
	}
	return nil
}
