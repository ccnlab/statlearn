// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"github.com/emer/etable/etable"   // include to get gui views
	_ "github.com/emer/etable/etview" // include to get gui views
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
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

type Gen struct {
	StructView *giv.StructView `view:"-" desc:"the params viewer"`
	Params     *etable.Table

	Syl1     []string `desc:"first position syllables (CVs)"`
	Syl2     []string `desc:"second position syllables (CVs)"`
	Syl3     []string `desc:"third position syllables (CVs)"`
	Syls123  []string `desc:"concatenated list of syl1, syl2, syl3 (CVs)"`
	SylsShuf []string `view:"no-inline" desc:"current shuffle of Syls123"`
	TriSyls  []string `view:"no-inline" desc:"the full list of generated trisyllables"`

	// these are for testing
	WholeWds []string `view:"no-inline" desc:"the list of all whole words"`
	PartWds  []string `view:"no-inline" desc:"the list of all part words (a part word is the last syllable of a whole word followed by the first 2 CVs of another word)"`

	ShufflesIn  string
	ShufflesOut string
	IndWavsDir  string `desc:"directory of wav files of individual syllables (CVs) (input dir)"`
	TriWavsDir  string `desc:"directory of wav files of concatenated tri-syllables (CVs) (output dir)"`
	SeqWavsDir  string `desc:"directory of the sequence wav files (i.e. the concatenated tri-syllables (CVs) (output dir)"`
}

func NewGen() *Gen {
	g := Gen{}

	// CV_III
	//g.Syl1 = []string{"su", "ro", "pa", "ho"} // whole words would be subahi, rolura, etc
	//g.Syl2 = []string{"ba", "lu", "go", "li"} // part words would be hirolu, hipago, rasuba, sapago, etc
	//g.Syl3 = []string{"hi", "ra", "di", "sa"}

	// CV_IV
	//g.Syl1 = []string{"do", "na", "hu", "ki"}
	//g.Syl2 = []string{"ka", "to", "mo", "mu"}
	//g.Syl3 = []string{"ru", "si", "ta", "po"}

	// CV_V
	//g.Syl1 = []string{"gu", "ma", "bi", "bu"}
	//g.Syl2 = []string{"ri", "gi", "tu", "ni"}
	//g.Syl3 = []string{"ha", "so", "ga", "bo"}

	// CV_VI
	g.Syl1 = []string{"da", "ti", "nu", "lo"}
	g.Syl2 = []string{"ku", "no", "pi", "du"}
	g.Syl3 = []string{"mi", "pu", "ko", "la"}

	g.Syls123 = append(g.Syls123, g.Syl1...)
	g.Syls123 = append(g.Syls123, g.Syl2...)
	g.Syls123 = append(g.Syls123, g.Syl3...)

	//g.IndWavsDir = "/Users/rohrlich/gnuspeech_sa-master/generated/CV_III_NoCoAr/individualCVsTrimmed"
	//g.TriWavsDir = "/Users/rohrlich/gnuspeech_sa-master/generated/CV_III_NoCoAr/triCVsTrimmed"
	//g.SeqWavsDir = "/Users/rohrlich/gnuspeech_sa-master/generated/CV_III_NoCoAr/seqsTrimmed"

	//g.IndWavsDir = "/Users/rohrlich/gnuspeech_sa-master/selfRecorded/individualCVs"
	//g.TriWavsDir = "/Users/rohrlich/gnuspeech_sa-master/selfRecorded/triCVs"
	//g.SeqWavsDir = "/Users/rohrlich/gnuspeech_sa-master/selfRecorded/seqs"

	g.IndWavsDir = "/Users/rohrlich/gnuspeech_sa-master/generated/CV_VI_NoCoAr/individualCVs"
	//g.TriWavsDir = "/Users/rohrlich/gnuspeech_sa-master/generated/temp/triCVs"
	g.TriWavsDir = "/Users/rohrlich/gnuspeech_sa-master/generated/temp/testWords"
	g.SeqWavsDir = "/Users/rohrlich/gnuspeech_sa-master/generated/temp/seqs"

	g.ShufflesIn = "/Users/rohrlich/ccnlab/lang-acq/cvShuffles"
	g.ShufflesOut = "/Users/rohrlich/gnuspeech_sa-master/generated/wavfiles"

	return &g
}

