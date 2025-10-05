package app

import (
	"log"
	"nasa-app/internal/airquality"
	"nasa-app/internal/auth"
	database "nasa-app/internal/db"
	"nasa-app/internal/middleware"
	"nasa-app/internal/mlclient"
	"nasa-app/internal/notification"
	user2 "nasa-app/internal/user"
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

	mailSender := func(email, riskLevel string, aqi int) error {
		log.Printf("SMTP configuration missing; skipping email to %s (risk=%s)", email, riskLevel)
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
		return mlclient.PredictionResponse{RiskLevel: "unknown"}, nil
	}

	if cfg.MLServiceURL != "" {
		log.Printf("Initializing ML client with URL: %s%s", cfg.MLServiceURL, cfg.MLPredictPath)
		mlc, err := mlclient.New(cfg.MLServiceURL, cfg.MLPredictPath, nil)
		if err != nil {
			log.Printf("ml client init failed: %v", err)
		} else {
			log.Println("ML client initialized successfully")
			mlPredictor = func(n notification.Notification, metrics airquality.Metrics) (mlclient.PredictionResponse, error) {
				log.Printf("Sending prediction request to ML service for user %d", n.UserID)
				req := mlclient.PredictionRequest{
					Temperature:       metrics.Temperature,
					Humidity:          metrics.Humidity,
					PM25:              metrics.PM25,
					PM10:              metrics.PM10,
					NO2:               metrics.NO2,
					SO2:               metrics.SO2,
					CO:                metrics.CO,
					PopulationDensity: metrics.PopulationDensity,
				}
				return mlc.Predict(req)
			}
		}
	} else {
		log.Println("ML service URL missing; using fallback predictor")
	}

	alertNotifier := func(n notification.Notification, metrics airquality.Metrics, prediction mlclient.PredictionResponse) error {
		riskLevel := prediction.RiskLevel
		if riskLevel == "" {
			riskLevel = "unknown"
		}
		return mailSender(n.Email, riskLevel, 0)
	}

	// ML predictor for air quality endpoint
	aqMLPredictor := func(latitude, longitude float64, metrics airquality.Metrics) (string, error) {
		prediction, err := mlPredictor(notification.Notification{Latitude: latitude, Longitude: longitude}, metrics)
		if err != nil {
			return "unknown", err
		}
		return prediction.RiskLevel, nil
	}

	/* ------------ Handlers ------------ */
	userHdl := user2.NewHandler(userSvc)
	notifHdl := notification.NewHandler(notifRepo, nil) // session store ileride eklenecek
	aqHdl := airquality.NewHandler(aqService, aqMLPredictor)

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
		AllowOrigins:     "http://localhost:3000, https://localhost:3000, https://clean-breathing-front-six.vercel.app",
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
	app.Get("/air-quality", aqHdl.GetAirQuality)

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
