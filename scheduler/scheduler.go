package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

type (
	WorkFunc             func() error
	UpcomingScheduleFunc func() (int64, bool)
	ErrHandler           func(error)
)

type Scheduler struct {
	nextSchedule          int64
	replaceNextScheduleCh chan int64
	closeFlg              int64
	logger                zerolog.Logger

	work       WorkFunc
	upcoming   UpcomingScheduleFunc
	errHandler ErrHandler

	outProcess bool
	withServer bool

	// if with server
	endpoint string
	server   http.Server
	client   http.Client
	timeout  int64
}

func NewScheduler(outProcess, withServer bool, work WorkFunc, upcoming UpcomingScheduleFunc, opts ...Opt) (s Scheduler) {
	s.nextSchedule = defaultNextSchedule()
	s.replaceNextScheduleCh = make(chan int64)
	s.errHandler = s.defaultErrHandler
	s.logger = DefaultLogger
	s.outProcess = outProcess
	s.withServer = withServer
	s.endpoint = DEFAULT_ENDPOINT
	s.work = work
	s.upcoming = upcoming

	for i := range opts {
		opts[i].Apply(&s)
	}

	if s.work == nil || s.upcoming == nil {
		panic("worker func or upcoming func is null")
	}

	if s.withServer {
		s.server = http.Server{
			Addr:    s.endpoint,
			Handler: &s,
		}
		s.client = http.Client{}
		timeoutDuration = time.Duration(s.timeout) * time.Second
	}

	return
}

func (s *Scheduler) NextSchedule() int64 {
	return atomic.LoadInt64(&s.nextSchedule)
}

func (s *Scheduler) RegistSchedule(schedule int64) error {
	if s.outProcess {
		var (
			msg = RegisterScheduleMsg{
				Schedule: schedule,
			}
			data, _ = json.Marshal(msg)
			url     = fmt.Sprintf("http://%s/register", s.endpoint)
		)

		resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			return fmt.Errorf("failed to post schedule(=%d). err: %w", schedule, err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected http response: %v", resp)
		}

		return nil
	}

	if schedule < s.NextSchedule() {
		s.ReplaceNextSchedule(schedule)
	}

	return nil
}

func (s *Scheduler) ReplaceNextSchedule(next int64) {
	s.replaceNextScheduleCh <- next
}

func (s *Scheduler) Start(cancelCtx context.Context) {
	if s.withServer {
		go func() {
			ln, err := net.Listen("tcp", s.endpoint)
			if err != nil {
				s.errHandler(err)
				return
			}
			s.logger.Info().Msgf("starting at %s", s.endpoint)
			if err := s.server.Serve(ln); err != nil && !errors.As(err, &http.ErrServerClosed) {
				s.errHandler(err)
			}
		}()
	}

	go func() {
		for {
			var (
				timeout     = s.nextSchedule - time.Now().Unix()
				ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
				wg          sync.WaitGroup
			)
			defer cancel()

			select {
			case <-ctx.Done():
				nextSchedule, hasSchedule := s.upcoming()
				if !hasSchedule {
					nextSchedule = defaultNextSchedule()
				}

				atomic.StoreInt64(&s.nextSchedule, nextSchedule)

				wg.Add(1)
				go func() {
					defer wg.Done()

					if err := s.work(); err != nil {
						s.errHandler(err)
					}
				}()
			case next := <-s.replaceNextScheduleCh:
				atomic.StoreInt64(&s.nextSchedule, next)
			case <-cancelCtx.Done():
				s.logger.Info().Msg("scheduler is closing...")
				// wait until proceeding work done
				wg.Wait()
				atomic.StoreInt64(&s.closeFlg, 1)
				return
			}
		}
	}()
}

func (s *Scheduler) Close(cancel context.CancelFunc) {
	if s.withServer {
		if err := s.server.Shutdown(context.Background()); err != nil {
			s.errHandler(err)
		}
	}

	cancel()

	for atomic.LoadInt64(&s.closeFlg) == 0 {
	}
}
