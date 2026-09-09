package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	"github.com/mxschmitt/playwright-go"
)

// User settings: edit these values, save this file, then rebuild.
var (
	username    = "" // Your RMC ID; the phone topic is paradebot- followed by this value.
	password    = "" // Your RMC password. Keep configured source and executables private.
	startHour   = 5  // Local time, 24-hour clock.
	startMinute = 30
	endHour     = 7
	endMinute   = 0
)

// Internal defaults.
var (
	URL            = "https://services.rmc.ca/apex/f?p=RMCC_CMRC:101&p_lang=fr-ca"
	paradeURL      = "https://services.rmc.ca/php_apps/forms/index.php/en/cadet/update_status"
	loggerFilePath = "logs/parade.log"
	logger         *CustomLogger
	startTime      = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), startHour, startMinute, 0, 0, time.Now().Location())
	endTime        = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), endHour, endMinute, 0, 0, time.Now().Location())
)

type CustomLogger struct {
	consoleLogger *log.Logger
	fileLogger    *log.Logger
	file          *os.File
}

func NewCustomLogger(filePath string) *CustomLogger {
	// Create a custom log format without the default timestamp
	logFlags := 0
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		//Create the logs directory if it doesn't exist
		if _, err := os.Stat("logs"); os.IsNotExist(err) {
			err = os.Mkdir("logs", 0755)
			if err != nil {
				log.Fatal(err)
			}
			//Create the file again
			file, err = os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if err != nil {
		log.Fatal(err)
	}
	return &CustomLogger{
		consoleLogger: log.New(os.Stdout, "", logFlags),
		fileLogger:    log.New(file, "", logFlags),
		file:          file,
	}
}

func (c *CustomLogger) Close() {
	c.file.Close()
}

func (c *CustomLogger) Println(v ...interface{}) {
	message := fmt.Sprintf("[%s] %s", time.Now().Format("2006/01/02 15:04:05"), v[0])
	c.consoleLogger.Println(message)
	c.fileLogger.Println(message)
}

func (c *CustomLogger) Fatal(v ...interface{}) {
	c.Println(fmt.Sprint(v...))
	c.Close()
	os.Exit(1)
}

func (c *CustomLogger) Printf(format string, v ...interface{}) {
	message := fmt.Sprintf("["+format+"]", v...)
	c.consoleLogger.Print(message)
	c.fileLogger.Print(message)
}

func (c *CustomLogger) Print(v ...interface{}) {
	message := fmt.Sprintf("[%s] %s", time.Now().Format("2006/01/02 15:04:05"), v[0])
	c.consoleLogger.Print(message)
	c.fileLogger.Print(message)
}

type Controller struct {
	manual          <-chan chan error
	update          func() error
	lastRun         time.Time
	ranToday        bool
	startTime       time.Time
	endTime         time.Time
	weekEndWarning  bool
	weekendBypass   bool
	ranTodayWarning bool
}

func startProcess() error {
	logger.Println("Starting the parade state update process...")
	// Your existing code...

	err := run()
	if err != nil {
		logger.Println(fmt.Sprintf("Error in parade state update process: %v", err))
		return err
	}

	logger.Println("Parade state update process completed successfully.")
	logger.Println("Waiting for the next parade state update...")
	return nil
}
func printAsciiArt(value string) {
	myFigure := figure.NewColorFigure(value, "standard", "green", true)
	myFigure.Print()
}

func printInstructions(controller Controller) {

	startTimeFormatted := controller.startTime.Format("15:04")
	endTimeFormatted := controller.endTime.Format("15:04")

	fmt.Println()
	color.Green("=============== INSTRUCTIONS =============== ")
	color.Cyan("This program will update your parade state automatically.")
	color.Yellow("It will run every weekday @ a random time between %v and %v.", startTimeFormatted, endTimeFormatted)
	color.Red("If you want to stop the program, press Ctrl+C.")
	color.Green("============================================ ")
	fmt.Println()
}

func main() {
	runOnce := flag.Bool("once", false, "Update parade status immediately, then exit (submits a real update).")
	testNotification := flag.Bool("test-notification", false, "Send a phone notification, then exit without updating parade status.")
	background := flag.Bool("background", false, "Run in the Windows notification area with no terminal (build the executable first).")
	foreground := flag.Bool("foreground", false, "Keep the scheduler in this terminal instead of launching the Windows tray.")
	tray := flag.Bool("tray", false, "Run the Windows tray worker (used by -background).")
	flag.Parse()
	if !*foreground && !*background && !*tray && !*runOnce && !*testNotification {
		*background = defaultBackground()
	}
	// Startup launches may inherit an unwritable directory such as System32.
	if *background || *tray {
		executable, err := os.Executable()
		if err == nil {
			err = os.Chdir(filepath.Dir(executable))
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	logger = NewCustomLogger(loggerFilePath)
	defer logger.Close()
	if *foreground && (*background || *tray || *runOnce || *testNotification) {
		logger.Fatal("Use -foreground separately from the other execution modes.")
	}
	if (*background || *tray) && (*runOnce || *testNotification) || (*background && *tray) {
		logger.Fatal("Use background mode separately from -once and -test-notification.")
	}
	if *background {
		if err := launchBackground(); err != nil {
			logger.Fatal(err)
		}
		return
	}
	release, err := acquireInstance()
	if err != nil {
		logger.Fatal(err)
	}
	defer release()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if *tray {
		if err := runTray(ctx, cancel); err != nil {
			logger.Fatal(err)
		}
		return
	}
	if *runOnce && *testNotification {
		logger.Fatal("Use either -once or -test-notification, not both.")
	}
	if *testNotification {
		if err := testPhoneNotification(); err != nil {
			logger.Fatal(err)
		}
		return
	}

	if err := runBot(ctx, *runOnce, nil); err != nil {
		logger.Fatal(err)
	}
}

func testPhoneNotification() error {
	endpoint, err := notificationURL()
	if err != nil {
		return err
	}
	logger.Printf("Phone notification topic: %s", endpoint)
	if err := sendNtfy("ParadeBot test", "Notifications work. No update sent."); err != nil {
		return err
	}
	logger.Println("Test notification accepted by ntfy. Check the ntfy app on your phone.")
	return nil
}

func runBot(ctx context.Context, runOnce bool, manual <-chan chan error) error {
	if err := validateCredentials(); err != nil {
		return err
	}
	if !runOnce {
		if err := validateTimeWindow(); err != nil {
			return err
		}
	}
	controller := Controller{
		manual:          manual,
		lastRun:         time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()-7, 23, 59, 59, 0, time.Now().Location()),
		startTime:       startTime,
		endTime:         endTime,
		weekEndWarning:  false,
		weekendBypass:   false,
		ranTodayWarning: false,
	}

	printAsciiArt("Parade Bot")
	if !runOnce {
		printInstructions(controller)
	}
	options := &playwright.RunOptions{
		Verbose:  false,
		Browsers: []string{"chromium"},
	}
	setActivity("Checking browser setup")
	err := playwright.Install(options)
	if err != nil {
		logger.Println(err)
		notify("ParadeBot: setup failed", "Browser unavailable. Open the log.")
		return fmt.Errorf("could not install playwright: %w", err)
	}
	if ctx.Err() != nil {
		return nil
	}
	if runOnce {
		return run()
	}
	notify("ParadeBot started", "Weekday updates are on.")
	for ctx.Err() == nil {
		if err := controller.checkTime(ctx); err != nil {
			return err
		}
		if _, err := controller.wait(ctx, 30*time.Second); err != nil {
			return err
		}
	}
	logger.Println("Parade Bot stopped.")
	notify("ParadeBot stopped", "Updates paused until restart.")
	return nil
}

func (controller *Controller) hasRunToday() bool {
	//Check if lasRun is today
	ranToday := controller.lastRun.Year() == time.Now().Year() && controller.lastRun.Month() == time.Now().Month() && controller.lastRun.Day() == time.Now().Day()
	controller.ranToday = ranToday
	return ranToday
}

func (controller *Controller) checkTime(ctx context.Context) error {
	//Check if it is time to run
	startTimeFormatted := controller.startTime.Format("15:04")
	endTimeFormatted := controller.endTime.Format("15:04")

	currentHour, currentMinute, _ := time.Now().Clock()
	startHour, startMinute, _ := controller.startTime.Clock()
	endHour, endMinute, _ := controller.endTime.Clock()

	currentTimeValue := currentHour*60 + currentMinute
	startTimeValue := startHour*60 + startMinute
	endTimeValue := endHour*60 + endMinute

	if isWeekDay(time.Now()) || controller.weekendBypass {
		if currentTimeValue >= startTimeValue && currentTimeValue <= endTimeValue {
			if !controller.hasRunToday() {
				delay := addRandomDelay(controller, time.Now())
				if delay > 0 {
					setActivity("Update scheduled at " + time.Now().Add(time.Duration(delay)*time.Second).Format("Mon 15:04:05"))
					logger.Printf("Random delay before running: %v seconds", delay)
					logger.Println("Going to bed...")
					if manualRan, err := controller.wait(ctx, time.Second*time.Duration(delay)); manualRan || err != nil || ctx.Err() != nil {
						return err
					}
					logger.Println("Waking up...")
				}

				// A wait interrupted by PC sleep may finish on a different day.
				if ctx.Err() != nil || (!controller.weekendBypass && !isWeekDay(time.Now())) {
					return nil
				}
				controller.weekEndWarning = false
				controller.ranTodayWarning = false
				err := controller.performUpdate()
				if err != nil {
					return err
				}
			}
			setActivity("Today's attempt completed; waiting for next weekday")
		} else {
			setActivity("Waiting for weekday " + startTimeFormatted + " - " + endTimeFormatted)
			if !controller.ranTodayWarning {
				logger.Printf("Will only run between %v and %v.", startTimeFormatted, endTimeFormatted)
				controller.ranTodayWarning = true
			}
		}

	} else {
		setActivity("Weekend pause; resumes Monday at " + startTimeFormatted)
		if !controller.weekEndWarning {
			logger.Println("It's the weekend, take a break! I'll automatically run again on Monday.")
			controller.weekEndWarning = true
		}

	}
	return nil
}

func addRandomDelay(controller *Controller, now time.Time) int {
	end := time.Date(now.Year(), now.Month(), now.Day(), controller.endTime.Hour(), controller.endTime.Minute(), 0, 0, now.Location())
	remainingTime := int(end.Sub(now).Seconds())

	if remainingTime <= 0 {
		return 0
	}

	// Générer un délai aléatoire entre maintenant et l'heure de fin
	randomDelay := rand.Intn(int(remainingTime))
	return randomDelay
}

func isWeekDay(now time.Time) bool {
	//Check if it is a weekday
	switch now.Weekday() {
	case time.Saturday:
		return false
	case time.Sunday:
		return false
	default:
		return true
	}
}

func run() error {
	begin := time.Now()
	logger.Println("Mise à jour du Parade State en cours...")
	err := updateParade()
	if err != nil {
		logger.Println("Erreur lors de la mise à jour du Parade State.")
		if errors.Is(err, errSaveUnverified) {
			notify("ParadeBot: not verified", "Check RMC before retrying.")
		} else {
			notify("ParadeBot: update failed", "Check the log and RMC.")
		}
		return err
	}
	logger.Println("Parade state save verified on the reloaded RMC page.")
	color.Green("Parade state saved and verified in %.3f seconds\n", time.Since(begin).Seconds())

	notify("ParadeBot: saved", "Saved and verified on RMC.")
	return nil
}

func validateCredentials() error {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" || username == "YOUR_RMC_ID" || password == "YOUR_RMC_PASSWORD" {
		return fmt.Errorf("setup required: enter your own username and password in main.go, then rebuild; see README.md")
	}
	return nil
}

func validateTimeWindow() error {
	if startHour < 0 || startHour > 23 || endHour < 0 || endHour > 23 || startMinute < 0 || startMinute > 59 || endMinute < 0 || endMinute > 59 || startHour*60+startMinute >= endHour*60+endMinute {
		return fmt.Errorf("invalid time window: use hours 0-23 and minutes 0-59, with the start before the end on the same day; edit main.go and rebuild")
	}
	return nil
}

func updateParade() error {
	//Create browser instance and update the parade state
	pw, err := playwright.Run()
	if err != nil {
		logger.Println(err)
		return err
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	//browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
	//	Headless: playwright.Bool(false), // Set Headless to false
	//	Timeout:  playwright.Float(10 * 1000),
	//	SlowMo: playwright.Float(50),
	//})
	if err != nil {
		logger.Println(err)
		return err
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		logger.Println(err)
		return err
	}
	defer page.Close()

	pagePointer := &page

	err = login(pagePointer)
	if err != nil {
		logger.Println("Error logging in:")
		logger.Println(err)
		return err
	}
	logger.Println("Connexion réussie !")

	err = goToParadePage(pagePointer)
	if err != nil {
		fmt.Println("Error updating parade state:")
		logger.Println(err)
		return err
	}
	return nil
}

func login(pagePointer *playwright.Page) error {
	var (
		usernameSelector = "#P101_USERNAME"
		passwordSelector = "#P101_PASSWORD"
	)

	page := *pagePointer
	logger.Print("Connexion en cours... ")
	if _, err := page.Goto(URL); err != nil {
		logger.Println(err)
		return err
	}

	usernameField := page.Locator(usernameSelector)
	passwordField := page.Locator(passwordSelector)

	if err := usernameField.Fill(username); err != nil {
		logger.Println(err)
		return err
	}
	if err := passwordField.Fill(password); err != nil {
		logger.Println(err)
		return err

	}
	//Press enter to submit the form
	if err := passwordField.Press("Enter"); err != nil {
		logger.Println(err)
		return err
	}
	return nil
}

func goToParadePage(pagePointer *playwright.Page) error {
	page := *pagePointer
	//Add a 2 second delay to make sure the page is loaded
	time.Sleep(2 * time.Second)

	return saveAndVerifyParade(page, paradeURL)
}

func notify(title, message string) {
	if _, err := notificationURL(); err != nil {
		return // The blank public template has no notification recipient.
	}
	if err := sendNtfy(title, message); err != nil {
		logger.Printf("Could not send phone notification: %v", err)
	}
}

func sendNtfy(title, message string) error {
	endpoint, err := notificationURL()
	if err != nil {
		return err
	}
	return sendNtfyTo(endpoint, title, message)
}

func notificationURL() (string, error) {
	if strings.TrimSpace(username) == "" || username == "YOUR_RMC_ID" {
		return "", fmt.Errorf("phone notifications need your username: set username in main.go and rebuild; your topic will be paradebot- followed by your username")
	}
	return "https://ntfy.sh/" + url.PathEscape("paradebot-"+username), nil
}

func sendNtfyTo(notificationEndpoint, title, message string) error {
	endpoint, err := url.Parse(notificationEndpoint)
	if err != nil || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.Host == "" || strings.Trim(endpoint.Path, "/") == "" {
		return fmt.Errorf("invalid notification URL: use a full HTTP(S) topic URL")
	}
	query := endpoint.Query()
	query.Set("title", title)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodPost, endpoint.String(), strings.NewReader(message))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification service returned HTTP %d", resp.StatusCode)
	}
	_, err = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return err
}
