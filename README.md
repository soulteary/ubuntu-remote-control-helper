# Ubuntu Remote Control Helper

English | [简体中文](README_zh.md)

<img src=".github/urch.jpg" width="180">

Make Ubuntu native remote control easy to use and reliable.

Ubuntu's built-in Desktop Sharing (RDP, provided by `gnome-remote-desktop`) is easy to break: the credentials in the keyring can change or become unreadable, the remote session becomes unusable once the screen locks, and the settings can be reset. `urch` (Ubuntu Remote Control Helper) checks the remote control settings and credentials, and corrects them to the values you expect, either once or continuously in the background.

- [Requirements](#requirements)
- [What urch changes](#what-urch-changes)
- [Install](#install)
- [Usage](#usage)
- [Configuration](#configuration)
- [Connect](#connect)
- [Upgrade from 1.7.0](#upgrade-from-170)
- [Uninstall](#uninstall)
- [Troubleshooting](#troubleshooting)
- [Docker](#docker)
- [Development](#development)

## Requirements

- Ubuntu 22.04 or later with the GNOME desktop (RDP support of `gnome-remote-desktop` starts from GNOME 42).
- `libsecret-tools` (provides `secret-tool`), installed automatically by the installers.
- The desktop user must be logged in, and the login keyring must be unlocked, see [Keyring and automatic login](#keyring-and-automatic-login).
- CPU architectures: amd64, arm64, armv7, armv6, 386.

urch manages **Desktop Sharing** (sharing the session of a logged-in user). On Ubuntu 24.04 and later, Settings also has a separate **Remote Login** feature (a system-level login screen over RDP), urch does not manage it.

## What urch changes

On every check, urch compares the following settings and the stored credentials of the current user with the expected values, and corrects only the ones that differ:

| Setting | Value | Why / side effect |
|---|---|---|
| `org.gnome.desktop.session idle-delay` | `0` | Keeps the remote session usable: the screen never blanks or locks automatically. **Anyone with physical access to the machine can use the session.** |
| `org.gnome.desktop.remote-desktop.rdp enable` | `true` | Enable RDP. |
| `org.gnome.desktop.remote-desktop.rdp screen-share-mode` | `mirror-primary` | Share the primary monitor. |
| `org.gnome.desktop.remote-desktop.rdp view-only` | `false` | Allow controlling the keyboard and mouse. |
| `org.gnome.desktop.remote-desktop.vnc enable` | `false` | Disable VNC. |
| RDP credentials in the keyring | your username and password | Stored with `secret-tool` (schema `org.gnome.RemoteDesktop.RdpCredentials`). |

After changing the RDP / VNC settings or the credentials, urch restarts the current user's `gnome-remote-desktop` processes (by killing them) to apply them. Correcting only `idle-delay` does not interrupt the remote sessions. When everything is already correct, nothing is changed.

## Install

Pick one of the following ways. All of them run urch as the desktop user who shares the screen. **Do not use `sudo` to run urch**: gsettings and the keyring are per user, running as root changes root's settings instead of yours, and urch refuses to run as root.

### Option 1: systemd user service (recommended)

The service runs inside your own desktop session, so urch can reach the session D-Bus directly, and the password stays in a file only you can read.

```bash
# install urch to /usr/local/bin
wget https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/installer-standalone.sh
bash installer-standalone.sh

# set your remote control username and password
printf "UBUNTU_REMOTE_USER='your-user'\nUBUNTU_REMOTE_PASS='your-strong-password'\n" > ~/.config/urch.env
chmod 600 ~/.config/urch.env

# install and start the service
mkdir -p ~/.config/systemd/user
wget -O ~/.config/systemd/user/urch.service https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/urch.service
systemctl --user daemon-reload
systemctl --user enable --now urch.service
```

### Option 2: supervisor

`installer.sh` installs urch and supervisor, asks for the remote control username and password, and writes `/etc/supervisor/conf.d/urch.conf` (readable only by root) which runs urch as you.

```bash
wget https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/installer.sh
bash installer.sh   # without sudo
```

To configure supervisor by hand, see [`example/supervisor-urch.conf`](example/supervisor-urch.conf).

### Option 3: the program only

```bash
wget https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/installer-standalone.sh
bash installer-standalone.sh
```

Or download the archive for your architecture from [Releases](https://github.com/soulteary/ubuntu-remote-control-helper/releases) and verify it with `urch_<version>_checksums.txt`. Then run it yourself, see [Usage](#usage).

The installers install the version pinned in the script by default, set `URCH_VER=x.y.z` to choose another one.

### Machines without a monitor

On a headless machine, the desktop may have no screen to share. `URCH_INSTALL_DUMMY_XORG=1 bash installer.sh` also installs a dummy display driver config to `/etc/X11/xorg.conf` (the original file is backed up to `/etc/X11/xorg.conf.urch-backup.*`).

> **Do not use it with a monitor attached, the screen will stay black.** See [Troubleshooting](#troubleshooting) to revert it.

### Keyring and automatic login

The RDP credentials are stored in the login keyring, which must be unlocked for urch and `gnome-remote-desktop` to read them. With a password login, the keyring is unlocked when you log in. With **automatic login** (common for remote machines), no password is typed, so the keyring stays locked unless its password is empty: open "Passwords and Keys" (seahorse), right-click the "Login" keyring, "Change Password", and leave the new password empty.

> **Security trade-off**: with an empty keyring password, everything in the login keyring (browser and Wi-Fi passwords, tokens, ...) is stored **unencrypted** on disk. Only do this on a machine you trust, preferably with full disk encryption.

See [the blog post (Chinese)](https://soulteary.com/2023/04/11/make-ubuntu-native-remote-control-reliable-with-urch.html) for a walk-through.

## Usage

Check and correct the settings once:

```bash
UBUNTU_REMOTE_USER=your-user UBUNTU_REMOTE_PASS=your-strong-password urch
```

Keep running and check every minute:

```bash
UBUNTU_DAEMON=true UBUNTU_REMOTE_USER=your-user UBUNTU_REMOTE_PASS=your-strong-password urch
```

In daemon mode, failures (e.g. the user has not logged in yet) are logged and retried in the next minute. urch prints its version at startup, please include it when reporting issues.

## Configuration

Every option can be set with an environment variable or a command line argument; command line arguments take precedence.

| Environment variable | Argument | Default | Description |
|---|---|---|---|
| `UBUNTU_REMOTE_USER` | `--user` | `soulteary` | Username for remote control. |
| `UBUNTU_REMOTE_PASS` | `--pass` | `soulteary` | Password for remote control. |
| `UBUNTU_DAEMON` | `--daemon` | `false` | Run in the background and check every minute. `1`, `on` and `true` enable it. |

> The default username and password are public, **always set your own**. urch prints a warning when the defaults are used.
> Prefer the environment variable for the password: command line arguments are visible to other users (e.g. with `ps`).

## Connect

Connect to `<ip-of-the-machine>:3389` (the default RDP port) with the username and password you set:

- Windows: Remote Desktop Connection (`mstsc`)
- macOS: Windows App (formerly Microsoft Remote Desktop)
- Linux: Remmina

If the firewall is enabled, allow the port: `sudo ufw allow 3389/tcp`.

## Upgrade from 1.7.0

Since 1.8.0, urch refuses to run as root, the installers must be run without `sudo`, and the supervisor config no longer uses `xvfb-run` (the old one starts a new, empty D-Bus session, which causes `secret-tool: The connection is closed`). Replacing only the binary is not enough if you used the old `installer.sh`:

1. Run the new `installer.sh` again, **without `sudo`**. It replaces the program and rewrites `/etc/supervisor/conf.d/urch.conf`.
2. If the machine has a monitor and the old installer wrote `/etc/X11/xorg.conf` (check with `grep -l dummy /etc/X11/xorg.conf`), remove it and reboot: `sudo rm /etc/X11/xorg.conf`.
3. Optionally, remove the old logs: `sudo rm -f /tmp/urch.log* /tmp/urch.err.log*`.

If you only used the program, run `installer-standalone.sh` again and make sure urch is not started with `sudo`.

## Uninstall

```bash
# systemd user service
systemctl --user disable --now urch.service
rm -f ~/.config/systemd/user/urch.service ~/.config/urch.env
systemctl --user daemon-reload

# supervisor
sudo rm -f /etc/supervisor/conf.d/urch.conf
sudo supervisorctl reread && sudo supervisorctl update

# the program
sudo rm -f /usr/local/bin/urch

# if the dummy display driver was installed, remove it (or restore /etc/X11/xorg.conf.urch-backup.*) and reboot
sudo rm -f /etc/X11/xorg.conf
```

Optionally, remove the stored credentials and disable remote control:

```bash
secret-tool clear xdg:schema org.gnome.RemoteDesktop.RdpCredentials
gsettings set org.gnome.desktop.remote-desktop.rdp enable false
```

## Troubleshooting

Status and logs:

```bash
# systemd user service
systemctl --user status urch.service
journalctl --user -u urch.service -f

# supervisor
sudo supervisorctl status urch
sudo tail -f /var/log/urch.log /var/log/urch.err.log
```

- **Black screen / cannot enter the desktop after installing**: `/etc/X11/xorg.conf` contains the dummy display driver config (installed by `URCH_INSTALL_DUMMY_XORG=1`, or by the installer of 1.7.0 and earlier). Switch to a text console (`Ctrl+Alt+F3`) or connect with ssh, run `sudo rm /etc/X11/xorg.conf` (or restore your backup) and reboot.
- **`secret-tool: The connection is closed`, `Cannot autolaunch D-Bus without X11 $DISPLAY`, or urch hangs**: urch cannot reach the desktop session of the user. Make sure it is not run with `sudo`, runs as the logged-in desktop user, and the login keyring is unlocked. When started from supervisor/cron/ssh, urch uses `/run/user/<uid>/bus` automatically, and commands time out after 30 seconds instead of hanging. If you upgraded from 1.7.0, see [Upgrade from 1.7.0](#upgrade-from-170).
- **urch reports success but the client cannot connect**: check that the port is listening with `ss -tln | grep 3389`, check the firewall, and make sure the desktop user is logged in and the session is not locked.

## Docker

The Docker images published to Docker Hub are **not supported**: urch must run inside the desktop user's session (gsettings, keyring, D-Bus), which is not available in a container. Install the program on the host instead.

## Development

```bash
go vet ./...
go test ./...
go build -o urch .
```

Pushing a `v*` tag runs the Release workflow, which builds the archives with GoReleaser and publishes them to GitHub Releases. The installers download `urch_<version>_linux_<arch>.tar.gz`, update `URCH_VER` in `example/installer*.sh` before tagging a new version.
