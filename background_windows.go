package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"
)

func defaultBackground() bool {
	// Built Windows executables open the tray; go run keeps its temporary
	// executable attached to the terminal so Go can manage its lifetime.
	_, err := backgroundExecutable()
	return err == nil
}

func launchBackground() error {
	if err := checkInstance(); err != nil {
		return err
	}
	executable, err := backgroundExecutable()
	if err != nil {
		return err
	}
	directory := filepath.Dir(executable)
	if err := os.MkdirAll(filepath.Join(directory, "logs"), 0755); err != nil {
		return err
	}
	output, err := os.OpenFile(filepath.Join(directory, "logs", "background.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer output.Close()
	command := exec.Command(executable, "-tray")
	command.Dir = directory
	command.Stdout, command.Stderr = output, output
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP}
	if err := command.Start(); err != nil {
		return err
	}
	fmt.Printf("Starting ParadeBot in the notification area (PID %d). You can close this terminal.\n", command.Process.Pid)
	return command.Process.Release()
}

func runTray(ctx context.Context, cancel context.CancelFunc) error {
	systray.Run(func() {
		systray.SetIcon(trayIcon())
		activity := systray.AddMenuItem("Starting", "")
		activity.Disable()
		schedule := systray.AddMenuItem("Weekdays "+startTime.Format("15:04")+" - "+endTime.Format("15:04")+" (PC local time)", "")
		schedule.Disable()
		last := systray.AddMenuItem("Last attempt: no attempt in this session", "")
		last.Disable()
		started := systray.AddMenuItem(fmt.Sprintf("Started %s | PID %d", time.Now().Format("Jan 02 15:04"), os.Getpid()), "")
		started.Disable()
		systray.AddSeparator()
		logs := systray.AddMenuItem("Open log", "Open the activity and error log")
		quit := systray.AddMenuItem("Stop and exit", "An active update or browser installation finishes before exiting")
		systray.AddSeparator()
		startup := systray.AddMenuItemCheckbox("Start with Windows", "Start in the background when you sign in; off by default", false)
		systray.AddSeparator()
		runNow := systray.AddMenuItem("Run once now (real update)", "Submit now, including weekends; counts as today's automatic attempt")
		runNow.Disable()
		testNotification := systray.AddMenuItem("Test notification", "Send a test to your phone without updating parade status")
		manual := make(chan chan error)
		manualResult := make(chan error, 1)
		notificationResult := make(chan error, 1)
		manualRunning, notificationRunning := false, false
		refreshStartup := func() {
			enabled, err := startupEnabled()
			if err != nil {
				logger.Println(fmt.Sprintf("Could not read startup setting: %v", err))
				startup.SetTitle("Start with Windows (unavailable; see log)")
				startup.Disable()
				return
			}
			if enabled {
				startup.Check()
			} else {
				startup.Uncheck()
			}
		}
		refreshStartup()
		finished := make(chan error, 1)
		go func() { finished <- runBot(ctx, false, manual) }()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		stopping, done := false, false
		stopSignal := ctx.Done()
		refresh := func() {
			if !stopping && !done && !manualRunning && canRunManually() {
				runNow.Enable()
			} else {
				runNow.Disable()
			}
			state, result := currentStatus()
			if stopping {
				state = "Stopping; waiting for current work to finish"
			}
			activity.SetTitle(state)
			last.SetTitle("Last attempt: " + result)
			systray.SetTooltip("ParadeBot: " + state)
		}
		refresh()
		for {
			select {
			case err := <-finished:
				done = true
				if err != nil {
					logger.Println(err)
					setActivity("Stopped after error; see log and restart")
				}
				if (stopping || err == nil) && !notificationRunning {
					systray.Quit()
					return
				}
				refresh()
			case <-runNow.ClickedCh:
				if !stopping && !done && !manualRunning {
					select {
					case manual <- manualResult:
						manualRunning = true
					default:
						logger.Println("Run once is unavailable while the bot is busy; try again when it is waiting.")
					}
				}
				refresh()
			case <-manualResult:
				manualRunning = false
				refresh()
			case <-testNotification.ClickedCh:
				if !stopping && !notificationRunning {
					notificationRunning = true
					testNotification.SetTitle("Sending test notification...")
					testNotification.Disable()
					go func() { notificationResult <- testPhoneNotification() }()
				}
			case err := <-notificationResult:
				notificationRunning = false
				if err != nil {
					logger.Println(fmt.Sprintf("Test notification failed: %v", err))
					testNotification.SetTitle("Test notification (failed; see log)")
				} else {
					testNotification.SetTitle("Test notification (sent; check phone)")
				}
				if !stopping {
					testNotification.Enable()
				} else if done {
					systray.Quit()
					return
				}
			case <-logs.ClickedCh:
				path, err := filepath.Abs(loggerFilePath)
				if err == nil {
					command := exec.Command("notepad.exe", path)
					err = command.Start()
					if err == nil {
						go command.Wait()
					}
				}
				if err != nil {
					logger.Println(err)
				}
			case <-quit.ClickedCh:
				cancel()
			case <-startup.ClickedCh:
				enabled, err := startupEnabled()
				if err == nil {
					err = setStartup(!enabled)
				}
				if err != nil {
					logger.Println(fmt.Sprintf("Could not change startup setting: %v", err))
					startup.SetTitle("Start with Windows (change failed; see log)")
				} else {
					startup.SetTitle("Start with Windows")
					logger.Println(fmt.Sprintf("Start with Windows enabled: %t", !enabled))
				}
				refreshStartup()
			case <-stopSignal:
				stopSignal = nil
				stopping = true
				quit.Disable()
				testNotification.Disable()
				if done && !notificationRunning {
					systray.Quit()
					return
				}
				refresh()
			case <-ticker.C:
				refresh()
			}
		}
	}, cancel)
	return nil
}

func backgroundExecutable() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(strings.ToLower(executable), strings.ToLower(filepath.Join(os.TempDir(), "go-build"))) {
		return "", fmt.Errorf("build first: go build -o ParadeBot.exe . ; then run .\\ParadeBot.exe -background")
	}
	return executable, nil
}

// A small blue tile with a white P, encoded as a 32-bit Windows ICO.
func trayIcon() []byte {
	const size = 16
	data := make([]byte, 22+40+size*size*4+size*4)
	binary.LittleEndian.PutUint16(data[2:], 1)
	binary.LittleEndian.PutUint16(data[4:], 1)
	data[6], data[7] = size, size
	binary.LittleEndian.PutUint16(data[10:], 1)
	binary.LittleEndian.PutUint16(data[12:], 32)
	binary.LittleEndian.PutUint32(data[14:], uint32(len(data)-22))
	binary.LittleEndian.PutUint32(data[18:], 22)
	binary.LittleEndian.PutUint32(data[22:], 40)
	binary.LittleEndian.PutUint32(data[26:], size)
	binary.LittleEndian.PutUint32(data[30:], size*2)
	binary.LittleEndian.PutUint16(data[34:], 1)
	binary.LittleEndian.PutUint16(data[36:], 32)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			i := 62 + ((size-1-y)*size+x)*4
			data[i], data[i+1], data[i+2], data[i+3] = 210, 110, 35, 255
			if (x >= 4 && x <= 6 && y >= 3 && y <= 12) || (x >= 6 && x <= 10 && (y == 3 || y == 4 || y == 7 || y == 8)) || (x >= 10 && x <= 11 && y >= 4 && y <= 7) {
				data[i], data[i+1], data[i+2] = 255, 255, 255
			}
		}
	}
	return data
}
