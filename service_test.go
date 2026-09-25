package main

import "testing"

func TestIsOtherRemoteDesktopMode(t *testing.T) {
	cases := map[string]bool{
		"/usr/libexec/gnome-remote-desktop-daemon\x00":               false,
		"/usr/libexec/gnome-remote-desktop-daemon\x00--handover\x00": true,
		"/usr/libexec/gnome-remote-desktop-daemon\x00--headless\x00": true,
		"/usr/libexec/gnome-remote-desktop-daemon\x00--system\x00":   true,
	}
	for cmdline, want := range cases {
		if got := IsOtherRemoteDesktopMode([]byte(cmdline)); got != want {
			t.Errorf("IsOtherRemoteDesktopMode(%q) = %v, want %v", cmdline, got, want)
		}
	}
}

func TestIsUnitFileStateEnabled(t *testing.T) {
	cases := map[string]bool{
		"enabled": true, "static": true, "enabled-runtime": true,
		"disabled": false, "masked": false, "": false, "not-found": false,
	}
	for state, want := range cases {
		if got := IsUnitFileStateEnabled(state); got != want {
			t.Errorf("IsUnitFileStateEnabled(%q) = %v, want %v", state, got, want)
		}
	}
}
