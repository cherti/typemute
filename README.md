# Typemute

`typemute` is a tool that mutes all your unmuted microphones while you type. Your conference call fellows will thank you for it!

It currently only supports [PulseAudio](https://www.freedesktop.org/wiki/Software/PulseAudio/) as an audio backend.


## Usage

Just call `typemute` (or `./typemute` if only locally installed) in your terminal.

It will ask you for your sudo-password (see FAQ) for obtaining keypresses and will start working.


## How to get it

### Arch

[Install from AUR.](https://aur.archlinux.org/packages/typemute/)

### Debian

Install the packages `libinput-tools`, `expect` (for `unbuffer`) and `sudo`, then proceed to "build from source".

### Build from source

Install external dependencies:

  * `pacmd`
  * `libinput` (the tool, not just the library)
  * `unbuffer` (likely from the `expect` package)
  * `sudo`

Then either

    go get -u github.com/cherti/typemute

or clone and

    go get -u github.com/sqp/pulseaudio  # get the dependency
    go build typemute.go  # and build locally


## FAQ

### Why does it need sudo/root privileges?

Superuser privileges are required for libinput to read keypresses from devices in `/dev/input/*`. This could theoretically be avoided by adjusting permissions, however, being allowed to read from there as a user means every software can log every keystroke, which is highly undesirable. Therefore typemute simply requires elevated privileges for monitoring keystrokes (and only for that, typemute itself runs with user privileges only).
