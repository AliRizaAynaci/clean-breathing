package notification

import (
	"log"
	"strings"
	"time"

	"nasa-app/internal/airquality"
	"nasa-app/internal/mlclient"
)

func StartScheduler(
	repo *Repository,
	interval time.Duration,
	metricsFunc func(Notification) (airquality.Metrics, error),
	predictFunc func(Notification, airquality.Metrics) (mlclient.PredictionResponse, error),
	notifyFunc func(Notification, airquality.Metrics, mlclient.PredictionResponse) error,
) {
	if interval <= 0 {
		interval = 30 * time.Minute
	}

	go func() {
		for {
			notifs, err := repo.GetAllNotifications()
			if err != nil {
				log.Println("Scheduler DB error:", err)
				time.Sleep(time.Minute)
				continue
			}

			for _, n := range notifs {
				metrics, err := metricsFunc(n)
				if err != nil {
					log.Println("Metrics fetch error:", err)
					continue
				}

				prediction, err := predictFunc(n, metrics)
				if err != nil {
					log.Println("Prediction error:", err)
					continue
				}

				riskLevel := strings.ToLower(strings.TrimSpace(prediction.RiskLevel))
				switch riskLevel {
				case "poor", "hazardous":
					if err := notifyFunc(n, metrics, prediction); err != nil {
						log.Println("Notification error:", err)
					}
				case "good", "moderate":
					log.Printf("Risk level %s does not require alert for user %d", riskLevel, n.UserID)
				default:
					log.Printf("Unknown risk level '%s' for user %d, skipping notification", riskLevel, n.UserID)
				}
			}

			time.Sleep(interval)
		}
	}()
}
