package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"notifier/internal/api"
	"notifier/internal/db/postgres"
	"notifier/internal/db/redis"
	"notifier/internal/queue"
	"notifier/internal/repository"
	"notifier/internal/sender"
	"notifier/internal/service"
	"notifier/internal/worker"

	"github.com/rs/zerolog/log"

	"notifier/internal/config"
)

func main() {
	cf, err := config.Load()
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Pool(ctx, cf.DatabaseURL)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}
	defer pool.Close()

	r, err := redis.Client(cf.RedisURL)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	q, err := queue.NewRabbitMQ(cf.RabbitMQURL)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	s, err := sender.New(
		sender.WithEmail(cf.Email.Host, cf.Email.Port, cf.Email.User, cf.Email.Password, cf.Email.From),
		sender.WithTelegram(cf.BotToken),
	)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	svc := service.New(repository.New(pool, r), q, s)

	w, err := worker.New(cf.RabbitMQURL, svc)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}
	w.Start(ctx)

	a := api.New(svc)
	err = http.ListenAndServe(cf.Addr, a)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

}
