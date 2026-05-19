package xcursor

import (
	"testing"
)

func TestEncodeDecodeCursor(t *testing.T) {
	fields := map[string]any{
		"created_at": "2025-01-01T00:00:00Z",
		"id":         "abc-123",
	}

	encoded := EncodeCursor(fields)
	if encoded == "" {
		t.Fatal("encoded cursor should not be empty")
	}

	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor: %v", err)
	}
	if decoded["created_at"] != "2025-01-01T00:00:00Z" {
		t.Errorf("created_at = %v", decoded["created_at"])
	}
	if decoded["id"] != "abc-123" {
		t.Errorf("id = %v", decoded["id"])
	}
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	_, err := DecodeCursor("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecodeCursor_NotJSON(t *testing.T) {
	_, err := DecodeCursor("!!!invalid!!!")
	if err == nil {
		t.Error("expected error for invalid cursor")
	}
}

func TestComputeFilterHash(t *testing.T) {
	h1 := ComputeFilterHash("proj-1", "0", "", "")
	h2 := ComputeFilterHash("proj-1", "0", "", "")
	h3 := ComputeFilterHash("proj-1", "1", "", "")

	if h1 != h2 {
		t.Errorf("same params should produce same hash: %s != %s", h1, h2)
	}
	if h1 == h3 {
		t.Errorf("different params should produce different hash")
	}
	if len(h1) == 0 {
		t.Error("hash should not be empty")
	}
}

func TestDecodeCursor_Empty(t *testing.T) {
	fields, err := DecodeCursor("")
	if err != nil {
		t.Fatalf("DecodeCursor empty string: %v", err)
	}
	if len(fields) != 0 {
		t.Errorf("empty cursor should return empty fields, got %v", fields)
	}
}

func TestEncodeCursor_Empty(t *testing.T) {
	encoded := EncodeCursor(nil)
	if encoded != "" {
		t.Errorf("nil fields should encode to empty string, got %s", encoded)
	}
}
