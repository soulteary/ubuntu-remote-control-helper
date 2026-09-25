# Ubuntu Remote Control Helper

[English](README.md) | 简体中文

<img src=".github/urch.jpg" width="180">

让 Ubuntu 自带的远程控制简单好用、稳定可靠。

Ubuntu 自带的桌面共享（RDP，由 `gnome-remote-desktop` 提供）很容易出问题：keyring 里的凭据可能被改掉或读不出来，屏幕锁定后远程会话就用不了，相关设置也可能被重置。`urch`（Ubuntu Remote Control Helper）会检查远程控制的设置和凭据，并把它们修正为你期望的值。可以只运行一次，也可以在后台持续运行。

- [系统要求](#系统要求)
- [urch 会修改哪些设置](#urch-会修改哪些设置)
- [安装](#安装)
- [使用](#使用)
- [配置](#配置)
- [连接](#连接)
- [从 1.7.0 升级](#从-170-升级)
- [卸载](#卸载)
- [常见问题](#常见问题)
- [Docker](#docker)
- [开发](#开发)

## 系统要求

- Ubuntu 22.04 或更高版本，使用 GNOME 桌面（`gnome-remote-desktop` 从 GNOME 42 开始支持 RDP）。
- `libsecret-tools`（提供 `secret-tool` 命令），安装脚本会自动安装。
- 桌面用户必须已经登录，并且登录 keyring 处于解锁状态，见[keyring 与自动登录](#keyring-与自动登录)。
- 支持的 CPU 架构：amd64、arm64、armv7、armv6、386。

urch 管理的是**桌面共享**（共享已登录用户的桌面会话）。Ubuntu 24.04 及以后的设置里还有一个独立的**远程登录**功能（通过 RDP 访问系统级的登录界面），urch 不管理它。

## urch 会修改哪些设置

当 keyring 中保存的凭据与期望值不一致时，urch 会为当前用户应用以下设置：

| 设置 | 值 | 原因 / 副作用 |
|---|---|---|
| `org.gnome.desktop.session idle-delay` | `0` | 保证远程会话可用：屏幕永远不会自动熄灭或锁定。**任何能接触到这台机器的人都可以直接使用这个会话。** |
| `org.gnome.desktop.remote-desktop.rdp enable` | `true` | 开启 RDP。 |
| `org.gnome.desktop.remote-desktop.rdp screen-share-mode` | `mirror-primary` | 共享主显示器。 |
| `org.gnome.desktop.remote-desktop.rdp view-only` | `false` | 允许远程控制键盘和鼠标。 |
| `org.gnome.desktop.remote-desktop.vnc enable` | `false` | 关闭 VNC。 |
| keyring 中的 RDP 凭据 | 你设置的用户名和密码 | 通过 `secret-tool` 保存（schema 为 `org.gnome.RemoteDesktop.RdpCredentials`）。 |
| systemd 用户单元 `gnome-remote-desktop.service` | 已启用（enabled） | 只有启用了这个单元，登录后才会启动远程桌面守护进程。在设置里打开远程桌面时会自动启用它，但只改 gsettings 不会，重启后 RDP 就会失效。单元被 mask，或者 `gnome-remote-desktop-headless.service` 正在运行时，不做修改。 |

修改完成后，urch 会用 `systemctl --user restart` 重启 `gnome-remote-desktop.service` 用户单元，让新设置生效。只有在无法使用 systemd 时，才会退回到结束当前用户的桌面共享守护进程（远程登录使用的 `--handover`、`--headless`、`--system` 进程不会被结束）。

## 安装

下面三种方式任选其一。无论哪种方式，urch 都以共享屏幕的桌面用户身份运行。**不要用 `sudo` 运行 urch**：gsettings 和 keyring 都是按用户存储的，以 root 运行只会修改 root 自己的设置，而且 urch 会拒绝以 root 身份运行。

### 方式一：systemd 用户服务（推荐）

服务运行在你自己的桌面会话里，urch 可以直接连到会话 D-Bus，密码也只保存在一个只有你能读的文件里。

```bash
# 安装 urch 到 /usr/local/bin
wget https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/installer-standalone.sh
bash installer-standalone.sh

# 设置远程控制的用户名和密码
printf "UBUNTU_REMOTE_USER='your-user'\nUBUNTU_REMOTE_PASS='your-strong-password'\n" > ~/.config/urch.env
chmod 600 ~/.config/urch.env

# 安装并启动服务
mkdir -p ~/.config/systemd/user
wget -O ~/.config/systemd/user/urch.service https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/urch.service
systemctl --user daemon-reload
systemctl --user enable --now urch.service
```

### 方式二：supervisor

`installer.sh` 会安装 urch 和 supervisor，询问远程控制的用户名和密码，并生成 `/etc/supervisor/conf.d/urch.conf`（只有 root 可读），以你的身份运行 urch。

```bash
wget https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/installer.sh
bash installer.sh   # 不要加 sudo
```

如果想手动配置 supervisor，可以参考 [`example/supervisor-urch.conf`](example/supervisor-urch.conf)。

### 方式三：只安装程序

```bash
wget https://github.com/soulteary/ubuntu-remote-control-helper/raw/main/example/installer-standalone.sh
bash installer-standalone.sh
```

也可以从 [Releases](https://github.com/soulteary/ubuntu-remote-control-helper/releases) 下载对应架构的压缩包，用 `urch_<version>_checksums.txt` 校验后自行运行，见[使用](#使用)。

安装脚本默认安装脚本里指定的版本，可以通过 `URCH_VER=x.y.z` 指定其他版本。

### 没有接显示器的机器

没有接显示器时，桌面可能没有可以共享的屏幕。执行 `URCH_INSTALL_DUMMY_XORG=1 bash installer.sh`，会额外把一个虚拟显卡驱动配置安装到 `/etc/X11/xorg.conf`（原文件会备份为 `/etc/X11/xorg.conf.urch-backup.*`）。

> **接了显示器的机器不要使用这个选项，否则开机后会黑屏。** 恢复方法见[常见问题](#常见问题)。

### keyring 与自动登录

RDP 凭据保存在登录 keyring 中，keyring 必须处于解锁状态，urch 和 `gnome-remote-desktop` 才能读到它。用密码登录时，keyring 会在登录时自动解锁。如果开启了**自动登录**（远程机器上很常见），因为没有输入密码，keyring 会一直保持锁定，除非它的密码为空：打开"密码和密钥"（seahorse），右键"登录"keyring，选择"更改密码"，新密码留空即可。

> **安全提示**：keyring 密码为空时，登录 keyring 中的所有内容（浏览器和 Wi-Fi 密码、各种令牌等）都会以**明文**形式保存在磁盘上。只应在你信任的机器上这样做，最好同时开启全盘加密。

详细的图文步骤可以参考[这篇博客](https://soulteary.com/2023/04/11/make-ubuntu-native-remote-control-reliable-with-urch.html)。

## 使用

检查并修正一次设置：

```bash
UBUNTU_REMOTE_USER=your-user UBUNTU_REMOTE_PASS=your-strong-password urch
```

持续运行，每分钟检查一次：

```bash
UBUNTU_DAEMON=true UBUNTU_REMOTE_USER=your-user UBUNTU_REMOTE_PASS=your-strong-password urch
```

在后台模式下，遇到错误（例如用户还没有登录）时只会记录日志，下一分钟再重试。urch 启动时会打印版本号，反馈问题时请一并附上。

## 配置

所有选项都可以通过环境变量或命令行参数设置，命令行参数优先。

| 环境变量 | 命令行参数 | 默认值 | 说明 |
|---|---|---|---|
| `UBUNTU_REMOTE_USER` | `--user` | `soulteary` | 远程控制的用户名。 |
| `UBUNTU_REMOTE_PASS` | `--pass` | `soulteary` | 远程控制的密码。 |
| `UBUNTU_DAEMON` | `--daemon` | `false` | 在后台运行，每分钟检查一次。取值 `1`、`on`、`true` 表示开启。 |

> 默认的用户名和密码是公开的，**请务必设置你自己的**。使用默认值时，urch 会打印警告。
> 密码建议通过环境变量传入：命令行参数对其他用户可见（例如通过 `ps` 命令）。

## 连接

用你设置的用户名和密码连接 `<机器的 IP>:3389`（RDP 默认端口）：

- Windows：远程桌面连接（`mstsc`）
- macOS：Windows App（原 Microsoft Remote Desktop）
- Linux：Remmina

如果开启了防火墙，需要放行端口：`sudo ufw allow 3389/tcp`。

## 从 1.7.0 升级

从 1.8.0 开始：urch 拒绝以 root 身份运行；安装脚本必须在不加 `sudo` 的情况下执行；supervisor 配置不再使用 `xvfb-run`（旧配置会启动一个全新的空 D-Bus 会话，这正是 `secret-tool: The connection is closed` 报错的原因）。如果你用过旧版 `installer.sh`，只替换程序文件是不够的：

1. 重新执行新版 `installer.sh`，**不要加 `sudo`**。它会替换程序，并重写 `/etc/supervisor/conf.d/urch.conf`。
2. 如果机器接了显示器，而旧版安装脚本写入过 `/etc/X11/xorg.conf`（可以用 `grep -l dummy /etc/X11/xorg.conf` 检查），请删除它并重启：`sudo rm /etc/X11/xorg.conf`。
3. 可选：删除旧日志 `sudo rm -f /tmp/urch.log* /tmp/urch.err.log*`。

如果你只安装了程序本身，重新执行 `installer-standalone.sh` 即可，并确保运行 urch 时不加 `sudo`。

## 卸载

```bash
# systemd 用户服务
systemctl --user disable --now urch.service
rm -f ~/.config/systemd/user/urch.service ~/.config/urch.env
systemctl --user daemon-reload

# supervisor
sudo rm -f /etc/supervisor/conf.d/urch.conf
sudo supervisorctl reread && sudo supervisorctl update

# 程序本身
sudo rm -f /usr/local/bin/urch

# 如果安装过虚拟显卡驱动配置，删除它（或恢复 /etc/X11/xorg.conf.urch-backup.*）后重启
sudo rm -f /etc/X11/xorg.conf
```

可选：删除保存的凭据，并关闭远程控制：

```bash
secret-tool clear xdg:schema org.gnome.RemoteDesktop.RdpCredentials
gsettings set org.gnome.desktop.remote-desktop.rdp enable false
```

## 常见问题

查看运行状态和日志：

```bash
# systemd 用户服务
systemctl --user status urch.service
journalctl --user -u urch.service -f

# supervisor
sudo supervisorctl status urch
sudo tail -f /var/log/urch.log /var/log/urch.err.log
```

- **安装后黑屏、进不了桌面**：`/etc/X11/xorg.conf` 中是虚拟显卡驱动的配置（由 `URCH_INSTALL_DUMMY_XORG=1` 安装，或者是 1.7.0 及更早版本的安装脚本写入的）。切换到文本控制台（`Ctrl+Alt+F3`）或通过 ssh 登录，执行 `sudo rm /etc/X11/xorg.conf`（或恢复备份）后重启。
- **报错 `secret-tool: The connection is closed`、`无法在没有 X11 $DISPLAY 的情况下自动启动 D-Bus`，或者 urch 卡住不动**：urch 连不上该用户的桌面会话。请确认没有用 `sudo` 运行、是以已登录的桌面用户身份运行，并且登录 keyring 已解锁。从 supervisor、cron 或 ssh 启动时，urch 会自动使用 `/run/user/<uid>/bus`；命令超过 30 秒会超时退出，不会一直卡住。如果是从 1.7.0 升级上来的，请看[从 1.7.0 升级](#从-170-升级)。
- **urch 显示成功，但客户端连不上**：用 `ss -tln | grep 3389` 确认端口在监听，检查防火墙，并确认桌面用户已登录、屏幕没有锁定。

## Docker

发布到 Docker Hub 的镜像**不受支持**：urch 必须运行在桌面用户的会话里（需要 gsettings、keyring 和 D-Bus），容器里没有这些。请直接在主机上安装。

## 开发

```bash
go vet ./...
go test ./...
go build -o urch .
```

推送 `v*` 格式的 tag 会触发 Release workflow，用 GoReleaser 构建压缩包并发布到 GitHub Releases。安装脚本下载的是 `urch_<version>_linux_<arch>.tar.gz`，发布新版本前请先更新 `example/installer*.sh` 中的 `URCH_VER`。
