package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

const (
	UBUNTU_SETTING_KEY_RDP_ENABLE     = `enable`
	UBUNTU_SETTING_KEY_RDP_SHARE_MODE = `screen-share-mode`
	UBUNTU_SETTING_KEY_RDP_VIEW_ONLY  = `view-only`
	UBUNTU_SETTING_KEY_VNC_ENABLE     = `enable`
)

// Update the remote control related configuration in Ubuntu, and exit the program if any configuration update fails.
func UpdateSettings(username string, password string) {
	if !UpdateGnomeSettings(DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, UBUNTU_SETTING_KEY_RDP_ENABLE, `true`) {
		log.Fatalf("Update gnome settings %s:`%s` failed.", DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, UBUNTU_SETTING_KEY_RDP_ENABLE)
	}

	if !UpdateGnomeSettings(DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, UBUNTU_SETTING_KEY_RDP_SHARE_MODE, `mirror-primary`) {
		log.Fatalf("Update gnome settings %s:`%s` failed.", DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, UBUNTU_SETTING_KEY_RDP_SHARE_MODE)
	}

	if !UpdateGnomeSettings(DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, UBUNTU_SETTING_KEY_RDP_VIEW_ONLY, `false`) {
		log.Fatalf("Update gnome settings %s:`%s` failed.", DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, UBUNTU_SETTING_KEY_RDP_VIEW_ONLY)
	}

	if !UpdateGnomeSettings(DEFAULT_UBUNTU_REMOTE_DESKTOP_VNC, UBUNTU_SETTING_KEY_VNC_ENABLE, `false`) {
		log.Fatalf("Update gnome settings %s:`%s` failed.", DEFAULT_UBUNTU_REMOTE_DESKTOP_VNC, UBUNTU_SETTING_KEY_VNC_ENABLE)
	}

	if !UpdateRemoteControlCredentials(username, password, false) {
		log.Fatalf("Update remote control credentials, user: %s, pass: %s", username, password)
	}
}

// Execute system commands using Bash and obtain the normal and error log output contents.
func ExecuteShellCommand(command string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Update the settings in Gnome and check if the changes are actually applied.
func UpdateGnomeSettings(appName string, key string, value string) bool {
	setValue := ""
	if strings.EqualFold(strings.ToLower(value), "true") || strings.EqualFold(strings.ToLower(value), "false") {
		setValue = strings.ToLower(value)
	} else {
		setValue = fmt.Sprintf(`'%s'`, value)
	}

	const command = "gsettings"
	const actionSet = "set"
	const actionGet = "get"

	cmdLineSet := fmt.Sprintf("%s %s '%s' '%s' %s", command, actionSet, appName, key, setValue)
	_, stderr, err := ExecuteShellCommand(cmdLineSet)
	if err != nil {
		log.Fatal("Execute command failed: ", cmdLineSet, stderr)
	}

	cmdLineGet := fmt.Sprintf("%s %s '%s' '%s'", command, actionGet, appName, key)
	stdout, stderr, err := ExecuteShellCommand(cmdLineGet)
	if err != nil {
		log.Fatal("Execute command failed: ", cmdLineGet, stderr)
	}

	return strings.TrimSpace(stdout) == setValue
}

// Check if the account and password settings are correct for remote control.
func CheckRemoteControlCredentialsIsCorrect(inputUser string, inputPass string) bool {
	return UpdateRemoteControlCredentials(inputUser, inputPass, true)
}

// Update the account and password for remote control.
func UpdateRemoteControlCredentials(inputUser string, inputPass string, dryrun bool) bool {
	username := strings.TrimSpace(inputUser)
	password := strings.TrimSpace(inputPass)
	if username == "" || password == "" {
		return false
	}
	prompt := fmt.Sprintf(`{'username': <'%s'>, 'password': <'%s'>}`, username, password)

	if !dryrun {
		cmdLineSet := fmt.Sprintf(`printf "%s" | secret-tool store -l 'GNOME Remote Desktop RDP credentials' xdg:schema org.gnome.RemoteDesktop.RdpCredentials`, prompt)
		_, stderr, err := ExecuteShellCommand(cmdLineSet)
		if err != nil {
			log.Fatal("Execute command failed: ", cmdLineSet, stderr)
		}
	}

	cmdLineGet := `secret-tool lookup xdg:schema org.gnome.RemoteDesktop.RdpCredentials`
	stdout, stderr, err := ExecuteShellCommand(cmdLineGet)
	if err != nil {
		log.Fatal("Execute command failed: ", cmdLineGet, stderr)
	}

	return strings.TrimSpace(stdout) == prompt
}
