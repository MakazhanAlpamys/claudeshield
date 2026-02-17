package secrets

import (
	"os"
	"testing"
)

func TestEnvProvider_Available(t *testing.T) {
	p := &EnvProvider{}
	if !p.Available() {
		t.Error("EnvProvider should always be available")
	}
}

func TestEnvProvider_Load(t *testing.T) {
	os.Setenv("CS_TEST_KEY_1", "value1")
	os.Setenv("CS_TEST_KEY_2", "value2")
	defer os.Unsetenv("CS_TEST_KEY_1")
	defer os.Unsetenv("CS_TEST_KEY_2")

	p := &EnvProvider{}
	result, err := p.Load([]string{"CS_TEST_KEY_1", "CS_TEST_KEY_2"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if result["CS_TEST_KEY_1"] != "value1" {
		t.Errorf("expected value1, got %s", result["CS_TEST_KEY_1"])
	}
	if result["CS_TEST_KEY_2"] != "value2" {
		t.Errorf("expected value2, got %s", result["CS_TEST_KEY_2"])
	}
}

func TestEnvProvider_Missing(t *testing.T) {
	p := &EnvProvider{}
	_, err := p.Load([]string{"CS_NONEXISTENT_VAR"})
	if err == nil {
		t.Error("should error on missing env var")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	if _, err := r.Get("env"); err != nil {
		t.Errorf("env provider should exist: %v", err)
	}

	if _, err := r.Get("nonexistent"); err == nil {
		t.Error("nonexistent provider should error")
	}
}
