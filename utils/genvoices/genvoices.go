// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
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
	Params     *etable.Table   `view:"no-inline" desc:"training trial-level log data"`
	StructView *giv.StructView `view:"-" desc:"the params viewer"`
	vtlVals    []float64
	gpminmax   []float64
	rgp        []float64
	br         []float64
}

func NewGen() *Gen {
	g := Gen{}

	g.vtlVals = []float64{15.0, 16.25, 17.5}
	g.gpminmax = []float64{24.0, 27.0, 30.0}
	g.rgp = []float64{-12.0, -6.0, 0}
	g.br = []float64{0.5, 1.0, 1.5}

	g.Params = &etable.Table{}
	sch := etable.Schema{
		{"vtl", etensor.FLOAT64, nil, nil},
		{"gpminmax", etensor.FLOAT64, nil, nil},
		{"rgp", etensor.FLOAT64, nil, nil},
		{"br", etensor.FLOAT64, nil, nil},
	}
	g.Params.SetFromSchema(sch, 1)
	return &g
}

// Reset clears the seqs and words slices
func (gn *Gen) Reset() {

}

// GenSeqs generates the sequences one at a time call GenSeq
func (gn *Gen) GenVoices() {
	r := 0
	for _, vtl := range gn.vtlVals {
		for _, minmax := range gn.gpminmax {
			for _, rgp := range gn.rgp {
				for _, br := range gn.br {
					gn.Params.SetCellFloat("vtl", r, vtl)
					gn.Params.SetCellFloat("gpminmax", r, minmax)
					gn.Params.SetCellFloat("rgp", r, rgp)
					gn.Params.SetCellFloat("br", r, br)
					gn.WriteVoice(r)
					r++
					gn.Params.AddRows(1)
				}
			}
		}
	}
}

// WriteConfig writes one config file
func (gn *Gen) WriteVoice(row int) {
	vtl := gn.Params.CellString("vtl", row)
	gpminmax := gn.Params.CellString("gpminmax", row)
	rgp := gn.Params.CellString("rgp", row)
	br := gn.Params.CellString("br", row)

	s := ""
	s += "vocal_tract_length = " + vtl + "\n"
	s += "glottal_pulse_tp = 40.0\n"
	s += "glottal_pulse_tn_min = " + gpminmax + "\n"
	s += "glottal_pulse_tn_max = " + gpminmax + "\n"
	s += "reference_glottal_pitch = " + rgp + "\n"
	s += "breathiness = " + br + "\n"
	s += "aperture_radius = 3.05\n"
	s += "nose_radius_1 = 1.35\n"
	s += "nose_radius_2 = 1.96\n"
	s += "nose_radius_3 = 1.91\n"
	s += "nose_radius_4 = 1.3\n"
	s += "nose_radius_5 = 0.73\n"
	s += "global_nose_radius_coef = 1.0\n"
	s += "radius_1_coef = 1.0\n"
	s += "radius_2_coef = 1.0\n"
	s += "radius_3_coef = 1.0\n"
	s += "radius_4_coef = 1.0\n"
	s += "radius_5_coef = 1.0\n"
	s += "radius_6_coef = 1.0\n"
	s += "radius_7_coef = 1.0\n"
	s += "radius_8_coef = 1.0\n"
	s += "global_radius_coef = 1.0\n"

	paramStr := "_vtl_" + vtl + "_gp_" + gpminmax + "_rgp_" + rgp + "_br_" + br
	fn := "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/voices/voice" + paramStr
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

	tbar.AddAction(gi.ActOpts{Label: "Gen Voice Configs", Icon: "new", Tooltip: "Generate all combinations of voice configurations"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.GenVoices()
		})

	//tbar.AddAction(gi.ActOpts{Label: "Write Configs", Icon: "new", Tooltip: "write each sequence to a file"}, win.This(),
	//	func(recv, send ki.Ki, sig int64, data interface{}) {
	//		gn.WriteConfig()
	//	})

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
