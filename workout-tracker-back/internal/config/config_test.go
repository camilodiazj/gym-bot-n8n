package config

import (
	"os"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	// Set required environment variables
	os.Setenv("SUPABASE_DB_URL", "postgresql://test:test@localhost:5432/test")
	defer os.Unsetenv("SUPABASE_DB_URL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	if cfg.Database.URL != "postgresql://test:test@localhost:5432/test" {
		t.Errorf("Unexpected database URL: %s", cfg.Database.URL)
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	os.Unsetenv("SUPABASE_DB_URL")

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error when SUPABASE_DB_URL is missing")
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	os.Setenv("SUPABASE_DB_URL", "postgresql://test:test@localhost:5432/test")
	os.Setenv("PORT", "invalid")
	defer func() {
		os.Unsetenv("SUPABASE_DB_URL")
		os.Unsetenv("PORT")
	}()

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for invalid PORT")
	}
}

func TestLoad_CustomPort(t *testing.T) {
	os.Setenv("SUPABASE_DB_URL", "postgresql://test:test@localhost:5432/test")
	os.Setenv("PORT", "3000")
	defer func() {
		os.Unsetenv("SUPABASE_DB_URL")
		os.Unsetenv("PORT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", cfg.Server.Port)
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	os.Setenv("SUPABASE_DB_URL", "postgresql://test:test@localhost:5432/test")
	os.Unsetenv("PORT")
	defer os.Unsetenv("SUPABASE_DB_URL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}
}

func TestLoad_GinMode(t *testing.T) {
	os.Setenv("SUPABASE_DB_URL", "postgresql://test:test@localhost:5432/test")
	os.Setenv("GIN_MODE", "release")
	os.Setenv("INTERNAL_API_KEY", "test-api-key-for-production")
	defer func() {
		os.Unsetenv("SUPABASE_DB_URL")
		os.Unsetenv("GIN_MODE")
		os.Unsetenv("INTERNAL_API_KEY")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Server.GinMode != "release" {
		t.Errorf("Expected GinMode 'release', got '%s'", cfg.Server.GinMode)
	}

	if cfg.Server.InternalAPIKey != "test-api-key-for-production" {
		t.Errorf("Expected InternalAPIKey 'test-api-key-for-production', got '%s'", cfg.Server.InternalAPIKey)
	}
}

func TestLoad_InternalAPIKey_RequiredInProduction(t *testing.T) {
	os.Setenv("SUPABASE_DB_URL", "postgresql://test:test@localhost:5432/test")
	os.Setenv("GIN_MODE", "release")
	os.Unsetenv("INTERNAL_API_KEY")
	defer func() {
		os.Unsetenv("SUPABASE_DB_URL")
		os.Unsetenv("GIN_MODE")
	}()

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error when INTERNAL_API_KEY is missing in production")
	}
}

func TestLoad_InternalAPIKey_DefaultInDev(t *testing.T) {
	os.Setenv("SUPABASE_DB_URL", "postgresql://test:test@localhost:5432/test")
	os.Unsetenv("GIN_MODE")
	os.Unsetenv("INTERNAL_API_KEY")
	defer os.Unsetenv("SUPABASE_DB_URL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error in dev mode, got: %v", err)
	}

	if cfg.Server.InternalAPIKey != "dev-api-key-not-for-production" {
		t.Errorf("Expected default InternalAPIKey 'dev-api-key-not-for-production', got '%s'", cfg.Server.InternalAPIKey)
	}
}

func TestServerAddr(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 3000},
	}

	addr := cfg.ServerAddr()
	if addr != ":3000" {
		t.Errorf("Expected ':3000', got '%s'", addr)
	}
}

func TestGetEnv(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	value := getEnv("TEST_VAR", "default")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}

	// Test with non-existing env var
	value = getEnv("NON_EXISTING_VAR", "default")
	if value != "default" {
		t.Errorf("Expected 'default', got '%s'", value)
	}
}

func TestGetEnvAsInt(t *testing.T) {
	// Test with valid int
	os.Setenv("INT_VAR", "42")
	defer os.Unsetenv("INT_VAR")

	value, err := getEnvAsInt("INT_VAR", 0)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}

	// Test with empty value (default)
	value, err = getEnvAsInt("NON_EXISTING", 100)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if value != 100 {
		t.Errorf("Expected default 100, got %d", value)
	}

	// Test with invalid int
	os.Setenv("INVALID_INT", "not_a_number")
	defer os.Unsetenv("INVALID_INT")

	_, err = getEnvAsInt("INVALID_INT", 0)
	if err == nil {
		t.Error("Expected error for invalid int")
	}
}

func TestGetEnvAsBool(t *testing.T) {
	// Test with true value
	os.Setenv("BOOL_VAR", "true")
	defer os.Unsetenv("BOOL_VAR")

	value := getEnvAsBool("BOOL_VAR", false)
	if !value {
		t.Error("Expected true")
	}

	// Test with false value
	os.Setenv("BOOL_VAR", "false")
	value = getEnvAsBool("BOOL_VAR", true)
	if value {
		t.Error("Expected false")
	}

	// Test with empty value (default)
	value = getEnvAsBool("NON_EXISTING", true)
	if !value {
		t.Error("Expected default true")
	}

	// Test with invalid bool
	os.Setenv("INVALID_BOOL", "not_a_bool")
	defer os.Unsetenv("INVALID_BOOL")

	value = getEnvAsBool("INVALID_BOOL", true)
	if !value {
		t.Error("Expected default true for invalid bool")
	}
}
