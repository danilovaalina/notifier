package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// NotificationStatus — статусы жизненного цикла
type NotificationStatus string

const (
	StatusScheduled NotificationStatus = "scheduled" // Ожидает времени отправки
	StatusSent      NotificationStatus = "sent"      // Успешно доставлено получателю
	StatusFailed    NotificationStatus = "failed"    // Все попытки исчерпаны
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

type NotificationFilter struct {
	Offset uint64
	Limit  uint64
}

// Errors
var (
	ErrNotFound           = errors.New("notification not found")
	ErrCancelled          = errors.New("notification cancelled")
	ErrInvalidTime        = errors.New("scheduled time must be in the future")
	ErrUnsupportedChannel = errors.New("unsupported notification channel")
)
