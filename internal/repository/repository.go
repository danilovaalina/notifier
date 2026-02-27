package repository

import (
	"context"
	"time"

	"notifier/internal/model"

	sq "github.com/Masterminds/squirrel"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	pool    *pgxpool.Pool
	redis   *redis.Client
	builder sq.StatementBuilderType
}

func New(pool *pgxpool.Pool, redis *redis.Client) *Repository {
	return &Repository{
		pool:    pool,
		redis:   redis,
		builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// Create сохраняет уведомление в БД и кэширует статус в Redis
func (r *Repository) CreateNotification(ctx context.Context, n model.Notification) (model.Notification, error) {
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
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow[notificationRow](rows, pgx.RowToStructByNameLax[notificationRow])
	if err != nil {
		return model.Notification{}, errors.WithStack(err)
	}

	return r.notificationModel(row), nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (model.Notification, error) {
	n, err := r.getNotificationFromRedis(ctx, id)
	if err == nil {
		return n, nil
	} else if !errors.Is(err, redis.Nil) {
		return model.Notification{}, err
	}

	query := `
	select id, channel, recipient, message, status, retry_count, scheduled_time, created
	from notifications
	where id = $1`

	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return model.Notification{}, errors.WithStack(err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow[notificationRow](rows, pgx.RowToStructByNameLax[notificationRow])
	if err != nil {
		return model.Notification{}, errors.WithStack(err)
	}

	n = r.notificationModel(row)

	err = r.storeNotificationInRedis(ctx, n)
	if err != nil {
		return model.Notification{}, err
	}

	return n, nil
}

func (r *Repository) UpdateNotification(ctx context.Context, notification model.Notification) (model.Notification, error) {

	b := r.builder.Update("notifications").
		Where(sq.Eq{"id": notification.ID}).
		Set("updated", time.Now())

	if notification.Status != "" {
		b = b.Set("status", notification.Status)
	}

	if notification.RetryCount > 0 {
		b = b.Set("retry_count", notification.RetryCount)
	}

	if !notification.ScheduledTime.IsZero() {
		b = b.Set("scheduled_time", notification.ScheduledTime)
	}

	b = b.Suffix("returning id, channel, recipient, message, status, retry_count, scheduled_time, created, updated")

	sql, args, err := b.ToSql()
	if err != nil {
		return model.Notification{}, errors.WithStack(err)
	}

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return model.Notification{}, errors.WithStack(err)
	}

	row, err := pgx.CollectExactlyOneRow[notificationRow](rows, pgx.RowToStructByNameLax[notificationRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Notification{}, model.ErrNotFound
		}
		return model.Notification{}, errors.WithStack(err)
	}

	err = r.deleteNotificationFromRedis(ctx, notification.ID)
	if err != nil {
		return model.Notification{}, err
	}

	return r.notificationModel(row), nil
}

// GetReadyNotifications — возвращает уведомления для отправки (без кэширования)
func (r *Repository) GetReadyNotifications(ctx context.Context, limit int) ([]model.Notification, error) {
	query, args, _ := r.builder.Select(
		"id", "channel", "recipient", "message",
		"status", "retry_count", "scheduled_time", "created",
	).From("notification").
		Where(sq.And{
			sq.Eq{"status": model.StatusScheduled},
			sq.LtOrEq{"scheduled_time": time.Now()},
		}).
		OrderBy("scheduled_time ASC").
		Limit(uint64(limit)).
		ToSql()

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	notificationRows, err := pgx.CollectRows[notificationRow](rows, pgx.RowToStructByNameLax[notificationRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.WithStack(model.ErrNotFound)
		}
		return nil, errors.WithStack(err)
	}

	notifications := make([]model.Notification, 0, len(notificationRows))
	for _, row := range notificationRows {
		notifications = append(notifications, r.notificationModel(row))
	}

	return notifications, nil
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
