package worker

import (
	"context"
	"encoding/json"
	"time"

	"notifier/internal/model"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

type NotificationService interface {
	ProcessNotification(ctx context.Context, id uuid.UUID) error
}

type Worker struct {
	conn    *amqp.Connection
	ch      *amqp.Channel
	service NotificationService
}

func New(amqpURL string, svc NotificationService) (*Worker, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, errors.WithStack(err)
	}

	if err = ch.Qos(1, 0, false); err != nil {
		return nil, errors.WithStack(err)
	}

	return &Worker{
		conn:    conn,
		ch:      ch,
		service: svc,
	}, nil
}

func (w *Worker) Start(ctx context.Context) {
	msgs, err := w.ch.Consume(
		"notifications_q",
		"",
		false, // auto-ack ставим false, чтобы подтверждать только после успеха
		false, false, false, nil,
	)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("stopping worker: context cancelled")
				return

			case d, ok := <-msgs:
				if !ok {
					log.Info().Msg("RabbitMQ channel closed")
					return
				}

				var n model.Notification
				if err = json.Unmarshal(d.Body, &n); err != nil {
					log.Info().Msgf("error decoding message: %v", err)
					d.Ack(false)
					continue
				}

				log.Info().
					Interface("id", n.ID).
					Time("scheduled", n.ScheduledTime).
					Time("actual_received", time.Now()).
					Msg("worker received message from queue")

				err = w.service.ProcessNotification(ctx, n.ID)
				if err != nil {
					log.Info().Msgf("task %s failed: %v", n.ID, err)
					//d.Nack(false, true)
					//return
				}

				d.Ack(false)
			}
		}
	}()
}
