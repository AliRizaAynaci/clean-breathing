package notification

import (
	"errors"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type Mailer struct {
	cfg SMTPConfig
}

func NewMailer(cfg SMTPConfig) (*Mailer, error) {
	if cfg.Host == "" || cfg.Port == 0 {
		return nil, errors.New("smtp host/port required")
	}
	if cfg.From == "" {
		return nil, errors.New("smtp from address required")
	}
	return &Mailer{cfg: cfg}, nil
}

func (m *Mailer) SendAQIAlert(to, riskLevel string, aqi int) error {
	if to == "" {
		return errors.New("recipient email is empty")
	}

	if riskLevel == "" {
		riskLevel = "unknown"
	}

	subject := fmt.Sprintf("Hava Kalitesi Uyarısı: %s", strings.ToUpper(riskLevel))
	body := fmt.Sprintf(
		"Merhaba,\n\nBulunduğunuz konumdaki hava kalitesi uyarı seviyesine ulaştı.\n\nRisk Durumu: %s\n\nLütfen gerekli önlemleri alınız ve mümkünse dışarı çıkmayınız.\n\nSevgiler,\nClean Breathing",
		strings.ToUpper(riskLevel),
	)

	return m.sendMail(to, subject, body)
}

func (m *Mailer) sendMail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	headers := []string{
		fmt.Sprintf("From: %s", m.cfg.From),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
	}
	msg := strings.Join(headers, "\r\n") + body

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg))
}

func ConfigFromEnv(host, port, username, password, from string) (SMTPConfig, error) {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return SMTPConfig{}, fmt.Errorf("invalid smtp port: %w", err)
	}
	return SMTPConfig{
		Host:     host,
		Port:     portNum,
		Username: username,
		Password: password,
		From:     from,
	}, nil
}
