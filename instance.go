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

		for _, channel := range channels {
			for _, pathItem := range strings.Split(path, ":") {
				joinedPath := filepath.Join(pathItem, channel)
				if _, err := os.Stat(joinedPath); err == nil {
					possiblepath, _ := filepath.EvalSymlinks(joinedPath)
					if possiblepath != joinedPath {
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

	if _, err := os.Stat(instance.PathRes); err == nil {
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
