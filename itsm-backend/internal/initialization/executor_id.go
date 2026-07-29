package initialization

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

func NewExecutorID(prefix string) (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate executor id: %w", err)
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "initializer"
	}
	return prefix + "-" + hex.EncodeToString(random), nil
}
