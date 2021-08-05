package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/emer/auditory/agabor"
	"github.com/emer/emergent/env"
	"github.com/emer/empi/mpi"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"github.com/goki/gi/gi"
	"github.com/goki/ki/kit"
)

// For the current implementation all of the sound files (.wav files) are generated prior to the run.
// Each wav file is generated from a particular sequence of CVs. All of the combinations of CVs that might
// be used in the wav files are loaded at startup. Note that it is possible that any particular sequence might not
// have a corresponding wav file as all the generation is done randomly rather than factorially.

// Once a wave file is loaded at set of times for the start and stop of each CV in the wav file is loaded.
// This is done one at a time rather than loading all of these "label" files at the start.

// The wave files are also loaded one at a time. It is only the names of the wav files that are loaded at startup.

// SeqOrder is the order of sequences for each epoch
type SeqOrder int

// Sound sequence ordering
var KiT_SeqOrder = kit.Enums.AddEnum(SeqOrderN, kit.NotBitFlag, nil)

const (
	FixedOrder SeqOrder = iota
	RandomOrder
	CycleOrder
	SeqOrderN
)

//go:generate stringer -type=SeqOrder

// PartWhole
type PartWhole int

// Describes the transition from the first CV of the sound sequence to the second
// WholeWord means the transition is the same as it was in the training.
// e.g. if trained on "golatu" as a word and the first CV in testing is "go" and the "second" is "la" - call it "WholeWord"
// but if the test sound is "tuti" then call it "PartWord" because "tu" is from the end of a word and "ti is from the start of another
var KiT_PartWhole = kit.Enums.AddEnum(PartWholeN, kit.NotBitFlag, nil)

const (
	NotPartNorWhole PartWhole = iota
	PartWord
	WholeWord
	PartWholeN
)

//go:generate stringer -type=Predictable

// Predictable
type Predictable int32

//go:generate stringer -type=Predictable

// Describes the degree to which a segment is predictable
var KiT_Predictable = kit.Enums.AddEnum(PredictableN, kit.NotBitFlag, nil)

const (
	Ignore        Predictable = iota
	Fully                     // sequence is absolute - only one possibility
	Partially                 // multiple possibilities (1 of N)
	Unpredictable             // a single occurence so can't be predicted
	PredictableN
)

////////////////////////////////////////////////////////////////////////////////////////////
// Environment - params and config for the train/test environment

// CVCurrent holds consonant vowel information for the current segment of sound
type CVCurrent struct {
	Ordinal     int               `desc:"is this the first, second, ... CV of the sound sequence"`
	SubSeg      int               `desc:"segment of the current CV counting forwards"`
	Last        string            `desc:"consonant vowel (CV) of previous segment"`
	Cur         string            `desc:"consonant vowel (CV) of current segment"`
	Predictable Predictable       `desc:"is this CV fully or partially predictable"`
	Word        PartWhole         `desc:"was this CV fully predictable during training (wholeword)"`
	Predicted   map[string]string `view:"no-inline" desc:"layer name is key and predicted CV is value, for cases where the CV is fully predicatable, "`
}

// set strings to empty and ints to 0
func (cv *CVCurrent) Reset() {
	cv.SubSeg = 0
	cv.Cur = ""
	cv.Last = ""
	cv.Ordinal = -1
	cv.Predictable = Ignore
	cv.Word = NotPartNorWhole
}

// CVSegment
type CVSegment struct {
	CV     string
	SubSeg int
}

// CVSequence a sequence of CVs and information about each CV
type CVSequence struct {
	ID  string `desc:"identifier xxx_xxxxx where xxx identifies the group of cvs the sequence is generated from and xxxxx is the specific sequence"`
	Seq string `desc:"the full sequence of CVs"`
}

