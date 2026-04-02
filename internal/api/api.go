package api

import (
	"context"
	"net/http"
	"time"

	"notifier/internal/model"

	"github.com/cockroachdb/errors"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Service interface {
	CreateNotification(ctx context.Context, notification model.Notification) (model.Notification, error)
	GetNotification(ctx context.Context, id uuid.UUID) (model.Notification, error)
	CancelNotification(ctx context.Context, id uuid.UUID) error
	Notifications(ctx context.Context, opts model.NotificationFilter) ([]model.Notification, error)
}

// API — HTTP сервер на основе Echo
type API struct {
	*echo.Echo
	service Service
}

// New создаёт новый API сервер
func New(service Service) *API {
	a := &API{
		Echo:    echo.New(),
		service: service,
	}
	a.Validator = &CustomValidator{validator: validator.New()}

	a.Static("/", "./static")

	api := a.Group("/api")
	api.GET("/ping", a.ping)

	notifications := api.Group("/notify")
	{
		notifications.POST("", a.createNotification)
		notifications.GET("", a.notifications)
		notifications.GET("/:id", a.notification)
		notifications.DELETE("/:id", a.cancelNotification)
	}

	return a
}

// ping — health check эндпоинт
func (a *API) ping(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// createNotificationRequest — входные данные для создания уведомления
type createNotificationRequest struct {
	Channel       string    `json:"channel" validate:"required,oneof=email telegram"`
	Recipient     string    `json:"recipient" validate:"required"`
	Message       string    `json:"message" validate:"required,min=1,max=4096"`
	ScheduledTime time.Time `json:"scheduled_time" validate:"required"` // RFC3339
}

// createNotification — POST /api/notify
func (a *API) createNotification(c echo.Context) error {
	var req createNotificationRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	notification := model.Notification{
		Channel:       model.NotificationChannel(req.Channel),
		Recipient:     req.Recipient,
		Message:       req.Message,
		ScheduledTime: req.ScheduledTime.UTC(),
	}

	n, err := a.service.CreateNotification(c.Request().Context(), notification)
	if err != nil {
		if errors.Is(err, model.ErrInvalidTime) {
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, model.ErrUnsupportedChannel) {
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error": "unsupported channel, use 'email' or 'telegram'",
			})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create notification"})
	}

	return c.JSON(http.StatusCreated, a.notificationFromModel(n))
}

type notificationRequest struct {
	ID uuid.UUID `param:"id" validate:"required"`
}

func (a *API) notification(c echo.Context) error {
	var req notificationRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid ID format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	n, err := a.service.GetNotification(c.Request().Context(), req.ID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "notification not found"})
		}

		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to get notification"})
	}

	return c.JSON(http.StatusOK, a.notificationFromModel(n))
}

type notificationsRequest struct {
	Limit  uint64 `query:"limit"`
	Offset uint64 `query:"offset"`
}

func (a *API) notifications(c echo.Context) error {
	var req notificationsRequest

	err := c.Bind(&req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"reason": "invalid request format or params"})
	}

	opts := model.NotificationFilter{
		Offset: req.Offset,
		Limit:  req.Limit,
	}

	notifications, err := a.service.Notifications(c.Request().Context(), opts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to get notifications"})
	}

	return c.JSON(http.StatusOK, a.notificationsFromModel(notifications))
}

type cancelNotificationRequest struct {
	ID uuid.UUID `param:"id" validate:"required"`
}

func (a *API) cancelNotification(c echo.Context) error {
	var req cancelNotificationRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid ID format"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if err := a.service.CancelNotification(c.Request().Context(), req.ID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "notification not found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to cancel notification"})
	}

	return c.NoContent(http.StatusNoContent)
}

type notificationResponse struct {
	ID            uuid.UUID `json:"id"`
	Channel       string    `json:"channel"`
	Recipient     string    `json:"recipient"`
	Message       string    `json:"message"`
	Status        string    `json:"status"`
	RetryCount    int64     `json:"retry_count"`
	ScheduledTime time.Time `json:"scheduled_time"`
	Created       time.Time `json:"created"`
}

func (a *API) notificationFromModel(notification model.Notification) notificationResponse {
	return notificationResponse{
		ID:            notification.ID,
		Channel:       string(notification.Channel),
		Recipient:     notification.Recipient,
		Message:       notification.Message,
		Status:        string(notification.Status),
		RetryCount:    notification.RetryCount,
		ScheduledTime: notification.ScheduledTime.UTC(),
		Created:       notification.Created,
	}
}

func (a *API) notificationsFromModel(notifications []model.Notification) []notificationResponse {
	r := make([]notificationResponse, 0, len(notifications))
	for _, t := range notifications {
		r = append(r, a.notificationFromModel(t))
	}

	return r
}
