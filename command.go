package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func UpdateSettings(username string, password string) {
	if !UpdateGnomeSettings(DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, `enable`, `true`) {
		log.Fatal("Update gnome settings `enable` failed.")
	}

	if !UpdateGnomeSettings(DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, `screen-share-mode`, `mirror-primary`) {
		log.Fatal("Update gnome settings `screen-share-mode` failed.")
	}

	if !UpdateGnomeSettings(DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, `view-only`, `false`) {
		log.Fatal("Update gnome settings `view-only` failed.")
	}

	if !UpdateGnomeSettings(DEFAULT_UBUNTU_REMOTE_DESKTOP_VNC, `enable`, `false`) {
		log.Fatalf("Update gnome settings %s `enable` failed.", DEFAULT_UBUNTU_REMOTE_DESKTOP_VNC)
	}

	if !UpdateRemoteControlCredentials(username, password, false) {
		log.Fatalf("Update remote control credentials, user: %s, pass: %s", username, password)
	}
}

func ExecuteShellCommand(command string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

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

func CheckRemoteControlCredentialsIsCorrect(inputUser string, inputPass string) bool {
	return UpdateRemoteControlCredentials(inputUser, inputPass, true)
}

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
