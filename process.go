package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Process struct {
	PID int
	CMD string
}

// Check whether the command line of a process is the given program.
// Only argv[0] is matched, so e.g. an editor opening a file named after the program is not matched.
func IsProgramCmdline(cmdline []byte, program string) bool {
	argv0 := strings.SplitN(string(cmdline), "\x00", 2)[0]
	return strings.HasPrefix(filepath.Base(argv0), program)
}

// Check whether the command line is a gnome-remote-desktop daemon of another mode than desktop sharing,
// e.g. `--handover` used by remote login, or `--headless`.
func IsOtherRemoteDesktopMode(cmdline []byte) bool {
	for _, arg := range strings.Split(string(cmdline), "\x00")[1:] {
		if arg == "--handover" || arg == "--headless" || arg == "--system" {
			return true
		}
	}
	return false
}

// Filter processes of the current user running in the system according to the program name
func FilterProcess(program string) ([]Process, error) {
	d, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := d.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	var process []Process
	uid := uint32(os.Getuid()) // #nosec G115 -- uid always fits in uint32

	for {
		names, err := d.Readdirnames(10)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for _, name := range names {
			if name[0] < '0' || name[0] > '9' {
				continue
			}

			id, err := strconv.ParseInt(name, 10, 0)
			if err != nil {
				continue
			}

			pid := int(id)
			if pid == os.Getpid() {
				continue
			}

			// only handle processes of the current user, e.g. skip the system-level
			// gnome-remote-desktop daemon used for remote login on newer Ubuntu
			info, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
			if err != nil {
				continue
			}
			if stat, ok := info.Sys().(*syscall.Stat_t); !ok || stat.Uid != uid {
				continue
			}

			cmd, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
			if err != nil {
				continue
			}
			if !IsProgramCmdline(cmd, program) || IsOtherRemoteDesktopMode(cmd) {
				continue
			}

			cmdLine := strings.TrimSpace(strings.ReplaceAll(string(cmd), "\x00", " "))
			process = append(process, Process{PID: pid, CMD: cmdLine})
		}
	}

	return process, nil
}

// Check if a process exists based on its PID
func CheckProcessExistByPID(pid int) bool {
	process := fmt.Sprintf("/proc/%d", pid)
	_, err := os.Stat(process)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}

// Terminate relevant processes to apply new configuration
func KillProcessForApplyNewSettings() {
	processes, err := FilterProcess(UBUNTU_REMOTE_CONTROL_APPNAME)
	if err != nil {
		fmt.Printf("[%s] Find Process failed: %s.\n", UBUNTU_REMOTE_CONTROL_APPNAME, err)
		return
	}
	for _, process := range processes {
		fmt.Printf("[%s] Find Process: %d.\n", UBUNTU_REMOTE_CONTROL_APPNAME, process.PID)
		if err := syscall.Kill(process.PID, syscall.SIGKILL); err != nil {
			fmt.Printf("[%s] Killing process failed: %s.\n", UBUNTU_REMOTE_CONTROL_APPNAME, err)
			continue
		}
		killed := false
		for i := 0; i < 50; i++ {
			if !CheckProcessExistByPID(process.PID) {
				killed = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if killed {
			fmt.Printf("[%s] Process has been killed.\n", UBUNTU_REMOTE_CONTROL_APPNAME)
		} else {
			fmt.Printf("[%s] Process %d still exists after being killed.\n", UBUNTU_REMOTE_CONTROL_APPNAME, process.PID)
		}
	}
}
