package vector

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

const defaultConnectTimeout = 5 * time.Second

var identifierRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func stringConfig(c map[string]interface{}, key, fallback string) string {
	if v, ok := c[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return fallback
}
func intConfig(c map[string]interface{}, key string, fallback int) int {
	switch v := c[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if n, e := strconv.Atoi(v); e == nil {
			return n
		}
	}
	return fallback
}
func topK(n, fallback int) int {
	if n > 0 {
		return n
	}
	if fallback > 0 {
		return fallback
	}
	return DefaultTopK
}
func validIdentifier(s string) error {
	if !identifierRE.MatchString(s) {
		return fmt.Errorf("invalid SQL identifier %q", s)
	}
	return nil
}
