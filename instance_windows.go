package main

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

// The same mutex covers every execution mode, folder and Windows login session.
var instanceName = windows.StringToUTF16Ptr(`Global\ParadeBot.Instance`)

func checkInstance() error {
	handle, err := windows.OpenMutex(windows.SYNCHRONIZE, false, instanceName)
	if err == nil {
		windows.CloseHandle(handle)
		return fmt.Errorf("ParadeBot is already running; stop the existing copy before launching another")
	}
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return nil
	}
	return fmt.Errorf("cannot check ParadeBot's instance lock (another Windows session may own it): %w", err)
}

func acquireInstance() (func(), error) {
	handle, err := windows.CreateMutex(nil, false, instanceName)
	if err != nil {
		if handle != 0 {
			windows.CloseHandle(handle)
		}
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			return nil, fmt.Errorf("ParadeBot is already running; stop the existing copy before launching another")
		}
		return nil, fmt.Errorf("cannot acquire ParadeBot's instance lock (another Windows session may own it): %w", err)
	}
	// Closing the last handle, including on a crash, automatically removes the lock.
	return func() { windows.CloseHandle(handle) }, nil
}
