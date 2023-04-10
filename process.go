package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type Process struct {
	PID int
	CMD string
}

// Filter processes running in the system according to the filter rule
func FilterProcess(filter string) ([]Process, error) {
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
			cmd, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
			if err != nil {
				continue
			}

			cmdLine := strings.TrimSpace(string(cmd))
			if !strings.Contains(cmdLine, filter) {
				continue
			}

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
	} else {
		for _, process := range processes {
			fmt.Printf("[%s] Find Process: %d.\n", UBUNTU_REMOTE_CONTROL_APPNAME, process.PID)
			for {
				if !CheckProcessExistByPID(process.PID) {
					fmt.Printf("[%s] Process has been killed.\n", UBUNTU_REMOTE_CONTROL_APPNAME)
					break
				}
				err := syscall.Kill(process.PID, syscall.SIGKILL)
				if err != nil {
					fmt.Printf("[%s] Killing process failed: %s.\n", UBUNTU_REMOTE_CONTROL_APPNAME, err)
				}
			}
		}
	}
}
