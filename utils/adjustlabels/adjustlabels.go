// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	_ "github.com/emer/etable/etview" // include to get gui views
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

// this is the stub main for gogi that calls our actual
// mainrun function, at end of file
func main() {
	gimain.Main(func() {
		mainrun()
	})
}

// CVTime
type CVTime struct {
	Name  string  `desc:"the CV, da, go, ku, etc"`
	Start float64 `desc:"start time of this CV in a particular sequence in milliseconds"`
	End   float64 `desc:"end time of this CV in a particular sequence in milliseconds"`
}

type Proc struct {
	Factor     float64         `desc:"percentage change in tempo -- negative for slowing the sound, positive to speed up"`
	LabelsDir  string          `desc:"path to existing label files"`
	LabelFiles []string        `view:"no-inline" desc:"slice of filenames"`
	OutDir     string          `view:"no-inline" desc:"directory of where to write files"`
	StructView *giv.StructView `view:"-" desc:"the params viewer"`
}

func NewProc() *Proc {
	p := Proc{}
	p.Factor = 13
	p.OutDir = "/Users/rohrlich/ccn_images/word_seg_snd_files/labelFiles/"
	p.LabelsDir = "/Users/rohrlich/ccn_images/word_seg_snd_files/CV_FA_Times"
	return &p
}

// ReadNames
func (p *Proc) ReadNames() {
	files, err := ioutil.ReadDir(p.LabelsDir)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if f.Name()[0] != '.' {
			p.LabelFiles = append(p.LabelFiles, f.Name())
		}
	}
}

// AdjustTimes multiplies the times to match tempo changes done with Audacity
func (p *Proc) AdjustTimes() {
	for _, fn := range p.LabelFiles {
		cvTimes := p.AdjustTime(fn)
		p.Write(cvTimes, fn)
	}
}

// AdjustTimes multiplies the times to match tempo changes done with Audacity
func (p *Proc) AdjustTime(fn string) []CVTime {
	// load the CV start/end times produced by Audacity "sound finder"
	fp, err := os.Open(p.LabelsDir + "/" + fn)
	if err != nil {
		log.Println(err)
	}
	defer fp.Close() // we will be done with the file within this function

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)

	var cvTimes []CVTime
	i := 0
	for scanner.Scan() {
		cvt := new(CVTime)
		cvTimes = append(cvTimes, *cvt)
		t := scanner.Text()
		cvs := strings.Fields(t)
		f, err := strconv.ParseFloat(cvs[0], 64)
		if err == nil {
			cvTimes[i].Start = f * (1 - p.Factor/100)
		}
		f, err = strconv.ParseFloat(cvs[1], 64)
		if err == nil {
			cvTimes[i].End = f * (1 - p.Factor/100)
		}
		cvTimes[i].Name = strconv.Itoa(i)
		i++
	}
	return cvTimes
}

// Write
func (p *Proc) Write(cvTimes []CVTime, name string) {
	s := ""
	for _, cvt := range cvTimes {
		cs := fmt.Sprintf("%.2f\t%.2f\t%s\n", cvt.Start, cvt.End, cvt.Name)
		s += cs
	}
	fn := p.OutDir + "/" + name
	fn = strings.Replace(fn, "//", "/", 1) // user may have put slash at end of path or maybe not
	f, err := os.Create(fn)
	check(err)
	f.Write([]byte(s))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Gui

// ConfigGui configures the GoGi gui interface for this Aud
func (p *Proc) ConfigGui() *gi.Window {
	width := 1600
	height := 1200

	gi.SetAppName("Adjust Label Times")
	gi.SetAppAbout(`Adjust Label Times `)

	win := gi.NewMainWindow("one", "Proc ...", width, height)

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
	sv.SetStruct(p)
	p.StructView = sv

	tbar.AddAction(gi.ActOpts{Label: "Read In File Names", Icon: "new", Tooltip: ""}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			p.ReadNames()
		})

	tbar.AddAction(gi.ActOpts{Label: "Adjust Times", Icon: "new", Tooltip: ""}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			p.AdjustTimes()
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
	Proc := NewProc()
	win := Proc.ConfigGui()
	win.StartEventLoop()
}