// CVTime
type CVTime struct {
	Name       string  `desc:"the CV, da, go, ku, etc"`
	Start      float64 `desc:"start time of this CV in a particular sequence in milliseconds"`
	End        float64 `desc:"end time of this CV in a particular sequence in milliseconds"`
	StartAlpha float64 `desc:"start time of this CV in a particular sequence in milliseconds, adjusted for random start silence and aligned to alpha (100ms)"`
	EndAlpha   float64 `desc:"end time of this CV in a particular sequence in milliseconds, adjusted for random start silence and aligned to alpha (100ms)"`
}
type WEEnv struct {
	// the environment has the training/test data and the procedures for creating/choosing the input to the model
	// "Segment" in var name indicates that the data or value only applies to a segment of samples rather than the entire signal
	Nm         string          `desc:"name of this environment"`
	Dsc        string          `desc:"description of this environment"`
	Run        env.Ctr         `view:"inline" desc:"current run of model as provided during Init"`
	Epoch      env.Ctr         `view:"inline" desc:"number of times through a set of sequences"`
	Sequence   env.Ctr         `view:"inline" desc:"current sequence which is a series of trials (segments of sound in this simulation"`
	Trial      env.Ctr         `view:"inline" desc:"current trial which is 2 or more events"`
	Event      env.Ctr         `view:"inline" desc:"the current event of the trial"`
	TrialName  string          `desc:"if Table has a Name column, this is the contents of that for current trial"`
	SeqOrder   SeqOrder        `view:"+" desc:"order of sound sequences - ordered, random, cyclical"`
	Patterns   *etable.IdxView `desc:"this is a one row table with the set of patterns to output for the next event"`
	SndCur     string          `view:"+" desc:" name of current open sound file"`
	SeqCur     string          `view:"+" desc: identifier (filename) of currently loaded sound"`
	SndList    string          `view:"-" desc:" stash file name for reload"`
	SndFiles   []string        `view:"no-inline" desc:" the list of sound files"`
	SndTimit   bool            `view:"-" desc:" are the sound files timit files"`
	SndPath    string          `view:"-" desc:" base path to all sound, sequence and timing files"`
	SeqsPath   string          `desc:"path to the human readable files of the sound sequences"`
	WavsPath   string          `desc:"path to wav files"`
	TimesPath  string          `desc:"path to the timing information for wav files - also called labels"`
	SndShort   SndEnv          `view:"+" desc:" sound processing values and matrices for the short duration pathway"`
	SndLong    SndEnv          `view:"+" desc:" sound processing values and matrices for the long duration pathway"`
	MaxSegCnt  int             `desc:"this will be the minimum segment count of SndShort and SndLong (or others if there are more)"`
	CV         CVCurrent       `desc:"struct containing segment/CV state"`
	CVs        []string        `desc:"the full list of CVs in the training"`
	CVTimes    []CVTime        `desc:"a slice of all of the CVs and their start/end times for the currently loaded sequence of CVs"`
	CVsPerWord int             `desc:"how many CVs per word"`
	CVsPerPos  int             `desc:"how many CV possibilities per syllable position - assumes same for each position"`
	FirstCVs   []string        `desc:"the CVs in the first position of the trisyllabic words"`  // order is important
	SecondCVs  []string        `desc:"the CVs in the second position of the trisyllabic words"` // order is important
	ThirdCVs   []string        `desc:"the CVs in the third position of the trisyllabic words"`  // order is important
	Silence    bool            `desc:"add random period of silence at start of sequence"`
	SilenceMax int             `desc:"maximum milliseconds of silence to add at start of sequence - uniform random"`
	HoldoutPct int             `desc:"percentage of items to holdout for testing"`

	// specific to word break detection
	//PW       PartWhole `desc:" is the current segment beginning of part word"`
	RepeatOk bool `desc:" is it okay to immediately repeat a sound"`

	// internal state - view:"-"
	ToolBar      *gi.ToolBar `view:"-" desc:" the master toolbar"`
	MoreSegments bool        `view:"-" desc:" are there more samples to process"`
	SndIdx       int         `view:"-" desc:"the index into the soundlist of the sound from last trial"`
	BinThr       float32     `def:"0.4" desc:"threshold for binarizing"`
	msSilence    float64     `desc:"add this much random silence at front of signal"`
}

func (we *WEEnv) DefaultsTrn() {
	we.SeqOrder = RandomOrder
	we.SndIdx = -1
	we.RepeatOk = true
	we.CV.Reset()
	we.BinThr = 0.2
	we.CVsPerWord = 3
	we.CVsPerPos = 4
	we.Silence = true
	we.SilenceMax = 25.0
	we.HoldoutPct = 17
}

func (we *WEEnv) DefaultsTest() {
	we.SeqOrder = CycleOrder
	we.SndIdx = -1
	we.RepeatOk = false
	we.CV.Reset()
	we.CVsPerWord = 3
	we.CVsPerPos = 4
	we.Silence = true
	we.HoldoutPct = 0
	we.SilenceMax = 25.0
}

