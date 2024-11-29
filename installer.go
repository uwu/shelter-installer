package main

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"encoding/json"

	gopsutil "github.com/shirou/gopsutil/process"
	"github.com/sqweek/dialog"
)

type AsarInstallState int
const (
	AsarInstallStateNone AsarInstallState = iota
	AsarInstallStateShelter
	AsarInstallStateShelterArchLinux
	AsarInstallStateKernel
	AsarInstallStateOtherMod
)

func isDiscordRunning() bool {
	procs, err := gopsutil.Processes()
	if err != nil { return false }

	for _, p := range procs {
		name, err := p.Name()

		if err == nil && strings.Contains(strings.ToLower(name), "discord") {
			return true
		}
	}

	return false
}


func mayNeedElevation() bool {
	user, err := user.Current()

	if err != nil {
		panic(err)
	}

	return user.Username != "root" && runtime.GOOS != "windows" && runtime.GOOS != "darwin";
}

/* func canElevateWithPolkit() bool {
	return runtime.GOOS != "windows" && runtime.GOOS != "darwin"
} */

// checks if the traditional shelter installer exists
func checkTraditionalInstall(instance DiscordInstance) AsarInstallState {
	// see if app/package.json exists and read it
	pkgJson, err := os.ReadFile(filepath.Join(instance.PathRes, "app/package.json"))

	if err != nil {
		// there's no app folder.
		// Either your mod messed with the asar vencord style, or you're stock
		// Hopefully asar client mods don't cause issues!!!!! I sure hope they don't!!! that'd be bad!!!!
		return AsarInstallStateNone
	}

	// first, check for the arch shelter package
	pacmanProc, err := os.StartProcess("/usr/bin/pacman", []string{ "pacman", "-Q", "shelter" }, &os.ProcAttr{})
	if err == nil {
		// we are on arch!
		state, err := pacmanProc.Wait()
		// idk what would error here but im just gonna silently continue
		if err == nil {
			if state.Success() {
				// we have the shelter arch package!
				return AsarInstallStateShelterArchLinux
			}
		}
	}

	_, err = os.Stat(filepath.Join(instance.PathRes, "original.asar"));
	originalAsarExists := err == nil

	// assume we have shelter installed iff package.json has { name: "shelter" },
	// and original.asar exists
	if originalAsarExists && strings.Contains(string(pkgJson), "\"name\": \"shelter\"") {
		return AsarInstallStateShelter
	}

	// test for kernel
	if strings.Contains(string(pkgJson), "\"name\": \"kernel\"") {
		return AsarInstallStateKernel
	}

	return AsarInstallStateOtherMod
}

// the removing flag does not change behaviour, but changes the error reporting used.
func uninstallTraditionalShelter(instance DiscordInstance, removing bool) bool {
	// if we actually, don't want to uninstall shelter, just quietly report success
	switch checkTraditionalInstall(instance) {
		case AsarInstallStateNone:
			return true
		case AsarInstallStateOtherMod:
			if !removing {
				dialog.Message("%s", "You appear to have another client mod installed. This may cause issues with shelter.").Info()
			}
			return true

		case AsarInstallStateKernel:
			if !removing {
				dialog.Message("%s", "You have Kernel installed. Please install the Kernel shelter injector, or uninstall Kernel and try again.").Error()
			}
			return false

		case AsarInstallStateShelterArchLinux:
			dialog.Message("%s", "You appear to have the Arch Linux shelter package installed. Please remove it (sudo pacman -R shelter) and try again.").Error();
			return false

		case AsarInstallStateShelter:

			removingOrUpgrading := "upgrading"
			if removing {
				removingOrUpgrading = "removing"
			}

			// remove the app folder, put original.asar back
			// we are guaranteed that these two things exist by the fact we got AsarInstallStateShelter back
			// move the asar first as that way around is less likely to break stuff if the first succeeds and second fails.

			var err error

			// note: on macos, we need a special permission that can only be granted in system settings.
			// when we attempt to perform a file operation, macos will show an instruction to the user, and fail
			if mayNeedElevation() {
				dialog.Message("%s %s %s", "You have an old-style shelter install, and", removingOrUpgrading, "it requires root permissions. We will now ask for your password.").Info()

				cmd := "mv " + filepath.Join(instance.PathRes, "original.asar") +
				" " + filepath.Join(instance.PathRes, "app.asar") +
				" && rm -rf " + filepath.Join(instance.PathRes, "app")

				// bad but using polkit directly is horrible so pick your poison
				proc, err2 := os.StartProcess("/usr/bin/pkexec", []string{"pkexec", "sh", "-c", cmd}, &os.ProcAttr{})
				err = err2

				if err == nil {
					state, err2 := proc.Wait()
					err = err2
					if err != nil && !state.Success() {
						err = errors.New("pkexec process exited with failure")
					}
				}

			} else {
				err = os.Rename(filepath.Join(instance.PathRes, "original.asar"), filepath.Join(instance.PathRes, "app.asar"))

				if err == nil {
					err = os.RemoveAll(filepath.Join(instance.PathRes, "app"))
				}
			}

			if err != nil {
				dialog.Message("%s", "Failed to remove your old-style shelter installation.").Error()
				return false
			}

			return true

		default:
			panic("Invalid return type from checkTraditionalInstall()")
	}
}

