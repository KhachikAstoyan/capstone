package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	cpdomain "github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
)

// JobPublisher publishes job creation requests to the control-plane consumer.
type JobPublisher interface {
	PublishJobCreation(ctx context.Context, req cpdomain.CreateJobRequest) error
}

type noopJobPublisher struct{}

// NewNoopJobPublisher returns a publisher that does nothing (for tests or when RabbitMQ is disabled).
func NewNoopJobPublisher() JobPublisher {
	return noopJobPublisher{}
}

func (noopJobPublisher) PublishJobCreation(context.Context, cpdomain.CreateJobRequest) error {
	return nil
}

type amqpJobPublisher struct {
	pub        *Publisher
	routingKey string
}

// NewJobPublisher wraps a Publisher to send JSON job creation messages with the given routing key.
func NewJobPublisher(pub *Publisher, routingKey string) JobPublisher {
	if routingKey == "" {
		routingKey = "jobs.create"
	}
	return &amqpJobPublisher{pub: pub, routingKey: routingKey}
}

func (a *amqpJobPublisher) PublishJobCreation(ctx context.Context, req cpdomain.CreateJobRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal job creation request: %w", err)
	}
	return a.pub.Publish(ctx, a.routingKey, body)
}
