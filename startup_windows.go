package main

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

// Tests use an isolated, non-startup key. Nothing writes to Run until opted in.
var startupKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

func startupEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupKeyPath, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer key.Close()
	command, _, err := key.GetStringValue("ParadeBot")
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	return command != "", err
}

func setStartup(enabled bool) error {
	command := ""
	if enabled {
		executable, err := backgroundExecutable()
		if err != nil {
			return err
		}
		command = syscall.EscapeArg(executable) + " -background"
	}
	return writeStartup(command)
}

func writeStartup(command string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, startupKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if command != "" {
		return key.SetStringValue("ParadeBot", command)
	}
	err = key.DeleteValue("ParadeBot")
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}
