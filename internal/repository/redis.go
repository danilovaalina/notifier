package repository

import (
	"context"
	"strconv"
	"time"

	"notifier/internal/model"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	redisKeyPrefix = "notif:" // Префикс для ключей уведомлений
	redisTTL       = 5 * time.Minute
)

// notificationCache — структура для хранения в Redis (хэш)
type notificationCache struct {
	ID                  string `redis:"id"`
	NotificationChannel string `redis:"notification_channel"`
	Recipient           string `redis:"recipient"`
	Message             string `redis:"message"`
	Status              string `redis:"status"`
	RetryCount          string `redis:"retry_count"`
	ScheduledTime       string `redis:"scheduled_time"`
	Created             string `redis:"created"`
	Updated             string `redis:"updated"`
}

// convertToNotificationCache — конвертирует модель в кэш-структуру
func convertToNotificationCache(n model.Notification) notificationCache {
	return notificationCache{
		ID:                  n.ID.String(),
		NotificationChannel: string(n.Channel),
		Recipient:           n.Recipient,
		Message:             n.Message,
		Status:              string(n.Status),
		RetryCount:          strconv.FormatInt(n.RetryCount, 10),
		ScheduledTime:       n.ScheduledTime.UTC().Format(time.RFC3339Nano),
		Created:             n.Created.UTC().Format(time.RFC3339Nano),
		Updated:             n.Updated.UTC().Format(time.RFC3339Nano),
	}
}

// convertToNotification — конвертирует кэш-структуру обратно в модель
func convertToNotification(nc notificationCache) (model.Notification, error) {
	id, err := uuid.Parse(nc.ID)
	if err != nil {
		return model.Notification{}, err
	}

	retryCount, err := strconv.ParseInt(nc.RetryCount, 10, 64)
	if err != nil {
		return model.Notification{}, errors.WithDetail(err, "failed to parse retry_count")
	}

	scheduledTime, err := time.Parse(time.RFC3339Nano, nc.ScheduledTime)
	if err != nil {
		return model.Notification{}, errors.WithDetail(err, "failed to parse scheduled_time")
	}

	created, err := time.Parse(time.RFC3339Nano, nc.Created)
	if err != nil {
		return model.Notification{}, errors.WithDetail(err, "failed to parse created_at")
	}

	return model.Notification{
		ID:            id,
		Channel:       model.NotificationChannel(nc.NotificationChannel),
		Recipient:     nc.Recipient,
		Message:       nc.Message,
		Status:        model.NotificationStatus(nc.Status),
		RetryCount:    retryCount,
		ScheduledTime: scheduledTime,
		Created:       created,
	}, nil
}

// storeNotificationInRedis — сохраняет уведомление в кэш (хэш)
func (r *Repository) storeNotificationInRedis(ctx context.Context, n model.Notification) error {
	nc := convertToNotificationCache(n)

	key := redisKeyPrefix + nc.ID

	err := r.redis.HSet(ctx, key, map[string]interface{}{
		"id":                   nc.ID,
		"notification_channel": nc.NotificationChannel,
		"recipient":            nc.Recipient,
		"message":              nc.Message,
		"status":               nc.Status,
		"retry_count":          nc.RetryCount,
		"scheduled_time":       nc.ScheduledTime,
		"created":              nc.Created,
		"updated":              nc.Updated,
	}).Err()
	if err != nil {
		return errors.WithStack(err)
	}

	// Устанавливаем TTL для автоматической очистки
	_ = r.redis.Expire(ctx, key, redisTTL)

	return nil
}

// getNotificationFromRedis — получает уведомление из кэша
func (r *Repository) getNotificationFromRedis(ctx context.Context, id uuid.UUID) (model.Notification, error) {
	key := redisKeyPrefix + id.String()

	result, err := r.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return model.Notification{}, errors.WithStack(err)
	}

	// Если хэш пустой (ключ не существует)
	if len(result) == 0 {
		return model.Notification{}, redis.Nil
	}

	nc := notificationCache{
		ID:                  result["id"],
		NotificationChannel: result["notification_channel"],
		Recipient:           result["recipient"],
		Message:             result["message"],
		Status:              result["status"],
		RetryCount:          result["retry_count"],
		ScheduledTime:       result["scheduled_time"],
		Created:             result["created"],
	}

	n, err := convertToNotification(nc)
	if err != nil {
		return model.Notification{}, errors.WithStack(err)
	}

	return n, nil
}

// deleteNotificationFromRedis — удаляет уведомление из кэша (инвалидация)
func (r *Repository) deleteNotificationFromRedis(ctx context.Context, id uuid.UUID) error {
	key := redisKeyPrefix + id.String()
	_ = r.redis.Del(ctx, key)
	return nil
}
