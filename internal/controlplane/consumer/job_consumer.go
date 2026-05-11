// Package consumer contains the RabbitMQ consumer for the control plane.
// It receives async job creation requests published by the API service and
// forwards them to the control plane service layer.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/controlplane/domain"
	"github.com/KhachikAstoyan/capstone/internal/controlplane/service"
	"github.com/KhachikAstoyan/capstone/pkg/rabbitmq"
	"go.uber.org/zap"
)

// JobConsumer consumes job creation messages published by the API service.
type JobConsumer struct {
	consumer *rabbitmq.Consumer
	svc      service.Service
	log      *zap.Logger
}

// NewJobConsumer creates a JobConsumer backed by the given RabbitMQ consumer config.
func NewJobConsumer(cfg rabbitmq.ConsumerConfig, svc service.Service, log *zap.Logger) (*JobConsumer, error) {
	c, err := rabbitmq.NewConsumer(cfg)
	if err != nil {
		return nil, fmt.Errorf("job consumer: %w", err)
	}
	return &JobConsumer{consumer: c, svc: svc, log: log}, nil
}

// Run blocks until ctx is cancelled, processing messages from the queue.
func (jc *JobConsumer) Run(ctx context.Context) error {
	return jc.consumer.Run(ctx, jc.handle)
}

// Close shuts down the underlying AMQP channel and connection.
func (jc *JobConsumer) Close() error {
	return jc.consumer.Close()
}

func (jc *JobConsumer) handle(ctx context.Context, body []byte) error {
	var req domain.CreateJobRequest
	if err := json.Unmarshal(body, &req); err != nil {
		// Malformed message — nack without requeue would be ideal, but the
		// Consumer wrapper nacks with requeue=true on error. Log and return
		// a sentinel so the message is not silently dropped; operator must
		// inspect the dead-letter queue.
		jc.log.Error("malformed job creation message", zap.Error(err), zap.ByteString("body", body))
		return fmt.Errorf("unmarshal job creation: %w", err)
	}

	job, err := jc.svc.CreateJob(ctx, req)
	if err != nil {
		jc.log.Error("failed to create job from queue message",
			zap.String("submission_id", req.SubmissionID.String()),
			zap.Error(err),
		)
		return err
	}

	jc.log.Info("job created from queue message",
		zap.String("job_id", job.ID.String()),
		zap.String("submission_id", req.SubmissionID.String()),
		zap.String("language", req.Language),
	)
	return nil
}
