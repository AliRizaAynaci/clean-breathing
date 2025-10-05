package auth

import (
	"context"
	"encoding/json"
	"errors"
	"nasa-app/internal/user"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleCfg *oauth2.Config
	initOnce  sync.Once
)

func cfg() (*oauth2.Config, error) {
	var initErr error

	initOnce.Do(func() {
		_ = godotenv.Load()

		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
		redirectURL := os.Getenv("OAUTH_REDIRECT_URL")

		// ✅ Environment variable kontrolü
		if clientID == "" || clientSecret == "" || redirectURL == "" {
			initErr = errors.New("missing Google OAuth credentials in environment")
			return
		}

		googleCfg = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return googleCfg, nil
}

// GET /auth/google/login
func Login(c *fiber.Ctx) error {
	config, err := cfg()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "OAuth config error: "+err.Error())
	}

	url := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	return c.Redirect(url, fiber.StatusTemporaryRedirect)
}

// GET /auth/google/callback
func Callback(svc *user.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		code := c.Query("code")
		if code == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing code")
		}

		config, err := cfg()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "OAuth config error: "+err.Error())
		}

		ctx := context.Background()
		tok, err := config.Exchange(ctx, code)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "oauth exchange: "+err.Error())
		}

		client := config.Client(ctx, tok)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo?alt=json")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "userinfo fetch: "+err.Error())
		}
		defer resp.Body.Close()

		var info struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "decode: "+err.Error())
		}

		u, err := svc.FindOrCreate(info.ID, info.Email, info.Name, "")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "user create: "+err.Error())
		}

		// Issue JWT
		claims := jwt.MapClaims{
			"user_id": u.ID,
			"exp":     time.Now().Add(24 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "jwt sign: "+err.Error())
		}

		// ✅ Cookie düzeltmeleri
		c.Cookie(&fiber.Cookie{
			Name:     "session_token",
			Value:    signed,
			Path:     "/",
			HTTPOnly: true,
			SameSite: "None", // ✅ Localhost için Lax kullan
			Secure:   true,   // ✅ Localhost için false
			MaxAge:   86400,  // 24 saat
		})

		frontendRedirect := os.Getenv("FRONTEND_REDIRECT_URL")
		if frontendRedirect == "" {
			frontendRedirect = os.Getenv("FRONTEND_URI") + "/dashboard" // fallback
		}
		return c.Redirect(frontendRedirect, fiber.StatusSeeOther)
	}
}
