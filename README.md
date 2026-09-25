# Ubuntu Remote Control Helper

English | [简体中文](README_zh.md)

<img src=".github/urch.jpg" width="180">

Make Ubuntu native remote control easy to use and reliable.

Ubuntu's built-in Desktop Sharing (RDP, provided by `gnome-remote-desktop`) is easy to break: the credentials in the keyring can change or become unreadable, the remote session becomes unusable once the screen locks, and the settings can be reset. `urch` (Ubuntu Remote Control Helper) checks the remote control settings and credentials, and corrects them to the values you expect, either once or continuously in the background.

- [Requirements](#requirements)
- [Desktop Sharing or Remote Login?](#desktop-sharing-or-remote-login)
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

## Desktop Sharing or Remote Login?

urch manages **Desktop Sharing**: it shares the session of a user who is already logged in. On Ubuntu 24.04 and later, Settings also has a separate **Remote Login** feature, urch does not manage it:

| | Desktop Sharing (urch) | Remote Login (Ubuntu 24.04+) |
|---|---|---|
| What you get | the session that is shown on the local screen | a new session started from the login screen over RDP |
| Needs a logged-in user | yes, so it usually needs automatic login after reboot | no, only the login screen (GDM) must be running |
| Credentials stored in | the login keyring, which must be unlocked | a file of the system service, no keyring involved |
| Default port | 3389 (moves to the next free port when taken) | 3389 |

**If you only need to reach the machine over RDP after every reboot, prefer Remote Login** and keep automatic login disabled, you do not need urch:

```bash
sudo grdctl --system rdp set-credentials <rdp-user> <rdp-password>
sudo grdctl --system rdp enable
sudo systemctl enable --now gnome-remote-desktop.service
sudo grdctl --system status
```

Remote Login needs GDM on Wayland: make sure `/etc/gdm3/custom.conf` does not contain `WaylandEnable=false` (often set with proprietary NVIDIA drivers). On Ubuntu 24.04 a remote login cannot join a session that the same user already has on the local screen, so do not combine it with automatic login of that user.

Use urch when you need to control **the same session that runs on the local screen**, e.g. to continue work started at the machine. This needs automatic login and a login keyring without password, see [Keyring and automatic login](#keyring-and-automatic-login) for the security trade-off.

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

# set your remote control username and password, the file is readable only by you
mkdir -p ~/.config
install -m 600 /dev/null ~/.config/urch.env
cat > ~/.config/urch.env <<'EOF'
UBUNTU_REMOTE_USER='your-user'
UBUNTU_REMOTE_PASS='your-strong-password'
EOF

# install and start the service
mkdir -p ~/.config/systemd/user
wget -O ~/.config/systemd/user/urch.service https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/urch.service
systemctl --user daemon-reload
systemctl --user enable --now urch.service
```

Values in `urch.env` are wrapped in single quotes, so characters such as `$`, `%`, `"` and `\` in the password are kept as is. If the password contains a single quote `'`, wrap it in double quotes instead and escape `"` and `\` with a backslash. Leading and trailing spaces of the password are kept.

The service starts with the graphical session and stops when you log out. If you installed an older `urch.service` (with `WantedBy=default.target`), run `systemctl --user reenable urch.service` after replacing the file.

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

The installers install the version pinned in the script by default, set `URCH_VER=x.y.z` to choose another one. The downloaded archive is verified against `urch_<version>_checksums.txt` of the release.

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

Connect to `<ip-of-the-machine>:3389` (the default RDP port) with the username and password you set.

When the port is already taken, e.g. by Remote Login on Ubuntu 24.04, Desktop Sharing listens on the next free port (up to 3399, `negotiate-port` is enabled by default). Check the actual port with:

```bash
ss -tlnp | grep gnome-remote
```

Clients:

- Windows: Remote Desktop Connection (`mstsc`)
- macOS: Windows App (formerly Microsoft Remote Desktop)
- Linux: Remmina

If the firewall is enabled, allow the port from your local network only (replace the subnet and the port with yours):

```bash
sudo ufw allow from 192.168.1.0/24 to any port 3389 proto tcp
```

Do not forward the port to the internet on your router, use a VPN (e.g. WireGuard, Tailscale) to connect from outside.

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
- **`the login keyring is locked`**: urch found the login keyring locked and did not read it (reading a locked keyring shows an unlock prompt on the screen, which would pop up every minute in daemon mode). gnome-remote-desktop cannot read the credentials in this state either, so remote connections fail. Log in with your password once, or with automatic login, see [Keyring and automatic login](#keyring-and-automatic-login).
- **`secret-tool: The connection is closed`, `Cannot autolaunch D-Bus without X11 $DISPLAY`, or urch hangs**: urch cannot reach the desktop session of the user. Make sure it is not run with `sudo`, runs as the logged-in desktop user, and the login keyring is unlocked. When started from supervisor/cron/ssh, urch uses `/run/user/<uid>/bus` automatically, and commands time out after 30 seconds instead of hanging. If you upgraded from 1.7.0, see [Upgrade from 1.7.0](#upgrade-from-170).
- **urch reports success but the client cannot connect**: check the listening port with `ss -tlnp | grep gnome-remote` (it may be 3390 or later, see [Connect](#connect)), check the firewall, and make sure the desktop user is logged in and the session is not locked.
- **Nothing listens on the RDP port, the logs mention an invalid or missing certificate**: RDP needs a TLS certificate, which the Settings app generates the first time remote desktop is turned on there. gnome-remote-desktop does not generate it by itself, so a machine configured only by urch may have none. Check with `gsettings get org.gnome.desktop.remote-desktop.rdp tls-cert`, an empty value means no certificate. Turn on remote desktop in Settings once, or generate one:

  ```bash
  mkdir -p ~/.local/share/gnome-remote-desktop
  openssl req -new -newkey rsa:4096 -days 3650 -nodes -x509 -subj "/CN=$(hostname)" \
    -keyout ~/.local/share/gnome-remote-desktop/rdp-tls.key \
    -out ~/.local/share/gnome-remote-desktop/rdp-tls.crt
  gsettings set org.gnome.desktop.remote-desktop.rdp tls-key ~/.local/share/gnome-remote-desktop/rdp-tls.key
  gsettings set org.gnome.desktop.remote-desktop.rdp tls-cert ~/.local/share/gnome-remote-desktop/rdp-tls.crt
  ```

- **The machine is unreachable after a while**: it may have suspended. Ubuntu does not suspend on AC power by default, but it may have been changed in Settings > Power. Check with `gsettings get org.gnome.settings-daemon.plugins.power sleep-inactive-ac-type`, and disable it with `gsettings set org.gnome.settings-daemon.plugins.power sleep-inactive-ac-type 'nothing'`.

## Docker

No Docker image is published: urch must run inside the desktop user's session (gsettings, keyring, D-Bus), which is not available in a container. The images published to Docker Hub by earlier releases are not supported, install the program on the host instead.

## Development

```bash
go vet ./...
go test ./...
go build -o urch .
```

Pushing a `v*` tag runs the Release workflow, which builds the archives with GoReleaser and publishes them to GitHub Releases. The installers download `urch_<version>_linux_<arch>.tar.gz`, update `URCH_VER` in `example/installer*.sh` before tagging a new version.
