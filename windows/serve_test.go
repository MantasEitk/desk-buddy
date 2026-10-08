package main

import (
	"os"
	"testing"
	"time"
)

// For looking at the window in a browser on the Mac: DB_SERVE=1 with DB_TEST_SERVER / DB_TEST_KEY.
func TestServeUI(t *testing.T) {
	if os.Getenv("DB_SERVE") == "" {
		t.Skip("set DB_SERVE=1")
	}
	os.Setenv("APPDATA", t.TempDir())
	saveConfig(Config{Mode: os.Getenv("DB_MODE"), Server: os.Getenv("DB_TEST_SERVER"), Key: os.Getenv("DB_TEST_KEY")})
	go measure()
	startPage()
	os.WriteFile(os.Getenv("DB_SERVE_URL_FILE"), []byte(uiURL()), 0o600)
	time.Sleep(20 * time.Minute)
}
