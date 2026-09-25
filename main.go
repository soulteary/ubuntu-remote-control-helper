package main

import (
	"flag"
	"fmt"
	"log"
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

type Config struct {
	User   string
	Pass   string
	Daemon bool
}

func isTruthy(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "on" || s == "1"
}

// Parse the configuration, the priority is: cli > env > default.
func ParseConfig(args []string, getenv func(string) string) (Config, error) {
	config := Config{User: DEFAULT_USERNAME, Pass: DEFAULT_PASSWORD, Daemon: DEFAULT_DAEMON_MODE}

	if envUser := strings.TrimSpace(getenv("UBUNTU_REMOTE_USER")); envUser != "" {
		config.User = envUser
		fmt.Println(`set remote username by env:`, config.User)
	}
	if envPass := strings.TrimSpace(getenv("UBUNTU_REMOTE_PASS")); envPass != "" {
		config.Pass = envPass
		fmt.Println(`set remote password by env`)
	}
	if isTruthy(getenv("UBUNTU_DAEMON")) {
		config.Daemon = true
	}

	fs := flag.NewFlagSet("urch", flag.ContinueOnError)
	cliUser := fs.String("user", "", "set remote control username (env: UBUNTU_REMOTE_USER)")
	cliPass := fs.String("pass", "", "set remote control password (env: UBUNTU_REMOTE_PASS, preferred: command line arguments are visible to other users)")
	cliDaemon := fs.String("daemon", "", "let app running in daemon mode (env: UBUNTU_DAEMON)")
	if err := fs.Parse(args); err != nil {
		return config, err
	}

	// only flags which were explicitly passed override env and defaults
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "user":
			if v := strings.TrimSpace(*cliUser); v != "" {
				config.User = v
				fmt.Println(`set remote username by cli:`, config.User)
			}
		case "pass":
			if v := strings.TrimSpace(*cliPass); v != "" {
				config.Pass = v
				fmt.Println(`set remote password by cli`)
			}
		case "daemon":
			config.Daemon = isTruthy(*cliDaemon)
		}
	})
	return config, nil
}

// attempting to apply the correct configuration.
func TryToApplyChange(config Config) error {
	EnsureSessionBusEnv()

	fmt.Println("check remote control credentials and correct the problem...")
	ok, err := CheckRemoteControlCredentialsIsCorrect(config.User, config.Pass)
	if err != nil {
		return err
	}
	if !ok {
		if err := UpdateSettings(config.User, config.Pass); err != nil {
			return err
		}
		KillProcessForApplyNewSettings()
	}
	fmt.Println("the configuration has been ensured to be correct.")
	return nil
}

// create background task
func CreateBackgroundTask(config Config) error {
	fmt.Println("try to create background task...")
	c := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DiscardLogger)))
	_, err := c.AddFunc(DEFAULT_CRONTAB_INTERVAL, func() {
		if err := TryToApplyChange(config); err != nil {
			log.Println("apply change failed:", err)
		}
	})
	if err != nil {
		return fmt.Errorf("create background task failed: %w", err)
	}
	fmt.Println("create background task succeeded.")
	c.Start()
	return nil
}

func main() {
	fmt.Println(`Remote Control Helper`)

	config, err := ParseConfig(os.Args[1:], os.Getenv)
	if err != nil {
		os.Exit(2)
	}

	// gsettings and the keyring are per user, running as root (e.g. with sudo)
	// would modify root's settings instead of the desktop user's.
	if os.Geteuid() == 0 {
		log.Fatal("urch must be run as the desktop user who shares the screen, not as root (do not use sudo).")
	}

	if config.User == DEFAULT_USERNAME && config.Pass == DEFAULT_PASSWORD {
		fmt.Println("WARNING: using the default username and password, please set your own with UBUNTU_REMOTE_USER / UBUNTU_REMOTE_PASS.")
	}

	// Regardless of whether the program needs to run in the background or not,
	// try to execute system configuration updates first.
	if err := TryToApplyChange(config); err != nil {
		if !config.Daemon {
			log.Fatal(err)
		}
		// in daemon mode, keep running and retry later, e.g. the user is not logged in yet
		log.Println("apply change failed:", err)
	}

	// If the program needs to run in the background,
	// then add a background task and keep the program from exiting.
	if config.Daemon {
		if err := CreateBackgroundTask(config); err != nil {
			log.Fatal(err)
		}
		select {}
	}
}
