package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/robfig/cron/v3"
)

const (
	UBUNTU_REMOTE_CONTROL_APPNAME     = `gnome-remote-desktop`
	DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP = `org.gnome.desktop.remote-desktop.rdp`
	DEFAULT_UBUNTU_REMOTE_DESKTOP_VNC = `org.gnome.desktop.remote-desktop.vnc`
	DEFAULT_CRONTAB_INTERVAL          = `@every 1m`
)

var (
	URCH_USER   = `soulteary`
	URCH_PASS   = `soulteary`
	URCH_DAEMON = false
)

func init() {
	fmt.Println(`Remote Control Helper`)

	envUser := strings.TrimSpace(os.Getenv("UBUNTU_REMOTE_USER"))
	if envUser != "" {
		URCH_USER = envUser
		fmt.Println(`set remote username:`, URCH_USER)
	}

	envPass := strings.TrimSpace(os.Getenv("UBUNTU_REMOTE_PASS"))
	if envPass != "" {
		URCH_PASS = envPass
		fmt.Println(`set remote password:`, URCH_PASS)
	}

	envDaemon := strings.ToLower(strings.TrimSpace(os.Getenv("UBUNTU_DAEMON")))
	if envDaemon == "true" || envDaemon == "on" || envDaemon == "1" {
		URCH_DAEMON = true
	}

}

func applyChange() {
	fmt.Println("check remote control credentials and correct the problem...")
	if !CheckRemoteControlCredentialsIsCorrect(URCH_USER, URCH_PASS) {
		UpdateSettings(URCH_USER, URCH_PASS)
		KillProcessForApplyNewSettings()
	}
}

func task() {
	c := cron.New()
	_, err := c.AddFunc(DEFAULT_CRONTAB_INTERVAL, func() {
		applyChange()
	})
	if err != nil {
		fmt.Printf("Create cronjob failed: %s\n", err)
		return
	}
	fmt.Println("Create cronjob succeeded.")
	c.Start()
}

func main() {
	applyChange()
	if URCH_DAEMON {
		go task()
		var chHoldOn = make(chan struct{})
		<-chHoldOn
	}
}
