package xcursor

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func EncodeCursor(fields map[string]any) (string, error) {
	if len(fields) == 0 {
		return "", nil
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func DecodeCursor(cursor string) (map[string]any, error) {
	if cursor == "" {
		return nil, nil
	}
	b, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor base64: %w", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		return nil, fmt.Errorf("invalid cursor json: %w", err)
	}
	return fields, nil
}

func ComputeFilterHash(params ...string) string {
	h := sha256.New()
	h.Write([]byte(strings.Join(params, "|")))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
