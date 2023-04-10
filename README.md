# Ubuntu Remote Control Helper

<img src=".github/urch.jpg" width="180">

Make Ubuntu Native Remote Control Easy to use and Reliable.

## Usage

```bash
UBUNTU_REMOTE_USER=soulteary UBUNTU_REMOTE_PASS=soulteary ./urch
```

Run the program only once to check if the remote control settings in the system are correct and set the username and password of incorrectly configured settings to the values expected by the user.

You can combine the program with your preferred scheduling tasks or other programs to accomplish periodic checking tasks.

```bash
UBUNTU_DAEMON=true UBUNTU_REMOTE_USER=soulteary UBUNTU_REMOTE_PASS=soulteary ./urch
```

If you set the program directly to daemon mode, it will continuously check whether the remote control settings in the system are correct and set the username and password of incorrectly configured settings to the values expected by the user.

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

By default, the program runs as a simple command line interface that checks and corrects situations where the username and password in the system are different from what is expected. If this environment variable is set to "yes", the program will run continuously in the background and check the system settings every minute to ensure that your remote connection configuration is always correct.
