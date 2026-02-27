package queue

import (
	"context"
	"encoding/json"
	"time"

	"notifier/internal/model"

	"github.com/cockroachdb/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to rabbitmq")
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open a channel")
	}

	err = ch.ExchangeDeclare(
		"delayed_notifications",
		"x-delayed-message",
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-delayed-type": "direct",
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to declare delayed exchange")
	}

	_, err = ch.QueueDeclare(
		"notifications_q",
		true, false, false, false, nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to declare queue")
	}

	// Связываем очередь с обменником
	err = ch.QueueBind("notifications_q", "notify_key", "delayed_notifications", false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to bind queue")
	}

	return &RabbitMQ{conn: conn, ch: ch}, nil
}

func (r *RabbitMQ) Publish(ctx context.Context, n model.Notification) error {
	body, err := json.Marshal(n)
	if err != nil {
		return errors.Wrap(err, "failed to marshal notification")
	}

	// Считаем задержку: когда отправить МИНУС сейчас
	delay := time.Until(n.ScheduledTime).Milliseconds()
	if delay < 0 {
		delay = 0 // если время уже прошло, отправляем сразу
	}

	return r.ch.PublishWithContext(ctx,
		"delayed_notifications",
		"notify_key",
		false,
		false,
		amqp.Publishing{
			Headers: amqp.Table{
				"x-delay": delay, // заголовок для плагина в мс
			},
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// Channel предоставляет доступ к каналу RabbitMQ для расширенных операций (например, Consume)
func (r *RabbitMQ) Channel() *amqp.Channel {
	return r.ch
}

func (r *RabbitMQ) Close() {
	_ = r.ch.Close()
	_ = r.conn.Close()
}
