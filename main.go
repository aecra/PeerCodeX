package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/aecra/PeerCodeX/data"
)

var topWindow fyne.Window
var mainContent *container.Split

func main() {
	a := app.NewWithID("cn.aecra.PeerCodeX")
	a.Settings().SetTheme(theme.LightTheme())

	w := a.NewWindow("PeerCodeX")
	topWindow = w

	w.SetIcon(data.AppIcon)
	w.SetMaster()

	// content used for each page
	content := container.NewBorder(nil, nil, nil, nil, makeBlankContent())

	// nav list
	navList := widget.NewList(
		func() int {
			return len(GetPages())
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(GetPages()[i].Name)
		})

	navList.OnSelected = func(id widget.ListItemID) {
		if GetPages()[id].Content == nil {
			GetPages()[id].Content = makeBlankContent()
		}
		content.Objects[0] = GetPages()[id].Content
		content.Refresh()
	}
	// 默认选中第一个页面
	navList.Select(0)

	themes := container.NewGridWithColumns(2,
		widget.NewButton("Dark", func() {
			a.Settings().SetTheme(theme.DarkTheme())
		}),
		widget.NewButton("Light", func() {
			a.Settings().SetTheme(theme.LightTheme())
		}),
	)

	leftNav := container.NewBorder(nil, themes, nil, nil, navList)

	split := container.NewHSplit(leftNav, content)
	split.Offset = 0.2
	mainContent = split
	w.SetContent(split)

	// 设置最小窗口大小
	w.Resize(fyne.NewSize(800, 600))

	w.ShowAndRun()
}

// Open the masklayer
func openLoadingMask() {
	topWindow.SetContent(makeLoadingMask())
}

func closeLoadingMask() {
	topWindow.SetContent(mainContent)
	mainContent.Refresh()
}

// Create a mask layer showing the loading animation
func makeLoadingMask() *fyne.Container {
	loading := widget.NewProgressBarInfinite()
	loading.Show()
	loading.Start()

	loadingBox := container.NewCenter(loading)
	return loadingBox
}