func (we *WEEnv) InitSndShort() {
	we.MoreSegments = true
	we.SndShort.Defaults()

	we.SndShort.Nm = "SoundShort"
	we.SndShort.Dsc = "150 ms window onto sound"

	// these should match the number of neuron pools in the input layer
	we.SndShort.GborPoolsY = 12
	we.SndShort.GborPoolsX = 6

	// override defaults
	we.SndShort.Params.SegmentMs = 150
	we.SndShort.Params.WinMs = 25
	we.SndShort.Params.StepMs = 10
	we.SndShort.Params.StrideMs = 100
	we.SndShort.Params.BorderSteps = 6

	// for example, with Stride/StepMs equal to 10 and 3 border steps on either side there will be 16 values for the gabor stepping to cover
	// so the gbor (size of 6) will go from 0-5, 2-7, 4-9 ... 10-15

	// these overrides must follow Mel.Defaults
	we.SndShort.Mel.FBank.RenormMin = 2 // 2/9 better than 0/10
	we.SndShort.Mel.FBank.RenormMax = 9
	we.SndShort.Mel.FBank.LoHz = 20
	we.SndShort.Mel.FBank.HiHz = 6000

	g := new(agabor.Params)
	g.Defaults()
	g.TimeSize = 6
	g.TimeStride = 4
	g.FreqSize = 6
	g.FreqStride = 3
	g.WaveLen = 6.0
	g.HorizSigmaWidth = 0.2

	st := -1.0
	end := -1.0
	if we.SndTimit {
		st = we.CVTimes[0].Start * 1000
		end = we.CVTimes[len(we.CVTimes)-1].End * 1000
	}
	err, _ := we.SndShort.Init(*g, we.msSilence, st, end)
	if err != nil {
		fmt.Println("Error returned from NewSoundInit")
	}
}

func (we *WEEnv) InitSndLong() {
	we.MoreSegments = true
	we.SndLong.Defaults()

	we.SndLong.Nm = "SoundLong"
	we.SndLong.Dsc = "200 ms window onto sound"

	// these should match the number of neuron pools in the input layer
	we.SndLong.GborPoolsY = 12
	we.SndLong.GborPoolsX = 6

	// override defaults
	we.SndLong.Params.SegmentMs = 150
	we.SndLong.Params.WinMs = 25
	we.SndLong.Params.StepMs = 10
	we.SndLong.Params.StrideMs = 100
	we.SndLong.Params.BorderSteps = 5
	// with SegmentMs/StepMs equal to 30 and 6 border steps on either side there will be 42 values for the gabor stepping to cover
	// so the gbor (size of 8) will go from 0-7, 3-10, 6-13 ... 33-40
	// for a time size of 10 the border steps needs to go to 7, etc.

	// these overrides must follow Mel.Defaults
	we.SndLong.Mel.FBank.RenormMin = 2 // 2/9 better than 0/10
	we.SndLong.Mel.FBank.RenormMax = 9
	we.SndLong.Mel.FBank.LoHz = 20
	we.SndLong.Mel.FBank.HiHz = 6000

	g := new(agabor.Params)
	g.Defaults()
	g.TimeSize = 10
	g.TimeStride = 3
	g.FreqSize = 6
	g.FreqStride = 3
	g.WaveLen = 6.0
	g.HorizSigmaWidth = 0.2

	st := -1.0
	end := -1.0
	if we.SndTimit {
		st = we.CVTimes[0].Start * 1000
		end = we.CVTimes[len(we.CVTimes)-1].End * 1000
	}
	err, _ := we.SndLong.Init(*g, we.msSilence, st, end)
	if err != nil {
		fmt.Println("Error returned from NewSoundInit")
	}
}

// SetIsPredictable checks to if the first segment of the CV is one that is "fully" predictable
// (i.e. within an unchanging word)
// or partially predictable (i.e. one of multiple that are possible)
func (we *WEEnv) SetIsPredictable() {
	we.CV.Predictable = Ignore
	if we.Nm == "PreTrainEnv" { // no predicting when just pretraining
		return
	}
	if we.CV.Ordinal == 0 { // ignore first CV - prediction not possible
		return
	}

	if we.CV.Cur == "ss" { // silence
		we.CV.Predictable = Ignore
		return
	}

	if we.SndTimit == true {
		we.CV.Predictable = Partially
		return
	}
	if we.CV.Ordinal%we.CVsPerWord == 0 {
		we.CV.Predictable = Partially
		return
	} else {
		we.CV.Predictable = Fully
		return
	}
}

