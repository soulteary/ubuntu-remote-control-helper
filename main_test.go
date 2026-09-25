package main

import "testing"

func TestParseConfig(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	cases := []struct {
		name string
		args []string
		env  map[string]string
		want Config
	}{
		{"defaults", nil, nil, Config{DEFAULT_USERNAME, DEFAULT_PASSWORD, false}},
		{"env", nil, map[string]string{"UBUNTU_REMOTE_USER": "u", "UBUNTU_REMOTE_PASS": "p", "UBUNTU_DAEMON": "on"}, Config{"u", "p", true}},
		{"cli overrides env", []string{"--user=c", "--pass=cp", "--daemon=1"}, map[string]string{"UBUNTU_REMOTE_USER": "u", "UBUNTU_REMOTE_PASS": "p"}, Config{"c", "cp", true}},
		// passing a value equal to the default must still override env
		{"cli equal to default", []string{"--user=" + DEFAULT_USERNAME}, map[string]string{"UBUNTU_REMOTE_USER": "u"}, Config{DEFAULT_USERNAME, DEFAULT_PASSWORD, false}},
		{"password spaces are kept", []string{"--pass= cp "}, map[string]string{"UBUNTU_REMOTE_PASS": " p "}, Config{DEFAULT_USERNAME, " cp ", false}},
		{"env password spaces are kept", nil, map[string]string{"UBUNTU_REMOTE_PASS": " p "}, Config{DEFAULT_USERNAME, " p ", false}},
		{"blank password is ignored", []string{"--pass=  "}, nil, Config{DEFAULT_USERNAME, DEFAULT_PASSWORD, false}},
		{"cli daemon off overrides env", []string{"--daemon=false"}, map[string]string{"UBUNTU_DAEMON": "true"}, Config{DEFAULT_USERNAME, DEFAULT_PASSWORD, false}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseConfig(c.args, env(c.env))
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestBuildRemoteControlCredentials(t *testing.T) {
	cases := map[string][2]string{
		`{'username': <'user'>, 'password': <'pass'>}`:           {"user", "pass"},
		`{'username': <'o\'neil'>, 'password': <'a\\b"$(x)%s'>}`: {"o'neil", `a\b"$(x)%s`},
	}
	for want, in := range cases {
		if got := BuildRemoteControlCredentials(in[0], in[1]); got != want {
			t.Errorf("BuildRemoteControlCredentials(%q, %q) = %s, want %s", in[0], in[1], got, want)
		}
	}
}

func TestIsProgramCmdline(t *testing.T) {
	cases := []struct {
		cmdline string
		want    bool
	}{
		{"/usr/libexec/gnome-remote-desktop-daemon\x00", true},
		{"/usr/libexec/gnome-remote-desktop-daemon\x00--system\x00", true},
		{"vim\x00gnome-remote-desktop.txt\x00", false},
		{"grep\x00gnome-remote-desktop\x00", false},
	}
	for _, c := range cases {
		if got := IsProgramCmdline([]byte(c.cmdline), UBUNTU_REMOTE_CONTROL_APPNAME); got != c.want {
			t.Errorf("IsProgramCmdline(%q) = %v, want %v", c.cmdline, got, c.want)
		}
	}
}
