package main

import (
	"fmt"
	"strings"
)

const (
	// systemd user unit of the desktop sharing daemon, the same name on GNOME 42 (22.04) and GNOME 46 (24.04)
	UBUNTU_REMOTE_DESKTOP_SERVICE = `gnome-remote-desktop.service`
	// the headless daemon conflicts with the desktop sharing one, it is not touched when enabled
	UBUNTU_REMOTE_DESKTOP_HEADLESS_SERVICE = `gnome-remote-desktop-headless.service`
)

// Run `systemctl --user` and return the trimmed output.
func RunUserSystemctl(args ...string) (string, error) {
	stdout, stderr, err := ExecuteCommand("", "systemctl", append([]string{"--user"}, args...)...)
	output := strings.TrimSpace(stdout)
	if err != nil {
		return output, fmt.Errorf("systemctl --user %s: %w %s", strings.Join(args, " "), err, strings.TrimSpace(stderr))
	}
	return output, nil
}

// Whether a unit file state printed by `systemctl is-enabled` means the unit is started by systemd.
func IsUnitFileStateEnabled(state string) bool {
	switch state {
	case "enabled", "enabled-runtime", "static", "alias", "indirect", "generated", "transient":
		return true
	}
	return false
}

// The desktop sharing daemon is only started on login when its user unit is enabled,
// which the Settings app does when turning on remote desktop, but setting gsettings alone does not.
// Enable and start it when it is disabled, so the remote desktop keeps working after reboot.
func EnsureRemoteDesktopServiceEnabled() {
	state, err := RunUserSystemctl("is-enabled", UBUNTU_REMOTE_DESKTOP_SERVICE)
	if IsUnitFileStateEnabled(state) {
		return
	}
	switch state {
	case "disabled":
	case "masked":
		fmt.Printf("[%s] the service is masked, not enabling it.\n", UBUNTU_REMOTE_DESKTOP_SERVICE)
		return
	default:
		// no user systemd instance, or the unit does not exist
		if err != nil {
			fmt.Printf("[%s] can not check the service: %s\n", UBUNTU_REMOTE_DESKTOP_SERVICE, err)
		}
		return
	}

	if active, _ := RunUserSystemctl("is-active", UBUNTU_REMOTE_DESKTOP_HEADLESS_SERVICE); active == "active" {
		fmt.Printf("[%s] %s is running, not enabling it.\n", UBUNTU_REMOTE_DESKTOP_SERVICE, UBUNTU_REMOTE_DESKTOP_HEADLESS_SERVICE)
		return
	}

	fmt.Printf("[%s] the service is disabled, enable and start it.\n", UBUNTU_REMOTE_DESKTOP_SERVICE)
	if _, err := RunUserSystemctl("enable", "--now", UBUNTU_REMOTE_DESKTOP_SERVICE); err != nil {
		fmt.Printf("[%s] enable the service failed: %s\n", UBUNTU_REMOTE_DESKTOP_SERVICE, err)
	}
}

// Restart the desktop sharing daemon to apply new settings.
// Falls back to killing the processes when systemd can not be used, e.g. there is no user systemd instance.
func RestartRemoteDesktopService() {
	if _, err := RunUserSystemctl("restart", UBUNTU_REMOTE_DESKTOP_SERVICE); err != nil {
		fmt.Printf("[%s] restart the service failed, kill the processes instead: %s\n", UBUNTU_REMOTE_DESKTOP_SERVICE, err)
		KillProcessForApplyNewSettings()
		return
	}
	fmt.Printf("[%s] the service has been restarted.\n", UBUNTU_REMOTE_DESKTOP_SERVICE)
}
