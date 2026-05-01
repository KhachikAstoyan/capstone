package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

// Publisher publishes JSON messages to a durable topic exchange.
type Publisher struct {
	mu       sync.Mutex
	conn     *amqp091.Connection
	ch       *amqp091.Channel
	exchange string
}

// NewPublisher dials RabbitMQ, opens a channel, and declares the exchange (topic, durable).
func NewPublisher(amqpURL, exchange string) (*Publisher, error) {
	if exchange == "" {
		return nil, fmt.Errorf("rabbitmq: exchange name is required")
	}
	conn, err := amqp091.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	if err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq declare exchange: %w", err)
	}
	return &Publisher{conn: conn, ch: ch, exchange: exchange}, nil
}

// Publish sends a persistent message to the configured exchange.
func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch == nil {
		return errors.New("rabbitmq: publisher is closed")
	}
	return p.ch.PublishWithContext(ctx, p.exchange, routingKey, false, false, amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,
		Body:         body,
		Timestamp:    time.Now(),
	})
}

// Close closes the channel and connection.
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var errs []error
	if p.ch != nil {
		if err := p.ch.Close(); err != nil {
			errs = append(errs, err)
		}
		p.ch = nil
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			errs = append(errs, err)
		}
		p.conn = nil
	}
	return errors.Join(errs...)
}