// PredictableAsString const int returned as string
func (we *WEEnv) PredictableAsString(p Predictable) string {
	if p == Fully {
		return "Fully"
	} else if p == Partially {
		return "Partially"
	}
	return ""
}

// PartWholeAsString const int returned as string
func (we *WEEnv) PartWholeAsString(p PartWhole) string {
	if p == WholeWord {
		return "Whole"
	} else if p == PartWord {
		return "Part"
	}
	return ""
}

// SetIsPartWhole determines if the second CV is from the same word or different word (called part word in earlier literature)
// These words are set for the run (experiment)
func (we *WEEnv) SetIsPartWhole() {
	we.CV.Word = NotPartNorWhole
	if we.CV.Ordinal == 1 && we.CV.Last != we.CV.Cur { // i.e. only when we have just processed the first segment of the second CV
		last := we.CV.Last
		cur := we.CV.Cur

		for i := 0; i < we.CVsPerPos; i++ {
			if last == we.FirstCVs[i] && cur == we.SecondCVs[i] {
				we.CV.Word = WholeWord
				return
			}
		}

		if we.CVsPerWord == 2 {
			for i := 0; i < we.CVsPerPos; i++ {
				if last == we.SecondCVs[i] {
					for j := 0; j < 4; j++ {
						if cur == we.FirstCVs[j] {
							we.CV.Word = PartWord
							return
						}
					}
				}
			}
		} else if we.CVsPerWord == 3 {
			for i := 0; i < we.CVsPerPos; i++ {
				if last == we.ThirdCVs[i] {
					for j := 0; j < 4; j++ {
						if cur == we.FirstCVs[j] {
							we.CV.Word = PartWord
							return
						}
					}
				}
			}
		}
	}
}

// ClearSoundsAndData empties the sound list, sets current sound to nothing, etc
func (we *WEEnv) ClearSoundsAndData() {
	if we.SndFiles != nil {
		we.SndFiles = we.SndFiles[:0]
	}
	we.SndCur = ""
	we.SndIdx = -1
	we.MaxSegCnt = 0
	we.Trial.Max = 0
	we.MoreSegments = false // this will force a new sound to be loaded
}

// LoadWavNames reads in a list of sound files names
func (we *WEEnv) LoadWavNames() error {
	fp, err := os.Open(we.SndPath + we.SndList)
	if err != nil {
		log.Println(err)
		log.Println("Make sure you have the sound files rsyncd to your ccn_images directory and a link (ln -s) to ccn_images in your sim working directory")
		return err
	}
	defer fp.Close() // we will be done with the file within this function

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)
	// clear first in case it is called twice or the file changes
	if we.SndFiles != nil {
		we.SndFiles = we.SndFiles[:0]
	}
	for scanner.Scan() {
		we.SndFiles = append(we.SndFiles, scanner.Text())
	}
	return nil
}

// LoadSoundsAndData loads
func (we *WEEnv) LoadSoundsAndData() {
	we.ClearSoundsAndData()
	we.LoadWavNames()
}

// SplitSndFiles pulls some of the training files out for testing - pass in the unique test file name
func (we *WEEnv) SplitSndFiles(tstFileName string) {
	tstfiles := []string{}
	trnfiles := []string{}
	n := int(math.Trunc(float64(we.HoldoutPct) / 100 * float64(len(we.SndFiles))))
	rands := []int{}
	for i := 0; i < n; i++ {
		rnd := rand.Intn(len(we.SndFiles))
		dupe := false
		for _, r := range rands {
			if rnd == r {
				dupe = true
				break
			}
		}
		if dupe {
			i--
		} else {
			rands = append(rands, rnd)
		}
	}
	sort.Ints(rands)

	r := 0
	normalHold := true // quick hack - set to false for reverse holdout set
	for i := 0; i < len(we.SndFiles); i++ {
		if normalHold == true {
			if i != rands[r] {
				trnfiles = append(trnfiles, we.SndFiles[i])
			} else {
				tstfiles = append(tstfiles, we.SndFiles[i])
				r++
			}
			if r == len(rands) { // we found all the test files
				trnfiles = append(trnfiles, we.SndFiles[i+1:len(we.SndFiles)]...)
				break
			}
		} else {
			if i == rands[r] {
				trnfiles = append(trnfiles, we.SndFiles[i])
				r++
			} else {
				tstfiles = append(tstfiles, we.SndFiles[i])
			}
			if r == len(rands) { // we found all the test files
				tstfiles = append(tstfiles, we.SndFiles[i+1:len(we.SndFiles)]...)
				break
			}
		}
	}
	we.SndFiles = trnfiles
	// create a file of sound file names for later loading
	we.WriteTestList(tstfiles, tstFileName)
	fmt.Println("train:", trnfiles)
	fmt.Println("test:", tstfiles)
}

