// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
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
	TranscriptDir   string          `view:"no-inline" desc:"directory of transcripts"`
	TranscriptFile  string          `view:"no-inline" desc:"a single transcript file"`
	DirOut          string          `view:"no-inline" desc:"directory of where to write files"`
	TranscriptFiles []string        `view:"no-inline" desc:"list of transcript files in TranscriptDir"`
	Seqs            []string        `view:"no-inline" desc:"the generated word sequences"`
	NSeqs           int             `desc:"the number of sequences to create"`
	Separator       string          `view:"no-inline" desc:"between words string"`
	StructView      *giv.StructView `view:"-" desc:"the params viewer"`
}

func NewGen() *Gen {
	g := Gen{}
	g.TranscriptFile = "/Users/rohrlich/gnuspeech_sa-master/generated/temp.txt"
	g.TranscriptDir = "/Users/rohrlich/gnuspeech_sa-master/generated/ChildesNuffieldTranscripts"
	g.DirOut = "/Users/rohrlich/gnuspeech_sa-master/generated/ChildesNuffieldUtterances/"
	return &g
}

// Reset clears the seqs and words slices
func (gn *Gen) Reset() {
	gn.Seqs = gn.Seqs[:0]
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// WriteSeqs writes each sequence to a separate file
func (gn *Gen) WriteSeqs() {
	strs := strings.Split(gn.TranscriptFile, ".")
	file := strs[0]
	last := strings.LastIndex(file, "/")
	prefix := file[last+1 : len(file)]

	for i, s := range gn.Seqs {
		a := strconv.Itoa(i)
		id := ""
		if i > 9999 {
			id = "_" + a
		} else if i > 999 {
			id = "_0" + a
		} else if i > 99 {
			id = "_00" + a
		} else if i > 9 {
			id = "_000" + a
		} else {
			id = "_0000" + a
		}
		fn := gn.DirOut + prefix + id
		f, err := os.Create(fn)
		check(err)
		f.Write([]byte(s))
	}
}

// SplitTranscript
func (gn *Gen) SplitTranscript() {
	fp, err := os.Open(gn.TranscriptFile)
	if err != nil {
		log.Println(err)
	}
	defer fp.Close() // we will be done with the file within this function

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		txt := scanner.Text() + "\n"
		if strings.HasPrefix(txt, "*CDS:") {
			strs := strings.SplitAfter(txt, "*CDS:\t")
			if strings.HasPrefix(strs[1], "&") || strings.Contains(strs[1], "xxx") {
				continue
			}
			gn.Seqs = append(gn.Seqs, strs[1])
		}
	}
}

// LoadTranscripts
func (gn *Gen) LoadTranscripts() {
	files, err := ioutil.ReadDir(gn.TranscriptDir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		gn.TranscriptFiles = append(gn.TranscriptFiles, f.Name())
	}
}

// SplitTranscripts
func (gn *Gen) SplitTranscripts() {
	for _, f := range gn.TranscriptFiles {
		gn.TranscriptFile = gn.TranscriptDir + "/" + f
		gn.SplitTranscript()
		gn.WriteSeqs()
		gn.Seqs = gn.Seqs[:0]
	}
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

	tbar.AddAction(gi.ActOpts{Label: "Load Transcripts", Icon: "new", Tooltip: "Read the names of all the transcripts in the transcripts directory"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.LoadTranscripts()
		})

	tbar.AddAction(gi.ActOpts{Label: "Split Transcripts", Icon: "new", Tooltip: "Calls split transcript on each file in TranscriptFiles list"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.SplitTranscripts()
		})

	tbar.AddAction(gi.ActOpts{Label: "Split Transcript", Icon: "new", Tooltip: " Splits a single transcript"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.SplitTranscript()
		})

	tbar.AddAction(gi.ActOpts{Label: "Write Seqs", Icon: "new", Tooltip: "Write each utterance to file"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.WriteSeqs()
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
