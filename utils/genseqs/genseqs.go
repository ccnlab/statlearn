// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"

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
	Seqs        []string        `view:"no-inline" desc:"the generated word sequences"`
	Words       []string        `view:"no-inline" desc:"the words read from file from which to make the sequences"`
	WordFile    string          `view:"no-inline" desc:"the transcript file or other file to process"`
	DirOut      string          `view:"no-inline" desc:"directory of where to write files"`
	Prefix      string          `view:"no-inline" desc:"fixed part of file name before id"`
	StructView  *giv.StructView `view:"-" desc:"the params viewer"`
	NSeqs       int             `desc:"the number of sequences to create"`
	NWords      int             `desc:"the number of words to concatenate"`
	NWordGroups int             `desc:"the number of times to repeat concatenate a NWords"`
	Separator   string          `view:"no-inline" desc:"between words string"`
}

func NewGen() *Gen {
	g := Gen{}
	g.NSeqs = 24
	g.NWords = 12
	g.NWordGroups = 4

	g.WordFile = "/Users/rohrlich/gnuspeech_sa-master/generated/trisyllabicsITest"
	g.DirOut = "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/seqs/"
	g.Prefix = "001"
	return &g
}

// Reset clears the seqs and words slices
func (gn *Gen) Reset() {
	gn.Seqs = gn.Seqs[:0]
	gn.Words = gn.Words[:0]

}

// GenSeqs generates the sequences one at a time call GenSeq
func (gn *Gen) GenSeqs() {
	for i := 0; i < gn.NSeqs; i++ {
		seq := gn.GenSeq()
		if len(seq) != 0 {
			gn.Seqs = append(gn.Seqs, seq)
		}
	}
}

// GenSeq concatenates words randomly but never concatenating a string twice in succession
func (gn *Gen) GenSeq() string {
	catstr := ""
	prv := -1
	cnt := 0
	for cnt < gn.NWords {
		idx := rand.Intn(len(gn.Words))
		if idx != prv {
			catstr += gn.Words[idx]
			if cnt < gn.NWords-1 {
				catstr += gn.Separator
			}
			cnt++
			prv = idx
		}
	}
	catstr += "\n"
	return catstr
}

// GenSeqsII generates the sequences one at a time call GenSeqII
func (gn *Gen) GenSeqsII() {
	for i := 0; i < gn.NSeqs; i++ {
		seq := gn.GenSeqII()
		if len(seq) != 0 {
			gn.Seqs = append(gn.Seqs, seq)
		}
	}
}

// GenSeq concatenates words randomly within word list and never concatenates a string twice in succession
func (gn *Gen) GenSeqII() string {
	if len(gn.Words) == 0 {
		fmt.Println("gn.Words is empty, load words first")
		return ""
	}
	catstr := ""
	prv := ""

	for i := 0; i < gn.NWordGroups; i++ {
		gn.ShuffleWords()
		cnt := 0
		for {
			if prv != gn.Words[0] {
				for cnt < len(gn.Words) {
					catstr += gn.Words[cnt]
					if cnt < gn.NWords-1 {
						catstr += gn.Separator
					}
					cnt++
				}
				prv = gn.Words[len(gn.Words)-1] // last word is new previous
				break
			} else {
				gn.ShuffleWords()
			}
		}
		catstr += gn.Separator
	}
	catstr += "\n"
	return catstr
}

// GenSeq concatenates words randomly in groups but never concatenating a string twice in succession
func (gn *Gen) ShuffleWords() {
	rand.Shuffle(len(gn.Words), func(i, j int) { gn.Words[i], gn.Words[j] = gn.Words[j], gn.Words[i] })
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// WriteSeqs writes each sequence to a separate file
func (gn *Gen) WriteSeqs() {
	for i, s := range gn.Seqs {
		fn := gn.Prefix
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
		fn = gn.DirOut + fn + id
		f, err := os.Create(fn)
		check(err)
		f.Write([]byte(s))
	}
}

// LoadWords
func (gn *Gen) LoadWords() {
	fp, err := os.Open(gn.WordFile)
	if err != nil {
		log.Println(err)
		return
	}
	defer fp.Close() // we will be done with the file within this function

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)
	// clear first in case it is called twice or the file changes
	if gn.Words != nil {
		gn.Words = gn.Words[:0]
	}
	for scanner.Scan() {
		gn.Words = append(gn.Words, scanner.Text())
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

	tbar.AddAction(gi.ActOpts{Label: "Load Words", Icon: "new", Tooltip: ""}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.LoadWords()
		})

	tbar.AddAction(gi.ActOpts{Label: "Shuffle Words", Icon: "new", Tooltip: ""}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.ShuffleWords()
		})

	tbar.AddAction(gi.ActOpts{Label: "Gen Seqs", Icon: "new", Tooltip: "Generate N sequences of M words"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.GenSeqs()
		})

	tbar.AddAction(gi.ActOpts{Label: "GenSeqII", Icon: "new", Tooltip: "Generate N sequences of M words"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.GenSeqII()
		})

	tbar.AddAction(gi.ActOpts{Label: "Gen SeqsII", Icon: "new", Tooltip: "Generate N sequences of M words"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.GenSeqsII()
		})

	tbar.AddAction(gi.ActOpts{Label: "Write Seqs", Icon: "new", Tooltip: "write each sequence to a file"}, win.This(),
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
