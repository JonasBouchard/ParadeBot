package main

import (
	"errors"
	"sync"
	"time"
)

var botStatus = struct {
	sync.RWMutex
	activity    string
	result      string
	manualReady bool
}{activity: "Starting", result: "No attempt in this session"}

func setActivity(activity string) {
	botStatus.Lock()
	defer botStatus.Unlock()
	botStatus.activity = activity
}

func setManualReady(ready bool) {
	botStatus.Lock()
	defer botStatus.Unlock()
	botStatus.manualReady = ready
}

func canRunManually() bool {
	botStatus.RLock()
	defer botStatus.RUnlock()
	return botStatus.manualReady
}

func recordResult(err error) {
	result := "Save verified"
	if err != nil {
		result = "Failed; see log"
		if errors.Is(err, errSaveUnverified) {
			result = "Save not verified; check RMC"
		}
	}
	botStatus.Lock()
	defer botStatus.Unlock()
	botStatus.result = result + " at " + time.Now().Format("Jan 02 15:04:05")
}

func currentStatus() (string, string) {
	botStatus.RLock()
	defer botStatus.RUnlock()
	return botStatus.activity, botStatus.result
}
