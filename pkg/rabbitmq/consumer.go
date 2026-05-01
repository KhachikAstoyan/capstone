package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

// ConsumerConfig configures a durable queue bound to a topic exchange (same topology as Publisher).
type ConsumerConfig struct {
	URL         string
	Exchange    string
	Queue       string
	RoutingKey  string
	ConsumerTag string
}

// Consumer receives messages from a bound queue until Run returns.
type Consumer struct {
	mu          sync.Mutex
	conn        *amqp091.Connection
	ch          *amqp091.Channel
	queue       string
	consumerTag string
}

// NewConsumer dials RabbitMQ, declares the exchange and queue, and binds the queue with the routing key.
func NewConsumer(cfg ConsumerConfig) (*Consumer, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("rabbitmq: consumer URL is required")
	}
	if cfg.Exchange == "" {
		return nil, fmt.Errorf("rabbitmq: exchange name is required")
	}
	if cfg.Queue == "" {
		return nil, fmt.Errorf("rabbitmq: queue name is required")
	}
	if cfg.RoutingKey == "" {
		return nil, fmt.Errorf("rabbitmq: routing key is required")
	}
	tag := cfg.ConsumerTag
	if tag == "" {
		tag = "capstone-consumer"
	}

	conn, err := amqp091.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		cfg.Exchange,
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

	if _, err := ch.QueueDeclare(
		cfg.Queue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq declare queue: %w", err)
	}

	if err := ch.QueueBind(
		cfg.Queue,
		cfg.RoutingKey,
		cfg.Exchange,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq queue bind: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq qos: %w", err)
	}

	return &Consumer{conn: conn, ch: ch, queue: cfg.Queue, consumerTag: tag}, nil
}

// Run consumes deliveries until ctx is cancelled or the channel closes. The handler is invoked for each message body; ack is sent after the handler returns nil.
func (c *Consumer) Run(ctx context.Context, handler func(context.Context, []byte) error) error {
	c.mu.Lock()
	ch := c.ch
	tag := c.consumerTag
	c.mu.Unlock()
	if ch == nil {
		return errors.New("rabbitmq: consumer is closed")
	}

	deliveries, err := ch.Consume(
		c.queue,
		tag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("rabbitmq consume: %w", err)
	}

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = ch.Cancel(tag, false)
		case <-done:
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("rabbitmq: deliveries channel closed")
			}
			handleErr := handler(ctx, d.Body)
			if handleErr != nil {
				_ = d.Nack(false, true)
				continue
			}
			if err := d.Ack(false); err != nil {
				return fmt.Errorf("rabbitmq ack: %w", err)
			}
		}
	}
}

// Close closes the channel and connection.
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var errs []error
	if c.ch != nil {
		if err := c.ch.Close(); err != nil {
			errs = append(errs, err)
		}
		c.ch = nil
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, err)
		}
		c.conn = nil
	}
	return errors.Join(errs...)
}
