package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// EventTypeEmailVerification is the JSON "type" field for verification messages.
const EventTypeEmailVerification = "email.verification"

// EmailVerificationEvent is the payload for outbound verification emails.
type EmailVerificationEvent struct {
	UserID          uuid.UUID `json:"user_id"`
	Email           string    `json:"email"`
	VerificationURL string    `json:"verification_url"`
}

// EmailVerificationPublisher publishes email verification events to a queue consumer.
type EmailVerificationPublisher interface {
	PublishEmailVerification(ctx context.Context, ev EmailVerificationEvent) error
}

type noopEmailVerificationPublisher struct{}

// NewNoopEmailVerificationPublisher returns a publisher that does nothing (for tests or when RabbitMQ is disabled).
func NewNoopEmailVerificationPublisher() EmailVerificationPublisher {
	return noopEmailVerificationPublisher{}
}

func (noopEmailVerificationPublisher) PublishEmailVerification(context.Context, EmailVerificationEvent) error {
	return nil
}

type amqpEmailVerificationPublisher struct {
	pub        *Publisher
	routingKey string
}

// NewEmailVerificationPublisher wraps a Publisher to send JSON verification events with the given routing key.
func NewEmailVerificationPublisher(pub *Publisher, routingKey string) EmailVerificationPublisher {
	if routingKey == "" {
		routingKey = "email.verification"
	}
	return &amqpEmailVerificationPublisher{pub: pub, routingKey: routingKey}
}

func (a *amqpEmailVerificationPublisher) PublishEmailVerification(ctx context.Context, ev EmailVerificationEvent) error {
	msg := struct {
		Type            string    `json:"type"`
		UserID          uuid.UUID `json:"user_id"`
		Email           string    `json:"email"`
		VerificationURL string    `json:"verification_url"`
	}{
		Type:            EventTypeEmailVerification,
		UserID:          ev.UserID,
		Email:           ev.Email,
		VerificationURL: ev.VerificationURL,
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal email verification event: %w", err)
	}
	return a.pub.Publish(ctx, a.routingKey, body)
}
