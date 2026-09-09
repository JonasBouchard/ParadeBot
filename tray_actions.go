package main

import (
	"context"
	"time"
)

// Automatic and manual updates execute on the same worker, never concurrently.
func (controller *Controller) performUpdate() error {
	setManualReady(false)
	controller.lastRun = time.Now()
	setActivity("Updating parade status")
	update := controller.update
	if update == nil {
		update = startProcess
	}
	err := update()
	recordResult(err)
	return err
}

// A manual run replaces a pending random wait and counts as today's attempt.
// Unbuffered requests are accepted only while waiting, never queued behind work.
func (controller *Controller) wait(ctx context.Context, delay time.Duration) (bool, error) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	setManualReady(true)
	defer setManualReady(false)
	select {
	case <-ctx.Done():
		return false, nil
	case <-timer.C:
		return false, nil
	case result := <-controller.manual:
		if ctx.Err() != nil {
			result <- ctx.Err()
			return false, nil
		}
		logger.Println("Manual update requested from the tray.")
		err := controller.performUpdate()
		result <- err
		return true, err
	}
}
