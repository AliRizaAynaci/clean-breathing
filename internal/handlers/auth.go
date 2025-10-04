package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"nasa-app/internal/db"
	"nasa-app/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

	// JWT token oluştur
	token, err := generateJWTToken(u)
	if err != nil {
		log.Printf("❌ JWT token generation failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "token generation failed",
		})
	}

	// Frontend'e yönlendir
	frontendURL := os.Getenv("FRONTEND_REDIRECT_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000" // Default frontend URL
		log.Println("⚠️  FRONTEND_URI not set, using default:", frontendURL)
	}

	// Başarılı giriş sonrası frontend'e yönlendir (token ile)
	redirectURL := fmt.Sprintf("%s/home?token=%s&login=success", frontendURL, token)
	log.Printf("🔄 Redirecting to frontend: %s", redirectURL)
	return c.Redirect(redirectURL)
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

// generateJWTToken - Kullanıcı için JWT token oluşturur
func generateJWTToken(user *models.User) (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key" // Production'da mutlaka güvenli bir secret kullanın
		log.Println("⚠️  JWT_SECRET not set, using default")
	}

	// Token claims
	claims := jwt.MapClaims{
		"user_id":   user.ID,
		"email":     user.Email,
		"name":      user.Name,
		"google_id": user.GoogleID,
		"exp":       time.Now().Add(time.Hour * 24).Unix(), // 24 saat geçerli
		"iat":       time.Now().Unix(),
	}

	// Token oluştur
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	log.Printf("🔑 JWT token generated for user: %s", user.Email)
	return tokenString, nil
}
