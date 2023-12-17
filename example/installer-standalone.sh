#!/bin/env bash

# install dependencies
sudo apt-get install -y libsecret-tools wget

# download urch
URCH_VER=1.7.0
wget "https://github.com/soulteary/ubuntu-remote-control-helper/releases/download/v${URCH_VER}/urch_${URCH_VER}_linux_amd64.tar.gz" && \
tar zxvf "urch_${URCH_VER}_linux_amd64.tar.gz" && \
rm -rf "urch_${URCH_VER}_linux_amd64.tar.gz"

# copy urch to executable directory
sudo mv urch /usr/local/bin/
