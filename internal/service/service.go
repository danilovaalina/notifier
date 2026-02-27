package service

import (
	"context"
	"time"

	"notifier/internal/model"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

type Repository interface {
	CreateNotification(ctx context.Context, n model.Notification) (model.Notification, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Notification, error)
	UpdateNotification(ctx context.Context, notification model.Notification) (model.Notification, error)
	GetReadyNotifications(ctx context.Context, limit int) ([]model.Notification, error)
}

type Queue interface {
	Publish(ctx context.Context, n model.Notification) error
}

type Sender interface {
	Send(ctx context.Context, n model.Notification) error
}

type Service struct {
	repo   Repository
	queue  Queue
	sender Sender
}

func New(repo Repository, queue Queue, sender Sender) *Service {
	return &Service{repo: repo, queue: queue, sender: sender}
}

// CreateNotification создаёт уведомление с валидацией
func (s *Service) CreateNotification(ctx context.Context, notification model.Notification) (model.Notification, error) {
	// Валидация времени
	if notification.ScheduledTime.Before(time.Now().Add(10 * time.Second)) {
		return model.Notification{}, model.ErrInvalidTime
	}

	// Валидация канала
	if notification.Channel != "email" && notification.Channel != "telegram" {
		return model.Notification{}, model.ErrUnsupportedChannel
	}

	notification.ID = uuid.New()
	n, err := s.repo.CreateNotification(ctx, notification)
	if err != nil {
		return model.Notification{}, err
	}

	if err = s.queue.Publish(ctx, n); err != nil {
		return n, errors.Wrap(err, "failed to queue notification")
	}

	return n, nil
}

// GetNotification получает уведомление по ID
func (s *Service) GetNotification(ctx context.Context, id uuid.UUID) (model.Notification, error) {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return model.Notification{}, err
	}

	return n, nil
}

// CancelNotification отменяет уведомление
func (s *Service) CancelNotification(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.UpdateNotification(ctx, model.Notification{
		ID:     id,
		Status: model.StatusCancelled,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ProcessNotification(ctx context.Context, id uuid.UUID) error {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if n.Status != model.StatusScheduled || n.ScheduledTime.After(time.Now()) {
		return nil
	}

	err = s.sender.Send(ctx, n)
	if err != nil {
		return s.handleSendError(ctx, n)
	}

	_, err = s.repo.UpdateNotification(ctx, model.Notification{
		ID:     id,
		Status: model.StatusSent,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) handleSendError(ctx context.Context, n model.Notification) error {
	n.RetryCount++

	if n.RetryCount > 5 {
		_, err := s.repo.UpdateNotification(ctx, model.Notification{
			ID:         n.ID,
			Status:     model.StatusFailed,
			RetryCount: n.RetryCount,
		})
		if err != nil {
			return err
		}
		return nil
	}

	delaySec := 30 * (1 << n.RetryCount)
	newTime := time.Now().Add(time.Duration(delaySec) * time.Second)

	_, err := s.repo.UpdateNotification(ctx, model.Notification{
		ID:            n.ID,
		Status:        model.StatusScheduled,
		ScheduledTime: newTime,
		RetryCount:    n.RetryCount,
	})
	if err != nil {
		return err
	}

	return s.queue.Publish(ctx, n)
}
