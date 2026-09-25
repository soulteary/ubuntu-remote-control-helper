#!/usr/bin/env bash
#
# Install urch to /usr/local/bin and run it as a daemon with supervisor.
#
# Usage (run as the desktop user who shares the screen, not with sudo):
#   bash installer.sh
#
# Env:
#   URCH_VER                  version to install, default 1.7.0
#   UBUNTU_REMOTE_USER        remote control username, asked if not set
#   UBUNTU_REMOTE_PASS        remote control password, asked if not set
#   URCH_INSTALL_DUMMY_XORG   set to 1 to install a dummy display driver config for machines
#                             WITHOUT a monitor. Do NOT use it with a monitor attached, the
#                             screen will stay black. The original config is backed up.

set -euo pipefail

if [ "$(id -u)" -eq 0 ]; then
  echo "please run the installer as the desktop user who shares the screen, not as root (do not use sudo)." >&2
  exit 1
fi

URCH_VER="${URCH_VER:-1.7.0}"
DESKTOP_USER="$(id -un)"
DESKTOP_UID="$(id -u)"
DESKTOP_HOME="$HOME"

# remote control credentials
REMOTE_USER="${UBUNTU_REMOTE_USER:-}"
REMOTE_PASS="${UBUNTU_REMOTE_PASS:-}"
if [ -z "$REMOTE_USER" ]; then
  read -r -p "remote control username: " REMOTE_USER
fi
if [ -z "$REMOTE_PASS" ]; then
  read -r -s -p "remote control password: " REMOTE_PASS
  echo
fi
if [ -z "$REMOTE_USER" ] || [ -z "$REMOTE_PASS" ]; then
  echo "remote control username and password must not be empty." >&2
  exit 1
fi

# install urch
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$SCRIPT_DIR/installer-standalone.sh" ]; then
  URCH_VER="$URCH_VER" bash "$SCRIPT_DIR/installer-standalone.sh"
else
  STANDALONE="$(mktemp)"
  wget -qO "$STANDALONE" https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/installer-standalone.sh
  URCH_VER="$URCH_VER" bash "$STANDALONE"
  rm -f "$STANDALONE"
fi

# install supervisor
sudo apt-get install -y supervisor

# escape a value for a double quoted supervisor environment value
supervisor_escape() {
  local v="$1"
  v="${v//\\/\\\\}"
  v="${v//\"/\\\"}"
  v="${v//%/%%}"
  printf '%s' "$v"
}

# write supervisor conf, it contains the password so only root can read it
SUPERVISOR_CONF=/etc/supervisor/conf.d/urch.conf
sudo install -m 600 /dev/null "$SUPERVISOR_CONF"
sudo tee "$SUPERVISOR_CONF" >/dev/null <<EOF
[program:urch]
command=/usr/local/bin/urch --daemon=1
user=${DESKTOP_USER}
environment=HOME="$(supervisor_escape "$DESKTOP_HOME")",XDG_RUNTIME_DIR="/run/user/${DESKTOP_UID}",DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/${DESKTOP_UID}/bus",UBUNTU_REMOTE_USER="$(supervisor_escape "$REMOTE_USER")",UBUNTU_REMOTE_PASS="$(supervisor_escape "$REMOTE_PASS")"
autostart=true
startsecs=3
startretries=100000
autorestart=true
stderr_logfile=/var/log/urch.err.log
stderr_logfile_maxbytes=10MB
stderr_logfile_backups=10
stdout_logfile=/var/log/urch.log
stdout_logfile_maxbytes=10MB
stdout_logfile_backups=10
EOF

# optional: dummy display driver for headless machines
if [ "${URCH_INSTALL_DUMMY_XORG:-0}" = "1" ]; then
  sudo apt-get install -y xserver-xorg-core xserver-xorg-video-dummy
  if [ -f /etc/X11/xorg.conf ]; then
    BACKUP="/etc/X11/xorg.conf.urch-backup.$(date +%Y%m%d%H%M%S)"
    sudo cp /etc/X11/xorg.conf "$BACKUP"
    echo "the original /etc/X11/xorg.conf has been backed up to $BACKUP"
  fi
  if [ -f "$SCRIPT_DIR/x11-xorg.conf" ]; then
    sudo install -m 644 "$SCRIPT_DIR/x11-xorg.conf" /etc/X11/xorg.conf
  else
    wget -qO- https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/x11-xorg.conf | sudo tee /etc/X11/xorg.conf >/dev/null
  fi
  echo "dummy display driver installed, if the screen is black after reboot, remove /etc/X11/xorg.conf (or restore the backup) and reboot."
fi

sudo supervisorctl reread
sudo supervisorctl update

echo "done, urch is running as ${DESKTOP_USER}, logs: /var/log/urch.log /var/log/urch.err.log"
