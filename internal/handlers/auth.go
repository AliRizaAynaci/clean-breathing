package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"nasa-app/internal/db"
	"nasa-app/internal/models"
)

var googleOauthConfig *oauth2.Config

// InitGoogleOAuth - OAuth config'i başlatır
func InitGoogleOAuth() error {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("OAUTH_REDIRECT_URL")

	if clientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is not set")
	}
	if clientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET is not set")
	}

	// Redirect URL yoksa default localhost
	if redirectURL == "" {
		redirectURL = "http://localhost:8080/auth/google/callback"
		log.Println("⚠️  OAUTH_REDIRECT_URL not set, using default:", redirectURL)
	}

	googleOauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	log.Printf("🔐 OAuth Redirect URL: %s", redirectURL)
	return nil
}

// GET /auth/google/login
func GoogleLogin(c *fiber.Ctx) error {
	state := "random-state-token-12345" // Production'da güvenli random string kullanın
	url := googleOauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	log.Printf("🔗 Redirecting to Google OAuth: %s", url)
	return c.Redirect(url)
}

// GET /auth/google/callback
func GoogleCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		log.Println("❌ OAuth callback: missing code parameter")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "missing code parameter",
		})
	}

	state := c.Query("state")
	if state != "random-state-token-12345" {
		log.Printf("❌ OAuth callback: invalid state (got: %s)", state)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid state parameter",
		})
	}

	ctx := context.Background()
	tok, err := googleOauthConfig.Exchange(ctx, code)
	if err != nil {
		log.Printf("❌ OAuth exchange failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "oauth exchange failed",
			"details": err.Error(),
		})
	}

	// Kullanıcı bilgilerini al
	client := googleOauthConfig.Client(ctx, tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo?alt=json")
	if err != nil {
		log.Printf("❌ Userinfo fetch failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "userinfo fetch failed",
			"details": err.Error(),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Userinfo request failed with status: %d", resp.StatusCode)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "userinfo request failed",
			"status": resp.StatusCode,
		})
	}

	var info struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		log.Printf("❌ JSON decode failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "decode failed",
			"details": err.Error(),
		})
	}

	log.Printf("✅ User authenticated: %s (%s)", info.Name, info.Email)

	// Kullanıcıyı bul veya oluştur
	u, err := findOrCreateUser(info.ID, info.Email, info.Name)
	if err != nil {
		log.Printf("❌ User creation failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "user create failed",
			"details": err.Error(),
		})
	}

	// Kullanıcı bilgisini döndür (TEST İÇİN)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Login successful! 🎉",
		"user": fiber.Map{
			"id":        u.ID,
			"name":      u.Name,
			"email":     u.Email,
			"googleID":  u.GoogleID,
			"createdAt": u.CreatedAt,
			"updatedAt": u.UpdatedAt,
		},
	})
}

// findOrCreateUser - Kullanıcıyı veritabanında bul veya oluştur
func findOrCreateUser(googleID, email, name string) (*models.User, error) {
	database := db.GetDB()
	if database == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var user models.User

	// Google ID ile kullanıcıyı ara
	result := database.Where("google_id = ?", googleID).First(&user)
	if result.Error == nil {
		log.Printf("👤 Existing user found: %s", email)
		return &user, nil
	}

	// Kullanıcı yoksa oluştur
	user = models.User{
		GoogleID: googleID,
		Email:    email,
		Name:     name,
	}

	if err := database.Create(&user).Error; err != nil {
		return nil, err
	}

	log.Printf("✨ New user created: %s", email)
	return &user, nil
}
