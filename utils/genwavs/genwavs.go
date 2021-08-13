// Copyright (c) 2020, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This code generates wav files by calling gnuspeech
package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"math/rand"
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
	StructView   *giv.StructView `view:"-" desc:"the params viewer"`
	VoicesPath   string          `desc:"voice files are the voice config files that gnu_speech uses"`
	ControlsPath string          `desc:"control files are the trm_control_model config files that gnu_speech uses"`
	SeqsPath     string          `desc:"seqs are the cv sequences, pa bi ku go etc"`
	WavsPath     string          `desc:"where to write the wavs file to"`
	VoiceFiles   []string        `view:"no-inline"`
	ControlFiles []string        `view:"no-inline"`
	SeqFiles     []string        `view:"no-inline"`
	WavFiles     []string        `view:"no-inline"`
	NWavs        int             `desc:"the number of wav files to generate"`
}

func NewGen() *Gen {
	g := Gen{}

	g.VoicesPath = "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/voices"
	g.ControlsPath = "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/controls"
	g.SeqsPath = "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/seqs"
	g.WavsPath = "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/wavs/"
	g.NWavs = 110

	cfiles, cerr := ioutil.ReadDir(g.ControlsPath)
	if cerr != nil {
		log.Fatal(cerr)
	}
	for _, f := range cfiles {
		g.ControlFiles = append(g.ControlFiles, f.Name())
	}

	vfiles, verr := ioutil.ReadDir(g.VoicesPath)
	if verr != nil {
		log.Fatal(verr)
	}
	for _, f := range vfiles {
		g.VoiceFiles = append(g.VoiceFiles, f.Name())
	}

	sfiles, serr := ioutil.ReadDir(g.SeqsPath)
	if serr != nil {
		log.Fatal(serr)
	}
	for _, f := range sfiles {
		nm := f.Name()
		if nm[0] != '.' {
			g.SeqFiles = append(g.SeqFiles, f.Name())
		}
	}

	return &g
}

// Reset clears the seqs and words slices
func (gn *Gen) Reset() {

}

// GenWavs generates wav files using a voice file and a control file previously generated
func (gn *Gen) GenWavs() {
	arg1 := "-c"
	arg2 := "/Users/rohrlich/gnuspeech_sa-master/data/en" // path to the configuration file, dictionary etc
	arg3 := "-p"
	arg4 := "trm_param_file.txt"
	arg5 := "-o"
	seqtxt := ""

	for idx := 0; idx < gn.NWavs; idx++ {
		i := rand.Intn(len(gn.VoiceFiles))
		fvoice := gn.VoiceFiles[i]
		vs := strings.Replace(fvoice, "voice_", "", 1)

		j := rand.Intn(len(gn.ControlFiles))
		fcontrol := gn.ControlFiles[j]
		cs := strings.Replace(fcontrol, "control_", "", 1)

		k := rand.Intn(len(gn.SeqFiles))
		ks := gn.SeqFiles[k]
		fseq := "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/seqs/" + gn.SeqFiles[k]

		fp, err := os.Open(fseq)
		if err != nil {
			log.Fatal(err)
		}
		defer fp.Close() // we will be done with the file within this function
		scanner := bufio.NewScanner(fp)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			seqtxt = scanner.Text()
		}

		// the gnuspeech software expects a configuration file of specific name - easy just to copy file to that name
		fc := "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/controls/" + fcontrol
		copyCmd := exec.Command("cp", fc, "/Users/rohrlich/gnuspeech_sa-master/data/en/trm_control_model.config")
		copyCmd.Stdout = os.Stdout
		copyCmd.Stderr = os.Stderr
		err = copyCmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		// the control file is expecting a voice configuration file of specific name - easy just to copy file to that name
		fv := "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/voices/" + fvoice
		copyCmd = exec.Command("cp", fv, "/Users/rohrlich/gnuspeech_sa-master/data/en/voice_config.config")
		copyCmd.Stdout = os.Stdout
		copyCmd.Stderr = os.Stderr
		err = copyCmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		fout := gn.WavsPath + ks + "_" + cs + "_" + vs + ".wav"
		cmd := exec.Command("/Users/rohrlich/gnuspeech_sa-master/build/gnuspeech_sa", arg1, arg2, arg3, arg4, arg5, fout, seqtxt)
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// SplitWavs
func (gn *Gen) SplitWavs() {
	files, err := ioutil.ReadDir(gn.WavsPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		gn.WavFiles = append(gn.WavFiles, f.Name())
	}
	rand.Shuffle(len(gn.WavFiles), func(i, j int) { gn.WavFiles[i], gn.WavFiles[j] = gn.WavFiles[j], gn.WavFiles[i] })

	for i, f := range files {
		fw := "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/wavs/" + f.Name()
		splt := i % 3
		n := strconv.Itoa(splt)
		ss := ""
		if splt == 0 {
			ss = "_sil_20"
		} else if splt == 1 {
			ss = "_sil_25"
		} else {
			ss = "_sil_30"
		}
		ext := ".wav"
		nm := strings.Replace(f.Name(), ext, ss+ext, 1)
		fw2 := "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/" + "wavs" + n + "/" + nm
		cpCmd := exec.Command("mv", fw, fw2)
		cpCmd.Stdout = os.Stdout
		cpCmd.Stderr = os.Stderr
		err = cpCmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
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

	tbar.AddAction(gi.ActOpts{Label: "Gen Wavs", Icon: "new", Tooltip: "Generate the .wav files"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.GenWavs()
		})

	tbar.AddAction(gi.ActOpts{Label: "Split Wavs", Icon: "new", Tooltip: ""}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.SplitWavs()
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
