package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/sqweek/dialog"
)

func main() {

	if isDiscordRunning() {
		dialog.Message("%s", "Discord appears to be currently running. Please fully exit it prior to un/installing shelter.").Error()
	}

	a := app.New()
	a.Settings().SetTheme(discordTheme{})
	w := a.NewWindow("shelter installer")
	w.SetFixedSize(true)

	allAvailableInstances := GetChannels()

	var selectedInstance *DiscordInstance

	header := canvas.NewText("shelter installer", color.White)
	header.TextSize = 22
	header.Alignment = fyne.TextAlignCenter

	description := widget.NewLabel("Choose the version of Discord you'd like to install to, then click install.")

	installButton := widget.NewButton("Install", func() {})
	installButton.Disable()

	uninstallButton := widget.NewButton("Uninstall", func() {})

	uninstallButton.Hide()

	setInstallVisibilities := func() {
		if selectedInstance == nil {
			uninstallButton.Hide()
			installButton.Show()
			installButton.Disable()
		} else {
			installButton.Enable()
			if isShelterNewInstalled(*selectedInstance) {
				uninstallButton.Show()
				installButton.Hide()
			} else {
				installButton.Show()
				uninstallButton.Hide()
			}

			tis := checkTraditionalInstall(*selectedInstance)
			if tis == AsarInstallStateShelter || tis == AsarInstallStateShelterArchLinux {
				installButton.SetText("Upgrade")
			} else {
				installButton.SetText("Install")
			}
		}

		w.Resize(fyne.NewSize(0, 0))
	}

	installButton.OnTapped = func() {
		if (selectedInstance == nil) {return}
		installShelter(*selectedInstance)

		setInstallVisibilities()
	}

	uninstallButton.OnTapped = func() {
		if (selectedInstance == nil) {return}
		uninstallShelter(*selectedInstance)

		setInstallVisibilities()
	}

	// fyne's Select widge API is not designed well (only deals in strings) so we have to do this BS.

	channelDisplayMap := make(map[string]*DiscordInstance)
	channelDisplayList := []string{}
	for _, instance := range allAvailableInstances {
		channelDisplayMap[instance.Channel] = &instance
		channelDisplayList = append(channelDisplayList, instance.Channel)
	}

	w.SetContent(container.NewVBox(
		header,
		description,
		widget.NewSelect(channelDisplayList, func(s string) {
			selectedInstance = channelDisplayMap[s]

			setInstallVisibilities()
		}),
		installButton,
		uninstallButton,
	))

	w.ShowAndRun()
}
