# Additional details

[Back to the quick start](../README.md)

## Schedule and manual updates

- Automatic mode checks approximately every 30 seconds, Monday through Friday, using the PC's local clock. It chooses a random time within the remaining window each day.
- The comparisons use hours and minutes: the entire end minute is eligible. An update can finish after the window ends.
- Starting after the window waits for the next eligible day; missed days are not queued. Holidays and leave receive no special handling.
- The tray's **Run once now** replaces a pending random wait and records today's attempt. Another deliberate click can submit again. The action is disabled during setup, active updates, shutdown, or after an error stops the scheduler.
- Command-line `-once` bypasses the schedule and exits afterward. It can submit on weekends too.
- Daily history exists only in memory and records attempts before success is known. Restarting loses that history. An error stops scheduling; there is no automatic retry. The tray remains available to show the error and open logs.
- After sleep during a random wait, the bot rechecks the weekday but does not recheck the original date or time window. Pending work may run late on a weekday. Clock/time-zone changes can also affect waits.

## Power, stopping, and startup

Turning off the display or locking the session is fine if the PC remains awake and online. Sleep and hibernation suspend normal operation. Closing a laptop lid often puts it to sleep; check its power settings. The bot has no wake timer or sleep prevention.

Closing the launcher or terminal does not stop a Windows tray instance. Closing a foreground terminal normally stops its process. Shutdown, restart, power loss, and sign-out stop the bot.

**Stop and exit** or Ctrl+C interrupts idle/random waits. An active browser installation or update finishes first. Phone requests can add up to ten seconds. If you must end a stuck process in Task Manager, check RMC before retrying a possibly completed submission.

The Windows **Start with Windows** checkbox saves the executable's path with `-background` for your account. It is off until you opt in. It runs at **sign-in**, not at the login screen; Windows may delay it. It neither wakes the computer nor retries missed updates. See [Windows startup behavior](https://learn.microsoft.com/en-us/windows/win32/setupapi/run-and-runonce-registry-keys).

Uncheck the option to disable startup. **Stop and exit** leaves that choice unchanged. Before moving or deleting the executable, disable startup; enable it again from the new location if needed. Rebuilding at the same path preserves the choice. Remove any older Startup-folder shortcuts separately. If Windows Settings or Task Manager disabled the entry, re-enable it there too—the tray reflects the saved entry.

## Notifications and status

