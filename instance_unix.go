//go:build darwin || linux

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func acquireInstance() (func(), error) {
	directory, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	directory = filepath.Join(directory, "ParadeBot")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(directory, "instance.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("ParadeBot is already running or its instance lock is unavailable: %w", err)
	}
	// Keep the file in place; deleting it can allow two different inodes to be locked.
	return func() { file.Close() }, nil
}
