# upmerge - maintain local changes to /etc on macOS (and maybe other systems) across upgrades

Apple's [macOS](https://www.apple.com/macos/) tends to overwrite user's changes to files
in `/etc` on system upgrades, which means local modifications (such as disabling
password authentication for [sshd](https://man.openbsd.org/sshd_config.5), or enabling
[Touch ID for sudo](https://duckduckgo.com/?q=macos+pam_tid.so)) are regularly reset to
their defaults.

On other systems, such as OpenBSD, the system upgrade tools will prompt you to merge
your local changes (see [sysupgrade(8)](https://man.openbsd.org/sysupgrade.8),
[sysmerge(8)](https://man.openbsd.org/sysmerge.8)), putting the user (rather than the
system) in charge of deciding what goes and what doesn't.

`upmerge` attempts to fix this for macOS, by using the following approach:

1. Overwrite any system files in `/etc` with versions provided by the user (by default,
   stored in `/usr/local/upmerge/etc`);
2. Create a copy of any overwritten file by appending `.upmerge~` to its name, allowing
   for later inspection and/or merging.

There's nothing inherently macOS-specific about this tool - you can use it on other
systems as well; however it addresses a problem that is specific to macOS - you
shouldn't need it on e.g. Debian or OpenBSD.

## Installation

    go install github.com/rollcat/upmerge

## Usage

    upmerge [-hnv] [-s src] [-d dest]

Run `upmerge -nv` to preview changes. Flag `-n` means dry run, and `-v` means to be
verbose; together, these options will show which operations will be attempted.

Run `sudo upmerge` to apply your overrides - this is non-interactive, so you can run it
e.g. at every boot. However the recommended usage is to run it once after each system
upgrade, followed up by another reboot (to ensure all changes are applied). At the very
least, restart each affected service.

Upmerge will refuse destructive operations (such as overwriting the only known
backup). You should pay attention when it says things like `CHECK: /etc/foo.upmerge~`.
Inspect what changes have been made (e.g. `diff -u /etc/foo /etc/foo.upmerge~`), and once
you're happy with your system's state, delete the backup.

You can use the `-s` flag with a directory argument, to use a different directory
(default is `/usr/local/upmerge/etc`) as the "source of the truth". Similarly, you can
use `-d` to use a destination other than `/etc`.

## Word of caution and no warranty

This could eat your data, or make the system unbootable. There is no warranty.

Upmerge takes some reasonable precautions, e.g. always makes a backup before overwriting
a file; refuses to overwrite backups; actually it refuses to overwrite any existing
file. However, there is no warranty.

Don't use `/usr/local/etc` as the source directory for your overrides, since both
first-party and third-party software might already want to look there for some
configuration. If you don't like putting things in `/usr/local`, you can even use
something like `~/.upmerge/etc`.

Use at your own risk. Read the code. There is no warranty.

## Alternatives and inspiration

Inspired by this HN thread: <https://news.ycombinator.com/item?id=31750715>

Alternative approaches:

1. Run `sudo rsync -rb /usr/local/my-etc/ /etc/`. The `-b` is for backup.
2. <https://github.com/YuriyGuts/persistent-touch-id-sudo>