func isUpdateUrlInJson(jsonStr []byte) bool {
	var deserialized map[string] any

	err := json.Unmarshal(jsonStr, &deserialized)
	if err != nil {
		panic(err)
	}

	ue1 := deserialized["UPDATE_ENDPOINT"]
	ue2 := deserialized["NEW_UPDATE_ENDPOINT"]

	// i will have rob pike's decapitated head on a spike for the mess that is this language
	switch ue1 := ue1.(type) {
	case string:
		if strings.Contains(ue1, "inject.shelter.uwu.network") {
			return true
		}
	}

	switch ue2 := ue2.(type) {
	case string:
		if strings.Contains(ue2, "inject.shelter.uwu.network") {
			return true
		}
	}

	return false
}

func setUpdateUrlInJson(jsonStr []byte, branch string) []byte {
	var deserialized map[string] any

	err := json.Unmarshal(jsonStr, &deserialized)
	if err != nil {
		panic(err)
	}

	deserialized["UPDATE_ENDPOINT"] = "https://inject.shelter.uwu.network/" + branch
	deserialized["NEW_UPDATE_ENDPOINT"] = "https://inject.shelter.uwu.network/" + branch + "/"

	sered, err := json.MarshalIndent(deserialized, "", "	")
	if err != nil {
		panic(err)
	}

	return sered
}

func removeUpdateUrlFromJson(jsonStr []byte) []byte {
	var deserialized map[string] any

	err := json.Unmarshal(jsonStr, &deserialized)
	if err != nil {
		panic(err)
	}

	delete(deserialized, "UPDATE_ENDPOINT")
	delete(deserialized, "NEW_UPDATE_ENDPOINT")

	sered, err := json.MarshalIndent(deserialized, "", "	")
	if err != nil {
		panic(err)
	}

	return sered
}

func installShelter(instance DiscordInstance) {
	// step one: uninstall traditional shelter if necessary

	// this will also prompt users with errors and warnings about existing installs, and returns if we can proceed
	if !uninstallTraditionalShelter(instance, false) {
		return
	}

	// read the settings json
	// note if we fail here we remove the old shelter and don't replace it, lol oopsies.
	content, err := os.ReadFile(instance.PathCfg)
	if err == nil {

		// inject
		// TODO: other branches? not sure what I want to do with respect to that.
		newContent := setUpdateUrlInJson(content, "shelter")

		// write
		err = os.WriteFile(instance.PathCfg, newContent, 0644)
	}

	if err != nil {
		dialog.Message("%s", "Failed to install shelter. *If* you had an old-style injector present, it has been removed.").Error()
	} else {
		dialog.Message("%s", "shelter has been installed successfully 🎉. Please restart Discord if it is open.").Info()
	}
}

func isShelterNewInstalled(instance DiscordInstance) bool {
	content, err := os.ReadFile(instance.PathCfg)
	if err != nil { return false }

	return isUpdateUrlInJson(content)
}

func uninstallShelter(instance DiscordInstance) {
	// first, uninstall it traditionally
	// will tell the user if theres an issue itself
	_ = uninstallTraditionalShelter(instance, true)

	// next, remove it from settings.json
	content, err := os.ReadFile(instance.PathCfg)
	if err == nil {
		newContent := removeUpdateUrlFromJson(content)

		err = os.WriteFile(instance.PathCfg, newContent, 0644)
	}

	if err != nil {
		dialog.Message("%s", "Encountered an error while uninstalling shelter. shelter may or may not remain installed.")
	} else {
		dialog.Message("%s", "shelter has been uninstalled successfully.").Info()
	}
}

