// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	_ "github.com/emer/etable/etview" // include to get gui views
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
)

// this is the stub main for gogi that calls our actual
// mainrun function, at end of file
func main() {
	gimain.Main(func() {
		mainrun()
	})
}

type Gen struct {
	Dir        string
	Files      []string
	StructView *giv.StructView `view:"-" desc:"the params viewer"`
}

func NewGen() *Gen {
	g := Gen{}
	g.Dir = "/Users/rohrlich/gnuspeech_sa-master/generated/CV_III_NoCoAr/individualCVs"
	return &g
}

// Reset clears the seqs and words slices
func (gn *Gen) Reset() {
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// LoadTranscripts
func (gn *Gen) LoadFileNames() {
	gn.Files = gn.Files[:0]
	files, err := ioutil.ReadDir(gn.Dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		gn.Files = append(gn.Files, f.Name())
		fmt.Println(f.Name())
	}
}

func (gn *Gen) Rename() {
	for _, f := range gn.Files {
		if strings.HasSuffix(f, ".wav") == false {
			continue
		}
		// kludge to avoid processing .wav files already renamed
		if strings.Contains(f, "-") == false {
			continue
		}
		fo := strings.TrimSuffix(f, ".wav")
		ln := len(fo)
		pos := fo[ln-2:]
		i, err := strconv.Atoi(pos)
		if err != nil {
			fmt.Println("atoi error")
		}
		s := Chunks(fo, 2)
		fn := s[i-1] // audacity labels start at 1 not zero
		fmt.Println(fn)

		cmd := exec.Command("mv", gn.Dir+"/"+fo+".wav", gn.Dir+"/"+fn+"_"+fo+".wav")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

	}
}

func Chunks(s string, chunkSize int) []string {
	if chunkSize >= len(s) {
		return []string{s}
	}
	var chunks []string
	chunk := make([]rune, chunkSize)
	len := 0
	for _, r := range s {
		chunk[len] = r
		len++
		if len == chunkSize {
			chunks = append(chunks, string(chunk))
			len = 0
		}
	}
	if len > 0 {
		chunks = append(chunks, string(chunk[:len]))
	}
	return chunks
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Gui

// ConfigGui configures the GoGi gui interface for this Aud
func (gn *Gen) ConfigGui() *gi.Window {
	width := 1600
	height := 1200

	gi.SetAppName("Gen")
	gi.SetAppAbout(`Gen concatenated strings of syllables`)

	win := gi.NewMainWindow("one", "Gen ...", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := gi.AddNewToolBar(mfr, "tbar")
	tbar.SetStretchMaxWidth()
	// vi.ToolBar = tbar

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = gi.X
	split.SetStretchMaxWidth()
	split.SetStretchMaxHeight()

	sv := giv.AddNewStructView(split, "sv")
	sv.SetStruct(gn)
	gn.StructView = sv

	// tv := gi.AddNewTabView(split, "tv")

	tbar.AddAction(gi.ActOpts{Label: "Reset", Icon: "new", Tooltip: ""}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.Reset()
		})

	tbar.AddAction(gi.ActOpts{Label: "Load File Names", Icon: "new", Tooltip: "Read the names of all the files in the directory"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.LoadFileNames()
		})

	tbar.AddAction(gi.ActOpts{Label: "Rename", Icon: "new", Tooltip: ""}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.Rename()
		})

	vp.UpdateEndNoSig(updt)

	// main menu
	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu.AddCopyCutPaste(win)

	vp.UpdateEndNoSig(updt)

	win.MainMenuUpdated()
	return win
}

func mainrun() {
	Gen := NewGen()
	win := Gen.ConfigGui()
	win.StartEventLoop()
}
