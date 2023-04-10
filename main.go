package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/robfig/cron/v3"
)

const (
	UBUNTU_REMOTE_CONTROL_APPNAME     = `gnome-remote-desktop`
	DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP = `org.gnome.desktop.remote-desktop.rdp`
	DEFAULT_UBUNTU_REMOTE_DESKTOP_VNC = `org.gnome.desktop.remote-desktop.vnc`
	DEFAULT_UBUNTU_DESKTOP_SESSION    = `org.gnome.desktop.session`
	DEFAULT_CRONTAB_INTERVAL          = `@every 1m`
	DEFAULT_USERNAME                  = "soulteary"
	DEFAULT_PASSWORD                  = "soulteary"
	DEFAULT_DAEMON_MODE               = false
)

var (
	URCH_USER   = DEFAULT_USERNAME
	URCH_PASS   = DEFAULT_PASSWORD
	URCH_DAEMON = DEFAULT_DAEMON_MODE
)

// Initialize global variables
func init() {
	fmt.Println(`Remote Control Helper`)

	envUser := strings.TrimSpace(os.Getenv("UBUNTU_REMOTE_USER"))
	if envUser != "" {
		URCH_USER = envUser
		fmt.Println(`set remote username by env:`, URCH_USER)
	}

	envPass := strings.TrimSpace(os.Getenv("UBUNTU_REMOTE_PASS"))
	if envPass != "" {
		URCH_PASS = envPass
		fmt.Println(`set remote password by env:`, URCH_PASS)
	}

	envDaemon := strings.ToLower(strings.TrimSpace(os.Getenv("UBUNTU_DAEMON")))
	if envDaemon == "true" || envDaemon == "on" || envDaemon == "1" {
		URCH_DAEMON = true
	}

	var cliUser string
	var cliPass string
	var cliDaemon string

	flag.StringVar(&cliUser, "user", DEFAULT_USERNAME, "set remote control username")
	flag.StringVar(&cliPass, "pass", DEFAULT_PASSWORD, "set remote control password")
	flag.StringVar(&cliDaemon, "daemon", "", "let app running in daemon mode")
	flag.Parse()

	if cliUser != DEFAULT_USERNAME && cliUser != "" {
		URCH_USER = cliUser
		fmt.Println(`set remote username by cli:`, URCH_USER)
	}

	if cliPass != DEFAULT_PASSWORD && cliPass != "" {
		URCH_PASS = cliPass
		fmt.Println(`set remote password by cli:`, URCH_USER)
	}

	cliDaemon = strings.ToLower(strings.TrimSpace(cliDaemon))
	if cliDaemon == "true" || cliDaemon == "on" || cliDaemon == "1" {
		URCH_DAEMON = true
	}
}

// attempting to apply the correct configuration.
func TryToApplyChange() {
	fmt.Println("check remote control credentials and correct the problem...")
	if !CheckRemoteControlCredentialsIsCorrect(URCH_USER, URCH_PASS) {
		UpdateSettings(URCH_USER, URCH_PASS)
		KillProcessForApplyNewSettings()
	}
	fmt.Println("the configuration has been ensured to be correct.")
}

// create background task
func CreateBackgroundTask() {
	fmt.Println("try to create background task...")
	c := cron.New()
	_, err := c.AddFunc(DEFAULT_CRONTAB_INTERVAL, func() {
		TryToApplyChange()
	})
	if err != nil {
		fmt.Printf("create background task failed: %s\n", err)
		return
	}
	fmt.Println("create background task succeeded.")
	c.Start()
}

func main() {
	// Regardless of whether the program needs to run in the background or not,
	// try to execute system configuration updates first.
	TryToApplyChange()

	// If the program needs to run in the background,
	// then add a background task and keep the program from exiting.
	if URCH_DAEMON {
		go CreateBackgroundTask()
		var chHoldOn = make(chan struct{})
		<-chHoldOn
	}
}
