//go:build !windows

package main

import (
	"context"
	"fmt"
)

func launchBackground() error {
	return fmt.Errorf("the tray/background mode is Windows-only; see README.md for macOS/Linux background instructions")
}

func defaultBackground() bool {
	return false
}

func runTray(context.Context, context.CancelFunc) error {
	return launchBackground()
}
