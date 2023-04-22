package main

import (
	"context"
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/aecra/PeerCodeX/client"
	"github.com/aecra/PeerCodeX/data"
	"github.com/aecra/PeerCodeX/dc"
	"github.com/aecra/PeerCodeX/server"
)

type Page struct {
	// Name is the name of the page
	Name string
	// Icon is the icon of the page
	Icon fyne.Resource
	// Content is the content of the page
	Content fyne.CanvasObject
}

// NewPage creates a new page
func NewPage(name string, icon fyne.Resource, content fyne.CanvasObject) *Page {
	return &Page{
		Name:    name,
		Icon:    icon,
		Content: content,
	}
}

var pages []*Page
var mu = sync.Mutex{}

func GetPages() []*Page {
	mu.Lock()
	defer mu.Unlock()
	if len(pages) == 0 {
		pages = []*Page{
			NewPage("Home", nil, makeHomeContent()),
			NewPage("Service Status", nil, makeServiceStatusContent()),
			NewPage("File List", nil, makeFileListContent()),
			NewPage("Node List", nil, makeNodeListContent()),
			NewPage("Settings", nil, makeSettingContent()),
			NewPage("About", nil, makeAboutContent()),
		}
	}
	return pages
}

func makeHomeContent() fyne.CanvasObject {
	logo := canvas.NewImageFromResource(data.HomeImage)
	logo.FillMode = canvas.ImageFillContain

	logo.SetMinSize(fyne.NewSize(480, 270))

	home := container.NewCenter(container.NewVBox(
		widget.NewLabelWithStyle("Demonstration System of Network Coding in P2P", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		logo,
		widget.NewLabel(""), // balance the header on the tutorial screen we leave blank on this content
	))
	return home
}

var serverInstance = server.NewServer()

func makeServiceStatusContent() fyne.CanvasObject {
	var ServiceStatusPage *fyne.Container
	title := widget.NewLabel("Service Status")

	// p1 is a horizontal box with a label and an icon
	t1 := widget.NewLabel("Current Service Status: ")
	statusIcon := canvas.NewImageFromResource(data.StatusOff)
	statusIcon.FillMode = canvas.ImageFillContain
	statusIcon.SetMinSize(fyne.NewSize(16, 16))
	p1 := container.NewHBox(t1, statusIcon)

	status := false

	// p2 is a form with a label and a text input
	p2 := widget.NewForm(
		widget.NewFormItem("Service Host", widget.NewEntry()),
		widget.NewFormItem("Service Port", widget.NewEntry()),
	)
	// set default value
	p2.Items[0].Widget.(*widget.Entry).SetText("0.0.0.0")
	p2.Items[1].Widget.(*widget.Entry).SetText("8080")

	defer func() {
		// if panic occurs, show a dialog
		if err := recover(); err != nil {
			// dialog.ShowError(err.(error), topWindow)
			log.Println(err)
		}
	}()

	// p3 is a button with primary color
	var p3 *widget.Button
	p3 = &widget.Button{
		Text:       "Start Service",
		Importance: widget.HighImportance,
		OnTapped: func() {
			if status {
				if !serverInstance.IsRunning() {
					return
				}
				serverInstance.Stop()
				status = false
				statusIcon.Resource = data.StatusOff
				p3.Text = "Start Service"
				p3.Importance = widget.HighImportance
			} else {
				if serverInstance.IsRunning() {
					return
				}
				host := p2.Items[0].Widget.(*widget.Entry).Text
				port := p2.Items[1].Widget.(*widget.Entry).Text
				serverInstance.SetHost(host)
				serverInstance.SetPort(port)
				dc.SetHost(host)
				dc.SetPort(port)

				panicChan := make(chan error)
				go serverInstance.Start(panicChan)
				go func() {
					// watch at most 10 seconds
					ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
					defer cancel()
					select {
					case <-ctx.Done():
						log.Println("timeout")
					case err := <-panicChan:
						log.Println(err)
						dialog.ShowError(err, topWindow)
						status = false
						statusIcon.Resource = data.StatusOff
						p3.Text = "Start Service"
						p3.Importance = widget.HighImportance
					}
				}()
				time.Sleep(50 * time.Millisecond)
				if serverInstance.IsRunning() {
					status = true
					statusIcon.Resource = data.StatusOn
					p3.Text = "Stop Service"
					p3.Importance = widget.DangerImportance
				}
			}
			ServiceStatusPage.Refresh()
		},
	}

	// a note for the user
	intro := widget.NewLabel("Note: You can only start the service when the service host and port are not empty.")
	intro.Wrapping = fyne.TextWrapWord

	ServiceStatusPage = container.NewBorder(
		container.NewVBox(title, widget.NewSeparator(), p1, p2, p3, intro), nil, nil, nil, nil)
	return ServiceStatusPage
}

func makeFileListContent() fyne.CanvasObject {
	title := widget.NewLabel("File List")

	newSeedButton := makeNewSeedButton()

	var fileListWidget *widget.List
	fileListWidget = widget.NewList(
		func() int {
			return len(dc.FileList)
		},
		func() fyne.CanvasObject {
			return makeFileListItem(fileListWidget)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			downloaded := true
			f := dc.FileList[id]
			for _, d := range f.IsDownloaded {
				if d == false {
					downloaded = false
					break
				}
			}
			progressBar := item.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.ProgressBar)
			downloadedTextLabel := item.(*fyne.Container).Objects[1].(*fyne.Container).Objects[2].(*widget.Label)
			if downloaded {
				progressBar.SetValue(1)
				downloadedTextLabel.Show()
			} else {
				// sum ProcessRate
				var sumRate float64
				for _, v := range f.ProcessRate {
					sumRate += v
				}
				progressBar.SetValue(sumRate / float64(len(f.ProcessRate)))
				downloadedTextLabel.Hide()
			}
			item.(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*widget.Label).SetText(f.Path)
		},
	)

	t := makeFileListToolbar(fileListWidget.Refresh)

	return container.NewBorder(
		container.NewVBox(title,
			widget.NewSeparator(),
			newSeedButton,
		),
		container.NewVBox(widget.NewSeparator(), t),
		nil,
		nil,
		container.NewMax(fileListWidget))
}

