# go-scheduler
The "Shortest remaining time first" scheduler

# Sample
```go
package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/tak1827/go-scheduler/scheduler"
)

func main() {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		isSlave     = false
		withServer  = true
		counter     int64
		work        = func(any ...interface{}) error {
			atomic.AddInt64(&counter, 1)
			return nil
		}
		upcomingSchedule = func() (int64, bool) {
			return time.Now().Unix() + 1, true
		}
	)
	sch := scheduler.NewScheduler(isSlave, withServer, work, upcomingSchedule)
	sch.Start(ctx)

	if err := sch.RegistSchedule(time.Now().Unix()); err != nil {
		panic(err)
	}

	sch.Close(cancel)

	fmt.Printf("counger is %d", counter)
}
```