// WriteTestList
func (we *WEEnv) WriteTestList(list []string, filename string) {
	f, err := os.Create(we.SndPath + filename)
	if err != nil {
		log.Println("WriteTestList: ", err)
	}
	defer f.Close()

	for _, file := range list {
		_, err := f.WriteString(file + "\n")
		if err != nil {
			log.Println("WriteTestList: ", err)
		}
	}
}

// LoadCVSeq reads in a list of cv strings for decoding a particular sequence
func (we *WEEnv) LoadCVSeq(fn string) error {
	fp2, err2 := os.Open(we.SndPath + we.SeqsPath + fn)
	if err2 != nil {
		log.Println(err2)
		return err2
	}
	defer fp2.Close() // we will be done with the file within this function
	scanner2 := bufio.NewScanner(fp2)
	scanner2.Split(bufio.ScanLines)
	for scanner2.Scan() {
		we.SeqCur = scanner2.Text()
	}
	return nil
}

// SeqFields
func (we *WEEnv) SeqFields(seq string) []string {
	s := strings.Replace(seq, ".", "", -1)
	return strings.Split(s, " ")
}

// NextSound will determine the next sound to load and load it - return error if end of sound list or actual error
func (we *WEEnv) NextSound() (done bool, err error) {
	done = false
	if len(we.SndFiles) == 0 {
		log.Printf("No file names in sound list file\n")
		we.LoadWavNames()
		err = errors.New("wordsegenv.NextSound: SndFiles zero length")
		return true, err
	}

	stop := we.NextSndFile() // will set we.SndIndx
	if stop == true {
		done = true
		return done, nil
	}
	if we.SndIdx == -1 { // no sound or we exhausted the sounds - done
		fmt.Println("SndIdx == -1")
		err := error(nil)
		return true, err
	}

	we.SndCur = we.SndFiles[we.SndIdx]
	fp := we.SndPath + we.WavsPath + we.SndCur

	// add some random silence at start of sequence (up to 50ms)
	we.msSilence = 0.0
	if we.Silence {
		we.msSilence = float64(rand.Intn(we.SilenceMax))
	}

	err = we.SndShort.Sound.Load(fp)
	if err != nil {
		log.Printf("NextSegment: error loading sound -- %v\n, err", we.SndCur)
		return false, err
	}

	// before loading sound, load the sequence times so we can drop the silence
	// from the signal at start and end
	fn := strings.TrimSuffix(we.SndCur, ".wav")
	idx := 0
	cnt := 0
	for j := 0; j < len(fn); j++ {
		if fn[j] == '_' {
			cnt++
		}
		if cnt == 2 {
			idx = j
			break
		}
	}

	if idx == 0 {
		we.SeqCur = fn
	} else {
		we.SeqCur = fn[0:idx]
	}
	we.TrialName = fn
	if we.SndTimit == true {
		we.LoadTimitSeqsAndTimes(fn)
	} else {
		we.LoadCVSeq(fn)
		we.LoadCVTimes(fn)
	}

	we.SndShort.LoadSound()
	we.InitSndShort()

	err = we.SndLong.Sound.Load(fp)
	if err != nil {
		log.Printf("NextSegment: error loading sound -- %v\n, err", we.SndCur)
		return false, err
	}
	we.SndLong.LoadSound()
	we.InitSndLong()

	// do some checks and set trial max
	if we.SndLong.SegCnt < we.SndShort.SegCnt {
		we.Trial.Max += we.SndLong.SegCnt
		we.MaxSegCnt = we.SndLong.SegCnt
		log.Println("Segment count long < short duration count, use this for we.Trial.Max!")
		err = errors.New("Segment count long < short duration count, should only happen if the sounds are different")
	} else if we.SndLong.SegCnt > we.SndShort.SegCnt {
		we.Trial.Max += we.SndShort.SegCnt
		we.MaxSegCnt = we.SndShort.SegCnt
		log.Println("Segment count long > short duration count, use this for we.Trial.Max!")
		err = errors.New("Segment count long > short duration count, should only happen if the sounds are different")
	} else {
		we.Trial.Max += we.SndShort.SegCnt
		we.MaxSegCnt = we.SndShort.SegCnt
	}
	we.CV.Reset()

	return done, err
}

