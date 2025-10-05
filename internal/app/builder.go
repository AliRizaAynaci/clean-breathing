package app

import (
	"log"
	"math"
	"nasa-app/internal/airquality"
	"nasa-app/internal/auth"
	database "nasa-app/internal/db"
	"nasa-app/internal/middleware"
	"nasa-app/internal/mlclient"
	"nasa-app/internal/notification"
	user2 "nasa-app/internal/user"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"nasa-app/internal/config"
)

func New() *fiber.App {
	cfg := config.Load()

	/* ------------ DB ------------ */
	db, err := database.Connect(cfg.DSN)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	if err := database.Migrate(db,
		&user2.User{},
		&notification.Notification{},
	); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	/* ------------ Services ------------ */
	userSvc := user2.NewService(user2.NewGormRepo(db))
	notifRepo := notification.NewRepository(db)
	aqService := airquality.NewService(nil, cfg.AQIBaseURL)

	mailSender := func(email string, aqi int) error {
		log.Printf("SMTP configuration missing; skipping email to %s (AQI=%d)", email, aqi)
		return nil
	}

	if cfg.SMTPHost != "" && cfg.SMTPPort != "" && cfg.SMTPFrom != "" {
		smtpCfg, err := notification.ConfigFromEnv(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
		if err != nil {
			log.Printf("invalid SMTP configuration: %v", err)
		} else {
			mailer, err := notification.NewMailer(smtpCfg)
			if err != nil {
				log.Printf("mailer init failed: %v", err)
			} else {
				mailSender = mailer.SendAQIAlert
			}
		}
	} else {
		log.Println("SMTP configuration incomplete; notification emails disabled")
	}

	mlPredictor := func(n notification.Notification, metrics airquality.Metrics) (mlclient.PredictionResponse, error) {
		return mlclient.PredictionResponse{PredictedAQI: float64(metrics.AQI)}, nil
	}

	if cfg.MLServiceURL != "" {
		mlc, err := mlclient.New(cfg.MLServiceURL, cfg.MLPredictPath, nil)
		if err != nil {
			log.Printf("ml client init failed: %v", err)
		} else {
			mlPredictor = func(n notification.Notification, metrics airquality.Metrics) (mlclient.PredictionResponse, error) {
				return mlc.Predict(mlclient.PredictionRequest{
					Latitude:  n.Latitude,
					Longitude: n.Longitude,
					Metrics:   metrics,
				})
			}
		}
	} else {
		log.Println("ML service URL missing; falling back to raw AQI values")
	}

	alertNotifier := func(n notification.Notification, metrics airquality.Metrics, prediction mlclient.PredictionResponse) error {
		predicted := prediction.PredictedAQI
		if predicted == 0 {
			predicted = float64(metrics.AQI)
		}
		return mailSender(n.Email, int(math.Round(predicted)))
	}

	/* ------------ Handlers ------------ */
	userHdl := user2.NewHandler(userSvc)
	notifHdl := notification.NewHandler(notifRepo, nil) // session store ileride eklenecek

	/* ------------ Fiber ------------ */
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	srv := app.Server()
	srv.ReadTimeout = 10 * time.Second
	srv.WriteTimeout = 15 * time.Second

	// ✅ CORS düzeltmesi
	app.Use(cors.New(cors.Config{
		AllowOrigins:     os.Getenv("FRONTEND_URI"),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Accept, Authorization, Content-Type, X-CSRF-Token",
		AllowCredentials: true,
	}))

	// ✅ Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	/* ------------ Public routes ------------ */
	app.Get("/auth/google/login", auth.Login)
	app.Get("/auth/google/callback", auth.Callback(userSvc))
	app.Get("/logout", auth.Logout)

	/* ------------ Protected routes ------------ */
	api := app.Group("/", middleware.Auth())
	api.Get("/me", userHdl.Me)
	api.Post("/notifications/subscribe", notifHdl.Subscribe)

	// Bildirim scheduler'ı başlat
	interval := time.Duration(cfg.NotificationIntervalMinute) * time.Minute
	notification.StartScheduler(
		notifRepo,
		interval,
		func(n notification.Notification) (airquality.Metrics, error) {
			return aqService.GetMetrics(n.Latitude, n.Longitude)
		},
		mlPredictor,
		alertNotifier,
	)

	return app
}
