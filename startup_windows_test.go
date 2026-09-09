package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestStartupOptInAndRemoval(t *testing.T) {
	previous := startupKeyPath
	startupKeyPath = fmt.Sprintf(`Software\ParadeBotTest-%d`, os.Getpid())
	t.Cleanup(func() {
		if err := registry.DeleteKey(registry.CURRENT_USER, startupKeyPath); err != nil && !errors.Is(err, registry.ErrNotExist) {
			t.Error(err)
		}
		startupKeyPath = previous
	})
	if enabled, err := startupEnabled(); err != nil || enabled {
		t.Fatalf("startup must default to off: %t, %v", enabled, err)
	}
	if key, err := registry.OpenKey(registry.CURRENT_USER, startupKeyPath, registry.READ); err == nil {
		key.Close()
		t.Fatal("reading the default must not create a registry key")
	} else if !errors.Is(err, registry.ErrNotExist) {
		t.Fatal(err)
	}
	command := syscall.EscapeArg(`C:\folder with spaces\ParadeBot.exe`) + " -background"
	if err := writeStartup(command); err != nil {
		t.Fatal(err)
	}
	if enabled, err := startupEnabled(); err != nil || !enabled {
		t.Fatalf("startup must be on after opt-in: %t, %v", enabled, err)
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, startupKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer key.Close()
	if got, _, err := key.GetStringValue("ParadeBot"); err != nil || got != command {
		t.Fatalf("startup command lost quoting: %q, %v", got, err)
	}
	if err := key.SetStringValue("UnrelatedApp", "leave this alone"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := setStartup(false); err != nil {
			t.Fatal(err)
		}
		if enabled, err := startupEnabled(); err != nil || enabled {
			t.Fatalf("startup must be off after removal: %t, %v", enabled, err)
		}
	}
	if got, _, err := key.GetStringValue("UnrelatedApp"); err != nil || got != "leave this alone" {
		t.Fatalf("unrelated startup entry changed: %q, %v", got, err)
	}
}
