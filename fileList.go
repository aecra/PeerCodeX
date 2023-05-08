package main

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/aecra/PeerCodeX/client"
	"github.com/aecra/PeerCodeX/data"
	"github.com/aecra/PeerCodeX/dc"
	"github.com/aecra/PeerCodeX/seed"
	"github.com/aecra/PeerCodeX/tools"
)

func makeNewSeedButton() *widget.Button {
	return widget.NewButton("Create New Seed File", func() {
		log.Println("Create New Seed File")

		filePath := ""
		commentWidget := widget.NewEntry()
		announceWidget := widget.NewEntry()
		announceListWidget := widget.NewMultiLineEntry()
		items := []*widget.FormItem{
			widget.NewFormItem("File", widget.NewButton("Select File", func() {
				fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
					if err != nil {
						dialog.ShowError(err, topWindow)
						return
					}
					if reader == nil {
						log.Println("Cancelled")
						return
					}

					log.Println(reader.URI().Path())
					filePath = reader.URI().Path()
				}, topWindow)

				fd.Show()
			})),
			widget.NewFormItem("Comment", commentWidget),
			widget.NewFormItem("Announce", announceWidget),
			widget.NewFormItem("Announce List", announceListWidget),
		}
		formDialog := dialog.NewForm("Create New Seed File", "Create", "Cancel", items, func(b bool) {
			if !b {
				return
			}

			comment := commentWidget.Text
			announce := announceWidget.Text
			announceList := announceListWidget.Text
			{
				v1 := strings.Split(announceList, "\n")
				for i, v := range v1 {
					v1[i] = strings.TrimSpace(v)
				}
				announceList = strings.Join(v1, ",")
			}
			openLoadingMask()
			err := seed.CreateSeedFile(filePath, comment, announce, announceList)
			closeLoadingMask()
			if err != nil {
				dialog.ShowError(err, topWindow)
				return
			} else {
				dialog.ShowInformation("Create New Seed File", "Create New Seed File Success", topWindow)
			}
		}, topWindow)
		formDialog.Resize(fyne.NewSize(500, 350))
		formDialog.Show()
	})
}

func makeFileListToolbar(refresh func()) *widget.Toolbar {
	return widget.NewToolbar(widget.NewToolbarAction(theme.ContentAddIcon(), func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, topWindow)
				return
			}
			if reader == nil {
				log.Println("Cancelled")
				return
			}

			// add file
			log.Println(reader.URI().Path())
			openLoadingMask()
			err = dc.AddFile(reader.URI().Path())
			if err != nil {
				dialog.ShowError(err, topWindow)
				return
			}
			closeLoadingMask()
			refresh()
		}, topWindow)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".nc"}))
		fd.Show()
	}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() { log.Println("Refresh") }),
	)
}

var refreshGoroutine = make(map[string]struct{})
var downloadActive = sync.Mutex{}

func makeFileListItem(parent *widget.List) *fyne.Container {
	pathLabel := widget.NewLabel("File Name")
	progressBar := widget.NewProgressBar()
	progressBar.SetValue(0)
	var InfoAction, DownloadAction, DeleteAction *widget.ToolbarAction
	InfoAction = widget.NewToolbarAction(theme.InfoIcon(), func() {
		log.Println("Info of ", pathLabel.Text)
		f := dc.GetFileByPath(pathLabel.Text)
		if f == nil {
			dialog.ShowError(errors.New("file not found"), topWindow)
		}
		hashs := ""
		for _, v := range f.NcFile.Info.Hash {
			hashs += fmt.Sprintf("%x\n", v)
		}
		items := []*widget.FormItem{
			widget.NewFormItem("Name", widget.NewLabel(filepath.Base(f.Path))),
			widget.NewFormItem("Path", widget.NewLabel(f.Path)),
			widget.NewFormItem("SHA1 Hash", widget.NewLabel(hashs)),
			widget.NewFormItem("Comment", widget.NewLabel(f.NcFile.Comment)),
			widget.NewFormItem("Creation Date", widget.NewLabel(f.NcFile.CreationDate.String())),
			widget.NewFormItem("Announce", widget.NewLabel(f.NcFile.Announce)),
			widget.NewFormItem("Announce List", widget.NewLabel(strings.Join(f.NcFile.AnnounceList, "\n"))),
			widget.NewFormItem("Length", widget.NewLabel(tools.FormatByteSize(f.NcFile.Info.Length))),
		}
		form := &widget.Form{Items: items}
		formDialog := dialog.NewCustom("File Info", "OK", form, topWindow)
		formDialog.Show()
	})
	DownloadAction = widget.NewToolbarAction(data.DownloadOff, func() {
		downloadActive.Lock()
		defer downloadActive.Unlock()

		file := dc.GetFileByPath(pathLabel.Text)
		if file == nil {
			dialog.ShowError(errors.New("file not found"), topWindow)
		}

		if file.GetProcessRate() == 1 {
			progressBar.SetValue(1)
			return
		}

		// use a goroutine to watch the download status
		_, ok := refreshGoroutine[pathLabel.Text]
		if !ok {
			refreshGoroutine[pathLabel.Text] = struct{}{}
			go func() {
				for {
					_, ok := refreshGoroutine[pathLabel.Text]
					if !ok {
						break
					}

					progressBar.SetValue(file.GetProcessRate())
					if file.GetProcessRate() == 1 {
						break
					}
					time.Sleep(time.Second)
				}
			}()
		}

		if file.IsDownloading() {
			file.StopReceivingCodedPiece()
			parent.Refresh()
			return
		}

		client.RequestForFile(file)

		parent.Refresh()
	})
	DeleteAction = widget.NewToolbarAction(theme.DeleteIcon(), func() {
		log.Println("Delete")
		f := dc.GetFileByPath(pathLabel.Text)
		if f == nil {
			dialog.ShowError(errors.New("file not found"), topWindow)
		}

		f.StopReceivingCodedPiece()

		dc.DeleteFileByPath(pathLabel.Text)
		parent.Refresh()
	})

	return container.NewBorder(
		nil,
		container.NewVBox(progressBar),
		container.NewHBox(widget.NewIcon(theme.DocumentIcon()), pathLabel, widget.NewLabel("(Downloaded)")),
		container.NewHBox(widget.NewToolbar(
			widget.NewToolbarSpacer(),
			InfoAction,
			DownloadAction,
			DeleteAction,
		)),
	)
}
