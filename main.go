package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/pemistahl/lingua-go"
)

// --- Configuration Constants ---
const (
	DefaultPort           = "3000"
	DefaultMaxCharProcess = 1000
)

// --- DTOs (Data Transfer Objects) ---

type LanguageDetectionRequest struct {
	Text string `json:"text"`
}

type LanguageDetectionResponse struct {
	ISOCode    string  `json:"iso_code"`
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence"`
}

func main() {
	// 0. LOAD CONFIGURATION
	appPort := os.Getenv("PORT")
	if appPort == "" {
		appPort = DefaultPort
	}
	if !strings.HasPrefix(appPort, ":") {
		appPort = ":" + appPort
	}

	maxCharProcess := DefaultMaxCharProcess
	if val := os.Getenv("MAX_CHAR_PROCESS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxCharProcess = parsed
		}
	}

	// 1. LOAD LANGUAGE MODELS (Cold Start)
	// Loading models into memory. To optimize RAM usage, we select common languages.
	// If you need full support, use .FromAllLanguages() instead.
	log.Println("🚀 Lexis: Loading language models into memory...")

	targetLanguages := []lingua.Language{
		lingua.English, lingua.Turkish, lingua.German, lingua.French,
		lingua.Spanish, lingua.Italian, lingua.Portuguese, lingua.Russian,
		lingua.Arabic, lingua.Chinese, lingua.Japanese, lingua.Korean,
		lingua.Dutch, lingua.Azerbaijani, lingua.Persian,
	}

	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(targetLanguages...).
		WithPreloadedLanguageModels().
		Build()

	log.Println("✅ Lexis: Models are ready.")

	// 2. SETUP WEB SERVER
	app := NewApp(detector, maxCharProcess)

	// 4. START SERVER (Graceful Shutdown Implementation)
	// Run server in a separate goroutine so we can listen for OS signals
	go func() {
		log.Printf("👂 Lexis is listening on port %s\n", appPort)
		if err := app.Listen(appPort); err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal (SIGINT or SIGTERM from Kubernetes)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c // Block main thread until signal is received

	log.Println("\n🛑 Lexis: Shutting down...")
	if err := app.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	log.Println("👋 Lexis: Bye!")
}

func NewApp(detector lingua.LanguageDetector, maxCharProcess int) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "Hermes Lexis API",
		DisableStartupMessage: true, // Keep logs clean
	})

	// Middlewares
	app.Use(logger.New())  // Log HTTP requests
	app.Use(recover.New()) // Recover from panics to prevent crash
	app.Use(cors.New())    // Enable CORS for microservice communication

	// 3. DEFINE ROUTES

	// Kubernetes Liveness/Readiness Probe
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	// Main Detection Endpoint
	app.Post("/detect", func(c *fiber.Ctx) error {
		req := new(LanguageDetectionRequest)

		// Parse JSON Body
		if err := c.BodyParser(req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid JSON format",
				"code":  "INVALID_JSON",
			})
		}

		// Validation
		if len(req.Text) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Description cannot be empty",
				"code":  "EMPTY_TEXT",
			})
		}

		// OPTIMIZATION: Smart Truncation
		// Prevent CPU spikes on very large texts
		textToProcess := req.Text
		// Convert to runes to avoid slicing in the middle of a multi-byte character
		runes := []rune(textToProcess)
		if len(runes) > maxCharProcess {
			textToProcess = string(runes[:maxCharProcess])
		}

		// Perform Detection
		language, exists := detector.DetectLanguageOf(textToProcess)

		rsp_isoCode := "unknown"
		rsp_language := "unknown"
		rsp_confidence := 0.0

		if exists {
			rsp_isoCode = language.IsoCode639_1().String()
			rsp_language = language.String()
			rsp_confidence = detector.ComputeLanguageConfidence(textToProcess, language)
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Could not detect language with sufficient confidence",
				"code":  "DETECTION_FAILED",
			})
		}

		return c.JSON(LanguageDetectionResponse{
			ISOCode:    strings.ToLower(rsp_isoCode),
			Language:   strings.ToLower(rsp_language),
			Confidence: rsp_confidence,
		})
	})

	return app
}