func (we *WEEnv) NextSndFile() (stop bool) {
	stop = false
	sfc := len(we.SndFiles)
	nproc := mpi.WorldSize()

	if sfc%nproc != 0 {
		log.Printf("wordsegenv: number of sequences: %d is not an even multiple of number of MPI procs: %d -- must be!\n", sfc, nproc)
	}
	if we.SeqOrder == RandomOrder {
		if we.RepeatOk {
			n := sfc / nproc
			we.SndIdx = rand.Intn(n) + mpi.WorldRank()*n
			//log.Printf("SndIdx: %v, rank: %v, file: %v\n", we.SndIdx, mpi.WorldRank(), we.SndFiles[we.SndIdx])
		} else {
			for {
				n := sfc / nproc
				we.SndIdx = rand.Intn(n) + mpi.WorldRank()*n
				sndNext := strings.TrimSuffix(we.SndFiles[we.SndIdx], ".wav")
				sndNext = strings.TrimPrefix(sndNext, "")
				if len(we.SndCur) > 0 {
					sndCur := strings.TrimPrefix(we.SndCur, "")
					if sndNext[0] != sndCur[0] {
						break
					}
				} else {
					break
				}
			}
		}
	} else {
		we.SndIdx++
		n := sfc / nproc
		if we.SndIdx >= n+mpi.WorldRank()*n {
			if we.SeqOrder == CycleOrder { // done
				stop = true
			} else {
				we.SndIdx = mpi.WorldRank() * n // start over
			}
		}
	}
	return stop
	//fmt.Printf("rank: %d\t idx: %d\n", mpi.WorldRank(), we.SndIdx)
}

// NextSegment calls to process the next segment of sound, loading a new sound if the last sound was fully processed
func (we *WEEnv) NextSegment() error {
	//fmt.Println("seg / max seg", we.SndShort.Segment, we.MaxSegCnt)
	if we.MoreSegments == false || we.SndShort.Segment == we.MaxSegCnt {
		done, err := we.NextSound()
		if done && err == nil {
			return err
		}
		if err != nil {
			return err
		}
	}
	moreShort := we.SndShort.ProcessSegment()
	moreLong := we.SndLong.ProcessSegment()

	if moreShort != moreLong {
		return errors.New("Sequence lengths out of sync - could there be a bug in the padding of the signal?")
	}
	we.MoreSegments = moreShort
	return nil
}

// LoadCVTimes loads the timing and sequence (transcription) data for CV files
func (we *WEEnv) LoadCVTimes(fn string) error {
	we.CVTimes = nil

	// load the CV start/end times produced by Audacity "sound finder"
	fp, err := os.Open(we.SndPath + we.TimesPath + fn + ".txt")
	if err != nil {
		log.Println(err)
		log.Println("Make sure you have the sound files rsyncd to your ccn_images directory and a link (ln -s) to ccn_images in your sim working directory")
		return err
	}
	defer fp.Close() // we will be done with the file within this function

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)

	silence := we.msSilence / 1000.0
	offset := 0.0 // first start time might not be zero - could be a section of longer sequence
	flds := we.SeqFields(we.SeqCur)
	i := 0
	for scanner.Scan() {
		t := scanner.Text()
		if t == "" {
			break
		} else if strings.HasPrefix(t, "\\") { // lines starting with '/' are lines with frequency for start/end points
			continue
		}
		cvt := new(CVTime)
		we.CVTimes = append(we.CVTimes, *cvt)
		cvs := strings.Fields(t)
		f, err := strconv.ParseFloat(cvs[0], 64)
		if len(we.CVTimes) == 1 { // first start/end time - save offset
			offset = f
		}
		if err == nil {
			we.CVTimes[i].Start = f
			f += silence - offset
			we.CVTimes[i].StartAlpha = we.AdjustCVTime(f, true)
		}
		f, err = strconv.ParseFloat(cvs[1], 64)
		if err == nil {
			we.CVTimes[i].End = f
			f += silence - offset
			we.CVTimes[i].EndAlpha = we.AdjustCVTime(f, false)
		}
		we.CVTimes[i].Name = flds[i]
		i++
		if i == len(flds) {
			return nil
		} // handles case where there may be lines after last line of start, end, name
	}
	return nil
}

