package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	UBUNTU_SETTING_KEY_RDP_ENABLE     = `enable`
	UBUNTU_SETTING_KEY_RDP_SHARE_MODE = `screen-share-mode`
	UBUNTU_SETTING_KEY_RDP_VIEW_ONLY  = `view-only`
	UBUNTU_SETTING_KEY_VNC_ENABLE     = `enable`
	UBUNTU_SETTING_KEY_IDLE_DELAY     = `idle-delay`

	SECRET_TOOL_LABEL  = `GNOME Remote Desktop RDP credentials`
	SECRET_TOOL_SCHEMA = `org.gnome.RemoteDesktop.RdpCredentials`

	// If the keyring is locked, secret-tool waits for the user to unlock it,
	// so every command needs a timeout to keep the program from hanging.
	DEFAULT_COMMAND_TIMEOUT = 30 * time.Second
)

// A gnome setting that should be applied.
// `Value` is the GVariant text passed to `gsettings set`,
// `Expected` is what `gsettings get` prints once the value is applied.
type GnomeSetting struct {
	Schema   string
	Key      string
	Value    string
	Expected string
}

var REMOTE_CONTROL_SETTINGS = []GnomeSetting{
	// `gsettings get` prints uint32 values with their type annotation
	{DEFAULT_UBUNTU_DESKTOP_SESSION, UBUNTU_SETTING_KEY_IDLE_DELAY, `uint32 0`, `uint32 0`},
	{DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, UBUNTU_SETTING_KEY_RDP_ENABLE, `true`, `true`},
	{DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, UBUNTU_SETTING_KEY_RDP_SHARE_MODE, `'mirror-primary'`, `'mirror-primary'`},
	{DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP, UBUNTU_SETTING_KEY_RDP_VIEW_ONLY, `false`, `false`},
	{DEFAULT_UBUNTU_REMOTE_DESKTOP_VNC, UBUNTU_SETTING_KEY_VNC_ENABLE, `false`, `false`},
}

// Update the remote control related configuration in Ubuntu.
func UpdateSettings(username string, password string) error {
	for _, setting := range REMOTE_CONTROL_SETTINGS {
		if err := UpdateGnomeSettings(setting); err != nil {
			return fmt.Errorf("update gnome settings %s:`%s` failed: %w", setting.Schema, setting.Key, err)
		}
	}

	ok, err := UpdateRemoteControlCredentials(username, password, false)
	if err != nil {
		return fmt.Errorf("update remote control credentials failed: %w", err)
	}
	if !ok {
		return errors.New("update remote control credentials failed: the stored credentials do not match")
	}
	return nil
}

// Execute a command without shell, optionally feeding stdin, and obtain the normal and error log output contents.
func ExecuteCommand(stdin string, name string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DEFAULT_COMMAND_TIMEOUT)
	defer cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- the command name is always a constant
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("`%s` timed out after %s, is the keyring locked or the session D-Bus unavailable?", name, DEFAULT_COMMAND_TIMEOUT)
	}
	return stdout.String(), stderr.String(), err
}

// Update the settings in Gnome and check if the changes are actually applied.
func UpdateGnomeSettings(setting GnomeSetting) error {
	_, stderr, err := ExecuteCommand("", "gsettings", "set", setting.Schema, setting.Key, setting.Value)
	if err != nil {
		return fmt.Errorf("gsettings set: %w %s", err, strings.TrimSpace(stderr))
	}

	stdout, stderr, err := ExecuteCommand("", "gsettings", "get", setting.Schema, setting.Key)
	if err != nil {
		return fmt.Errorf("gsettings get: %w %s", err, strings.TrimSpace(stderr))
	}

	if actual := strings.TrimSpace(stdout); actual != setting.Expected {
		return fmt.Errorf("value is `%s` after update, expected `%s`", actual, setting.Expected)
	}
	return nil
}

// Quote a string as a GVariant text format string literal.
func QuoteGVariantString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return `'` + s + `'`
}

// Build the secret which gnome-remote-desktop reads from the keyring.
func BuildRemoteControlCredentials(username string, password string) string {
	return fmt.Sprintf(`{'username': <%s>, 'password': <%s>}`, QuoteGVariantString(username), QuoteGVariantString(password))
}

// Check if the account and password settings are correct for remote control.
func CheckRemoteControlCredentialsIsCorrect(inputUser string, inputPass string) (bool, error) {
	return UpdateRemoteControlCredentials(inputUser, inputPass, true)
}

// Update the account and password for remote control.
// When `dryrun` is true, only compare the stored credentials with the expected ones.
func UpdateRemoteControlCredentials(inputUser string, inputPass string, dryrun bool) (bool, error) {
	username := strings.TrimSpace(inputUser)
	password := strings.TrimSpace(inputPass)
	if username == "" || password == "" {
		return false, errors.New("username and password must not be empty")
	}
	credentials := BuildRemoteControlCredentials(username, password)

	if !dryrun {
		// secret-tool reads the secret from stdin, keep it out of the command line
		_, stderr, err := ExecuteCommand(credentials, "secret-tool", "store", "-l", SECRET_TOOL_LABEL, "xdg:schema", SECRET_TOOL_SCHEMA)
		if err != nil {
			return false, fmt.Errorf("secret-tool store: %w %s", err, strings.TrimSpace(stderr))
		}
	}

	stdout, stderr, err := ExecuteCommand("", "secret-tool", "lookup", "xdg:schema", SECRET_TOOL_SCHEMA)
	if err != nil {
		// secret-tool exits with 1 and prints nothing when no credentials are stored yet
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && strings.TrimSpace(stdout) == "" && strings.TrimSpace(stderr) == "" {
			return false, nil
		}
		return false, fmt.Errorf("secret-tool lookup: %w %s", err, strings.TrimSpace(stderr))
	}

	return strings.TrimSpace(stdout) == credentials, nil
}

// gsettings and secret-tool need the D-Bus session bus of the logged-in desktop user.
// When the program is started outside the desktop session (supervisor, cron, ssh),
// point it at the user's session bus instead of letting dbus-launch spawn a new one.
func EnsureSessionBusEnv() {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		return
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	bus := runtimeDir + "/bus"
	if _, err := os.Stat(bus); err != nil {
		return
	}
	_ = os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	_ = os.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+bus)
	fmt.Println("use session bus:", bus)
}
