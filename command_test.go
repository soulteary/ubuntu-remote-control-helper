package main

import (
	"reflect"
	"strings"
	"testing"
)

// A fake system which stores gsettings and the keyring secret in memory.
type fakeSystem struct {
	settings map[string]string
	secret   string
	calls    []string
}

func (f *fakeSystem) run(stdin string, name string, args ...string) (string, string, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	switch {
	case name == "gsettings" && args[0] == "get":
		return f.settings[args[1]+" "+args[2]] + "\n", "", nil
	case name == "gsettings" && args[0] == "set":
		f.settings[args[1]+" "+args[2]] = args[3]
		return "", "", nil
	case name == "secret-tool" && args[0] == "lookup":
		return f.secret, "", nil
	case name == "secret-tool" && args[0] == "store":
		f.secret = stdin
		return "", "", nil
	}
	panic("unexpected command: " + name)
}

func (f *fakeSystem) setCalls() []string {
	var calls []string
	for _, c := range f.calls {
		if strings.HasPrefix(c, "gsettings set") || strings.HasPrefix(c, "secret-tool store") {
			calls = append(calls, c)
		}
	}
	return calls
}

func newFakeSystem(t *testing.T) *fakeSystem {
	f := &fakeSystem{settings: map[string]string{}, secret: BuildRemoteControlCredentials("user", "pass")}
	for _, s := range REMOTE_CONTROL_SETTINGS {
		f.settings[s.Schema+" "+s.Key] = s.Expected
	}
	original := runCommand
	runCommand = f.run
	t.Cleanup(func() { runCommand = original })
	return f
}

func TestEnsureRemoteControlConfig(t *testing.T) {
	cases := []struct {
		name        string
		modify      func(f *fakeSystem)
		wantRestart bool
		wantSets    []string
	}{
		{"everything is correct", func(f *fakeSystem) {}, false, nil},
		{
			"rdp disabled while the credentials are correct",
			func(f *fakeSystem) { f.settings[DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP+" enable"] = "false" },
			true,
			[]string{"gsettings set " + DEFAULT_UBUNTU_REMOTE_DESKTOP_RDP + " enable true"},
		},
		{
			"only idle-delay changed does not restart",
			func(f *fakeSystem) { f.settings[DEFAULT_UBUNTU_DESKTOP_SESSION+" idle-delay"] = "uint32 300" },
			false,
			[]string{"gsettings set " + DEFAULT_UBUNTU_DESKTOP_SESSION + " idle-delay uint32 0"},
		},
		{
			"credentials changed",
			func(f *fakeSystem) { f.secret = BuildRemoteControlCredentials("user", "random") },
			true,
			[]string{"secret-tool store -l " + SECRET_TOOL_LABEL + " xdg:schema " + SECRET_TOOL_SCHEMA},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFakeSystem(t)
			c.modify(f)

			restart, err := EnsureRemoteControlConfig("user", "pass")
			if err != nil {
				t.Fatal(err)
			}
			if restart != c.wantRestart {
				t.Errorf("restart = %v, want %v", restart, c.wantRestart)
			}
			if got := f.setCalls(); !reflect.DeepEqual(got, c.wantSets) {
				t.Errorf("changes = %q, want %q", got, c.wantSets)
			}
			for _, s := range REMOTE_CONTROL_SETTINGS {
				if got := f.settings[s.Schema+" "+s.Key]; got != s.Expected {
					t.Errorf("%s %s = %s, want %s", s.Schema, s.Key, got, s.Expected)
				}
			}
			if want := BuildRemoteControlCredentials("user", "pass"); f.secret != want {
				t.Errorf("secret = %s, want %s", f.secret, want)
			}
		})
	}
}
