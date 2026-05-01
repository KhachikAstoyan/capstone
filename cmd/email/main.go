package main

import (
	"context"
	"encoding/json"
	"os/signal"
	"syscall"

	"github.com/KhachikAstoyan/capstone/internal/email"
	"github.com/KhachikAstoyan/capstone/pkg/logger"
	"github.com/KhachikAstoyan/capstone/pkg/rabbitmq"
	"go.uber.org/zap"
)

func main() {
	cfg, err := email.LoadConfig()
	if err != nil {
		tempLog := logger.Init("production")
		tempLog.Fatal("Failed to load config", zap.Error(err))
	}

	log := logger.Init(cfg.Environment)
	defer log.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ctx = logger.WithLogger(ctx, log)

	log.Info("Starting email worker",
		zap.String("environment", cfg.Environment),
		zap.String("exchange", cfg.RabbitMQExchange),
		zap.String("queue", cfg.RabbitMQQueue),
		zap.String("routing_key", cfg.RabbitMQEmailVerificationRoutingKey),
		zap.String("smtp_host", cfg.SMTPHost),
		zap.Int("smtp_port", cfg.SMTPPort))

	sender := email.NewSender(cfg)

	consumer, err := rabbitmq.NewConsumer(rabbitmq.ConsumerConfig{
		URL:         cfg.RabbitMQURL,
		Exchange:    cfg.RabbitMQExchange,
		Queue:       cfg.RabbitMQQueue,
		RoutingKey:  cfg.RabbitMQEmailVerificationRoutingKey,
		ConsumerTag: "capstone-email",
	})
	if err != nil {
		log.Fatal("Failed to connect RabbitMQ consumer", zap.Error(err))
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Error("RabbitMQ consumer close", zap.Error(err))
		}
	}()

	err = consumer.Run(ctx, func(ctx context.Context, body []byte) error {
		log := logger.FromContext(ctx)

		var msg struct {
			Type            string `json:"type"`
			Email           string `json:"email"`
			VerificationURL string `json:"verification_url"`
		}
		if err := json.Unmarshal(body, &msg); err != nil {
			log.Error("failed to parse message", zap.Error(err))
			return err
		}

		if msg.Type != rabbitmq.EventTypeEmailVerification {
			log.Warn("unknown message type, discarding", zap.String("type", msg.Type))
			return nil
		}

		log.Info("sending verification email", zap.String("to", msg.Email))
		if err := sender.SendVerificationEmail(msg.Email, msg.VerificationURL); err != nil {
			log.Error("failed to send verification email",
				zap.Error(err),
				zap.String("to", msg.Email))
			return err
		}

		log.Info("verification email sent", zap.String("to", msg.Email))
		return nil
	})
	if err != nil && err != context.Canceled {
		log.Fatal("Consumer stopped with error", zap.Error(err))
	}

	log.Info("Email worker stopped")
}
