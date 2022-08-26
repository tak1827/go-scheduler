package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func MockWork() error {
	return nil
}

func MockUpcoming() (int64, bool) {
	return time.Now().Unix() + 1, true
}

func TestClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv := NewScheduler(false, MockWork, MockUpcoming)

	srv.Start(ctx)
	srv.Close(cancel)
}

func TestRegistSchedule(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		counter     int64
		work        = func() error {
			atomic.AddInt64(&counter, 1)
			return nil
		}
		errHandler = func(err error) {
			require.NoError(t, err)
		}
	)

	srv := NewScheduler(false, work, MockUpcoming, WithErrHandler(errHandler))
	srv.Start(ctx)

	for i := 0; i < 3; i++ {
		_ = srv.RegistSchedule(time.Now().Unix())
		time.Sleep(1 * time.Millisecond)
	}

	srv.Close(cancel)

	require.Equal(t, int64(3), atomic.LoadInt64(&counter))
}