func makeNodeListContent() fyne.CanvasObject {
	title := widget.NewLabel("Node List")

	var nodeListWidget *widget.List
	nodeListWidget = widget.NewList(
		func() int {
			return len(dc.GetNodeStatusList())
		},
		func() fyne.CanvasObject {
			// address, status Icon, refresh button, delete button
			statusIcon := canvas.NewImageFromResource(data.StatusOff)
			statusIcon.FillMode = canvas.ImageFillContain
			statusIcon.SetMinSize(fyne.NewSize(16, 16))
			address := widget.NewLabel("")
			return container.NewBorder(
				nil,
				nil,
				container.NewHBox(widget.NewLabel(" "), statusIcon, address),
				container.NewHBox(widget.NewToolbar(
					widget.NewToolbarSpacer(),
					widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
						client.CkeckServerStatus(address.Text)
						nodeListWidget.Refresh()
					}),
					widget.NewToolbarAction(theme.DeleteIcon(), func() {
						dc.DeleteNode(address.Text)
						nodeListWidget.Refresh()
					}),
				)),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			// add data to the widget
			item.(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*widget.Label).SetText(dc.GetNodeStatusList()[id].Addr)
			if dc.GetNodeStatusList()[id].IsOn {
				item.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*canvas.Image).Resource = data.StatusOn
			} else {
				item.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*canvas.Image).Resource = data.StatusOff
			}
		},
	)

	go func() {
		for {
			nodeListWidget.Refresh()
			time.Sleep(30 * time.Second)
		}
	}()

	t := widget.NewToolbar(widget.NewToolbarAction(theme.ContentAddIcon(), func() {
		host := widget.NewEntry()
		port := widget.NewEntry()
		items := []*widget.FormItem{
			widget.NewFormItem("Host", host),
			widget.NewFormItem("Port", port),
		}

		formDialog := dialog.NewForm("Add Node", "Add", "Cancel", items, func(b bool) {
			if !b {
				return
			}

			log.Println("Server Address: ", host.Text+":"+port.Text)
			dc.AddNode(host.Text + ":" + port.Text)
			nodeListWidget.Refresh()
		}, topWindow)
		formDialog.Resize(fyne.NewSize(300, 150))
		formDialog.Show()
	}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			client.CkeckAllServerStatus()
			nodeListWidget.Refresh()
		}),
	)

	return container.NewBorder(
		container.NewVBox(title,
			widget.NewSeparator(),
		),
		container.NewVBox(widget.NewSeparator(), t),
		nil,
		nil,
		container.NewMax(nodeListWidget))
}

func makeSettingContent() fyne.CanvasObject {
	return nil
}

func makeAboutContent() fyne.CanvasObject {
	title := widget.NewLabel("PeerCodeX")
	intro := widget.NewLabel("PeerCodeX is a demonstration system for the application of network coding in P2P systems.")
	intro.Wrapping = fyne.TextWrapWord

	return container.NewBorder(
		container.NewVBox(title, widget.NewSeparator(), intro), nil, nil, nil, nil)
}

func makeBlankContent() fyne.CanvasObject {
	title := widget.NewLabel("left to be realized")
	intro := widget.NewLabel("The current page code has not been written yet.")
	intro.Wrapping = fyne.TextWrapWord

	return container.NewBorder(
		container.NewVBox(title, widget.NewSeparator(), intro), nil, nil, nil, nil)
}
