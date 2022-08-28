package scheduler

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

const (
	DEFAULT_ENDPOINT = "127.0.0.1:8080"
	DEFAULT_TIMEOUT  = int64(60)
)

var (
	DefaultLogger = zerolog.New(os.Stderr).Level(zerolog.InfoLevel).With().Timestamp().Logger()
)

func defaultNextSchedule() int64 {
	return time.Now().Unix() + 60*60*24*365 // 365 days later
}

func (s *Scheduler) defaultErrHandler(err error) {
	s.logger.Error().Stack().Err(err).Msg("handled by default err handler")
}

type Opt interface {
	Apply(s *Scheduler)
}

type EndpointOpt string

func (o EndpointOpt) Apply(s *Scheduler) {
	s.endpoint = string(o)
}
func WithEndpoint(endpoint string) EndpointOpt {
	return EndpointOpt(endpoint)
}

type TimeoutOpt int64

func (t TimeoutOpt) Apply(s *Scheduler) {
	s.timeout = int64(t)
}
func WithTimeout(t int64) TimeoutOpt {
	if t <= 0 {
		panic("Timeout should be positive")
	}
	return TimeoutOpt(t)
}

type LoggerOpt zerolog.Logger

func (o LoggerOpt) Apply(s *Scheduler) {
	s.logger = zerolog.Logger(o)
}
func WithLoggerOpt(logger zerolog.Logger) LoggerOpt {
	return LoggerOpt(logger)
}

func (f ErrHandler) Apply(s *Scheduler) {
	s.errHandler = f
}
func WithErrHandler(f func(error)) ErrHandler {
	return ErrHandler(f)
}
