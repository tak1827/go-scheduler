package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var timeoutDuration time.Duration

func (s *Scheduler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/register", s.handleRegistSchedule)

	middles := []Middleware{
		s.setTimeout,
		s.setHeaders,
	}

	handler := applyMiddlewares(mux.ServeHTTP, middles)
	handler(w, r)
}

func (s *Scheduler) handleRoot(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("{\"msg\":\"hello world!\"}"))
}

type RegisterScheduleMsg struct {
	Schedule int64 `json:"schedule"`
}

func (s *Scheduler) handleRegistSchedule(w http.ResponseWriter, req *http.Request) {
	var m RegisterScheduleMsg
	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
		s.handleErr(w, fmt.Errorf("failed to decode RegisterScheduleMsg, err: %w", err), nil)
		return
	}

	if err := s.RegistSchedule(m.Schedule); err != nil {
		s.errHandler(err)
	}
}

type errMsg struct {
	Err string `json:"err"`
}

func (s *Scheduler) handleErr(w http.ResponseWriter, err error, code *int) {
	s.logger.Warn().Msgf("err handled, %s", err.Error())

	if code == nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(*code)
	}

	msg, err := json.Marshal(errMsg{Err: err.Error()})
	if err != nil {
		s.logger.Fatal().Msgf("failed to marshal err msg(%s)", err.Error())
	}
	_, _ = w.Write(msg)
}

type Middleware func(next http.HandlerFunc) http.HandlerFunc

func applyMiddlewares(f http.HandlerFunc, middle []Middleware) http.HandlerFunc {
	last := f
	for i := len(middle) - 1; i >= 0; i-- {
		last = middle[i](last)
	}
	return last
}

func (s *Scheduler) setTimeout(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
		defer cancel()

		next(w, r.Clone(ctx))
	}
}

func (s *Scheduler) setHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next(w, r)
	}
}
