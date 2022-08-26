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
	sch := NewScheduler(false, false, MockWork, MockUpcoming)

	sch.Start(ctx)
	sch.Close(cancel)
}

// func TestCloseWithSrv(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	sch := NewScheduler(false, true, MockWork, MockUpcoming)

// 	sch.Start(ctx)
// 	sch.Close(cancel)
// }

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

	sch := NewScheduler(false, false, work, MockUpcoming, WithErrHandler(errHandler))
	sch.Start(ctx)

	for i := 0; i < 3; i++ {
		err := sch.RegistSchedule(time.Now().Unix())
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	// while waiting upcoming will be exected
	time.Sleep(1 * time.Second)

	sch.Close(cancel)

	require.Equal(t, int64(4), atomic.LoadInt64(&counter))
}

func TestSlave(t *testing.T) {
	var (
		ctxForIn, cancelForIn   = context.WithCancel(context.Background())
		ctxForOut, cancelForOut = context.WithCancel(context.Background())
		counter                 int64
		work                    = func() error {
			atomic.AddInt64(&counter, 1)
			return nil
		}
		errHandler = func(err error) {
			require.NoError(t, err)
		}
	)

	inProcessSch := NewScheduler(false, true, work, MockUpcoming, WithErrHandler(errHandler))
	inProcessSch.Start(ctxForIn)

	outProcessSch := NewScheduler(true, false, MockWork, MockUpcoming, WithErrHandler(errHandler))
	outProcessSch.Start(ctxForOut)

	for i := 0; i < 3; i++ {
		err := outProcessSch.RegistSchedule(time.Now().Unix())
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	// while waiting upcoming will be exected
	time.Sleep(1 * time.Second)

	inProcessSch.Close(cancelForIn)
	outProcessSch.Close(cancelForOut)

	require.Equal(t, int64(4), atomic.LoadInt64(&counter))
}
