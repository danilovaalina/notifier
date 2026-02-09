package repository

import (
	"context"
	"time"

	"notifier/internal/model"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func New(pool *pgxpool.Pool, redis *redis.Client) *Repository {
	return &Repository{
		pool:  pool,
		redis: redis,
	}
}

// Create сохраняет уведомление в БД и кэширует статус в Redis
func (r *Repository) Create(ctx context.Context, n model.Notification) (model.Notification, error) {
	query := `
	insert into notifications (id, channel, recipient, message, status, retry_count, scheduled_time)
	values ($1, $2, $3, $4, $5, $6, $7)
	returning id, channel, recipient, message, status, retry_count, scheduled_time, created`

	rows, err := r.pool.Query(ctx, query,
		n.ID, n.Channel, n.Recipient, n.Message, n.Status, n.RetryCount, n.ScheduledTime,
	)
	if err != nil {
		return model.Notification{}, errors.WithStack(err)
	}

	row, err := pgx.CollectExactlyOneRow[notificationRow](rows, pgx.RowToStructByNameLax[notificationRow])
	if err != nil {
		return model.Notification{}, errors.WithStack(err)
	}

	return r.notificationModel(row), nil
}

func (r *Repository) notificationModel(row notificationRow) model.Notification {
	return model.Notification{
		ID:            row.ID,
		Channel:       model.NotificationChannel(row.Channel),
		Recipient:     row.Recipient,
		Message:       row.Message,
		Status:        model.NotificationStatus(row.Status),
		RetryCount:    row.RetryCount,
		ScheduledTime: row.ScheduledTime,
		Created:       row.Created,
		Updated:       row.Updated,
	}
}

type notificationRow struct {
	ID            uuid.UUID `db:"id"`
	Channel       string    `db:"channel"`
	Recipient     string    `db:"recipient"`
	Message       string    `db:"message"`
	Status        string    `db:"status"`
	RetryCount    int64     `db:"retry_count"`
	ScheduledTime time.Time `db:"scheduled_time"`
	Created       time.Time `db:"created"`
	Updated       time.Time `db:"updated"`
}