// GenTriSyllables generates all possible trisyllables adhering to a circular order
// i.e. 1 2 3, 2 3 1, 3 1 2
func (gn *Gen) GenTriSyllables() {
	s := ""
	for i := 0; i < 3; i++ {
		for _, syl1 := range gn.Syl1 {
			for _, syl2 := range gn.Syl2 {
				for _, syl3 := range gn.Syl3 {
					if i == 0 {
						s = syl3 + " " + syl1 + " " + syl2
					} else if i == 1 {
						s = syl1 + " " + syl2 + " " + syl3
					} else {
						s = syl2 + " " + syl3 + " " + syl1
					}
					gn.TriSyls = append(gn.TriSyls, s)
				}
			}
		}
	}
}

// ShuffledCVs generates a sequence of CVs in shuffled order
func (gn *Gen) ShuffleCVs() {
	rand.Shuffle(len(gn.Syls123), func(i, j int) { gn.Syls123[i], gn.Syls123[j] = gn.Syls123[j], gn.Syls123[i] })
	s := gn.Syls123[rand.Intn(len(gn.Syls123))] + " "
	for _, cv := range gn.Syls123 {
		s += cv + " "
	}
	s += gn.Syls123[rand.Intn(len(gn.Syls123))]
	gn.SylsShuf = append(gn.SylsShuf, s)
}

// LoadFileNamesFromFile is a utility to load a list of file names stored in a single text file
func (gn *Gen) LoadFileNamesFromFile(fn string) []string {
	filenames := []string{}

	fp, err := os.Open(fn)
	if err != nil {
		log.Println(err)
		return filenames
	}
	defer fp.Close() // we will be done with the file within this function

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		filenames = append(filenames, scanner.Text())
	}
	return filenames
}

// LoadFileNamesFromDir is a utility to create a list of file names found in specified directory.
// List can be limited to those with particular extension
func (gn *Gen) LoadFileNamesFromDir(dn string, ext string) []string {
	nms := []string{}
	files, err := ioutil.ReadDir(dn)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if len(ext) > 0 {
			if strings.HasSuffix(f.Name(), ext) == false {
				continue
			}
		}
		nms = append(nms, f.Name())
	}
	return nms
}

// Perm calls f with each permutation of a.
func Perm(a []string, f func([]string)) {
	perm(a, f, 0)
}

// perm the values at index i to len(a)-1.
func perm(a []string, f func([]string), i int) {
	if i > len(a) {
		f(a)
		return
	}
	perm(a, f, i+1)
	for j := i + 1; j < len(a); j++ {
		a[i], a[j] = a[j], a[i]
		perm(a, f, i+1)
		a[i], a[j] = a[j], a[i]
	}
}

// WriteTriSyllables
func (gn *Gen) WriteTriSyllables() {
	for _, s := range gn.TriSyls {
		fn := "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/coTriSyls/" + s
		fn = strings.Replace(fn, " ", "_", 2)
		f, err := os.Create(fn)
		check(err)
		_, err = f.Write([]byte(s))
		check(err)
	}
}

// WriteShuffles writes an individual file for each of the shuffled CV lists generated
// and also writes a file called "ls" that is a list of the files written!
func (gn *Gen) WriteShuffles() {
	var list []string // list of the files we are writing
	for _, s := range gn.SylsShuf {
		l := s
		l = strings.Replace(l, " ", "", -1) // this will be the file name without the path (same as an ls command)
		list = append(list, l)
		fn := "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/cvShuffles/" + s
		fn = strings.Replace(fn, " ", "", -1) // remove space from file name
		s = strings.Replace(s, " ", ", ", -1) // add a comma after each CV in the sequence itself
		f, err := os.Create(fn)
		check(err)
		_, err = f.Write([]byte(s))
		check(err)
	}
	// create a file that lists the individual files just written
	fn := "/Users/rohrlich/go/src/github.com/ccnlab/lang-acq/cvShuffles/" + "ls"
	f, err := os.Create(fn)
	check(err)
	s := ""
	for _, l := range list {
		s += l
		s += "\n"
	}
	_, err = f.Write([]byte(s))
	check(err)
}

