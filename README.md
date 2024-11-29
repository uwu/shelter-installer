# shelter-installer
shelter-installer is a cross-platform installer for the [shelter](https://shelter.uwu.network) Discord client mod.

It will install shelter using [sheltupdate](https://github.com/uwu/sheltupdate),
a robust, rootless, host-update-resistant client mod injection mechanism.

If it detects that you have a conflicting client mod install, it will warn or back out,
and if it detects an existing shelter install using the traditional injector mechanism, it will upgrade you to
sheltupdate (this will be denoted in the UI as "upgrade" instead of "install" showing on the button).

It supports all OSes and all Discord channels.

It currently does not allow specifying custom sheltupdate branches.

![Screenshot of the shelter Installer](https://i.uwu.network/61318cfbc.png)
