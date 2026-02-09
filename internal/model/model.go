package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// NotificationStatus — статусы жизненного цикла
type NotificationStatus string

const (
	StatusScheduled NotificationStatus = "scheduled" // В ожидании времени отправки
	StatusPublished NotificationStatus = "published" // Опубликовано в очередь
	StatusSent      NotificationStatus = "sent"      // Успешно отправлено
	StatusFailed    NotificationStatus = "failed"    // Превышено кол-во попыток
	StatusCancelled NotificationStatus = "cancelled" // Отменено пользователем
)

// NotificationChannel — поддерживаемые каналы доставки
type NotificationChannel string

const (
	ChannelEmail    NotificationChannel = "email"
	ChannelTelegram NotificationChannel = "telegram"
)

// Notification — основная сущность уведомления
type Notification struct {
	ID            uuid.UUID
	Channel       NotificationChannel
	Recipient     string
	Message       string
	Status        NotificationStatus
	RetryCount    int64
	ScheduledTime time.Time
	Created       time.Time
	Updated       time.Time
}

// Errors
var (
	ErrNotFound    = errors.New("notification not found")
	ErrCancelled   = errors.New("notification cancelled")
	ErrInvalidTime = errors.New("scheduled time must be in the future")
)
