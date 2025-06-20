// Package config provides configuration management for the IMS application.
// It handles environment variable loading and validation using envconfig.
package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	RabbitMQ  RabbitMQConfig
	Webhook   WebhookConfig
	Scheduler SchedulerConfig
	Log       LogConfig
	Message   MessageConfig
}

type ServerConfig struct {
	Port         string        `envconfig:"SERVER_PORT" default:"8080"`
	ReadTimeout  time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"15s"`
	WriteTimeout time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"15s"`
}

type DatabaseConfig struct {
	URL                string `envconfig:"DATABASE_URL" required:"true"`
	MaxConnections     int    `envconfig:"DATABASE_MAX_CONNECTIONS" default:"25"`
	MaxIdleConnections int    `envconfig:"DATABASE_MAX_IDLE_CONNECTIONS" default:"5"`
}

type RedisConfig struct {
	URL      string        `envconfig:"REDIS_URL"`
	CacheTTL time.Duration `envconfig:"REDIS_CACHE_TTL" default:"168h"`
}

type RabbitMQConfig struct {
	URL                  string `envconfig:"RABBITMQ_URL"`
	Enabled              bool   `envconfig:"RABBITMQ_ENABLED" default:"false"`
	MessagesQueue        string `envconfig:"RABBITMQ_MESSAGES_QUEUE" default:"messages.pending"`
	RetryQueue           string `envconfig:"RABBITMQ_RETRY_QUEUE" default:"messages.retry"`
	DeadLetterQueue      string `envconfig:"RABBITMQ_DLQ" default:"messages.dead_letter"`
	MaxRetries           int    `envconfig:"RABBITMQ_MAX_RETRIES" default:"5"`
	RetryDelayMultiplier int    `envconfig:"RABBITMQ_RETRY_DELAY_MULTIPLIER" default:"60"` // seconds
}

type WebhookConfig struct {
	URL        string        `envconfig:"WEBHOOK_URL" required:"true"`
	AuthKey    string        `envconfig:"WEBHOOK_AUTH_KEY" required:"true"`
	Timeout    time.Duration `envconfig:"WEBHOOK_TIMEOUT" default:"30s"`
	MaxRetries int           `envconfig:"WEBHOOK_MAX_RETRIES" default:"3"`
}

type SchedulerConfig struct {
	Interval  time.Duration `envconfig:"SCHEDULER_INTERVAL" default:"2m"`
	BatchSize int           `envconfig:"SCHEDULER_BATCH_SIZE" default:"2"`
}

type LogConfig struct {
	Level  string `envconfig:"LOG_LEVEL" default:"info"`
	Format string `envconfig:"LOG_FORMAT" default:"json"`
}

type MessageConfig struct {
	MaxLength int `envconfig:"MESSAGE_MAX_LENGTH" default:"160"`
}

func Load() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	return &cfg, err
}
