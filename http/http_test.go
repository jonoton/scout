package http

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestGetHttpDirectory(t *testing.T) {
	dir := getHttpDirectory()
	if dir == "" {
		t.Fatalf("Expected non-empty http directory")
	}

	templatePath := filepath.Join(dir, "templates", "index.html")
	if _, err := os.Stat(templatePath); err != nil {
		t.Errorf("Expected template file to exist at %s: %v", templatePath, err)
	}

	publicPath := filepath.Join(dir, "public", "css", "scout.css")
	if _, err := os.Stat(publicPath); err != nil {
		t.Errorf("Expected scout.css to exist at %s: %v", publicPath, err)
	}
}

func TestIndexTemplateParse(t *testing.T) {
	dir := getHttpDirectory()
	templatePath := filepath.Join(dir, "templates", "index.html")

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		t.Fatalf("Failed to parse index.html template: %v", err)
	}

	if tmpl == nil {
		t.Fatalf("Parsed template is nil")
	}
}

func TestRequiredLocalAssetsExist(t *testing.T) {
	dir := getHttpDirectory()
	requiredFiles := []string{
		"public/css/bootstrap.min.css",
		"public/css/scout.css",
		"public/js/jquery-3.5.1.min.js",
		"public/js/bootstrap.bundle.min.js",
		"public/js/bootstrap.min.js",
		"public/js/pako.min.js",
		"public/js/crypto-js.min.js",
		"public/js/js.cookie.min.js",
		"public/js/index.js",
		"public/js/login.js",
		"public/images/hawk-wing.png",
		"templates/index.html",
	}

	for _, relPath := range requiredFiles {
		fullPath := filepath.Join(dir, relPath)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("Required offline local asset missing: %s (%v)", fullPath, err)
		}
	}
}

func TestAuthStatusEndpoints(t *testing.T) {
	newTestHttp := func(conf *Config) *Http {
		h := &Http{
			httpConfig:          conf,
			fiber:               fiber.New(),
			loginLogger:         &log.Logger{},
			accessLogger:        &log.Logger{},
			linkClients:         make([]*linkClient, 0),
			linkRetry:           2,
			loginNeeded:         false,
			loginSigningKey:     "test-key",
			twoFactorCheck:      make(map[string]twoFactorAttempt),
			twoFactorTimeoutSec: 60,
			secTick:             time.NewTicker(time.Second),
			done:                make(chan bool),
		}
		h.setup()
		return h
	}

	// 1. Without users (loginNeeded = false)
	hNoAuth := newTestHttp(&Config{})
	defer func() {
		close(hNoAuth.done)
		hNoAuth.secTick.Stop()
	}()

	reqAuthStatus, _ := http.NewRequest("GET", "http://localhost/auth/status", nil)
	resp, err := hNoAuth.fiber.Test(reqAuthStatus)
	if err != nil {
		t.Fatalf("Failed /auth/status test request: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]bool
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if result["required"] != false {
		t.Errorf("Expected required=false, got %v", result["required"])
	}

	// Test GET /login endpoint
	reqLogin, _ := http.NewRequest("GET", "http://localhost/login", nil)
	respLogin, err := hNoAuth.fiber.Test(reqLogin)
	if err != nil {
		t.Fatalf("Failed GET /login test request: %v", err)
	}
	if respLogin.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", respLogin.StatusCode)
	}

	// 2. With users (loginNeeded = true)
	hWithAuth := newTestHttp(&Config{
		Users: []UserAuth{
			{User: "testuser", Password: "testpassword"},
		},
	})
	defer func() {
		close(hWithAuth.done)
		hWithAuth.secTick.Stop()
	}()

	respWithAuth, err := hWithAuth.fiber.Test(reqAuthStatus)
	if err != nil {
		t.Fatalf("Failed /auth/status with auth test request: %v", err)
	}
	if respWithAuth.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", respWithAuth.StatusCode)
	}

	bodyWithAuth, _ := io.ReadAll(respWithAuth.Body)
	var resultWithAuth map[string]bool
	if err := json.Unmarshal(bodyWithAuth, &resultWithAuth); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if resultWithAuth["required"] != true {
		t.Errorf("Expected required=true, got %v", resultWithAuth["required"])
	}
}