// LoadTimitSeqsTimes loads the timing and sequence (transcription) data for timit files
func (we *WEEnv) LoadTimitSeqsAndTimes(fn string) error {
	we.CVTimes = nil // the sounds aren't CVs but the idea is the same

	// load the sound start/end times shipped with the TIMIT database
	// wav files stored in a different directory so fix the filename
	fn = strings.Replace(fn, "_Wavs", "", 1)
	fp, err := os.Open(we.SndPath + we.TimesPath + fn + ".PHN.MS") // PHN is "Phone" and MS is milliseconds
	if err != nil {
		log.Println(err)
		log.Println("Make sure you have the sound files rsyncd to ccn_images directory and a link (ln -s) to ccn_images in your sim working directory")
		return err
	}
	defer fp.Close() // we will be done with the file within this function

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)

	silence := we.msSilence / 1000.0
	i := 0
	for scanner.Scan() {
		t := scanner.Text()
		if t == "" {
			break
		}
		if strings.Contains(t, "h#") { // silence at start or silence at end
			if len(we.CVTimes) == 0 { // starting silence
				continue
			} else { // silence at end
				cvs := strings.Fields(t)
				f, _ := strconv.ParseFloat(cvs[0], 64)
				f = f / 1000 // convert from ms to seconds
				we.CVTimes[i-1].End = f
				f += silence
				we.CVTimes[i-1].EndAlpha = we.AdjustCVTime(f, true)
				break // we're done!
			}
		}
		cvt := new(CVTime)
		we.CVTimes = append(we.CVTimes, *cvt)
		cvs := strings.Fields(t)
		f, err := strconv.ParseFloat(cvs[0], 64)
		f = f / 1000 // convert from ms to seconds
		if err == nil {
			we.CVTimes[i].Start = f
			f += silence
			we.CVTimes[i].StartAlpha = we.AdjustCVTime(f, true)
		}
		if len(we.CVTimes) > 1 {
			we.CVTimes[i-1].End = we.CVTimes[i].Start
			we.CVTimes[i-1].EndAlpha = we.CVTimes[i].StartAlpha
		}
		we.CVTimes[i].Name = cvs[1] //
		i++
	}
	return nil
}

// AdjustCVTimes adds some leeway around the absolute times.
// We need this because we only collect stats every 100ms and with the absolute times
// you can miss whole CVs if under 100ms (rare) but also we don't want to miss the first
// segment of a CV if 70% of the 100 ms is the first segment.
// Todo: is 70% the best division point? Yes seems to be good
func (we *WEEnv) AdjustCVTime(v float64, start bool) float64 {
	vadj := 0.0
	vrem := v*10 - math.Floor(v*10)
	if start {
		if vrem > .7 {
			vadj = 1000 * math.Ceil(v*10) / 10
		} else {
			vadj = 1000 * math.Floor(v*10) / 10
		}
	} else {
		if vrem > .7 {
			vadj = 1000 * math.Ceil(v*10) / 10
		} else {
			vadj = 1000 * math.Floor(v*10) / 10
		}
	}
	return vadj
}

// CVLookup uses the current segment position and the CVTimes to find the current CV as well as which segment of this particular CV
// Example papapabibikukuku, for segment 4, the second segment of "bi" (zero based of course)
// cv is "bi", subseg is 2
// sequence is the full sound sequence loaded from file, a subseq is a sequence of segments of a particular CV, e.g. papapa
func (we *WEEnv) CVLookup() {
	cv := ""
	stride := float64(we.SndShort.Params.StrideMs)
	time := float64(we.CurSeg())*stride + stride // add one stride to get to end of the segment
	last := len(we.CVTimes) - 1
	for _, cvt := range we.CVTimes {
		if time > we.CVTimes[last].EndAlpha { // if past the last cv
			cv = "ss"
			break
		}
		if time > cvt.EndAlpha {
			continue
		} else if time >= cvt.StartAlpha && time <= cvt.EndAlpha {
			cv = cvt.Name
			break
		} else {
			cv = "ss" // silence
			break
		}
	}

	if cv != "ss" && we.CV.Cur != "ss" { // never set last to ss (silence)
		we.CV.Last = we.CV.Cur
	}
	we.CV.Cur = cv
	if we.CV.Last == we.CV.Cur {
		we.CV.SubSeg++
	} else {
		if we.CV.Last != "ss" && we.CV.Cur != "ss" { // only update if next CV, silence doesn't count!
			we.CV.Ordinal++
		}
		we.CV.SubSeg = 0 // 0 for new CV or if silent segment part
	}
}

