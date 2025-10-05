package notification

import (
	"log"
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

				predictedAQI := prediction.PredictedAQI
				if predictedAQI == 0 {
					predictedAQI = float64(metrics.AQI)
				}
				if predictedAQI >= float64(n.Threshold) {
					if err := notifyFunc(n, metrics, prediction); err != nil {
						log.Println("Notification error:", err)
					}
				} else {
					log.Printf("Prediction below threshold for user %d: %.2f < %d", n.UserID, predictedAQI, n.Threshold)
				}
			}

			time.Sleep(interval)
		}
	}()
}
