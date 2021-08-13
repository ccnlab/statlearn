// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/emer/etable/etable"   // include to get gui views
	"github.com/emer/etable/etensor"  // include to get gui views
	_ "github.com/emer/etable/etview" // include to get gui views
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"os"
)

// this is the stub main for gogi that calls our actual
// mainrun function, at end of file
func main() {
	gimain.Main(func() {
		mainrun()
	})
}

type Gen struct {
	StructView *giv.StructView `view:"-" desc:"the params viewer"`
	Params     *etable.Table
	Rate       []float64
	Tempo      []float64
	Intonation bool
}

func NewGen() *Gen {
	g := Gen{}

	g.Params = &etable.Table{}
	g.Rate = []float64{215.0, 225.0, 235.0}
	g.Tempo = []float64{1.15, 1.25, 1.35}
	g.Intonation = true

	g.Params = &etable.Table{}
	sch := etable.Schema{
		{"rate", etensor.FLOAT64, nil, nil},
		{"tempo", etensor.FLOAT64, nil, nil},
	}
	g.Params.SetFromSchema(sch, 1)

	return &g
}

// Reset clears the seqs and words slices
func (gn *Gen) Reset() {

}

// GenSeqs generates the sequences one at a time call GenSeq
func (gn *Gen) GenControls() {
	r := 0
	for _, rt := range gn.Rate {
		for _, tempo := range gn.Tempo {
			gn.Params.SetCellFloat("rate", r, rt)
			gn.Params.SetCellFloat("tempo", r, tempo)
			gn.WriteControl(r)
			r++
			gn.Params.AddRows(1)
		}
	}
}

// WriteConfig writes one config file
func (gn *Gen) WriteControl(row int) {
	rt := gn.Params.CellString("rate", row)
	tempo := gn.Params.CellString("tempo", row)

	s := ""
	s += "control_rate = " + rt + "\n"
	s += "voice_name = config\n"
	s += "tempo = " + tempo + "\n"
	s += "pitch_offset = -4.0\n"
	s += "drift_deviation = 0.5\n"
	s += "drift_lowpass_cutoff = 0.5\n"
	if gn.Intonation == false {
		s += "micro_intonation = 0.0\n"
		s += "macro_intonation = 0.0\n"
		s += "intonation_drift = 0.0\n"
		s += "random_intonation = 0.0\n"
	} else {
		s += "micro_intonation = 1.0\n"
		s += "macro_intonation = 1.0\n"
		s += "intonation_drift = 1.0\n"
		s += "random_intonation = 1.0\n"
	}

	s += "notional_pitch = 2.0\n"
	s += "pretonic_range = -2.0\n"
	s += "pretonic_lift = 4.0\n"
	s += "tonic_range = -8.0\n"
	s += "tonic_movement = 4.0\n"

	s += "dictionary_1_file = none\n"
	s += "dictionary_2_file = none\n"
	s += "dictionary_3_file = MainDictionaryModified\n"

	paramStr := "_rt_" + rt + "_tmpo_" + tempo
	fn := "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/controls/control" + paramStr
	f, err := os.Create(fn)
	check(err)
	f.Write([]byte(s))
}

// LoadParams
func (gn *Gen) LoadParams() {
}

func check(e error) {
	if e != nil {
		panic(e)
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

	tbar.AddAction(gi.ActOpts{Label: "Load Params", Icon: "new", Tooltip: ""}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.LoadParams()
		})

	tbar.AddAction(gi.ActOpts{Label: "Gen Control Configs", Icon: "new", Tooltip: "Generate all combinations of control configurations"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.GenControls()
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
