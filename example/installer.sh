#!/bin/env bash

# install dependencies
sudo apt-get install -y libsecret-tools supervisor xserver-xorg-core xserver-xorg-video-dummy xvfb dbus-x11

# download urch
URCH_VER=1.6.0 \
wget https://github.com/soulteary/ubuntu-remote-control-helper/releases/download/v${URCH_VER}/urch_${URCH_VER}_linux_amd64.tar.gz && \
tar zxvf urch_${URCH_VER}_linux_amd64.tar.gz
rm -rf zxvf urch_${URCH_VER}_linux_amd64.tar.gz

# copy urch to executable directory
sudo mv urch /usr/local/bin/

# download supervisor conf
curl -sSL https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/supervisor-urch.conf -o /etc/supervisor/conf.d/urch.conf

# download x11 config
curl -sSL https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/x11-xorg.conf -o /etc/X11/xorg.conf