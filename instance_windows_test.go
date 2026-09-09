package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestInstanceLockAcrossProcesses(t *testing.T) {
	previous := instanceName
	name := fmt.Sprintf(`Local\ParadeBot.Test-%d`, os.Getpid())
	instanceName = windows.StringToUTF16Ptr(name)
	t.Cleanup(func() { instanceName = previous })
	release, err := acquireInstance()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := checkInstance(); err == nil {
		t.Fatal("launcher must detect the running instance")
	}
	child := exec.Command(os.Args[0], "-test.run=^TestInstanceLockChild$")
	child.Env = append(os.Environ(), "PARADEBOT_TEST_LOCK="+name, "PARADEBOT_TEST_EXPECT_BLOCKED=1")
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("second process was not blocked: %v, %s", err, output)
	}
}

func TestInstanceLockReleasedAfterCrash(t *testing.T) {
	previous := instanceName
	name := fmt.Sprintf(`Local\ParadeBot.CrashTest-%d`, os.Getpid())
	instanceName = windows.StringToUTF16Ptr(name)
	t.Cleanup(func() { instanceName = previous })
	child := exec.Command(os.Args[0], "-test.run=^TestInstanceLockChild$")
	child.Env = append(os.Environ(), "PARADEBOT_TEST_LOCK="+name)
	output, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { child.Process.Kill() })
	ready := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(output).ReadString('\n')
		ready <- line
	}()
	select {
	case line := <-ready:
		if line != "LOCKED\n" {
			t.Fatalf("child failed to acquire lock: %q", line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("child did not acquire lock")
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	release, err := acquireInstance()
	if err != nil {
		t.Fatalf("crashed process left a stale lock: %v", err)
	}
	release()
	if err := checkInstance(); err != nil {
		t.Fatalf("normal release left a stale lock: %v", err)
	}
}

func TestInstanceLockChild(t *testing.T) {
	name := os.Getenv("PARADEBOT_TEST_LOCK")
	if name == "" {
		return
	}
	instanceName = windows.StringToUTF16Ptr(name)
	release, err := acquireInstance()
	if os.Getenv("PARADEBOT_TEST_EXPECT_BLOCKED") == "1" {
		if err != nil {
			return
		}
		release()
		t.Fatal("acquired a lock already held by another process")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	fmt.Println("LOCKED")
	time.Sleep(30 * time.Second)
}
