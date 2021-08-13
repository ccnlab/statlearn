// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math/rand"

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
	syls1 []string
	n     int `view:"no-inline" desc:"the number of trisyllables to concatenate"`

	StructView *giv.StructView `view:"-" desc:"the params viewer"`
}

func NewGen() *Gen {
	g := Gen{}
	g.syls1 = []string{"pabiku", "tibudo", "golatu", "daropi"}
	g.n = 120
	return &g
}

// CatNoRepeat concatenates strings randomly but never concatenating a string twice in succession
func (gn *Gen) CatNoRepeat(strs []string) string {
	catstr := ""
	prv := -1
	cnt := 0
	for cnt < gn.n {
		idx := rand.Intn(len(strs))
		if idx != prv {
			catstr += strs[idx]
			cnt++
			prv = idx
		}
	}
	return catstr
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

	tbar.AddAction(gi.ActOpts{Label: "Gen cat string", Icon: "new", Tooltip: "Generate a new initial random seed to get different results.  By default, Init re-establishes the same initial seed every time."}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.CatNoRepeat(gn.syls1)
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
	catstr := Gen.CatNoRepeat(Gen.syls1)
	fmt.Println(catstr)
	win.StartEventLoop()
}
