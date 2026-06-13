package biz

import (
	"encoding/json"
	"fmt"
	"log"
)

func jsonDetail(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("ERROR: operation log detail marshal failed: %v", err)
		// Preserve diagnostic info in the audit log so the row shows that
		// detail was lost and why, rather than silently storing {}.
		return fmt.Sprintf(`{"error":"marshal_failed","reason":%q}`, err.Error())
	}
	return string(b)
}