// LoadParams
func (gn *Gen) LoadParams() {
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// GenWholeWordWavs generates the cv sequences for "whole words" (predictable cv sequences)
// These are used for training and for test
func (gn *Gen) GenWholeWordWavs() {
	// first the whole words
	for i := 0; i < 4; i++ {
		gn.GenTriCVWavs(gn.Syl1[i], gn.Syl2[i], gn.Syl3[i], 3)
	}
}

// GenPartWordWavs generates the cv sequences for "part words" (words where the second cv is one of a set and thus not fully predictable)
// These are only used during test
func (gn *Gen) GenPartWordWavs() {
	// next the part words
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			k := j % 4
			if k == i {
				continue
			}
			gn.GenTriCVWavs(gn.Syl3[i], gn.Syl1[k], gn.Syl2[k], 3)
		}
	}
}

//TriCVWavsFromSingle concatenates wav files of single CVs into wav files
//containing ordered combinations based on the list of CVs allowed in each position
func (gn *Gen) GenTriCVWavs(firstCV, secondCV, thirdCV string, nCombos int) {
	f1 := ""
	f2 := ""
	f3 := ""

	dirIn := gn.IndWavsDir
	dirOut := gn.TriWavsDir
	files := gn.LoadFileNamesFromDir(dirIn, ".wav")
	nm := ""

	for m := 0; m < nCombos; m++ {
		rand.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
		for _, f := range files {
			if strings.HasPrefix(f, firstCV) {
				f1 = f
				break
			}
		}
		for _, f := range files {
			if strings.HasPrefix(f, secondCV) {
				f2 = f
				break
			}
		}
		for _, f := range files {
			if strings.HasPrefix(f, thirdCV) {
				f3 = f
				break
			}
		}
		nm = f1[0:2] + f2[0:2] + f3[0:2]
		id := ""
		a := strconv.Itoa(m)
		if m > 99 {
			id = "_" + a
		} else if m > 9 {
			id = "_0" + a
		} else {
			id = "_00" + a
		}
		nm += id
		nm += ".wav"
		ffmpegConcatTris(dirIn, f1, f2, f3, dirOut, nm)
	}
}

// SequenceFromTriCVs
func (gn *Gen) SequenceFromTriCVs() {
	dirIn := gn.TriWavsDir
	dirOut := gn.SeqWavsDir
	fl := gn.LoadFileNamesFromDir(dirIn, ".wav") // file list
	fns := []string{}

	f1 := ""
	f2 := ""
	f3 := ""
	f4 := ""

	// make a list of the words formed by the syllables
	// e.g. subahi rolura pagodi holisa
	list := []string{}
	for i := 0; i < len(gn.Syl1); i++ {
		s := gn.Syl1[i] + gn.Syl2[i] + gn.Syl3[i]
		list = append(list, s)
	}
	fmt.Println(list)
	Perm(list, func(a []string) {
		rand.Shuffle(len(fl), func(i, j int) { fl[i], fl[j] = fl[j], fl[i] })
		fns = fns[:0]
		nm := ""

		for _, f := range fl {
			if strings.Contains(f, a[0]) {
				fns = append(fns, f)
				f1 = f
				break
			}
		}
		for _, f := range fl {
			if strings.Contains(f, a[1]) {
				fns = append(fns, f)
				f2 = f
				break
			}
		}
		for _, f := range fl {
			if strings.Contains(f, a[2]) {
				fns = append(fns, f)
				f3 = f
				break
			}
		}
		for _, f := range fl {
			if strings.Contains(f, a[3]) {
				fns = append(fns, f)
				f4 = f
				break
			}
		}
		nm = f1[0:6] + f2[0:6] + f3[0:6] + f4[0:6]
		nm += ".wav"
		ffmpegConcatSeqs(dirIn, f1, f2, f3, f4, dirOut, nm)
	})
}

