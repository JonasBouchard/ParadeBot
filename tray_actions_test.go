package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestTrayManualRunReplacesWaitAndCountsToday(t *testing.T) {
	for _, wantErr := range []error{nil, errors.New("simulated browser failure")} {
		t.Run("result="+fmtError(wantErr), func(t *testing.T) {
			previous := logger
			logger = NewCustomLogger(filepath.Join(t.TempDir(), "test.log"))
			t.Cleanup(func() { logger.Close(); logger = previous })
			manual := make(chan chan error)
			started := make(chan struct{})
			finishUpdate := make(chan struct{})
			defer close(finishUpdate)
			calls := 0
			controller := &Controller{manual: manual, update: func() error {
				calls++
				close(started)
				<-finishUpdate
				return wantErr
			}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			finished := make(chan error, 1)
			go func() {
				ran, err := controller.wait(ctx, 90*time.Minute)
				if !ran {
					finished <- errors.New("manual command did not replace wait")
					return
				}
				finished <- err
			}()
			reply := make(chan error, 1)
			select {
			case manual <- reply:
			case <-time.After(time.Second):
				t.Fatal("manual run was not accepted while waiting")
			}
			<-started
			if canRunManually() {
				t.Fatal("manual action remained available during update")
			}
			select {
			case manual <- make(chan error, 1):
				t.Fatal("a second update was queued while busy")
			default:
			}
			finishUpdate <- struct{}{}
			if err := <-finished; !errors.Is(err, wantErr) {
				t.Fatalf("worker result = %v, want %v", err, wantErr)
			}
			if err := <-reply; !errors.Is(err, wantErr) {
				t.Fatalf("tray result = %v, want %v", err, wantErr)
			}
			if calls != 1 || !controller.hasRunToday() {
				t.Fatal("manual run must perform one update and consume today's automatic attempt")
			}
		})
	}
}

func fmtError(err error) string {
	if err == nil {
		return "success"
	}
	return "failure"
}
