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

	// the collection secret-tool stores the credentials in, usually the login keyring
	SECRET_SERVICE_NAME               = `org.freedesktop.secrets`
	SECRET_SERVICE_DEFAULT_COLLECTION = `/org/freedesktop/secrets/aliases/default`

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

// Make sure all remote control settings and the credentials have the expected values,
// only the incorrect ones are updated.
// Returns whether gnome-remote-desktop needs to be restarted to apply the changes.
func EnsureRemoteControlConfig(username string, password string) (bool, error) {
	var incorrect []GnomeSetting
	for _, setting := range REMOTE_CONTROL_SETTINGS {
		ok, err := CheckGnomeSetting(setting)
		if err != nil {
			return false, fmt.Errorf("check gnome settings %s:`%s` failed: %w", setting.Schema, setting.Key, err)
		}
		if !ok {
			incorrect = append(incorrect, setting)
		}
	}

	credentialsOK, err := CheckRemoteControlCredentialsIsCorrect(username, password)
	if err != nil {
		return false, err
	}

	restart := false
	for _, setting := range incorrect {
		fmt.Printf("gnome settings %s:`%s` is not `%s`, correct it.\n", setting.Schema, setting.Key, setting.Expected)
		if err := UpdateGnomeSettings(setting); err != nil {
			return false, fmt.Errorf("update gnome settings %s:`%s` failed: %w", setting.Schema, setting.Key, err)
		}
		// the session settings (idle-delay) are not used by gnome-remote-desktop,
		// avoid interrupting the remote sessions for them
		if setting.Schema != DEFAULT_UBUNTU_DESKTOP_SESSION {
			restart = true
		}
	}

	if !credentialsOK {
		fmt.Println("remote control credentials are not correct, update them.")
		ok, err := UpdateRemoteControlCredentials(username, password, false)
		if err != nil {
			return false, fmt.Errorf("update remote control credentials failed: %w", err)
		}
		if !ok {
			return false, errors.New("update remote control credentials failed: the stored credentials do not match")
		}
		restart = true
	}
	return restart, nil
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

// The function used to execute commands, replaced in tests.
var runCommand = ExecuteCommand

// Read a setting in Gnome.
func GetGnomeSetting(setting GnomeSetting) (string, error) {
	stdout, stderr, err := runCommand("", "gsettings", "get", setting.Schema, setting.Key)
	if err != nil {
		return "", fmt.Errorf("gsettings get: %w %s", err, strings.TrimSpace(stderr))
	}
	return strings.TrimSpace(stdout), nil
}

// Check if a setting in Gnome already has the expected value.
func CheckGnomeSetting(setting GnomeSetting) (bool, error) {
	actual, err := GetGnomeSetting(setting)
	if err != nil {
		return false, err
	}
	return actual == setting.Expected, nil
}

// Update the settings in Gnome and check if the changes are actually applied.
func UpdateGnomeSettings(setting GnomeSetting) error {
	_, stderr, err := runCommand("", "gsettings", "set", setting.Schema, setting.Key, setting.Value)
	if err != nil {
		return fmt.Errorf("gsettings set: %w %s", err, strings.TrimSpace(stderr))
	}

	actual, err := GetGnomeSetting(setting)
	if err != nil {
		return err
	}
	if actual != setting.Expected {
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

var ErrKeyringLocked = errors.New("the login keyring is locked, gnome-remote-desktop can not read the credentials either. " +
	"Log in with your password once to unlock it, or, with automatic login, set an empty password for the login keyring, " +
	"see https://github.com/soulteary/ubuntu-remote-control-helper#keyring-and-automatic-login")

// Parse the output of `gdbus call` for a boolean property, e.g. `(<true>,)`.
func ParseGdbusBoolean(output string) (value bool, ok bool) {
	switch strings.TrimSpace(output) {
	case "(<true>,)":
		return true, true
	case "(<false>,)":
		return false, true
	}
	return false, false
}

// Check if the default keyring, which stores the credentials, is locked.
// Reading a secret from a locked keyring makes secret-tool show an unlock prompt on the screen
// and wait for it, while reading the `Locked` property does not prompt.
// Returns false when the state can not be determined, e.g. gdbus is missing or no default keyring exists yet,
// so that secret-tool still gets the chance to create or unlock it.
func IsDefaultKeyringLocked() bool {
	stdout, _, err := ExecuteCommand("", "gdbus", "call", "--session",
		"--dest", SECRET_SERVICE_NAME,
		"--object-path", SECRET_SERVICE_DEFAULT_COLLECTION,
		"--method", "org.freedesktop.DBus.Properties.Get", "org.freedesktop.Secret.Collection", "Locked")
	if err != nil {
		return false
	}
	locked, ok := ParseGdbusBoolean(stdout)
	return ok && locked
}

// Check if the account and password settings are correct for remote control.
func CheckRemoteControlCredentialsIsCorrect(inputUser string, inputPass string) (bool, error) {
	return UpdateRemoteControlCredentials(inputUser, inputPass, true)
}

// Update the account and password for remote control.
// When `dryrun` is true, only compare the stored credentials with the expected ones.
func UpdateRemoteControlCredentials(inputUser string, inputPass string, dryrun bool) (bool, error) {
	username := strings.TrimSpace(inputUser)
	password := inputPass
	if username == "" || strings.TrimSpace(password) == "" {
		return false, errors.New("username and password must not be empty")
	}
	credentials := BuildRemoteControlCredentials(username, password)

	if !dryrun {
		// secret-tool reads the secret from stdin, keep it out of the command line
		_, stderr, err := runCommand(credentials, "secret-tool", "store", "-l", SECRET_TOOL_LABEL, "xdg:schema", SECRET_TOOL_SCHEMA)
		if err != nil {
			return false, fmt.Errorf("secret-tool store: %w %s", err, strings.TrimSpace(stderr))
		}
	}

	stdout, stderr, err := runCommand("", "secret-tool", "lookup", "xdg:schema", SECRET_TOOL_SCHEMA)
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
