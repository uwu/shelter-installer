package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func cfgDir() string {
	cfg, err := os.UserConfigDir()

	if (err != nil) {
		panic(err)
	}

	return cfg
}

type DiscordInstance struct {
	// path to "resources" directory
	PathRes string
	// path to settings location
	PathCfg string
	Channel string
	// when true, PathRes has a known location but does not exist yet
	// this generally occurs when running the installer before the given
	// instance has been launched yet, on a self-host-updating platform
	NoRes   bool
}

func GetInstance(channel string) (DiscordInstance, error) {
	channelStringRes := "Discord"
	channelStringCfg := "discord"

	// Generate channel strings (e.g discord-canary, DiscordCanary, Discord Canary)
	if channel != "Stable" {
		channelStringCfg = channelStringCfg + strings.ToLower(channel)

		switch os := runtime.GOOS; os {
		case "darwin":
			channelStringRes = channelStringRes + " " + channel
		case "windows":
			channelStringRes = channelStringRes + channel
		default: // Linux and BSD are basically the same thing
			channelStringRes = channelStringRes + "-" + channel
		}
	}

	instance := DiscordInstance{
		PathRes: "",
		PathCfg: filepath.Join(cfgDir(), channelStringCfg, "settings.json"),
		Channel: channel,
		NoRes: false,
	}

	switch OS := runtime.GOOS; OS {
	case "darwin":
		instance.PathRes = filepath.Join("/Applications", channelStringRes+".app", "Contents", "Resources")
	case "windows":
		starterPath := filepath.Join(os.Getenv("localappdata"), channelStringRes, "/")
		filepath.Walk(starterPath, func(path string, _ fs.FileInfo, _ error) error {

			if strings.HasPrefix(filepath.Base(path), "app-") {
				instance.PathRes = filepath.Join(path, "resources")
			}

			return nil
		})
	default: // Linux and BSD are *still* basically the same thing
		channels := []string{channelStringRes, strings.ToLower(channelStringRes)}
		path := os.Getenv("PATH")

		// check both i.e, Discord and discord, Discord-Canary and discord-canary
		for _, channel := range channels {
			// check for those executables at every element of $PATH
			for _, pathItem := range strings.Split(path, ":") {
				joinedPath := filepath.Join(pathItem, channel)
				if _, err := os.Stat(joinedPath); err == nil {
					possiblepath, _ := filepath.EvalSymlinks(joinedPath)

					// possiblepath is either:
					// >= 1.0.136: a path (symlink or not) to the discord shim shell script
					// <= 1.0.135, possiblepath != joinedPath, a symlink to the real discord install location
					// <= 1.0.135, possiblepath == joinedPath, something weird we don't recognise, ignore it

					// check if its a shell script

					f, err := os.Open(possiblepath)
					if err == nil {
						defer f.Close()

						bexp := []byte{0x23, 0x21}
						bact := make([]byte, 2)
						_, err = f.Read(bact)

						if err == nil && bexp[0] == bact[0] && bexp[1] == bact[1] {
							// it is fine that we have no res folder because the legacy injector has been deprecated
							// before this install location started being used, and no version of the installer will
							// install to here, so we can ~safely just assume there is nothing to uninstall and say
							// "just install sheltupdate it'll be okay :)"
							instance.NoRes = true
							continue
						}
					}

					if possiblepath != joinedPath {
						// old style install
						instance.PathRes = filepath.Join(possiblepath, "..", "resources")
					}
				}
			}
		}

		// flatpak
		if instance.PathRes == "" && channel == "Stable" {
			instance.PathRes = "/var/lib/flatpak/app/com.discordapp.Discord/x86_64/stable/active/files/discord/resources/"
			instance.PathCfg = filepath.Join(filepath.Dir(cfgDir()), ".var/app/com.discordapp.Discord/config/discord/settings.json")
		}
	}

	if _, err := os.Stat(instance.PathRes); err == nil || instance.NoRes {
		return instance, nil
	} else {
		return instance, errors.New("Instance doesn't exist")
	}
}

func GetChannels() []DiscordInstance {
	possible := []string{"Stable", "PTB", "Canary", "Development"}
	var channels []DiscordInstance

	for _, channel := range possible {
		c, err := GetInstance(channel)
		if err == nil {
			channels = append(channels, c)
		}
	}

	return channels
}

func NewDiscordInstance(path string) (*DiscordInstance, error) {
	instance := DiscordInstance{
		PathRes:    path,
		Channel: "Unknown",
	}

	if _, err := os.Stat(filepath.Join(instance.PathRes, "app.asar")); err == nil {
		return &instance, nil
	} else {
		return nil, errors.New("Instance doesn't exist")
	}
}
