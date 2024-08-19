package utils

import (
	"testing"
)

func TestGenerateTokenLength(t *testing.T) {
	expected := 32
	token := GenerateToken()
	actual := len(token)
	if actual != expected {
		t.Errorf("expected token length: %v\n", expected)
	}
}

func TestPseudoUUIDLength(t *testing.T) {
	expected := 36
	uuid, err := PseudoUUID()
	if err != nil {
		t.Errorf("expected PseudoUUID() without error: %v\n", err)
	}
	actual := len(uuid)
	if actual != expected {
		t.Errorf("expected uuid length: %v\n", expected)
	}
}
