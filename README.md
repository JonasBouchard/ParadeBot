# ParadeBot

Updates your RMC parade status on weekdays and sends the result to your phone. Follow these steps on **Windows**. For macOS/Linux, see [other operating systems](docs/DETAILS.md#builds-and-other-operating-systems).

## 1. Download the project

On this GitHub page, select **Code → Download ZIP**. Right-click the downloaded ZIP, choose **Extract All**, and open the extracted folder containing `main.go` and `go.mod`.

## 2. Install Go

Download and run the Windows installer from [go.dev/dl](https://go.dev/dl/). Use Go 1.22 or newer.

After installation, open a **new PowerShell window** and check:

```powershell
go version
```

You should see a Go version. If the command is not recognized, follow the [Go installation guide](https://go.dev/doc/install) before continuing.

## 3. Enter your settings

Open `main.go` in a text editor. At the top, replace the values **inside the quotes** with your own credentials:

```go
username    = "YOUR_RMC_ID"
password    = "YOUR_RMC_PASSWORD"
startHour   = 5
startMinute = 30
endHour     = 7
endMinute   = 0
```

The default window is **05:30–07:00**, Monday through Friday, in your PC's local time. Change the hour/minute values if needed; the start must be before the end on the same day. **Save the file.**

## 4. Build the program

In PowerShell, open the extracted project folder. Replace the example path below with your actual folder:

```powershell
cd "C:\path\to\ParadeBot"
go build -o ParadeBot.exe .
```

Wait for the command to finish. A successful build creates `ParadeBot.exe` in that folder. **After any settings change, save and build again.** Stop a running bot before rebuilding.

## 5. Enable notifications on your phone

1. Install **ntfy** from [Google Play](https://play.google.com/store/apps/details?id=io.heckel.ntfy) or the [App Store](https://apps.apple.com/us/app/ntfy/id1625396347).
2. Open ntfy and allow it to send notifications.
3. Add a topic subscription with these values:

   | Field | Enter |
   | --- | --- |
   | Server | `https://ntfy.sh` |
   | Topic | `paradebot-` followed by the exact username you entered in step 3 |

   For example, username `ID_EXAMPLE` gives topic **`paradebot-ID_EXAMPLE`**. Match capitalization and enter only the topic name in the Topic field.
4. Save the subscription. No paid plan or ntfy account is required for this setup.

The program calculates the topic automatically; there is no notification URL to edit. [ntfy's phone guide](https://docs.ntfy.sh/subscribe/phone/) has additional help.

## 6. Start the bot and test notifications

1. Double-click `ParadeBot.exe`.
2. Find the blue **P** beside the Windows clock. Check the **^** hidden-icons menu if needed. The launcher closes; the tray app keeps running.
3. Right-click the **P** and select **Test notification**.
4. Confirm that **ParadeBot test** appears on your phone. This test does not submit a parade update.

If it does not arrive, check the topic spelling and phone notification permissions. Use **Open log** in the tray menu for errors. The first normal launch may take longer while it downloads browser components.

## 7. Optional: start automatically after a restart

Right-click the **P** and check **Start with Windows**. It is off by default. Once enabled, the bot starts when you sign in after restarting your PC. Uncheck it to disable startup.

## Everyday use

- **Keep the PC on, awake, and online.** The bot cannot work normally during sleep or wake the computer.
- **Run once now (real update)** submits immediately, including weekends. Success is reported only after verifying the saved result on a reloaded RMC page.
- **Stop and exit** stops the bot. Only one copy can run at a time.
- If a save is **not verified**, check RMC before retrying. Restarting loses today's history and can allow another submission.

Keep configured source and executables private: they contain your credentials. Public ntfy topics are guessable from your username. Never publish personal settings or old Git history containing them.

[Command-line use, other operating systems, and troubleshooting](docs/DETAILS.md)
