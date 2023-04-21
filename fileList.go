package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"strconv"
	"strings"

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
		pieceLengthWidget := widget.NewSelect(
			[]string{"4 KB", "8 KB", "16 KB", "32 KB", "64 KB", "128 KB", "256 KB", "512 KB", "1024 KB"},
			func(s string) {},
		)
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
			widget.NewFormItem("Piece Length", pieceLengthWidget),
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
			pieceLength, _ := strconv.Atoi(strings.Split(pieceLengthWidget.Selected, " ")[0])
			err := seed.CreateSeedFile(filePath, pieceLength, comment, announce, announceList)
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
			err = dc.AddFileItem(reader.URI().Path())
			if err != nil {
				dialog.ShowError(err, topWindow)
				return
			}
			refresh()
		}, topWindow)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".nc"}))
		fd.Show()
	}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() { fmt.Println("Refresh") }),
	)
}

func makeFileListItem(parent *widget.List) *fyne.Container {
	pathLabel := widget.NewLabel("File Name")
	processBar := container.NewVBox(widget.NewProgressBarInfinite())
	processBar.Hide()
	var InfoAction, DownloadAction, DeleteAction *widget.ToolbarAction
	InfoAction = widget.NewToolbarAction(theme.InfoIcon(), func() {
		log.Println("Info of ", pathLabel.Text)
		f := dc.GetFileItemByPath(pathLabel.Text)
		if f == nil {
			dialog.ShowError(errors.New("file not found"), topWindow)
		}
		items := []*widget.FormItem{
			widget.NewFormItem("Name", widget.NewLabel(filepath.Base(f.Path))),
			widget.NewFormItem("Path", widget.NewLabel(f.Path)),
			widget.NewFormItem("SHA1 Hash", widget.NewLabel(fmt.Sprintf("%x", f.NcFile.Info.Hash))),
			widget.NewFormItem("Comment", widget.NewLabel(f.NcFile.Comment)),
			widget.NewFormItem("Creation Date", widget.NewLabel(f.NcFile.CreationDate.String())),
			widget.NewFormItem("Announce", widget.NewLabel(f.NcFile.Announce)),
			widget.NewFormItem("Announce List", widget.NewLabel(strings.Join(f.NcFile.AnnounceList, "\n"))),
			widget.NewFormItem("Piece Length", widget.NewLabel(tools.FormatByteSize(f.NcFile.Info.PieceLength))),
			widget.NewFormItem("Length", widget.NewLabel(tools.FormatByteSize(f.NcFile.Info.Length))),
		}
		form := &widget.Form{Items: items}
		formDialog := dialog.NewCustom("File Info", "OK", form, topWindow)
		formDialog.Show()
	})
	DownloadAction = widget.NewToolbarAction(data.DownloadOff, func() {
		log.Println("download")
		f := dc.GetFileItemByPath(pathLabel.Text)
		if f == nil {
			dialog.ShowError(errors.New("file not found"), topWindow)
		}

		if f.IsDownloaded {
			processBar.Hide()
			return
		}

		if f.IsDownloading {
			f.IsDownloading = false
			// close all connections
			for _, c := range f.Conns {
				c.Close()
			}
			// clear f.Conns
			f.Conns = []net.Conn{}
			for _, node := range f.Nodes {
				node.HaveClient = false
			}
			processBar.Hide()
			parent.Refresh()
			return
		}

		f.IsDownloading = true
		f.StartReceiveCodedPiece()
		for _, node := range f.Nodes {
			if node.IsOn == true && node.HaveClient == false {
				// start a new client
				c := client.NewClient(node.Addr, f.NcFile.Info.Hash, dc.GetPort(), f.AddCodedPieceChan)
				connChan := make(chan net.Conn)
				go c.Start(connChan)
				if f.Conns == nil {
					f.Conns = make([]net.Conn, 0)
				}
				c.Conn = <-connChan
				f.Conns = append(f.Conns, c.Conn)
				node.HaveClient = true
			}
		}
		var refresh = func() {
			processBar.Hide()
			parent.Refresh()
		}
		f.HideRefresh = refresh

		processBar.Show()
		parent.Refresh()
	})
	DeleteAction = widget.NewToolbarAction(theme.DeleteIcon(), func() {
		log.Println("Delete")
		f := dc.GetFileItemByPath(pathLabel.Text)
		if f == nil {
			dialog.ShowError(errors.New("file not found"), topWindow)
		}
		if f.IsDownloading {
			// close all connections
			for _, c := range f.Conns {
				c.Close()
			}
			// clear f.Conns
			f.Conns = []net.Conn{}
			for _, node := range f.Nodes {
				node.HaveClient = false
			}
		}

		dc.DeleteFileItemByPath(pathLabel.Text)
		parent.Refresh()
	})

	return container.NewBorder(
		nil,
		processBar,
		container.NewHBox(widget.NewIcon(theme.DocumentIcon()), pathLabel, widget.NewLabel("(Downloaded)")),
		container.NewHBox(widget.NewToolbar(
			widget.NewToolbarSpacer(),
			InfoAction,
			DownloadAction,
			DeleteAction,
		)),
	)
}
