package biz

import (
	"encoding/json"
	"log"
)

func jsonDetail(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("ERROR: operation log detail marshal failed: %v", err)
		return `{}`
	}
	return string(b)
}
