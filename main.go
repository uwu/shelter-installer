package main

import (
	"image/color"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	//"github.com/sqweek/dialog"
)

func mayNeedElevation() bool {
	user, err := user.Current()

	if err != nil {
		panic(err)
	}

	return user.Username != "root" && runtime.GOOS == "linux";
}

func main() {
	/* if mayNeedElevation() {
		dialog.Message("%s", "Installing shelter requires root permissions. Please rerun shelter installer as root to continue.").Error()
		return
	} */

	a := app.New()
	a.Settings().SetTheme(discordTheme{})
	w := a.NewWindow("shelter installer")
	w.SetFixedSize(true)

	selectedInstancePath := make([]string, 1)

	instances := make(map[string]string)
	var channels []string
	for _, instance := range GetChannels() {
		instances[instance.Channel] = instance.Path

		channels = append(channels, instance.Channel)
	}

	header := canvas.NewText("shelter installer", color.White)
	header.TextSize = 22
	header.Alignment = fyne.TextAlignCenter

	description := widget.NewLabel("Choose the version of Discord you'd like to install to, then click install.")

	installButton := widget.NewButton("Install", func() {})
	installButton.Disable()

	installedContainer := container.NewVBox()

	updateButton := widget.NewButton("Update", func() {})
	uninstallButton := widget.NewButton("Uninstall", func() {})

	installedContainer.Add(updateButton)
	installedContainer.Add(uninstallButton)

	installedContainer.Hide()

	showInstall := func() {
		installButton.Show()
		installButton.Enable()
		installedContainer.Hide()

		w.Resize(fyne.NewSize(0, 0))
	}

	showUninstall := func() {
		installButton.Hide()
		installButton.Disable()
		installedContainer.Show()

		w.Resize(fyne.NewSize(0, 0))
	}

	installButton.OnTapped = func() {
		shelterZip, err := os.CreateTemp("", "shelter.zip")
		if err != nil {
			a.Quit()
		}

		tempDirectory := os.TempDir()

		resp, err := http.Get("https://github.com/uwu/shelter/archive/refs/heads/main.zip")
		if err != nil {
			a.Quit()
		}
		defer resp.Body.Close()

		_, err = io.Copy(shelterZip, resp.Body)

		Unzip(shelterZip.Name(), tempDirectory)
		os.Remove(shelterZip.Name())

		shelterDir := filepath.Join(tempDirectory, "shelter-main")
		injectorPath := filepath.Join(shelterDir, "injectors/desktop/app")

		os.Rename(injectorPath, filepath.Join(selectedInstancePath[0], "app"))
		os.Remove(shelterDir)

		os.Rename(filepath.Join(selectedInstancePath[0], "app.asar"), filepath.Join(selectedInstancePath[0], "original.asar"))

		showUninstall()
	}

	updateButton.OnTapped = func() {
		os.RemoveAll(filepath.Join(selectedInstancePath[0], "app"))
		installButton.OnTapped()
	}

	uninstallButton.OnTapped = func() {
		os.RemoveAll(filepath.Join(selectedInstancePath[0], "app"))
		os.Rename(filepath.Join(selectedInstancePath[0], "original.asar"), filepath.Join(selectedInstancePath[0], "app.asar"))

		showInstall()
	}

	w.SetContent(container.NewVBox(
		header,
		description,
		widget.NewSelect(channels, func(s string) {
			selectedInstancePath[0] = instances[s]

			if _, err := os.Stat(filepath.Join(selectedInstancePath[0], "original.asar")); os.IsNotExist(err) {
				showInstall()
			} else {
				showUninstall()
			}
		}),
		installButton,
		installedContainer,
	))

	w.ShowAndRun()
}
