package worker

import (
	"context"
	"encoding/json"
	"log"

	"notifier/internal/model"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
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
		return nil, errors.Wrap(err, "worker failed to connect to rabbitmq")
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, errors.Wrap(err, "worker failed to open channel")
	}

	if err = ch.Qos(1, 0, false); err != nil {
		return nil, errors.Wrap(err, "worker failed to set QoS")
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
		log.Fatal("failed to register a consumer:", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping worker: context cancelled")
				return

			case d, ok := <-msgs:
				if !ok {
					log.Println("RabbitMQ channel closed")
					return
				}

				var n model.Notification
				if err = json.Unmarshal(d.Body, &n); err != nil {
					log.Printf("Error decoding message: %v", err)
					d.Ack(false)
					continue
				}

				err = w.service.ProcessNotification(ctx, n.ID)
				if err != nil {
					log.Printf("Task %s failed: %v", n.ID, err)
				}

				d.Ack(false)
			}
		}
	}()
}