// IndexFromCV (consonant-vowel)
func (we *WEEnv) IndexFromCV(cv string) int {
	for i, s := range we.CVs {
		if s == cv {
			return i
		}
	}
	fmt.Println("IndexFromCV: Error - fell through CV switch")
	return -1
}

// CVFromIndex
func (we *WEEnv) CVFromIndex(idx int) string {
	if idx >= 0 && idx < len(we.CVs) {
		return we.CVs[idx]
	}
	fmt.Println("CVFromIndex: Error - fell through Index switch")
	return ""
}

func (we *WEEnv) Name() string { return we.Nm }
func (we *WEEnv) Desc() string { return we.Dsc }

func (we *WEEnv) Validate() error {
	we.Run.Scale = env.Run
	we.Epoch.Scale = env.Epoch
	we.Sequence.Scale = env.Sequence
	we.Trial.Scale = env.Trial
	we.Event.Scale = env.Event
	return nil
}

func (we *WEEnv) Counters() []env.TimeScales {
	return []env.TimeScales{env.Run, env.Epoch, env.Sequence, env.Trial, env.Event}
}

func (we *WEEnv) States() env.Elements {
	els := env.Elements{}
	//els.FromSchema(we.Patterns.Table.Schema())
	return els
}

func (we *WEEnv) Actions() env.Elements {
	return nil
}

func (we *WEEnv) Init(run int) {
	we.Run.Cur = run
	we.Trial.Cur = -1 // so first Trial is zero based

	we.CV.Predicted = make(map[string]string)
	we.CV.Predicted["CBTh_CV"] = ""
	we.CV.Predicted["RBTh_CV"] = ""
	we.CV.Predicted["CPBTh_CV"] = ""
	we.CV.Predicted["RPBTh_CV"] = ""
	we.CV.Predicted["STSTh_CV"] = ""
}

func (we *WEEnv) Step() bool {
	we.Epoch.Same() // good idea to just reset all non-inner-most counters at start
	we.Sequence.Same()

	max := we.Trial.Max
	cur := we.Trial.Cur
	if we.Trial.Incr() {
		we.Trial.Max = max     // reset to current - keep counting trials across sequences
		we.Trial.Cur = cur + 1 // Trial.Max is set based on sequence length and accumulated during epoch
		if we.Sequence.Incr() {
			we.Trial.Init()
			we.Trial.Max = 0
			we.Epoch.Incr()
		}
	}
	err := we.NextSegment()
	if err != nil {
		fmt.Printf("trial name: %v segment: %d\n", we.TrialName, we.CurSeg())
		//panic(err)
	}
	return true
}

func (we *WEEnv) State(element string) (et etensor.Tensor) {
	if element == "A1" {
		if we.SndShort.Kwta.On == true {
			et = &we.SndShort.GborKwta
		} else {
			et = &we.SndShort.GborOutput
		}
	} else if element == "R" {
		if we.SndLong.Kwta.On == true {
			et = &we.SndLong.GborKwta
		} else {
			et = &we.SndLong.GborOutput
		}
	} else {
		log.Println("State: element not known - check spelling, especially case!")
	}
	return et
}

func (we *WEEnv) Action(element string, input etensor.Tensor) {
	// nop
}

func (we *WEEnv) Counter(scale env.TimeScales) (cur, prv int, chg bool) {
	switch scale {
	case env.Run:
		return we.Run.Query()
	case env.Epoch:
		return we.Epoch.Query()
	case env.Sequence:
		return we.Sequence.Query()
	case env.Trial:
		return we.Trial.Query()
	case env.Event:
		return we.Event.Query()
	}
	return -1, -1, false
}

func (we *WEEnv) CurSeg() int {
	return we.SndShort.Segment
}