Your topic is always `paradebot-` plus the configured username, including its capitalization, on `https://ntfy.sh`. Changing the username changes the topic after rebuilding; follow the new topic on your phone. No paid subscription or ntfy account is needed for public topics. These topics are guessable and public; see [ntfy topic guidance](https://docs.ntfy.sh/publish/).

The bot sends startup, orderly-stop, verified-save, unverified-save, update-failure, and browser-setup-failure alerts. One-time mode sends only its result alert. Blank credentials stop setup; a blank username has no notification destination.

The tray's **Test notification** does not open RMC or affect today's attempt. Only one notification test runs at a time. The menu shows sending, sent, or failed, with errors in the log. “Sent” confirms acceptance by ntfy, not display on the phone. Check app permissions, topic spelling, network access, and Do Not Disturb if it does not appear.

Requests use HTTPS verification and a ten-second timeout, without retries. Notification failures do not repeat a parade submission. Power loss, crashes, and network failures may prevent alerts entirely.

**Save verified** means RMC returned a successful save response with a new `last_updated` value, and a fresh page reload showed the same submitted status/comment and that acknowledged timestamp. The bot requests cache revalidation and checks the saved page rather than relying on a button click or the page's immediate visual change.

Missing/invalid acknowledgements, unchanged timestamps, reload failures, expired sessions, or mismatched data report **Save not verified**. The save might still have reached RMC, so check the website before retrying. The bot makes only one save request and stops scheduling on an unverified result. Two saves within the same timestamp second may be reported as unverified because a new save cannot be distinguished. A waiting tray status means the scheduler is alive, not that the next login will succeed.

## Builds and other operating systems

Windows, macOS, and Linux builds are supported. The browser also requires a compatible OS; check [Playwright's system requirements](https://playwright.dev/docs/intro#system-requirements). Windows is runtime-tested here. macOS Intel/Apple Silicon and Linux x86-64/ARM64 have been cross-compiled, but their browser flows have not been runtime-tested here. Android and iOS receive notifications; they do not run the bot.

Browser components download automatically on the first normal or one-time run, once credentials are configured. Internet access and writable disk space are required. No separate Chrome or Node.js installation is needed. Linux additionally requires system browser libraries.

### macOS

Install Go, configure `main.go`, then run from the project folder:

```sh
go build -o ParadeBot .
./ParadeBot -test-notification
./ParadeBot
```

### Linux

Use a supported Ubuntu/Debian release. Install Go, configure `main.go`, and run:

```sh
go run github.com/mxschmitt/playwright-go/cmd/playwright install --with-deps chromium
go build -o ParadeBot .
./ParadeBot -test-notification
./ParadeBot
```

Installing system dependencies may need administrator privileges. Run the bot itself as your normal user. See [Playwright's dependency instructions](https://playwright.dev/docs/browsers#install-system-dependencies).

### Terminal and background use

On every supported OS, `go run .` runs the source in the terminal. Add `-once` or `-test-notification` after the `.` for those actions. Keep the terminal open and stop with Ctrl+C. `go run . -background` is unsupported because its executable is temporary.

Built Windows executables open the tray by default; `-foreground` keeps them in the terminal. The tray and startup checkbox are Windows-only. macOS/Linux executables remain in the terminal; for a simple detached process:

```sh
mkdir -p logs
nohup ./ParadeBot > logs/background.log 2>&1 &
echo $!  # Save this process ID
```

Check `ps -p PROCESS_ID`, stop with `kill -TERM PROCESS_ID`, and read logs with `tail -n 30 logs/parade.log`. This does not configure startup after a reboot or prevent sleep. Use `launchd`/`systemd` separately if needed.

Single-instance protection covers all modes, folders, and login sessions on Windows. On macOS/Linux it covers one OS user/cache location. Separate PCs or WSL environments still run independently. An OS lock is released on exit or crash; there is no stale PID file to delete. Stop older builds without this lock before launching a newer one.

## Troubleshooting and maintenance

| Problem | What to check |
| --- | --- |
| Setup required | Fill `username` and `password` in `main.go`, save, and rebuild. Example placeholders are rejected. |
| Wrong topic or old settings | Save and rebuild the same source folder used for the executable. Restart the old process. |
| Invalid time window | Hours must be 0–23 and minutes 0–59. Start must precede end on the same day. |
| Already running | Use the existing tray actions or stop that copy before launching another command. |
| Build cannot overwrite executable | Stop the existing bot before rebuilding. |
| `go` is not recognized | Install Go and open a new terminal so PATH is refreshed. |
| Driver download 404 | Rebuild using this repository's pinned Playwright dependency; an older executable may reference retired downloads. |
| Removing unused browser | Playwright is cleaning outdated cached browser versions. |
| Login or update fails | Open the log, check internet/RMC availability and credentials, and verify the website before retrying. |

Windows tray logs live beside the executable in `logs/parade.log` and `logs/background.log`. Foreground logs use the current folder. Logs append without rotation, may include the topic URL, and do not store daily scheduling history. Some logging helpers omit extra arguments. Keep logs private and manage their size yourself.

To publish a template, keep credentials blank and exclude generated executables/logs. Ignoring files does not remove old commits. If a repository once contained personal settings, publish a clean source export as a new repository without the old `.git` folder. Keep your configured copy separate; change any credentials or topics already exposed publicly.

After editing code or settings, rebuild and restart. Run `go test ./...` and `go vet ./...` when changing behavior. Tests use simulated updates/local HTTP servers and do not submit to RMC. Browser verification tests need the Playwright driver and Chromium installed; they skip if these are unavailable. Install them with `go run github.com/mxschmitt/playwright-go/cmd/playwright install chromium` (plus Linux system dependencies if needed).