// ffmpegConcatTris is a utility function to concatenate exising wav files - currently hard coded to 3 wav files
func ffmpegConcatTris(dirIn, f1, f2, f3, dirOut, f4 string) {
	f1 = dirIn + "/" + f1
	f2 = dirIn + "/" + f2
	f3 = dirIn + "/" + f3
	f4 = dirOut + "/" + f4
	cmd := exec.Command("ffmpeg", "-i", f1, "-i", f2, "-i", f3, "-filter_complex", "[0:0][1:0][2:0]concat=n=3:v=0:a=1[out]", "-map", "[out]", f4)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// ffmpegConcatTris is a utility function to concatenate exising wav files - currently hard coded to 3 wav files
func ffmpegConcatSeqs(dirIn, f1, f2, f3, f4, dirOut, f5 string) {
	f1 = dirIn + "/" + f1
	f2 = dirIn + "/" + f2
	f3 = dirIn + "/" + f3
	f4 = dirIn + "/" + f4
	f5 = dirOut + "/" + f5
	cmd := exec.Command("ffmpeg", "-i", f1, "-i", f2, "-i", f3, "-i", f4, "-filter_complex", "[0:0][1:0][2:0][3:0]concat=n=4:v=0:a=1[out]", "-map", "[out]", f5)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func (gn *Gen) Rename() {
	dirIn := gn.IndWavsDir
	files := gn.LoadFileNamesFromDir(dirIn, ".wav")

	for _, f := range files {
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

		cmd := exec.Command("mv", dirIn+"/"+fo+".wav", dirIn+"/"+fn+"_"+fo+".wav")
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
	ln := 0
	for _, r := range s {
		chunk[ln] = r
		ln++
		if ln == chunkSize {
			chunks = append(chunks, string(chunk))
			ln = 0
		}
	}
	if ln > 0 {
		chunks = append(chunks, string(chunk[:ln]))
	}
	return chunks
}

// GenSpeech calls gnuspeech on the content of each file in dir
func (gn *Gen) GenSpeech(dirIn, dirOut string) {
	files := gn.LoadFileNamesFromDir(dirIn, "")

	ex := "/Users/rohrlich/gnuspeech_sa-master/build/gnuspeech_sa"
	data := "/Users/rohrlich/gnuspeech_sa-master/data/en"

	for _, fn := range files {
		if fn == "ls" {
			continue
		}
		fp, err := os.Open(dirIn + "/" + fn)
		if err != nil {
			log.Println(err)
		}
		defer fp.Close() // we will be done with the file within this function

		scanner := bufio.NewScanner(fp)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			tx := scanner.Text()
			fout := dirOut + "/" + fn + ".wav"
			cmd := exec.Command(ex, "-c", data, "-p", "trm_param_file.txt", "-o", fout, tx)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		}
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

	tbar.AddAction(gi.ActOpts{Label: "Gen TriSyllable Strings", Icon: "new", Tooltip: "Generate all combinations of tri syllabic strings from the sets of syllables for each position"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.GenTriSyllables()
		})

	tbar.AddAction(gi.ActOpts{Label: "Write TriSyls Strings", Icon: "new", Tooltip: ""}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.WriteTriSyllables()
		})

	tbar.AddAction(gi.ActOpts{Label: "Shuffle CVs", Icon: "new", Tooltip: "Shuffle the syllables and add this shuffle to the list of shuffles"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.ShuffleCVs()
		})

	tbar.AddAction(gi.ActOpts{Label: "Write Shuffled CVs", Icon: "new", Tooltip: "WriteShuffles writes an individual file for each of the shuffled CV lists generated\n// and also writes a file called \"ls\" that is a list of the files written!"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.WriteShuffles()
		})

	tbar.AddAction(gi.ActOpts{Label: "Gen Speech", Icon: "new", Tooltip: "Calls GnuSpeech on content of files\n// and also writes a file called \"ls\" that is a list of the files written!"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.GenSpeech(gn.ShufflesIn, gn.ShufflesOut)
		})

	tbar.AddAction(gi.ActOpts{Label: "Rename Individual CVs", Icon: "new", Tooltip: "Must run this after splitting shuffle files into individual CVs before concatenating!"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.Rename()
		})

	tbar.AddAction(gi.ActOpts{Label: "Gen Whole Word Wavs", Icon: "new", Tooltip: "Generates wav files of 3 CVs where the second and third are fully predictable based on first CV"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.GenWholeWordWavs()
		})

	tbar.AddAction(gi.ActOpts{Label: "Gen Part Word Wavs", Icon: "new", Tooltip: "Generates wav files of 3 CVs, the second CV is of a set (so partially predictable), the third CV is predictable based on second"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.GenPartWordWavs()
		})

	tbar.AddAction(gi.ActOpts{Label: "Wav sequence from tri wavs", Icon: "new", Tooltip: "Write wav file that is the concatenation of wav files of tri CVs"}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			gn.SequenceFromTriCVs()
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
