package main

import (
	"context"
	"testing"
	"time"
)

func TestStopInterruptsRandomWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan bool, 1)
	controller := &Controller{}
	go func() {
		manualRan, _ := controller.wait(ctx, 90*time.Minute)
		finished <- manualRan
	}()
	cancel()
	select {
	case elapsed := <-finished:
		if elapsed {
			t.Fatal("cancelled wait must not start an update")
		}
	case <-time.After(time.Second):
		t.Fatal("stop did not interrupt the wait")
	}
}

func TestRandomDelayUsesTodaysWindow(t *testing.T) {
	controller := &Controller{endTime: time.Date(2026, 9, 9, 7, 0, 0, 0, time.UTC)}
	now := time.Date(2026, 9, 10, 5, 30, 0, 0, time.UTC)
	positive := false
	for i := 0; i < 20; i++ {
		delay := addRandomDelay(controller, now)
		if delay < 0 || delay >= 90*60 {
			t.Fatalf("delay %d is outside today's window", delay)
		}
		positive = positive || delay > 0
	}
	if !positive {
		t.Fatal("delay still uses yesterday's end time")
	}
	for _, now := range []time.Time{
		time.Date(2026, 9, 10, 6, 59, 59, 500000000, time.UTC),
		time.Date(2026, 9, 10, 7, 0, 30, 0, time.UTC),
	} {
		if delay := addRandomDelay(controller, now); delay != 0 {
			t.Fatalf("expected no delay at boundary, got %d", delay)
		}
	}
}
