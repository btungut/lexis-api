package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pemistahl/lingua-go"
)

func TestHealthEndpoint(t *testing.T) {
	languages := []lingua.Language{lingua.English, lingua.Spanish, lingua.French}
	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		WithPreloadedLanguageModels().
		Build()
	app := NewApp(detector, 1000)

	req, _ := http.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestDetectEndpoint(t *testing.T) {
	// Setup detector with specific languages for testing
	languages := []lingua.Language{lingua.English, lingua.Spanish, lingua.Turkish, lingua.Arabic}
	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		WithPreloadedLanguageModels().
		Build()

	app := NewApp(detector, 100) // Low max char process for testing truncation

	tests := []struct {
		name           string
		payload        interface{} // Use interface to allow passing invalid JSON strings or structs
		expectedStatus int
		expectedLang   string
	}{
		{
			name:           "Valid English",
			payload:        LanguageDetectionRequest{Text: "Hello, this is a software engineering test."},
			expectedStatus: 200,
			expectedLang:   "en",
		},
		{
			name:           "Valid Spanish",
			payload:        LanguageDetectionRequest{Text: "Hola, esto es una prueba de ingeniería de software."},
			expectedStatus: 200,
			expectedLang:   "es",
		},
		{
			name:           "Empty Text",
			payload:        LanguageDetectionRequest{Text: ""},
			expectedStatus: 400,
			expectedLang:   "",
		},
		{
			name:           "Truncation Logic",
			// Text is longer than 100 chars (limit set in app setup above)
			payload:        LanguageDetectionRequest{Text: strings.Repeat("This is a very long english sentence that should be truncated but still detected as english successfully.", 5)},
			expectedStatus: 200,
			expectedLang:   "en",
		},
		{
			name: "Valid Arabic",
			payload: LanguageDetectionRequest{Text: "مرحبا، هذا اختبار هندسة البرمجيات."},
			expectedStatus: 200,
			expectedLang:   "ar",
		},
		{
			name:           "Malformed JSON",
			payload:        `{"text": "This is a test",`, // Invalid JSON
			expectedStatus: 400,
			expectedLang:   "",
		},
		{
			name:           "Non-string Text Field",
			payload:        map[string]interface{}{"text": 12345}, // Invalid type
			expectedStatus: 400,
			expectedLang:   "",
		},
		{
			name:           "Unsupported Language",
			payload:        LanguageDetectionRequest{Text: "これはソフトウェアエンジニアリングのテストです。"}, // Japanese not in detector
			expectedStatus: 200,
			expectedLang:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			// Handle different payload types
			if v, ok := tt.payload.(string); ok {
				body = []byte(v) // Raw string (for malformed JSON tests)
			} else {
				body, err = json.Marshal(tt.payload)
				if err != nil {
					t.Fatalf("Failed to marshal JSON: %v", err)
				}
			}

			req, _ := http.NewRequest("POST", "/detect", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedStatus == 200 {
				var response LanguageDetectionResponse
				respBody, _ := io.ReadAll(resp.Body)
				if err := json.Unmarshal(respBody, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if response.Language != tt.expectedLang {
					t.Errorf("Expected language %s, got %s", tt.expectedLang, response.Language)
				}
			}
		})
	}
}