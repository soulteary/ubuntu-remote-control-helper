# Ubuntu Remote Control Helper

<img src=".github/urch.jpg" width="180">

Make Ubuntu Native Remote Control Easy to use and Reliable.

## Usage

> Run urch as the desktop user who shares the screen. **Do not use `sudo`**: gsettings and the keyring are per user, running as root changes root's settings instead of yours, and urch refuses to run as root.

```bash
UBUNTU_REMOTE_USER=soulteary UBUNTU_REMOTE_PASS=soulteary ./urch
```

Run the program only once to check if the remote control settings in the system are correct and set the username and password of incorrectly configured settings to the values expected by the user.

You can combine the program with your preferred scheduling tasks or other programs to accomplish periodic checking tasks.

```bash
UBUNTU_DAEMON=true UBUNTU_REMOTE_USER=soulteary UBUNTU_REMOTE_PASS=soulteary ./urch
```

If you set the program directly to daemon mode, it will continuously check whether the remote control settings in the system are correct and set the username and password of incorrectly configured settings to the values expected by the user.

The same options are available as command line arguments, which take precedence over the environment variables:

```bash
./urch --user=soulteary --pass=soulteary --daemon=1
```

Prefer the environment variable for the password, command line arguments are visible to other users (e.g. with `ps`).

## Install

```bash
# only install urch to /usr/local/bin
wget https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/installer-standalone.sh
bash installer-standalone.sh

# install urch and run it as a daemon with supervisor (run without sudo, it asks for the remote control username and password)
wget https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/installer.sh
bash installer.sh
```

The installers detect the CPU architecture (amd64, arm64, armv7, armv6, 386).

For a machine **without a monitor**, `URCH_INSTALL_DUMMY_XORG=1 bash installer.sh` also installs a dummy display driver config to `/etc/X11/xorg.conf` (the original file is backed up). Do not use it with a monitor attached, the screen will stay black.

The screen can only be shared when the desktop user is logged in and the login keyring is unlocked, e.g. by enabling automatic login and setting an empty password for the "Login" keyring in "Passwords and Keys" (seahorse). See [the blog post](https://soulteary.com/2023/04/11/make-ubuntu-native-remote-control-reliable-with-urch.html) for details.

## Uninstall

```bash
# stop and remove the supervisor program (if installed with installer.sh)
sudo rm -f /etc/supervisor/conf.d/urch.conf
sudo supervisorctl reread && sudo supervisorctl update

# remove the program
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

- **Black screen / cannot enter the desktop after installing**: an old installer overwrote `/etc/X11/xorg.conf` with a dummy display driver config. Switch to a text console (`Ctrl+Alt+F3`) or connect with ssh, run `sudo rm /etc/X11/xorg.conf` (or restore your backup) and reboot.
- **`secret-tool: The connection is closed`, `Cannot autolaunch D-Bus without X11 $DISPLAY`, or urch hangs**: urch can not reach the desktop session of the user. Make sure it is not run with `sudo`, runs as the logged-in desktop user, and the login keyring is unlocked. When started from supervisor/cron/ssh, urch uses `/run/user/<uid>/bus` automatically, commands time out after 30 seconds instead of hanging.

## Environment Variables

The program has only three variables: `UBUNTU_REMOTE_USER`, `UBUNTU_REMOTE_PASS`, and `UBUNTU_DAEMON`.

### `UBUNTU_REMOTE_USER`

Default value: `soulteary`

Usage:

```bash
# set remote control username to `soulteary`
UBUNTU_REMOTE_USER=soulteary
```

This variable represents the username used for connecting to Ubuntu remote control functionality. The program will set and continuously check if the system configuration matches the content specified by the user.

### `UBUNTU_REMOTE_PASS`

Default value: `soulteary`

Usage:

```bash
# set remote control password to `soulteary`
UBUNTU_REMOTE_PASS=soulteary
```

This variable represents the password used for connecting to Ubuntu remote control functionality. The program will set and continuously check if the system configuration matches the content specified by the user.

### `UBUNTU_DAEMON`

Default value: `false`

Usage:

```bash
# enable the Ubuntu Remote Control Helper running as daemon
UBUNTU_DAEMON=1
# or
UBUNTU_DAEMON=on
# or
UBUNTU_DAEMON=true
```

By default, the program runs as a simple command line interface that checks and corrects situations where the username and password in the system are different from what is expected. If this environment variable is set to `1`, `on` or `true`, the program will run continuously in the background and check the system settings every minute to ensure that your remote connection configuration is always correct.
