// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// wordends uses deep leabra predictive learning to predict word endings and form abstract categories of endings
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/emer/emergent/emer"
	"github.com/emer/emergent/env"
	"github.com/emer/emergent/netview"
	"github.com/emer/empi/empi"
	"github.com/emer/empi/mpi"
	"github.com/emer/etable/agg"
	"github.com/emer/etable/eplot"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"github.com/emer/etable/etview"
	"github.com/emer/leabra/deep"
	"github.com/emer/leabra/leabra"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// !! really 3 groups of 4, first, second and third position of trisyllabic word - keep the order!!
var CVs_I = []string{"da", "go", "pa", "ti", "ro", "la", "bi", "bu", "pi", "tu", "ku", "do"}
var CVs_III = []string{"su", "ro", "pa", "ho", "ba", "lu", "go", "li", "hi", "ra", "di", "sa"}
var CVs_IV = []string{"do", "na", "hu", "ki", "ka", "to", "mo", "mu", "ru", "si", "ta", "po"}
var CVs_V = []string{"gu", "ma", "bi", "bu", "ri", "gi", "tu", "ni", "ha", "so", "ga", "bo"}
var CVs_VI = []string{"da", "ti", "nu", "lo", "ku", "no", "pi", "du", "mi", "pu", "ko", "la"}
var CVs_MT = []string{"ti", "do", "ga", "mo", "may", "bu", "pi", "ku"}

// TestType
type TestType int

// Describes the degree to which a segment is predictable
var KiT_TestType = kit.Enums.AddEnum(TestTypeN, kit.NotBitFlag, nil)

const (
	TestingTypeNotSet TestType = iota
	SequenceTesting            // the test items are like the training sound sequences
	PartWholeTesting           // the test items are "part words" and "whole words" see saffran 1996/1998 and Graf-Estes 2015
	TestTypeN
)

//go:generate stringer -type=TestType

func main() {
	TheSim.New()
	if len(os.Args) > 1 {
		TheSim.CmdArgs() // simple assumption is that any args = no gui -- could add explicit arg if you want
	} else {
		TheSim.Config()
		gimain.Main(func() { // this starts gui -- requires valid OpenGL display connection (e.g., X11)
			guirun()
		})
	}
}

func guirun() {
	TheSim.Init()
	win := TheSim.ConfigGui()
	win.StartEventLoop()
}

// LogPrec is precision for saving float values in logs
const LogPrec = 4

type LogFile struct {
	Name          string `desc:"unique descriptive part of file name"`
	File          *os.File
	Header        bool `desc:"include header in file"`
	HeaderWritten bool `desc:"set to true if header has been written"`
	Run           bool `desc:"include the run number in the file name"`
	Epoch         bool `desc:"include the epoch number in the file name"`
}

// Sim encapsulates the entire simulation model, and we define all the
// functionality as methods on this struct.  This structure keeps all relevant
// state information organized and available without having to pass everything around
// for the fields which provide hints to how things should be displayed).
type Sim struct {
	Net             *WordNet          `view:"no-inline" desc:"the network -- click to view / edit parameters for layers, prjns, etc"`
	Snd             string            `inactive:"+" desc:"the current sound file - set in env - here for view only"`
	TrnList         string            `desc:"name of file with list of training sounds for this run - only used for cmdline runs"`
	TstList         string            `desc:"name of file with list of testing sounds for this run - only used for cmdline runs"`
	PreTrnList      string            `desc:"name of file with list of pre-training sounds for this run"`
	PreTstList      string            `desc:"name of file with list of pre-testing sounds for this run"`
	Tag             string            `desc:"extra tag string to add to any file names output from sim (e.g., weights files, log files, params for run)"`
	HoldoutID       string            `desc:"unique id to specify the holdout testing file for the job, must specify for each job using holdout testing!"`
	StartRun        int               `desc:"starting run number -- typically 0 but can be set in command args for parallel runs on a cluster"`
	MaxRuns         int               `desc:"maximum number of model runs to perform"`
	MaxEpcs         int               `desc:"maximum number of epochs to run per model run"`
	MaxSeqs         int               `desc:"maximum number of sequences to run per epoch"`
	MaxPreEpcs      int               `desc:"maximum number of epochs to run for pretraining"`
	MaxPreSeqs      int               `desc:"maximum number of sequences to run per epoch of pretraining"`
	WtsInterval     int               `desc:"if SaveWts is true save wts every N epochs"`
	TestInterval    int               `desc:"how often to test (i.e. after how many training epochs) - use -1 if not testing!!!"`
	PreTestInterval int               `desc:"how often to test (i.e. after how many pretraining epochs) - use -1 if not testing!!!"`
	TrainEnv        WEEnv             `desc:"training environment -- contains everything about iterating over input / output patterns over training"`
	PreTrainEnv     WEEnv             `desc:"pre training environment -- contains everything about iterating over input / output patterns for pre training"`
	TestEnv         WEEnv             `desc:"testing environment -- manages iterating over testing"`
	PreTestEnv      WEEnv             `desc:"testing environment -- manages iterating over testing"`
	Env             *WEEnv            `desc:"the current environment - pretrain/train/test"`
	Time            leabra.Time       `desc:"leabra timing parameters and state"`
	ViewOn          bool              `desc:"whether to update the network view while running"`
	TrainUpdt       leabra.TimeScales `desc:"at what time scale to update the display during training?  Anything longer than Epoch updates at Epoch in this model"`
	TestUpdt        leabra.TimeScales `desc:"at what time scale to update the display during testing?  Anything longer than Epoch updates at Epoch in this model"`
	LayStatNms      []string          `desc:"names of layers to collect more detailed stats on (avg act, etc)"`
	LayStatNmsHog   []string          `desc:"names of layers to collect more detailed stats on (avg act, etc)"`
	RSA             RSA               `view:"no-inline" desc:"RSA data"`
	NoLearn         bool              `desc:"if NoLearn is true learning off for first segment of sound"`
	SeqCnt          int               `desc:"tracks number of sound sequences per epoch - will vary (i.e. the number of sound files loaded for the epoch)"`
	Wts             string            `desc:"name of weights file saved from previous training"`
	WtsPath         string            `desc:"path to weights file - local, set on command line for server"`
	OpenWts         bool              `desc:"if true load weights from "Wts" file"`
	SaveWts         bool              `desc:"if true save the weights"`
	Pretrain        bool              `desc:"flag for pretrain - default is false"`
	TestType        TestType          `desc:"which variety of test item will be tested - different stats!"`
	Holdout         bool              `desc:"holdout some of the training data for test"`
	HoldoutPct      int               `desc:"percent of items to holdout for testing"`

	// activation values and related for generating similarity matrices
	CatLayActs *etable.Table `view:"no-inline" desc:"super layer activations per category / object"`

	// statistics: note use float64 as that is best for etable.Table
	TrnTrlLog           *etable.Table    `view:"no-inline" desc:"training trial-level log data"`
	TrnTrlLogAll        *etable.Table    `view:"no-inline" desc:"all training trial-level log data (aggregated from MPI)"`
	TrnEpcLog           *etable.Table    `view:"no-inline" desc:"training epoch-level log data"`
	TstTrlLog           *etable.Table    `view:"no-inline" desc:"testing trial-level log data"`
	TstTrlLogAll        *etable.Table    `view:"no-inline" desc:"all testing trial-level log data (aggregated from MPI)"`
	TstEpcLog           *etable.Table    `view:"no-inline" desc:"testing epoch-level log data"`
	TstEpcTidyLog       *etable.Table    `view:"no-inline" desc:"testing epoch-level log data in Tidy format for R stats"`
	RunLog              *etable.Table    `view:"no-inline" desc:"summary log of each run"`
	RunStats            *etable.Table    `view:"no-inline" desc:"aggregate stats on all runs"`
	CVConfusion         etensor.Float64  `view:"no-inline" desc:"confusion matrix for test run CV decoding"`
	NetData             *netview.NetData `view:"-" desc:"net data for recording in nogui mode"`
	TrlErr              float64          `inactive:"+" desc:"1 if trial was error, 0 if correct -- based on SSE = 0 (subject to .5 unit-wise tolerance)"`
	TrlSSE              float64          `inactive:"+" desc:"current trial's sum squared error"`
	TrlAvgSSE           float64          `inactive:"+" desc:"current trial's average sum squared error"`
	TrlCosDiffTRC       []float64        `inactive:"+" desc:"current trial's cosine difference for pulvinar (TRC) layers"`
	EpcCosDiffTRC       []float64        `inactive:"+" desc:"last epoch's average cosine difference for TRC layers (a normalized error measure, maximum of 1 when the minus phase exactly matches the plus)"`
	EpcBtwCosDiffTRC    []float64        `inactive:"+" desc:"last epoch's average cosine difference for between words for TRC layers (a normalized error measure, maximum of 1 when the minus phase exactly matches the plus)"`
	EpcInWordCosDiffTRC []float64        `inactive:"+" desc:"last epoch's average cosine difference for within for TRC layers"`
	EpcPartCosDiffTRC   []float64        `inactive:"+" desc:"last epoch's average cosine difference for 'part words' for TRC layers"`
	EpcWholeCosDiffTRC  []float64        `inactive:"+" desc:"last epoch's average cosine difference for 'whole words' for TRC layers"`

	// intermediary vars for various stats - view:"-"
	SumCosDiffTRC       []float64 `view:"-" inactive:"+" desc:"sum to increment as we go through epoch, per TRC"`
	SumBtwCosDiffTRC    []float64 `view:"-" inactive:"+" desc:"Btw means for trials that fall between words"`
	SumInWordCosDiffTRC []float64 `view:"-" inactive:"+" desc:"Within means for trials that fall between words"`
	SumPartCosDiffTRC   []float64 `view:"-" inactive:"+" desc:"part is for 'part word' cos diff"`
	SumWholeCosDiffTRC  []float64 `view:"-" inactive:"+" desc:"whole is for 'whole word' cos diff"`
	CntErr              int       `view:"-" inactive:"+" desc:"sum of errs to increment as we go through epoch"`
	BtwWordCnt          int       `inactive:"+" desc:"number of trials of epoch that end between words"`
	InWordCnt           int       `inactive:"+" desc:"number of trials of epoch that end within words"`
	PartWordCnt         int       `inactive:"+" desc:"number of trials of epoch that end between words"`
	WholeWordCnt        int       `inactive:"+" desc:"number of trials of epoch that end within words"`
	TestWordsPart       []string  `desc:"all the words for testing"`
	TestWordsWhole      []string  `desc:"the whole words for testing"`

	// tensors for holding various layer state info
	LogActsTsr *etensor.Float32 `view:"-" desc:"for holding layer activations for each exemplar"`

	// state
	NoGui       bool    `view:"-" desc:"if true, runing in no GUI mode"`
	IsRunning   bool    `view:"-" desc:"true if sim is running"`
	StopNow     bool    `view:"-" desc:"flag to stop running"`
	NeedsNewRun bool    `view:"-" desc:"flag to initialize NewRun if last one finished"`
	RndSeeds    []int64 `view:"-" desc:"a list of random seeds to use for each run"`

	// gui
	Win           *gi.Window         `view:"-" desc:"main GUI window"`
	NetView       *netview.NetView   `view:"-" desc:"the network viewer"`
	StructView    *giv.StructView    `view:"-" desc:"the params viewer"`
	ToolBar       *gi.ToolBar        `view:"-" desc:"the master toolbar"`
	TrnEpcPlot    *eplot.Plot2D      `view:"-" desc:"the training epoch plot"`
	TrnTrlPlot    *eplot.Plot2D      `view:"-" desc:"the training trial plot"`
	TstEpcPlot    *eplot.Plot2D      `view:"-" desc:"the testing epoch plot"`
	TstTrlPlot    *eplot.Plot2D      `view:"-" desc:"the test-trial plot"`
	RunPlot       *eplot.Plot2D      `view:"-" desc:"the run plot"`
	PowerGridS    *etview.TensorGrid `view:"-" desc:"power grid view for the current segment"`
	MelFBankGridS *etview.TensorGrid `view:"-" desc:"melfbank grid view for the current segment"`
	PowerGridL    *etview.TensorGrid `view:"-" desc:"power grid view for the current segment"`
	MelFBankGridL *etview.TensorGrid `view:"-" desc:"melfbank grid view for the current segment"`

	Comm    *mpi.Comm `view:"-" desc:"mpi communicator"`
	AllDWts []float32 `view:"-" desc:"buffer of all dwt weight changes -- for mpi sharing"`
	SumDWts []float32 `view:"-" desc:"buffer of MPI summed dwt weight changes"`

	// options
	UseMPI        bool `view:"-" desc:"if true, use MPI to distribute computation across nodes"`
	TestRun       bool `desc:" only for no gui -- run test not train"`
	UseRateSched  bool `desc:"change lrate over epochs using schedule - see LrateSched()"`
	CalcBtwWthin  bool `desc:"if true calc separate cos diff for between vs within"`
	CalcPartWhole bool `desc:"if true calc separate cos diff for part words and whole words"`
	CalcCosDiff   bool `desc:"if true normal cos diff for all trials"`
	SaveSimMat    bool `view:"-" desc:"for command-line run only, save simalarity matrix at end of run"`
	SaveActs      bool `view:"-" desc:"for command-line run only, log activations after each trial"`

	// files
	RunFile           *LogFile `view:"-" desc:"log file"`
	TrnTrlFile        *LogFile `view:"-" desc:"log file"`
	TrnEpcFile        *LogFile `view:"-" desc:"log file"`
	PreTrnEpcFile     *LogFile `view:"-" desc:"log file"`
	TstTrlFile        *LogFile `view:"-" desc:"log file"`
	TstEpcFile        *LogFile `view:"-" desc:"log file"`
	TstEpcTidyFile    *LogFile `view:"-" desc:"log file in "tidy" format for R stats"`
	PreTstEpcFile     *LogFile `view:"-" desc:"log file"`
	PreTstEpcTidyFile *LogFile `view:"-" desc:"log file in "tidy" format for R stats"`
	TrnCondEpcFile    *LogFile `view:"-" desc:"train epc by condition (part/whole word) log file - summary"`
	TstCondEpcFile    *LogFile `view:"-" desc:"test epc by condition (part/whole word) log file - summary"`
	CatActsFile       *LogFile `view:"-" desc:"category x layer activations"`

	saveProcLog       bool `desc:"save logs for every mpi process separately"`
	saveRunLog        bool `desc:"log file for the run"`
	saveTrnTrlLog     bool `desc:"training log file by trial"`
	saveTrnEpcLog     bool `desc:"training log file by epoch"`
	savePreTrnEpcLog  bool `desc:"training log file by epoch"`
	saveTstTrlLog     bool `desc:"testing log file by trial"`
	saveTstEpcLog     bool `desc:"testing log file by epoch"`
	saveTstEpcTidy    bool `desc:"testing log in tidy format for R stats"`
	savePreTstEpcLog  bool `desc:"pretesting log file by epoch"`
	savePreTstEpcTidy bool `desc:"pretesting log in tidy format for R stats"`
	saveTrnCondEpcLog bool `desc:"training log file by epoch by condition"`
	saveTstCondEpcLog bool `desc:"testing log file by epoch by condition"`
	saveActsLog       bool `desc:"log file for neuron activations by layer"`
}

// this registers this Sim Type and gives it properties that e.g.,
// prompt for filename for save methods.
var KiT_Sim = kit.Types.AddType(&Sim{}, SimProps)

// TheSim is the overall state for this simulation
var TheSim Sim

// New creates new blank elements and initializes defaults
func (ss *Sim) New() {
	ss.Net = NewWordNet()

	ss.TrnTrlLog = &etable.Table{}
	ss.TrnTrlLogAll = &etable.Table{}
	ss.TrnEpcLog = &etable.Table{}

	ss.TstTrlLog = &etable.Table{}
	ss.TstEpcLog = &etable.Table{}
	ss.TstEpcTidyLog = &etable.Table{}

	ss.RunLog = &etable.Table{}
	ss.RunStats = &etable.Table{}
	ss.CatLayActs = &etable.Table{}
	ss.RndSeeds = make([]int64, 100) // make enough for plenty of runs
	for i := 0; i < 100; i++ {
		ss.RndSeeds[i] = int64(i) + 1 // exclude 0
	}
	ss.ViewOn = true
	ss.TrainUpdt = leabra.AlphaCycle
	ss.TestUpdt = leabra.AlphaCycle
	ss.TestInterval = -1
	ss.PreTestInterval = -1
	ss.LayStatNms = []string{"CB", "RB", "CBCT", "RBCT", "STS", "STSCT"}
	ss.LayStatNmsHog = []string{"CB", "RB", "CBCT", "RBCT", "CPB", "CPBCT", "RPB", "RPBCT", "STS", "STSCT"}
	ss.NoLearn = false
	ss.SeqCnt = 0
	ss.WtsInterval = 100
	ss.OpenWts = false
	ss.SaveWts = false
	ss.WtsPath = "/Users/rohrlich/gruntdat/wc/blanca/rohrlich/wordseg/jobs/active/"
	ss.Wts = "roh001193/wordseg/WordSeg_Base_000_00000.wts"
	ss.SaveActs = false
	ss.SaveSimMat = false
	ss.TestRun = false
	ss.UseRateSched = false
	ss.CalcCosDiff = true
	ss.CalcBtwWthin = true
	ss.CalcPartWhole = true
	ss.Pretrain = false
	ss.RSA.Interval = -1
	ss.Holdout = false
	ss.HoldoutPct = 0

	// don't save for gui runs
	ss.saveProcLog = false
	ss.saveRunLog = false
	ss.saveTrnTrlLog = false
	ss.saveTrnEpcLog = false
	ss.savePreTrnEpcLog = false
	ss.saveTstTrlLog = false
	ss.saveTstEpcLog = false
	ss.saveTstEpcTidy = false
	ss.savePreTstEpcLog = false
	ss.savePreTstEpcTidy = false
	ss.saveTrnCondEpcLog = false
	ss.saveTstCondEpcLog = false
	ss.saveActsLog = false
}

////////////////////////////////////////////////////////////////////////////////////////////
// Configs

// Config configures all the elements using the standard functions
func (ss *Sim) Config() {
	ss.ConfigEnv()
	ss.Net.Config()
	ss.InitStats()
	ss.ConfigCatLayActs(ss.CatLayActs)

	ss.ConfigTrnTrlLog(ss.TrnTrlLogAll)
	ss.ConfigTrnEpcLog(ss.TrnEpcLog)
	ss.ConfigTrnTrlLog(ss.TrnTrlLog)

	ss.ConfigTstTrlLog(ss.TstTrlLog)
	ss.ConfigTstEpcLog(ss.TstEpcLog)
	ss.ConfigTstEpcTidy(ss.TstEpcTidyLog)

	ss.ConfigRunLog(ss.RunLog)
	ss.ConfigLogFiles()
	ss.Tag = ""
}

func (ss *Sim) ConfigEnv() {
	if ss.MaxRuns == 0 {
		ss.MaxRuns = 1
	}
	if ss.MaxEpcs == 0 {
		ss.MaxEpcs = 2
	}
	if ss.MaxSeqs == 0 {
		ss.MaxSeqs = 2
	}
	if ss.MaxPreEpcs == 0 {
		ss.MaxPreEpcs = 2
	}
	if ss.MaxPreSeqs == 0 {
		ss.MaxPreSeqs = 2
	}

	ss.TrainEnv.DefaultsTrn()
	ss.TrainEnv.Nm = "TrainEnv"
	ss.TrainEnv.Dsc = "training params and state"
	ss.TrainEnv.Validate()
	ss.TrainEnv.Run.Max = 1
	ss.TrainEnv.Epoch.Max = ss.MaxEpcs
	ss.TrainEnv.Sequence.Max = ss.MaxSeqs
	ss.TrainEnv.Trial.Max = 0
	ss.TrainEnv.SndTimit = false

	ss.TestEnv.DefaultsTest()
	ss.TestEnv.Nm = "TestEnv"
	ss.TestEnv.Dsc = "testing params and state"
	ss.TestEnv.Validate()
	ss.TestEnv.Run.Max = 1
	ss.TestEnv.Epoch.Max = 1
	ss.TestEnv.Sequence.Max = -1
	ss.TestEnv.Trial.Max = 0
	ss.TestEnv.SndTimit = false

	ss.PreTrainEnv.DefaultsTrn()
	ss.PreTrainEnv.Nm = "PreTrainEnv"
	ss.PreTrainEnv.Dsc = "pretraining params and state"
	ss.PreTrainEnv.Validate()
	ss.PreTrainEnv.Run.Max = 1
	ss.PreTrainEnv.Epoch.Max = ss.MaxPreEpcs
	ss.PreTrainEnv.Sequence.Max = ss.MaxPreSeqs
	ss.PreTrainEnv.Trial.Max = 0
	ss.PreTrainEnv.SndTimit = false

	ss.PreTestEnv.DefaultsTest()
	ss.PreTestEnv.Nm = "PreTestEnv"
	ss.PreTestEnv.Dsc = "testing params and state"
	ss.PreTestEnv.Validate()
	ss.PreTestEnv.Run.Max = 1
	ss.PreTestEnv.Epoch.Max = 1
	ss.PreTestEnv.Sequence.Max = -1
	ss.PreTestEnv.Trial.Max = 0
	ss.PreTestEnv.SndTimit = false

	run := ss.TrainEnv.Run.Cur
	ss.TrainEnv.Init(run)
	ss.TestEnv.Init(0)
	ss.PreTrainEnv.Init(0)
	ss.PreTestEnv.Init(0)

	if len(ss.TrnList) == 0 {
		ss.TrnList = "CVs_I"
		//ss.TrnList = "TIMIT_ALL_SX_F"
	}
	if len(ss.TstList) == 0 {
		//ss.TstList = "CVs_I"
		ss.TstList = "Holdouts"
	}
	if len(ss.PreTrnList) == 0 {
		ss.PreTrnList = "TIMIT_ALL_SX_F"
	}
	if len(ss.PreTstList) == 0 {
		ss.PreTstList = "Holdouts"
	}
	ss.SetTrainingFiles(ss.TrnList)
	ss.SetTestingFiles(ss.TstList)
	ss.SetPretrainingFiles(ss.PreTrnList)
	ss.SetPretestingFiles(ss.PreTstList)
}

// ConfigLogFiles
func (ss *Sim) ConfigLogFiles() {
	ss.RunFile = &LogFile{}
	ss.RunFile.Name = "run"
	ss.RunFile.Header = true
	ss.RunFile.HeaderWritten = false
	ss.RunFile.Run = true
	ss.RunFile.Epoch = false

	ss.TrnTrlFile = &LogFile{}
	ss.TrnTrlFile.Name = "trnTrl"
	ss.TrnTrlFile.Header = true
	ss.TrnTrlFile.HeaderWritten = false
	ss.TrnTrlFile.Run = true
	ss.TrnTrlFile.Epoch = false

	ss.TrnEpcFile = &LogFile{}
	ss.TrnEpcFile.Name = "trnEpc"
	ss.TrnEpcFile.Header = true
	ss.TrnEpcFile.HeaderWritten = false
	ss.TrnEpcFile.Run = true
	ss.TrnEpcFile.Epoch = false

	ss.PreTrnEpcFile = &LogFile{}
	ss.PreTrnEpcFile.Name = "preTrnEpc"
	ss.PreTrnEpcFile.Header = true
	ss.PreTrnEpcFile.HeaderWritten = false
	ss.PreTrnEpcFile.Run = true
	ss.PreTrnEpcFile.Epoch = false

	ss.TstTrlFile = &LogFile{}
	ss.TstTrlFile.Name = "tstTrl"
	ss.TstTrlFile.Header = true
	ss.TstTrlFile.HeaderWritten = false
	ss.TstTrlFile.Run = true
	ss.TstTrlFile.Epoch = false

	ss.TstEpcFile = &LogFile{}
	ss.TstEpcFile.Name = "tstEpc"
	ss.TstEpcFile.Header = true
	ss.TstEpcFile.HeaderWritten = false
	ss.TstEpcFile.Run = true
	ss.TstEpcFile.Epoch = false

	ss.TstEpcTidyFile = &LogFile{}
	ss.TstEpcTidyFile.Name = "tstEpcTidy"
	ss.TstEpcTidyFile.Header = false // don't write the header!
	ss.TstEpcTidyFile.HeaderWritten = false
	ss.TstEpcTidyFile.Run = true
	ss.TstEpcTidyFile.Epoch = false

	ss.PreTstEpcFile = &LogFile{}
	ss.PreTstEpcFile.Name = "preTstEpc"
	ss.PreTstEpcFile.Header = true
	ss.PreTstEpcFile.HeaderWritten = false
	ss.PreTstEpcFile.Run = true
	ss.PreTstEpcFile.Epoch = false

	ss.PreTstEpcTidyFile = &LogFile{}
	ss.PreTstEpcTidyFile.Name = "preTstEpcTidy"
	ss.PreTstEpcTidyFile.Header = false // don't write the header!
	ss.PreTstEpcTidyFile.HeaderWritten = false
	ss.PreTstEpcTidyFile.Run = true
	ss.PreTstEpcTidyFile.Epoch = false

	ss.TrnCondEpcFile = &LogFile{}
	ss.TrnCondEpcFile.Name = "trnCondEpc"
	ss.TrnCondEpcFile.Header = true
	ss.TrnCondEpcFile.HeaderWritten = false
	ss.TrnCondEpcFile.Run = true
	ss.TrnCondEpcFile.Epoch = false

	ss.TstCondEpcFile = &LogFile{}
	ss.TstCondEpcFile.Name = "tstCondEpc"
	ss.TstCondEpcFile.Header = true
	ss.TstCondEpcFile.HeaderWritten = false
	ss.TstCondEpcFile.Run = true
	ss.TstCondEpcFile.Epoch = false

	ss.CatActsFile = &LogFile{}
	ss.CatActsFile.Name = "catActs"
	ss.CatActsFile.Header = true
	ss.CatActsFile.HeaderWritten = false
	ss.CatActsFile.Run = true
	ss.CatActsFile.Epoch = true // a different file each epoch (based on interval)
}

////////////////////////////////////////////////////////////////////////////////
// 	    Init, utils

// Init restarts the run, and initializes everything, including network weights
// and resets the epoch log table
func (ss *Sim) Init() {
	ss.InitRndSeed()
	ss.StopNow = false
	ss.Net.SetParams("", ss.Net.LogSetParams) // all sheets
	ss.NewRun()
	ss.UpdateView(true) // ToDo: too early - find out why
}

// InitRndSeed initializes the random seed based on current training run number
func (ss *Sim) InitRndSeed() {
	run := ss.TrainEnv.Run.Cur
	rand.Seed(ss.RndSeeds[run])
}

// NewRndSeed gets a new set of random seeds based on current time -- otherwise uses
// the same random seeds for every run
func (ss *Sim) NewRndSeed() {
	rs := time.Now().UnixNano()
	for i := 0; i < 100; i++ {
		ss.RndSeeds[i] = rs + int64(i)
	}
}

// Counters returns a string of the current counter state
// use tabs to achieve a reasonable formatting overall
// and add a few tabs at the end to allow for expansion..
func (ss *Sim) Counters(train bool) string {
	phase := "training"
	if ss.Env == &ss.TestEnv || ss.Env == &ss.PreTestEnv {
		phase = "testing"
	}
	return fmt.Sprintf("Phase:  %v\tRun:  %d\tEpoch:  %d\tSequence:  %d\tTrial:  %d\tCycle:  %d\t\tName:  %v\t\tSegment:  %d\tCV:  %v\tSubSeg:  %d\t\t\t", phase, ss.Env.Run.Cur, ss.Env.Epoch.Cur, ss.Env.Sequence.Cur, ss.Env.Trial.Cur,
		ss.Time.Cycle, ss.Env.TrialName, ss.Env.CurSeg(), ss.Env.CV.Cur, ss.Env.CV.SubSeg)
}

func (ss *Sim) UpdateView(train bool) {
	if ss.NetView != nil && ss.NetView.IsVisible() {
		ss.NetView.Record(ss.Counters(train))
		// note: essential to use Go version of update when called from another goroutine
		ss.NetView.GoUpdate() // note: using counters is significantly slower..
	}
}

////////////////////////////////////////////////////////////////////////////////
// 	    Running the Network, starting bottom-up..

// AlphaCyc runs one alpha-cycle (100 msec, 4 quarters) of processing.
// External inputs must have already been applied prior to calling,
// using ApplyExt method on relevant layers (see TrainTrial, TestTrial).
// If train is true, then learning DWt or WtFmDWt calls are made.
// Handles netview updating within scope of AlphaCycle
func (ss *Sim) AlphaCyc(train bool) {
	viewUpdt := ss.TrainUpdt
	if !train {
		viewUpdt = ss.TestUpdt
	}

	// update prior weight changes at start, so any DWt values remain visible at end
	// you might want to do this less frequently to achieve a mini-batch update
	// in which case, move it out to the TrainTrial method where the relevant
	// counters are being dealt with.
	if train {
		ss.MPIWtFmDWt()
	}

	ss.ApplyToA1(ss.Env)
	net := ss.Net.Net
	net.AlphaCycInit()
	ss.Time.AlphaCycStart()
	for qtr := 0; qtr < 4; qtr++ {
		for cyc := 0; cyc < ss.Time.CycPerQtr; cyc++ {
			if qtr == 0 && cyc == 0 {
				ss.ApplyToR(ss.Env)
			}
			net.Cycle(&ss.Time)
			ss.Time.CycleInc()
			if ss.ViewOn {
				switch viewUpdt {
				case leabra.Cycle:
					ss.UpdateView(train)
				case leabra.FastSpike:
					if (cyc+1)%10 == 0 {
						ss.UpdateView(train)
					}
				}
			}
		}
		net.QuarterFinal(&ss.Time)
		ss.Time.QuarterInc()
		if ss.ViewOn {
			switch {
			case viewUpdt <= leabra.Quarter:
				ss.UpdateView(train)
			case viewUpdt == leabra.Phase:
				if qtr >= 2 {
					ss.UpdateView(train)
				}
			}
		}
		if qtr == 3 { // calc cosine difference
			idx := 0
			for _, ly := range net.Layers {
				if ly.Type() == deep.TRC && ly.IsOff() == false {
					lyLy := ly.(leabra.LeabraLayer).AsLeabra()
					ss.CosDiffStd(lyLy, idx)
					idx += 1
				}
			}
		}
	}

	if train {
		if ss.NoLearn == false || ss.TrainEnv.CurSeg() > 0 { // no learn on first segment of sound - unpredictable
			net.DWt()
		}
	}
	if ss.ViewOn && viewUpdt == leabra.AlphaCycle {
		ss.UpdateView(train)
	}
	ss.TrnTrlPlot.GoUpdate()
}

// ApplyToA1
func (ss *Sim) ApplyToA1(en env.Env) {
	net := ss.Net.Net
	a1s := net.LayerByName("A1").(leabra.LeabraLayer).AsLeabra()
	if a1s != nil {
		a1s.InitExt()
		A1Pat := en.State(a1s.Nm)
		a1s.ApplyExt(A1Pat)
	}
}

// ApplyToR
func (ss *Sim) ApplyToR(en env.Env) {
	net := ss.Net.Net
	rs := net.LayerByName("R").(leabra.LeabraLayer).AsLeabra()
	if rs != nil {
		rs.InitExt()
		rsPat := en.State(rs.Nm)
		rs.ApplyExt(rsPat)
	}
}

// LrateSched implements the learning rate schedule
func (ss *Sim) LrateSched(epc int) {
	net := ss.Net.Net
	q := ss.MaxEpcs / 5
	switch epc {
	case q:
		net.LrateMult(0.5)
		mpi.Printf("dropped lrate to 0.50 of initial lrate at epoch: %d\n", epc)
	case q * 2:
		net.LrateMult(0.25)
		mpi.Printf("dropped lrate to 0.25 of initial lrate at epoch: %d\n", epc)
	case q * 3:
		net.LrateMult(0.125)
		mpi.Printf("dropped lrate to 0.125 of initial lrate at epoch: %d\n", epc)
	}
}

// TrainTrial runs one trial of training using TrainEnv
func (ss *Sim) TrainTrial() {
	net := ss.Net.Net
	//start := time.Now()
	if ss.NeedsNewRun {
		ss.NewRun()
	}

	epcPrvStep, _, _ := ss.TrainEnv.Counter(env.Epoch) // before stepping
	ss.TrainEnv.Step()                                 // the Env encapsulates and manages all counter state

	epc, _, chg := ss.TrainEnv.Counter(env.Epoch)
	if chg {
		ss.LogTrnEpc(ss.TrnEpcLog, ss.TrnEpcFile, ss.saveTrnEpcLog)

		if ss.UseRateSched {
			ss.LrateSched(epc)
		}
		if ss.ViewOn && ss.TrainUpdt > leabra.AlphaCycle {
			ss.UpdateView(true)
		}
		// without the < MaxEpcs test it will start testing and run forever!!
		// note: epc is *next* so won't trigger first time
		// test every epoch up to test interval
		if ss.TestInterval > 0 && ss.TestInterval < ss.MaxEpcs && ss.TrainEnv.Epoch.Cur < ss.TestInterval {
			ss.TestAll(&ss.TestEnv)
		} else if ss.TestInterval > 0 && ss.TestInterval < ss.MaxEpcs && epc%ss.TestInterval == 0 {
			ss.TestAll(&ss.TestEnv)
		}

		ss.Env = &ss.TrainEnv           // reset
		if epcPrvStep > 0 && epc == 0 { // if epc was reset to 0 after reaching max!
			// done with training..
			ss.RunEnd()
			//fmt.Println("Run: ", ss.TrainEnv.Run.Cur)
			if ss.TrainEnv.Run.Incr() { // we are done!
				ss.StopNow = true
				return
			} else {
				ss.NeedsNewRun = true
				return
			}
		}
		if ss.SaveWts && epc%ss.WtsInterval == 0 {
			fnm := ss.WeightsFileName()
			fmt.Printf("Saving Weights to: %v\n", fnm)
			net.SaveWtsJSON(gi.FileName(fnm))
		}
	}

	if ss.TrainEnv.CurSeg() == 0 {
		ss.SeqCnt += 1
	}
	ss.TrialFieldUpdates()

	if ss.CalcBtwWthin {
		ss.TrainEnv.CVLookup()
	}

	//net.InitExt() // clear any existing inputs -- do layer by layer if applying to different layers at different cycles
	ss.AlphaCyc(true)    // train
	ss.TrnTrlStats(true) // accumulate
	ss.TrainEnv.Event.Cur = ss.TrainEnv.CurSeg()
	ss.LogTrnTrl(ss.TrnTrlLog)
	p := ss.TrainEnv.CV.Predictable
	if ss.TrainEnv.Epoch.Cur >= 0 {
		if ss.SaveActs && (p == Fully || p == Partially) {
			if ss.TrainEnv.CV.SubSeg == 0 {
				ss.RecordLayActs(ss.CatLayActs)
			}
		}
	}
	ss.TrialGuiUpdates()
}

// PreTrainTrial runs one trial of pretraining using TrainEnv
func (ss *Sim) PreTrainTrial() {
	//start := time.Now()
	net := ss.Net.Net
	if ss.NeedsNewRun {
		ss.NewRun()
	}

	epcPrvStep, _, _ := ss.PreTrainEnv.Counter(env.Epoch) // before stepping
	ss.PreTrainEnv.Step()                                 // the Env encapsulates and manages all counter state

	// Key to query counters FIRST because current state is in NEXT epoch
	// if epoch counter has changed
	epc, _, chg := ss.PreTrainEnv.Counter(env.Epoch)
	if chg {
		ss.LogTrnEpc(ss.TrnEpcLog, ss.PreTrnEpcFile, ss.savePreTrnEpcLog) // reusing TrnEpcLog - but write to diff file
		if ss.ViewOn && ss.TrainUpdt > leabra.AlphaCycle {
			ss.UpdateView(true)
		}

		if ss.PreTestInterval > 0 && ss.PreTestInterval < ss.MaxPreEpcs && epc%ss.PreTestInterval == 0 {
			ss.TestAll(&ss.PreTestEnv)
		}

		ss.Env = &ss.PreTrainEnv
		if epcPrvStep > 0 && epc == 0 { // if epc was reset to 0 after reaching max!
			// done with training..
			ss.RunEnd()
			if ss.PreTrainEnv.Run.Incr() { // we are done!
				ss.StopNow = true
				return
			} else {
				ss.NeedsNewRun = true
				return
			}
		}
		if ss.SaveWts && epc%ss.WtsInterval == 0 {
			fnm := ss.WeightsFileName()
			fmt.Printf("Saving Weights to: %v\n", fnm)
			net.SaveWtsJSON(gi.FileName(fnm))
		}
	}

	if ss.PreTrainEnv.CurSeg() == 0 {
		ss.SeqCnt += 1
	}
	ss.TrialFieldUpdates()

	//ss.ApplyInputs(ss.Env)
	ss.AlphaCyc(true)    // train
	ss.TrnTrlStats(true) // accumulate
	ss.PreTrainEnv.Event.Cur = ss.PreTrainEnv.CurSeg()
	ss.LogTrnTrl(ss.TrnTrlLog)

	//elapsed := time.Since(start)
	//log.Printf("trial took %v", elapsed)
	ss.TrialGuiUpdates()
}

// TrialGuiUpdates
func (ss *Sim) TrialGuiUpdates() {
	if ss.Env.Trial.Cur == 0 { // make sure the counter text is shown
		if ss.Win != nil {
			vp := ss.Win.WinViewport2D()
			if ss.ToolBar != nil {
				ss.ToolBar.UpdateActions()
			}
			vp.SetNeedsFullRender()
		}
	}
	if !ss.NoGui {
		ss.PowerGridS.UpdateSig()
		ss.MelFBankGridS.UpdateSig()
		ss.PowerGridL.UpdateSig()
		ss.MelFBankGridL.UpdateSig()
	}
}

// TrialFieldUpdates
func (ss *Sim) TrialFieldUpdates() {
	ss.Snd = ss.Env.SndCur
	if ss.StructView != nil {
		ss.StructView.UpdateField("Snd")
	}
}

// RunEnd is called at the end of a run -- save weights, record final log, etc here
func (ss *Sim) RunEnd() {
	net := ss.Net.Net
	ss.LogRun(ss.RunLog)
	if ss.SaveWts {
		fnm := ss.WeightsFileName()
		fmt.Printf("Saving Weights to: %v\n", fnm)
		net.SaveWtsJSON(gi.FileName(fnm))
	}
}

// NewRun intializes a new run of the model, using the TrainEnv.Run counter for the new run value
func (ss *Sim) NewRun() {
	ss.InitRndSeed()
	ss.Time.Reset()
	ss.OpenTrainedWts(ss.Net.Net)
	ss.InitStats()
	ss.TrnTrlLog.SetNumRows(0)
	ss.TrnEpcLog.SetNumRows(0)
	ss.TstTrlLog.SetNumRows(0)
	ss.TstEpcLog.SetNumRows(0)
	ss.TstEpcTidyLog.SetNumRows(0)
	ss.NeedsNewRun = false
}

// OpenTrainedWts
func (ss *Sim) OpenTrainedWts(net *deep.Network) {
	if ss.OpenWts {
		if ss.Wts == "" {
			log.Println("OpenWts is true but variable Wts is empty")
			return
		}
		wf := ss.WtsPath + ss.Wts
		ab, err := ioutil.ReadFile(wf)
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("Loading weights from %v at %v\n", ss.Wts, ss.WtsPath)
		net.ReadWtsJSON(bytes.NewBuffer(ab))
	}
}

// InitStats initializes all the statistics, especially important for the
// cumulative epoch stats -- called at start of new run
func (ss *Sim) InitStats() {
	ss.CntErr = 0

	// clear rest just to make Sim look initialized
	ss.TrlSSE = 0
	ss.TrlAvgSSE = 0

	net := ss.Net // ss.Net.Net is the actual network

	net.TRCLays = []string{}
	net.HidLays = []string{}
	net.SuperLays = []string{"A1", "R"} // A1 and R are input layers - add to super list manually

	for _, ly := range net.Net.Layers {
		if ly.IsOff() {
			continue
		}
		if ly.Type() == emer.Hidden && ly.IsOff() == false {
			ss.Net.SuperLays = append(ss.Net.SuperLays, ly.Name())
		}
		if (ly.Type() == emer.Hidden || ly.Type() == deep.CT) && ly.IsOff() == false {
			ss.Net.HidLays = append(ss.Net.HidLays, ly.Name())
		}
		if ly.Type() == deep.TRC && ly.IsOff() == false {
			ss.Net.TRCLays = append(net.TRCLays, ly.Name())
		}
	}

	nTRC := len(ss.Net.TRCLays)
	if len(ss.TrlCosDiffTRC) != nTRC {
		ss.TrlCosDiffTRC = make([]float64, nTRC) // for each trial regardless of tick/segment
		ss.SumCosDiffTRC = make([]float64, nTRC) // sum over trials
		ss.EpcCosDiffTRC = make([]float64, nTRC) // for each epoch regardless of tick/segment

		ss.SumBtwCosDiffTRC = make([]float64, nTRC)
		ss.SumInWordCosDiffTRC = make([]float64, nTRC)
		ss.EpcBtwCosDiffTRC = make([]float64, nTRC)
		ss.EpcInWordCosDiffTRC = make([]float64, nTRC)

		ss.SumPartCosDiffTRC = make([]float64, nTRC)
		ss.SumWholeCosDiffTRC = make([]float64, nTRC)
		ss.EpcPartCosDiffTRC = make([]float64, nTRC)
		ss.EpcWholeCosDiffTRC = make([]float64, nTRC)
	}

	ss.RSA.Init(net.SuperLays)
	//ss.RSA.SetCats([]string{"b", "d", "g", "h", "k", "l", "m", "n", "p", "r", "s", "t"})
	ss.RSA.SetCats([]string{})
}

// TrialStats computes the trial-level statistics and adds them to the epoch accumulators if
// accum is true.  Note that we're accumulating stats here on the Sim side so the
// core algorithm side remains as simple as possible, and doesn't need to worry about
// different time-scales over which stats could be accumulated etc.
// You can also aggregate directly from log data, as is done for testing stats
func (ss *Sim) TrnTrlStats(accum bool) (sse, avgsse, cosdiff float64) {
	net := ss.Net.Net
	rc := net.LayerByName("STS").(leabra.LeabraLayer).AsLeabra()
	ss.TrlSSE, ss.TrlAvgSSE = rc.MSE(0.5) // 0.5 = per-unit tolerance -- right side of .5

	if ss.TrlSSE > 0 {
		ss.TrlErr = 1
	} else {
		ss.TrlErr = 0
	}
	ss.Env.SetIsPredictable() // call every trial
	ss.TrnTrlStatsTRC(accum)
	return
}

// TstTrlStats computes the trial-level statistics and adds them to the epoch accumulators if accum is true.
func (ss *Sim) TstTrlStats(accum bool) (sse, avgsse, cosdiff float64) {
	if ss.CalcBtwWthin {
		if ss.Env == &ss.PreTestEnv {
			ss.PreTestEnv.SetIsPredictable() // call every trial
		} else {
			ss.TestEnv.SetIsPredictable() // call every trial
		}
	}
	if ss.CalcPartWhole { // if we are testing individual "part" words vs individual "whole" words or specific sequence words
		if ss.Env == &ss.PreTestEnv {
			ss.PreTestEnv.SetIsPartWhole() // call every trial
		} else {
			ss.TestEnv.SetIsPartWhole() // call every trial
		}
	}
	if ss.TestType == SequenceTesting {
		ss.TstTrlStatsTRC(accum)
	} else if ss.TestType == PartWholeTesting {
		ss.PartWholeStatsTrc(accum)
	}
	return
}

// TrnTriStatsTRC computes the trial-level statistics and adds them to the epoch accumulators if accum is true.
// Must run after LogTrnTrl
func (ss *Sim) TrnTrlStatsTRC(accum bool) {
	if accum { // don't accumultate if not learning on first trial of sequence
		for i := range ss.Net.TRCLays {
			ss.SumCosDiffTRC[i] += ss.TrlCosDiffTRC[i] // sum per layer across all trials
			if ss.Env.CV.Ordinal > 0 && (ss.Env.CV.SubSeg == 0) {
				last := ss.Env.CV.Last
				cur := ss.Env.CV.Cur
				if ss.Env.CV.Predictable == Partially && ss.IsTestWordPart(last, cur) {
					if i == 0 { // only count once - not for every layer
						ss.BtwWordCnt++
					}
					ss.SumBtwCosDiffTRC[i] += ss.TrlCosDiffTRC[i]
				} else if ss.Env.CV.Predictable == Fully && ss.IsTestWordWhole(last, cur) {
					if i == 0 { // only count once - not for every layer
						ss.InWordCnt++
					}
					ss.SumInWordCosDiffTRC[i] += ss.TrlCosDiffTRC[i]
				}
			}
		}
	}
}

// TstTrlStatsTrc computes the trial-level statistics and adds them to the epoch accumulators if accum is true
// Must run after LogTstTrl
func (ss *Sim) TstTrlStatsTRC(accum bool) {
	if accum {
		for i := range ss.Net.TRCLays {
			ss.SumCosDiffTRC[i] += ss.TrlCosDiffTRC[i]            // sum per layer across all trials
			if ss.Env.CV.Ordinal > 0 && (ss.Env.CV.SubSeg == 0) { // only collect stat for first segment of sound
				last := ss.Env.CV.Last
				cur := ss.Env.CV.Cur
				if ss.Env.CV.Predictable == Partially && ss.IsTestWordPart(last, cur) {
					if i == 0 { // only count once - not for every layer
						ss.BtwWordCnt++
					}
					ss.SumBtwCosDiffTRC[i] += ss.TrlCosDiffTRC[i]
				} else if ss.Env.CV.Predictable == Fully && ss.IsTestWordWhole(last, cur) {
					if i == 0 { // only count once - not for every layer
						ss.InWordCnt++
					}
					ss.SumInWordCosDiffTRC[i] += ss.TrlCosDiffTRC[i]
				}
			}
		}
	}
}

// PartWholeStatsTrc computes the trial-level statistics for runs where the sequences are part words and whole words
func (ss *Sim) PartWholeStatsTrc(accum bool) {
	// == 1 because we only look at the 2nd CV and only the first segment of the 2nd CV
	if ss.Env.CV.Ordinal == 1 && ss.Env.CV.SubSeg == 0 {
		if ss.Env.CV.Word == PartWord {
			ss.PartWordCnt++
		} else if ss.Env.CV.Word == WholeWord {
			ss.WholeWordCnt++
		}

		if accum {
			for i := range ss.Net.TRCLays {
				if ss.Env.CV.Word == PartWord {
					ss.SumPartCosDiffTRC[i] += ss.TrlCosDiffTRC[i]
				} else if ss.Env.CV.Word == WholeWord {
					ss.SumWholeCosDiffTRC[i] += ss.TrlCosDiffTRC[i]
				}
			}
		}
	}
}

// EpochStatsTRC computes the epoch-level statistics for TRC layers
// nt is the number of trials
func (ss *Sim) EpochStatsTRC(nt float64) {
	if ss.NoLearn {
		nt = nt - float64(ss.SeqCnt)
	}
	for i := range ss.Net.TRCLays {
		ss.EpcCosDiffTRC[i] = ss.SumCosDiffTRC[i] / nt
		ss.SumCosDiffTRC[i] = 0

		ss.EpcBtwCosDiffTRC[i] = ss.SumBtwCosDiffTRC[i] / float64(ss.BtwWordCnt)
		ss.SumBtwCosDiffTRC[i] = 0
		ss.EpcInWordCosDiffTRC[i] = ss.SumInWordCosDiffTRC[i] / float64(ss.InWordCnt)
		ss.SumInWordCosDiffTRC[i] = 0

		ss.EpcPartCosDiffTRC[i] = ss.SumPartCosDiffTRC[i] / float64(ss.PartWordCnt)
		ss.SumPartCosDiffTRC[i] = 0
		ss.EpcWholeCosDiffTRC[i] = ss.SumWholeCosDiffTRC[i] / float64(ss.WholeWordCnt)
		ss.SumWholeCosDiffTRC[i] = 0

	}
	ss.BtwWordCnt = 0 // reset for next epoch
	ss.InWordCnt = 0

	ss.PartWordCnt = 0
	ss.WholeWordCnt = 0
}

// HogDead computes the proportion of units in given layer name with ActAvg over hog thr
// and under dead threshold
func (ss *Sim) HogDead(lnm string) (hog, dead float64) {
	net := ss.Net.Net

	ly := net.LayerByName(lnm).(leabra.LeabraLayer).AsLeabra()
	n := len(ly.Neurons)
	for ni := range ly.Neurons {
		nrn := &ly.Neurons[ni]
		if nrn.ActAvg > 0.3 {
			hog += 1
		} else if nrn.ActAvg < 0.01 {
			dead += 1
		}
	}
	hog /= float64(n)
	dead /= float64(n)
	return
}

// RecordLayActs records the minus phase activations for named layer
func (ss *Sim) RecordLayActs(dt *etable.Table) {
	if ss.Env.CV.Cur == "ss" {
		fmt.Println("RecordLayActs - ss!!!!!! - should not happen")
		return
	}
	net := ss.Net.Net

	if ss.LogActsTsr == nil {
		ss.LogActsTsr = &etensor.Float32{}
	}

	exists := false
	erow := -1
	cv := ss.Env.CV.Cur
	seg := strconv.Itoa(ss.Env.CV.SubSeg)
	for r := 0; r < dt.Rows; r++ {
		if dt.CellString("CV", r) == cv && dt.CellString("Seg", r) == seg {
			exists = true
			erow = r
			break
		}
	}

	if exists == true { // we have already recording an instance of this CV, phoneme, phone, etc
		cnt := dt.CellFloat("Count", erow) + 1
		dt.SetCellFloat("Count", erow, cnt)
		for _, lyNm := range ss.Net.SuperLays {
			ly := net.LayerByName(lyNm).(leabra.LeabraLayer).AsLeabra()
			ss.LogActsTsr.SetShape(ly.Shp.Shp, nil, nil)
			ly.UnitValsTensor(ss.LogActsTsr, "ActM") // get minus phase act
			colname := lyNm
			_, err := dt.ColByNameTry(colname)
			if err != nil {
				log.Println("LogActs: col not found")
				return
			}
			ct := dt.CellTensor(lyNm, erow)
			for j, vl := range ss.LogActsTsr.Values {
				val := float64(vl)
				x := ct.FloatVal1D(j)
				if !ss.LogActsTsr.IsNull1D(j) && !math.IsNaN(val) {
					nval := val + x
					ss.LogActsTsr.SetFloat1D(j, nval)
				}
			}
			dt.SetCellTensor(lyNm, erow, ss.LogActsTsr)
		}
	} else {
		dt.AddRows(1)
		row := dt.Rows - 1
		dt.SetCellString("Seg", row, strconv.Itoa(ss.Env.CV.SubSeg))
		dt.SetCellString("CV", row, ss.Env.CV.Cur)
		c := ""
		if ss.TrainEnv.SndTimit == false { // i.e. we are training consonant vowels not phones
			c = string(ss.Env.CV.Cur[0])
		}
		dt.SetCellString("Cons", row, c)
		dt.SetCellString("MannerCat", row, MannerCats[ss.Env.CV.Cur])

		// ToDo: what are the place categories for all the phones
		if ss.TrainEnv.SndTimit == false { // i.e. we are training consonant vowels not phones
			dt.SetCellString("PlaceCat", row, PlaceCats[c])
		}
		for _, lyNm := range ss.Net.SuperLays {
			ly := net.LayerByName(lyNm).(leabra.LeabraLayer).AsLeabra()
			ss.LogActsTsr.SetShape(ly.Shp.Shp, nil, nil)
			ly.UnitValsTensor(ss.LogActsTsr, "ActM") // get minus phase act
			colname := lyNm
			_, err := dt.ColByNameTry(colname)
			if err != nil {
				log.Println("LogActs: col not found")
				return
			}
			dt.SetCellTensor(lyNm, row, ss.LogActsTsr)
		}
	}
	//s := MannerCats[c] + "/" + c
	//fmt.Println(s)
}

// RSAAnal does a bit of preprocessing and then calls the RSA code
func (ss *Sim) RSAAnal(acts *etable.Table, layNms []string) {
	// calculate the mean activation values for each sound for which activations were recorded (instance count varies)
	dt := ss.CatLayActs
	for r := 0; r < ss.CatLayActs.Rows; r++ {
		cnt := dt.CellFloat("Count", r)
		if cnt > 1 {
			for _, lyNm := range ss.Net.SuperLays {
				_, err := dt.ColByNameTry(lyNm)
				if err != nil {
					log.Println("LogActs: col not found")
					return
				}
				ct := dt.CellTensor(lyNm, r).(*etensor.Float32)
				for j, vl := range ct.Values {
					val := float64(vl)
					if !math.IsNaN(val) {
						nval := val / cnt
						ct.SetFloat1D(j, nval)
					}
				}
			}
		}
	}
	ss.RSA.StatsFmActs(ss.CatLayActs, ss.Net.SuperLays)

}

// CosDiffStd - use this if not computing cosine difference directly from activations
// see CosDiffFmActs
func (ss *Sim) CosDiffStd(ly *leabra.Layer, idx int) {
	ss.TrlCosDiffTRC[idx] = float64(ly.CosDiff.Cos)
}

// TrainSequence runs training trials for the remainder of this sequence
func (ss *Sim) TrainSequence() {
	ss.StopNow = false
	curSeq := ss.TrainEnv.Sequence.Cur
	for {
		ss.TrainTrial()
		if ss.StopNow || ss.TrainEnv.Sequence.Cur != curSeq {
			break
		}
	}
	ss.Stopped()
}

// TrainEpoch runs training trials for remainder of this epoch
func (ss *Sim) TrainEpoch() {
	ss.StopNow = false
	curEpc := ss.TrainEnv.Epoch.Cur
	for {
		ss.TrainTrial()
		if ss.StopNow || ss.TrainEnv.Epoch.Cur != curEpc {
			break
		}
	}
	ss.Stopped()
}

// TrainRun runs training trials for remainder of run
func (ss *Sim) TrainRun() {
	ss.StopNow = false
	curRun := ss.TrainEnv.Run.Cur
	for {
		ss.TrainTrial()
		if ss.StopNow || ss.TrainEnv.Run.Cur != curRun {
			break
		}
	}
	ss.Stopped()
}

func (ss *Sim) TrainInit() {
	ss.Env = &ss.TrainEnv
	ss.Env.LoadSoundsAndData()
	if ss.Holdout == true { // holdout some files for testing - must have unique name!
		r := rand.Intn(99999)
		ss.TestEnv.SndList = "testHoldouts_" + ss.HoldoutID + "_" + strconv.Itoa(r) + ".txt"
		ss.TrainEnv.SplitSndFiles(ss.TestEnv.SndList)
	}
	ss.TrainEnv.Epoch.Max = ss.MaxEpcs
	ss.TrainEnv.Sequence.Max = ss.MaxSeqs
	ss.CalcBtwWthin = true
	ss.CalcPartWhole = false
}

// Train runs the full training from this point onward
func (ss *Sim) Train() {
	if ss.Env != &ss.TrainEnv {
		//fmt.Println("We weren't training so calling Init()")
		//ss.Init()
	}
	if ss.TrainEnv.Trial.Cur < 0 {
		ss.TrainInit()
	}
	ss.StopNow = false

	for {
		ss.TrainTrial()
		if ss.StopNow {
			break
		}
	}
	ss.Stopped()
}

func (ss *Sim) PreTrainInit() {
	//ss.Pretrain = true
	ss.Env = &ss.PreTrainEnv
	ss.PreTrainEnv.Silence = true
	ss.PreTrainEnv.LoadSoundsAndData()
	ss.TrainEnv.LoadSoundsAndData() // load these now so we can get the test holdouts for pretesting!

	if ss.Holdout == true { // holdout some files for testing - must have unique name!
		r := rand.Intn(99999)
		ss.PreTestEnv.SndList = "testHoldouts_" + ss.HoldoutID + "_" + strconv.Itoa(r) + ".txt"
		ss.TrainEnv.SplitSndFiles(ss.PreTestEnv.SndList)
	}
	ss.PreTrainEnv.Run.Max = 1
	ss.PreTrainEnv.Epoch.Max = ss.MaxPreEpcs
	ss.PreTrainEnv.Sequence.Max = ss.MaxPreSeqs

	ss.CalcBtwWthin = false
	ss.CalcPartWhole = false
}

// PreTrain runs pre-training, saves weights to PreTrainWts
func (ss *Sim) PreTrain() {
	if ss.Env != &ss.PreTrainEnv {
		fmt.Println("We weren't pretraining so calling Init()")
		//ss.Init()
	}
	if ss.PreTrainEnv.Trial.Cur < 0 {
		ss.PreTrainInit()
	}
	ss.StopNow = false

	//curRun := ss.PreTrainEnv.Run.Cur
	for {
		ss.PreTrainTrial()
		if ss.StopNow {
			//if ss.StopNow || ss.PreTrainEnv.Run.Cur != curRun {
			break
		}
	}
	b := &bytes.Buffer{}
	ss.Net.Net.WriteWtsJSON(b)
	ss.Stopped()
}

// SaveWeights saves the network weights -- when called with giv.CallMethod
func (ss *Sim) SaveWeights(filename gi.FileName) {
	// it will auto-prompt for filename
	net := ss.Net.Net

	net.SaveWtsJSON(filename)
}

// Stop tells the sim to stop running
func (ss *Sim) Stop() {
	ss.StopNow = true
}

// Stopped is called when a run method stops running -- updates the IsRunning flag and toolbar
func (ss *Sim) Stopped() {
	ss.IsRunning = false
	if ss.Win != nil {
		vp := ss.Win.WinViewport2D()
		if ss.ToolBar != nil {
			ss.ToolBar.UpdateActions()
		}
		vp.SetNeedsFullRender()
	}
}

////////////////////////////////////////////////////////////////////////////////////////////
// Testing

// TestInit
func (ss *Sim) TestInit() {
	//ss.Env = &ss.TestEnv
	ss.Env.LoadSoundsAndData()
	ss.Env.Init(ss.Env.Run.Cur)
	ss.Env.Run.Max = 1
	ss.Env.Epoch.Max = 1
	ss.Env.Sequence.Max = len(ss.Env.SndFiles)
	ss.Env.Silence = true
	ss.CalcBtwWthin = true
}

// TestTrial runs one trial of testing -- always sequentially presented inputs
func (ss *Sim) TestTrial(epcFile *LogFile, epcFileTidy *LogFile) {
	if ss.Env.Sequence.Cur == 0 && ss.Env.Trial.Cur == -1 {
		ss.TestInit()
	}

	epcPrvStep, _, _ := ss.Env.Counter(env.Epoch) // before stepping
	ss.Env.Step()

	epc, _, chg := ss.Env.Counter(env.Epoch)
	if chg {
		if ss.ViewOn && ss.TestUpdt > leabra.AlphaCycle {
			ss.UpdateView(true)
		}
		// log file reused for pretest and test but separate files
		ss.LogTstEpc(ss.TstEpcLog, epcFile)
		ss.LogTstEpcTidy(ss.TstEpcTidyLog, epcFileTidy)
		if epcPrvStep > 0 && epc == 0 { // if epc was reset to 0 after reaching max!
			// done with testing
			ss.RunEnd()
			if ss.TrainEnv.Run.Incr() { // we are done!
				ss.StopNow = true
				return
			} else {
				ss.NeedsNewRun = true
				return
			}
		}
	}

	if ss.Env.CurSeg() == 0 {
		ss.SeqCnt += 1
	}
	ss.TrialFieldUpdates()

	if ss.CalcBtwWthin {
		ss.Env.CVLookup()
	}

	ss.AlphaCyc(false)   // !train
	ss.TstTrlStats(true) // !accumulate
	ss.Env.Event.Cur = ss.Env.CurSeg()
	ss.LogTstTrl(ss.TstTrlLog)
	p := ss.TrainEnv.CV.Predictable
	if ss.SaveActs && (p == Fully || p == Partially) {
		if ss.Env.CV.SubSeg == 0 {
			ss.RecordLayActs(ss.CatLayActs)
		}
	}
	ss.TrialGuiUpdates()

	if ss.NetData != nil { // offline record net data from testing, just final state
		ss.NetData.Record(ss.Counters(false))
	}
}

// TestAll runs through the full set of testing items, has stop running = false at end -- for gui
func (ss *Sim) TestAll(env *WEEnv) {
	if env == &ss.PreTestEnv {
		ss.Env = &ss.PreTestEnv
		ss.TestInit()
		for {
			ss.TestTrial(ss.PreTstEpcFile, ss.PreTstEpcTidyFile) // return on change -- don't wrap
			_, _, chg := ss.Env.Counter(env.Epoch.Scale)
			if chg || ss.StopNow {
				break
			}
		}
	} else {
		ss.Env = &ss.TestEnv
		ss.TestInit()
		for {
			ss.TestTrial(ss.TstEpcFile, ss.TstEpcTidyFile) // return on change -- don't wrap
			_, _, chg := ss.Env.Counter(env.Epoch.Scale)
			if chg || ss.StopNow {
				break
			}
		}
	}
}

// RunTestAll runs through the full set of testing items, has stop running = false at end -- for gui
func (ss *Sim) RunTestAll() {
	ss.StopNow = false
	ss.TestAll(&ss.TestEnv)
	ss.Stopped()

	if ss.NoGui {
		os.Exit(0)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Logging

// RunName returns a name for this run that combines Tag and Params -- add this to
// any file names that are saved.
func (ss *Sim) RunName() string {
	rn := ""
	if ss.Tag != "" {
		rn += ss.Tag + "_"
	}
	rn += ss.Net.ParamsName()
	if ss.StartRun > 0 {
		rn += fmt.Sprintf("_%03d", ss.StartRun)
	}
	return rn
}

// RunEpochName returns a string with the run and epoch numbers with leading zeros, suitable
// for using in weights file names.  Uses 3, 5 digits for each.
func (ss *Sim) RunEpochName(run, epc int) string {
	return fmt.Sprintf("%03d_%05d", run, epc)
}

// WeightsFileName returns default current weights file name
func (ss *Sim) WeightsFileName() string {
	net := ss.Net.Net

	return net.Nm + "_" + ss.RunName() + "_" + ss.RunEpochName(ss.TrainEnv.Run.Cur, ss.TrainEnv.Epoch.Cur) + ".wts"
}

// ActsFileName returns default current acts file name
func (ss *Sim) ActsFileName() string {
	net := ss.Net.Net

	return net.Nm + "_" + ss.RunName() + "_" + ss.RunEpochName(ss.TrainEnv.Run.Cur, ss.TrainEnv.Epoch.Cur) + ".csv"
}

func (ss *Sim) CreateLogFile(lf LogFile) *os.File {
	net := ss.Net.Net
	lognm := ""
	if lf.Run == true && lf.Epoch == false {
		lognm = net.Nm + "_" + ss.RunName() + "_" + lf.Name
	} else if lf.Run == true && lf.Epoch == true {
		lognm = net.Nm + "_" + ss.RunName() + "_" + strconv.Itoa(ss.Env.Epoch.Cur) + "_" + lf.Name
	} else {
		lognm = net.Nm + "_" + lf.Name // probably never used
	}

	if mpi.WorldRank() > 0 {
		lognm += fmt.Sprintf("_%d", mpi.WorldRank())
	}
	lognm += ".tsv"

	f, err := os.Create(lognm)
	if err != nil {
		log.Println(err)
		f = nil
	} else {
		mpi.Printf("Saving epoch log to: %v\n", lognm)
	}
	return f
}

//////////////////////////////////////////////
//  TrnEpcLog

func (ss *Sim) ConfigTrnEpcLog(dt *etable.Table) {
	dt.SetMetaData("name", "TrnEpcLog")
	dt.SetMetaData("desc", "Train epoch log")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	sch := etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Epoch", etensor.INT64, nil, nil},
		{"AvgSSE", etensor.FLOAT64, nil, nil},
		{"PctErr", etensor.FLOAT64, nil, nil},
		{"PctCor", etensor.FLOAT64, nil, nil},
	}

	for _, lnm := range ss.Net.TRCLays {
		if ss.CalcBtwWthin {
			sch = append(sch, etable.Column{lnm + " CosDiff_Btw", etensor.FLOAT64, nil, nil})
			sch = append(sch, etable.Column{lnm + " CosDiff_InWord", etensor.FLOAT64, nil, nil})
		}
		if ss.CalcCosDiff {
			sch = append(sch, etable.Column{lnm + " CosDiff", etensor.FLOAT64, nil, nil})
		}
	}

	for _, lnm := range ss.LayStatNms {
		sch = append(sch, etable.Column{lnm + " ActAvg", etensor.FLOAT64, nil, nil})
	}
	for _, lnm := range ss.LayStatNmsHog {
		sch = append(sch, etable.Column{lnm + " Hog", etensor.FLOAT64, nil, nil})
		sch = append(sch, etable.Column{lnm + " Dead", etensor.FLOAT64, nil, nil})
	}
	dt.SetFromSchema(sch, 1)
}

// LogTrnEpc adds data from current epoch to the TrnEpcLog table.
// computes epoch averages prior to logging.
func (ss *Sim) LogTrnEpc(dt *etable.Table, logFile *LogFile, save bool) {
	net := ss.Net

	row := dt.Rows
	dt.SetNumRows(row + 1)

	// use Env not TrainEnv because it could be pretraining!
	epc := ss.Env.Epoch.Prv          // this is triggered by increment so use previous value
	nt := float64(ss.TrnTrlLog.Rows) // number of trials in view

	trl := ss.TrnTrlLog
	if ss.UseMPI {
		empi.GatherTableRows(ss.TrnTrlLogAll, ss.TrnTrlLog, ss.Comm)
		trl = ss.TrnTrlLogAll
	}

	if ss.RSA.Interval > 0 && (epc%ss.RSA.Interval) == 0 {
		if ss.saveActsLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
			// this one is special because there is a new file for each epoch
			// so we create it here rather than once at start of run
			ss.CatActsFile.File = ss.CreateLogFile(*ss.CatActsFile)
			if ss.CatActsFile.File != nil {
				defer ss.CatActsFile.File.Close()
			} else {
				log.Println("CatActsFile creation failed")
			}
		}

		ss.RSAAnal(ss.CatLayActs, net.SuperLays)
		ss.CatLayActs.WriteCSV(ss.CatActsFile.File, etable.Tab, true)
		ss.CatLayActs = nil
		ss.CatLayActs = &etable.Table{}
		ss.ConfigCatLayActs(ss.CatLayActs)
	}

	tix := etable.NewIdxView(trl)
	pcterr := agg.Mean(tix, "Err")[0]

	ss.EpochStatsTRC(nt)
	dt.SetCellFloat("Run", row, float64(ss.TrainEnv.Run.Cur))
	dt.SetCellFloat("Epoch", row, float64(epc))
	dt.SetCellFloat("AvgSSE", row, agg.Mean(tix, "AvgSSE")[0])
	dt.SetCellFloat("PctErr", row, pcterr)
	dt.SetCellFloat("PctCor", row, 1-agg.Mean(tix, "Err")[0])
	ss.SeqCnt = 0

	for i, lnm := range ss.Net.TRCLays {
		if ss.CalcBtwWthin {
			dt.SetCellFloat(lnm+" CosDiff_Btw", row, float64(ss.EpcBtwCosDiffTRC[i]))
			dt.SetCellFloat(lnm+" CosDiff_InWord", row, float64(ss.EpcInWordCosDiffTRC[i]))
		}
		if ss.CalcCosDiff {
			dt.SetCellFloat(lnm+" CosDiff", row, float64(ss.EpcCosDiffTRC[i]))
		}
	}

	for _, lnm := range ss.LayStatNms {
		ly, err := net.Net.LayerByNameTry(lnm)
		if err == nil && ly.IsOff() == false {
			lyl := ly.(leabra.LeabraLayer).AsLeabra()
			dt.SetCellFloat(lyl.Nm+" ActAvg", row, float64(lyl.Pools[0].ActAvg.ActPAvgEff))
		}
	}

	for _, lnm := range ss.LayStatNmsHog {
		ly, err := net.Net.LayerByNameTry(lnm)
		if err == nil && ly.IsOff() == false {
			lyl := ly.(leabra.LeabraLayer).AsLeabra()
			hog, dead := ss.HogDead(lnm)
			dt.SetCellFloat(lyl.Nm+" Hog", row, hog)
			dt.SetCellFloat(lyl.Nm+" Dead", row, dead)
		}
	}

	// note: essential to use Go version of update when called from another goroutine
	ss.TrnEpcPlot.GoUpdate()

	if save == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if logFile.Header == true && logFile.HeaderWritten == false {
			dt.WriteCSVHeaders(logFile.File, etable.Tab)
			logFile.HeaderWritten = true
		}
		err := dt.WriteCSVRow(logFile.File, row, etable.Tab)
		if err != nil {
			log.Println("Error writing log: ", logFile.Name)
		}
	}
	ss.TrnTrlLog.SetNumRows(0)
}

func (ss *Sim) ConfigTrnEpcPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Train epoch plot"
	plt.Params.XAxisCol = "Epoch"
	plt.Params.Scale = 3
	plt.Params.LineWidth = 1.5
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Run", false, true, 0, false, 0)
	plt.SetColParams("Epoch", false, true, 0, false, float64(ss.MaxEpcs))

	for _, lnm := range ss.Net.TRCLays {
		if ss.CalcBtwWthin {
			plt.SetColParams(lnm+" CosDiff_Btw", true, true, 0, true, 1)
			plt.SetColParams(lnm+" CosDiff_InWord", true, true, 0, true, 1)
		}
		if ss.CalcCosDiff {
			plt.SetColParams(lnm+" CosDiff", false, true, 0, true, 1)
		}
	}

	for _, lnm := range ss.LayStatNmsHog {
		plt.SetColParams(lnm+" Hog", false, true, 0, true, 1)
		plt.SetColParams(lnm+" Dead", false, true, 0, true, 1)
	}
	return plt
}

//////////////////////////////////////////////
//  TrnTrlLog

func (ss *Sim) ConfigTrnTrlLog(dt *etable.Table) {
	dt.SetMetaData("name", "TrnTrlLog")
	dt.SetMetaData("desc", "Train trial log")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	sch := etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Epoch", etensor.INT64, nil, nil},
		{"Trial", etensor.INT64, nil, nil},
		{"Err", etensor.FLOAT64, nil, nil},
		{"SSE", etensor.FLOAT64, nil, nil},
		{"AvgSSE", etensor.FLOAT64, nil, nil},
		{"Segment", etensor.FLOAT64, nil, nil},
	}

	for _, lnm := range ss.Net.TRCLays {
		if ss.CalcBtwWthin {
			sch = append(sch, etable.Column{lnm + " CosDiff_Btw", etensor.FLOAT64, nil, nil})
			sch = append(sch, etable.Column{lnm + " CosDiff_InWord", etensor.FLOAT64, nil, nil})
		}
		if ss.CalcCosDiff {
			sch = append(sch, etable.Column{lnm + " CosDiff", etensor.FLOAT64, nil, nil})
		}
	}
	dt.SetFromSchema(sch, 0)
}

// LogTrnTrl adds data from current trial to the TrnTrlLog table.
func (ss *Sim) LogTrnTrl(dt *etable.Table) {
	row := dt.Rows
	if dt.Rows <= row {
		dt.SetNumRows(row + 1)
	}

	dt.SetCellFloat("Run", row, float64(ss.TrainEnv.Run.Cur))
	dt.SetCellFloat("Epoch", row, float64(ss.TrainEnv.Epoch.Cur))
	dt.SetCellFloat("Trial", row, float64(ss.TrainEnv.Trial.Cur))
	dt.SetCellFloat("Segment", row, float64(ss.TrainEnv.CurSeg())*.01)
	dt.SetCellFloat("Err", row, ss.TrlErr)
	dt.SetCellFloat("SSE", row, ss.TrlSSE)
	dt.SetCellFloat("AvgSSE", row, ss.TrlAvgSSE)

	// are we within a word or at start of word
	if ss.CalcBtwWthin {
		if ss.TrainEnv.CV.SubSeg == 0 { // only saving stat for first segment of CV
			last := ss.TrainEnv.CV.Last
			cur := ss.TrainEnv.CV.Cur
			if ss.TrainEnv.CV.Predictable == Partially && ss.IsTestWordPart(last, cur) {
				for i, lnm := range ss.Net.TRCLays {
					dt.SetCellFloat(lnm+" CosDiff_Btw", row, float64(ss.TrlCosDiffTRC[i]))
					dt.SetCellFloat(lnm+" CosDiff_InWord", row, float64(0))
				}
			} else if ss.TrainEnv.CV.Predictable == Fully && ss.IsTestWordWhole(last, cur) {
				for i, lnm := range ss.Net.TRCLays {
					dt.SetCellFloat(lnm+" CosDiff_InWord", row, float64(ss.TrlCosDiffTRC[i]))
					dt.SetCellFloat(lnm+" CosDiff_Btw", row, float64(0))
				}
			} else {
				for _, lnm := range ss.Net.TRCLays {
					dt.SetCellFloat(lnm+" CosDiff_InWord", row, float64(0))
					dt.SetCellFloat(lnm+" CosDiff_Btw", row, float64(0))
					//fmt.Println(last, cur)
				}
			}
		}
		if ss.CalcCosDiff {
			for i, lnm := range ss.Net.TRCLays {
				dt.SetCellFloat(lnm+" CosDiff", row, float64(ss.TrlCosDiffTRC[i]))
			}
		}
	}
	ss.TrnTrlPlot.GoUpdate()

	if ss.saveTrnTrlLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.TrnTrlFile.Header == true && ss.TrnTrlFile.HeaderWritten == false {
			dt.WriteCSVHeaders(ss.TrnTrlFile.File, etable.Tab)
			ss.TrnTrlFile.HeaderWritten = true
		}
		err := dt.WriteCSVRow(ss.TrnTrlFile.File, row, etable.Tab)
		if err != nil {
			log.Println("Error writing log: ", ss.TrnTrlFile.Name)
		}
	}
}

// ConfigTrnTrlPlot
func (ss *Sim) ConfigTrnTrlPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Train trial plot"
	plt.Params.XAxisCol = "Trial"
	plt.Params.Scale = 3
	plt.Params.LineWidth = 1.5
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Run", false, true, 0, false, 0)
	plt.SetColParams("Epoch", false, true, 0, false, 0)
	plt.SetColParams("Trial", false, true, 0, false, 0)
	plt.SetColParams("Segment", false, true, 0, false, 0)

	on := false
	for _, lnm := range ss.Net.TRCLays {
		if lnm == "STS" {
			on = true
		}
		if ss.CalcBtwWthin {
			plt.SetColParams(lnm+" CosDiff_Btw", on, true, 0, true, 1)
			plt.SetColParams(lnm+" CosDiff_InWord", on, true, 0, true, 1)
		}
		if ss.CalcCosDiff {
			plt.SetColParams(lnm+" CosDiff", false, true, 0, true, 1)
		}
		on = false
	}
	return plt
}

//////////////////////////////////////////////
//  TstTrlLog

func (ss *Sim) ConfigTstTrlLog(dt *etable.Table) {
	dt.SetMetaData("name", "TstTrlLog")
	dt.SetMetaData("desc", "Test trial log")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	sch := etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Epoch", etensor.INT64, nil, nil},
		{"Trial", etensor.INT64, nil, nil},
		{"Segment", etensor.FLOAT64, nil, nil},
		{"CosDiff", etensor.FLOAT64, nil, nil},
	}

	for _, lnm := range ss.Net.TRCLays {
		if ss.CalcBtwWthin {
			sch = append(sch, etable.Column{lnm + " CosDiff_Btw", etensor.FLOAT64, nil, nil})
			sch = append(sch, etable.Column{lnm + " CosDiff_InWord", etensor.FLOAT64, nil, nil})
		}
		if ss.CalcPartWhole {
			sch = append(sch, etable.Column{lnm + " CosDiff_WholeWord", etensor.FLOAT64, nil, nil})
			sch = append(sch, etable.Column{lnm + " CosDiff_PartWord", etensor.FLOAT64, nil, nil})
		}
		if ss.CalcCosDiff {
			sch = append(sch, etable.Column{lnm + " CosDiff", etensor.FLOAT64, nil, nil})
		}
	}
	dt.SetFromSchema(sch, 0)
}

// LogTstTrl adds data from current trial to the TstTrlLog table.
// log always contains number of testing items
func (ss *Sim) LogTstTrl(dt *etable.Table) {
	row := dt.Rows
	if dt.Rows <= row {
		dt.SetNumRows(row + 1)
	}

	dt.SetCellFloat("Run", row, float64(ss.TestEnv.Run.Cur))
	dt.SetCellFloat("Epoch", row, float64(ss.TestEnv.Epoch.Cur))
	dt.SetCellFloat("Trial", row, float64(ss.TestEnv.Trial.Cur))
	dt.SetCellFloat("Segment", row, float64(ss.TestEnv.CurSeg())*.01)

	// are we within a word or at start of word
	if ss.TestType == SequenceTesting && ss.CalcBtwWthin { // only saving stat for first segment of CV
		if ss.TestEnv.CV.SubSeg == 0 {
			last := ss.TestEnv.CV.Last
			cur := ss.TestEnv.CV.Cur
			if ss.TestEnv.CV.Predictable == Partially && ss.IsTestWordPart(last, cur) {
				for i, lnm := range ss.Net.TRCLays {
					dt.SetCellFloat(lnm+" CosDiff_Btw", row, float64(ss.TrlCosDiffTRC[i]))
					dt.SetCellFloat(lnm+" CosDiff_InWord", row, float64(0))
				}
			} else if ss.TestEnv.CV.Predictable == Fully && ss.IsTestWordWhole(last, cur) {
				for i, lnm := range ss.Net.TRCLays {
					dt.SetCellFloat(lnm+" CosDiff_InWord", row, float64(ss.TrlCosDiffTRC[i]))
					dt.SetCellFloat(lnm+" CosDiff_Btw", row, float64(0))
				}
			} else {
				for _, lnm := range ss.Net.TRCLays {
					dt.SetCellFloat(lnm+" CosDiff_InWord", row, float64(0))
					dt.SetCellFloat(lnm+" CosDiff_Btw", row, float64(0))
				}
			}
		}
		if ss.CalcCosDiff {
			for i, lnm := range ss.Net.TRCLays {
				dt.SetCellFloat(lnm+" CosDiff", row, float64(ss.TrlCosDiffTRC[i]))
			}
		}
	}

	if ss.TestType == PartWholeTesting && ss.CalcPartWhole {
		// is the second consonantVowel pair from a whole word or a "part" word
		if ss.TestEnv.CV.SubSeg == 0 { // only get stat for first segment of the CV
			if ss.TestEnv.CV.Word == PartWord {
				for i, lnm := range ss.Net.TRCLays {
					dt.SetCellFloat(lnm+" CosDiff_PartWord", row, float64(ss.TrlCosDiffTRC[i]))
					dt.SetCellFloat(lnm+" CosDiff_WholeWord", row, float64(0))
				}
			} else if ss.TestEnv.CV.Word == WholeWord {
				for i, lnm := range ss.Net.TRCLays {
					dt.SetCellFloat(lnm+" CosDiff_WholeWord", row, float64(ss.TrlCosDiffTRC[i]))
					dt.SetCellFloat(lnm+" CosDiff_PartWord", row, float64(0))
				}
			} else {
				for _, lnm := range ss.Net.TRCLays {
					dt.SetCellFloat(lnm+" CosDiff_WholeWord", row, float64(0))
					dt.SetCellFloat(lnm+" CosDiff_PartWord", row, float64(0))
				}
			}
		}
	}
	ss.TstTrlPlot.GoUpdate()

	if ss.saveTstTrlLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.TstTrlFile.Header == true && ss.TstTrlFile.HeaderWritten == false {
			dt.WriteCSVHeaders(ss.TstTrlFile.File, etable.Tab)
			ss.TstTrlFile.HeaderWritten = true
		}
		err := dt.WriteCSVRow(ss.TstTrlFile.File, row, etable.Tab)
		if err != nil {
			log.Println("Error writing log: ", ss.TstTrlFile.Name)
		}
	}
}

func (ss *Sim) ConfigTstTrlPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Test trial plot"
	plt.Params.XAxisCol = "Trial"
	plt.Params.Scale = 3
	plt.Params.LineWidth = 1.5
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Run", false, true, 0, false, 0)
	plt.SetColParams("Epoch", false, true, 0, false, 0)
	plt.SetColParams("Trial", false, true, 0, false, 0)
	plt.SetColParams("Segment", true, true, 0, false, 0)

	on := false
	for _, lnm := range ss.Net.TRCLays {
		if lnm == "STS" {
			on = true
		}
		if ss.CalcBtwWthin {
			plt.SetColParams(lnm+" CosDiff_Btw", on, true, 0, true, 1)
			plt.SetColParams(lnm+" CosDiff_InWord", on, true, 0, true, 1)
		}
		if ss.CalcPartWhole {
			plt.SetColParams(lnm+" CosDiff_PartWord", on, true, 0, true, 1)
			plt.SetColParams(lnm+" CosDiff_WholeWord", on, true, 0, true, 1)
		}
		if ss.CalcCosDiff {
			plt.SetColParams(lnm+" CosDiff", false, true, 0, true, 1)
		}
		on = false
	}
	return plt
}

//////////////////////////////////////////////
//  TstEpcLog

func (ss *Sim) ConfigTstEpcLog(dt *etable.Table) {
	dt.SetMetaData("name", "TstEpcLog")
	dt.SetMetaData("desc", "Test epoch log")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	sch := etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Epoch", etensor.INT64, nil, nil},
	}

	for _, lnm := range ss.Net.TRCLays {
		if ss.CalcBtwWthin {
			sch = append(sch, etable.Column{lnm + " CosDiff_Btw", etensor.FLOAT64, nil, nil})
			sch = append(sch, etable.Column{lnm + " CosDiff_InWord", etensor.FLOAT64, nil, nil})
		}
		if ss.CalcPartWhole {
			sch = append(sch, etable.Column{lnm + " CosDiff_PartWord", etensor.FLOAT64, nil, nil})
			sch = append(sch, etable.Column{lnm + " CosDiff_WholeWord", etensor.FLOAT64, nil, nil})
		}
		if ss.CalcCosDiff {
			sch = append(sch, etable.Column{lnm + " CosDiff", etensor.FLOAT64, nil, nil})
		}
	}
	dt.SetFromSchema(sch, 1)
}

func (ss *Sim) LogTstEpc(dt *etable.Table, logFile *LogFile) {
	row := dt.Rows
	dt.SetNumRows(row + 1)
	nt := float64(ss.TstTrlLog.Rows) // number of trials in view

	//trl := ss.TstTrlLog
	//if ss.UseMPI {
	//	empi.GatherTableRows(ss.TstTrlLogAll, ss.TstTrlLog, ss.Comm)
	//	trl = ss.TstTrlLogAll
	//}

	ss.EpochStatsTRC(nt)
	if ss.Env == &ss.PreTestEnv {
		dt.SetCellFloat("Run", row, float64(ss.PreTestEnv.Run.Cur))
		dt.SetCellFloat("Epoch", row, float64(ss.PreTrainEnv.Epoch.Prv)) // use train epoch
	} else {
		dt.SetCellFloat("Run", row, float64(ss.TestEnv.Run.Cur))
		dt.SetCellFloat("Epoch", row, float64(ss.TrainEnv.Epoch.Prv)) // use train epoch
	}

	ss.SeqCnt = 0

	for i, lnm := range ss.Net.TRCLays {
		if ss.TestType == SequenceTesting && ss.CalcBtwWthin {
			dt.SetCellFloat(lnm+" CosDiff_Btw", row, float64(ss.EpcBtwCosDiffTRC[i]))
			dt.SetCellFloat(lnm+" CosDiff_InWord", row, float64(ss.EpcInWordCosDiffTRC[i]))
		}
		if ss.TestType == PartWholeTesting && ss.CalcPartWhole {
			dt.SetCellFloat(lnm+" CosDiff_PartWord", row, float64(ss.EpcPartCosDiffTRC[i]))
			dt.SetCellFloat(lnm+" CosDiff_WholeWord", row, float64(ss.EpcWholeCosDiffTRC[i]))
		}
		if ss.CalcCosDiff {
			dt.SetCellFloat(lnm+" CosDiff", row, float64(ss.EpcCosDiffTRC[i]))
		}
	}

	ss.TstEpcPlot.GoUpdate()
	if ss.saveTstEpcLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		//if ss.saveTstEpcLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if logFile.Header == true && logFile.HeaderWritten == false {
			dt.WriteCSVHeaders(logFile.File, etable.Tab)
			logFile.HeaderWritten = true
		}
		err := dt.WriteCSVRow(logFile.File, row, etable.Tab)
		if err != nil {
			log.Println("Error writing log: ", logFile.Name)
		}
	}
	ss.TstTrlLog.SetNumRows(0)
}

// ConfigTstEpcTidy formats the data as "Tidy Data" which works well with R statistics
// Each variable in its own column
// Each observation in it own row
// Each value it its own cell
func (ss *Sim) ConfigTstEpcTidy(dt *etable.Table) {
	dt.SetMetaData("name", "TstEpcTidy")
	dt.SetMetaData("desc", "Test epoch tidy")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	sch := etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Phase", etensor.STRING, nil, nil}, // pretest or test
		{"Epoch", etensor.INT64, nil, nil},
		{"Layer", etensor.STRING, nil, nil},
		{"Condition", etensor.STRING, nil, nil},
		{"Cosine", etensor.FLOAT64, nil, nil},
	}
	dt.SetFromSchema(sch, 1)
}

// LogTstEpcTidy is the log to be used with R statistics - see ConfigTstEpcLog for explanation
func (ss *Sim) LogTstEpcTidy(dt *etable.Table, logFile *LogFile) {
	row := dt.Rows
	dt.SetNumRows(row + 1)

	conditions := []string{"in---word", "next-word"} // same length for aligning tabs
	for l, lnm := range ss.Net.TRCLays {
		for _, cond := range conditions {
			if ss.Env == &ss.PreTestEnv {
				dt.SetCellFloat("Run", row, float64(ss.PreTestEnv.Run.Cur))
				dt.SetCellString("Phase", row, "pretest")
				dt.SetCellFloat("Epoch", row, float64(ss.PreTrainEnv.Epoch.Prv)) // use train epoch
			} else {
				dt.SetCellFloat("Run", row, float64(ss.TestEnv.Run.Cur))
				dt.SetCellString("Phase", row, "test")
				dt.SetCellFloat("Epoch", row, float64(ss.TrainEnv.Epoch.Prv+100)) // add 100 to get beyond pretest epoch numbers
			}
			dt.SetCellString("Layer", row, lnm)
			dt.SetCellString("Condition", row, cond)
			if cond == "in---word" {
				dt.SetCellFloat("Cosine", row, ss.EpcInWordCosDiffTRC[l])
			} else {
				dt.SetCellFloat("Cosine", row, ss.EpcBtwCosDiffTRC[l])
			}

			if ss.saveTstEpcTidy == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
				if logFile.Header == true && logFile.HeaderWritten == false {
					dt.WriteCSVHeaders(logFile.File, etable.Tab)
					logFile.HeaderWritten = true
				}
				err := dt.WriteCSVRow(logFile.File, row, etable.Tab)
				if err != nil {
					log.Println("Error writing log: ", logFile.Name)
				}
			}
			dt.SetNumRows(row + 1)
		}
	}
}

func (ss *Sim) ConfigTstEpcPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Testing Epoch Plot"
	plt.Params.XAxisCol = "Epoch"
	plt.Params.Scale = 3
	plt.Params.LineWidth = 1.5
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Run", false, true, 0, false, 0)
	plt.SetColParams("Epoch", false, true, 0, false, 0)

	for _, lnm := range ss.Net.TRCLays {
		if ss.CalcBtwWthin {
			plt.SetColParams(lnm+" CosDiff_Btw", false, true, 0, true, 1)
			plt.SetColParams(lnm+" CosDiff_InWord", true, true, 0, true, 1)
		}
		if ss.CalcPartWhole {
			plt.SetColParams(lnm+" CosDiff_PartWord", false, true, 0, true, 1)
			plt.SetColParams(lnm+" CosDiff_WholeWord", true, true, 0, true, 1)
		}
		if ss.CalcCosDiff {
			plt.SetColParams(lnm+" CosDiff", true, true, 0, true, 1)
		}
	}
	return plt
}

//////////////////////////////////////////////
//  TstCycLog

// LogTstCyc adds data from current trial to the TstCycLog table.
// log just has 100 cycles, is overwritten
func (ss *Sim) LogTstCyc(dt *etable.Table, cyc int) {
	net := ss.Net.Net

	if dt.Rows <= cyc {
		dt.SetNumRows(cyc + 1)
	}

	dt.SetCellFloat("Cycle", cyc, float64(cyc))
	for _, lnm := range ss.LayStatNms {
		ly := net.LayerByName(lnm).(leabra.LeabraLayer).AsLeabra()
		dt.SetCellFloat(ly.Nm+" Ge.Avg", cyc, float64(ly.Pools[0].Inhib.Ge.Avg))
		dt.SetCellFloat(ly.Nm+" Act.Avg", cyc, float64(ly.Pools[0].Inhib.Act.Avg))
	}
}

func (ss *Sim) ConfigActLog(dt *etable.Table, layName string) {
	net := ss.Net.Net

	dt.SetMetaData("name", layName+"ActLog")
	dt.SetMetaData("desc", "Record of activity for one tick")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	ly := net.LayerByName(layName).(leabra.LeabraLayer).AsLeabra()
	sch := etable.Schema{
		{"CV", etensor.STRING, nil, nil},
		{"Cons", etensor.STRING, nil, nil},
		{layName, etensor.FLOAT64, ly.Shape().Shp, nil},
	}
	dt.SetFromSchema(sch, 0)
}

func (ss *Sim) ConfigCatLayActs(dt *etable.Table) {
	dt.SetMetaData("name", "CatLayActs")
	dt.SetMetaData("desc", "layer activations for each cat / obj")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	sch := etable.Schema{
		{"Seg", etensor.INT64, nil, nil},
		{"CV", etensor.STRING, nil, nil},
		{"Count", etensor.INT64, nil, nil}, // how many instances were summed - use for mean
		{"Cons", etensor.STRING, nil, nil}, // consonant
		{"MannerCat", etensor.STRING, nil, nil},
		{"PlaceCat", etensor.STRING, nil, nil},
	}
	for _, lnm := range ss.Net.SuperLays {
		ly := ss.Net.Net.LayerByName(lnm).(leabra.LeabraLayer).AsLeabra()
		sch = append(sch, etable.Column{lnm, etensor.FLOAT32, ly.Shp.Shp, ly.Shp.Nms})
	}
	dt.SetFromSchema(sch, 0)
}

func (ss *Sim) ConfigTstCycPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Test cycle plot"
	plt.Params.XAxisCol = "Cycle"
	plt.Params.Scale = 3
	plt.Params.LineWidth = 1.5
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Cycle", false, true, 0, false, 0)
	for _, lnm := range ss.LayStatNms {
		plt.SetColParams(lnm+" Ge.Avg", true, true, 0, true, .5)
		plt.SetColParams(lnm+" Act.Avg", true, true, 0, true, .5)
	}
	return plt
}

func (ss *Sim) ConfigTstCycLog(dt *etable.Table) {
	dt.SetMetaData("name", "TstCycLog")
	dt.SetMetaData("desc", "Test cycle log")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	np := 100 // max cycles
	sch := etable.Schema{
		{"Cycle", etensor.INT64, nil, nil},
	}
	for _, lnm := range ss.LayStatNms {
		sch = append(sch, etable.Column{lnm + " Ge.Avg", etensor.FLOAT64, nil, nil})
		sch = append(sch, etable.Column{lnm + " Act.Avg", etensor.FLOAT64, nil, nil})
	}
	dt.SetFromSchema(sch, np)
}

func (ss *Sim) ResetTestSet() {
	nm := ss.TstList
	var dlg *gi.Dialog
	dlg = gi.StringPromptDialog(ss.Win.Viewport, nm, "enter name of sound list corresponding to the loaded tsttrldecode log", gi.DlgOpts{Title: "Set sound list for decode analysis", Prompt: "Update existing Message"},
		ss.Win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				ss.TstList = gi.StringPromptDialogValue(dlg)
				ss.SetTestingFiles(nm)
			}
		})
}

//////////////////////////////
//  RunLog

// LogRun adds data from current run to the RunLog table.
func (ss *Sim) LogRun(dt *etable.Table) {
	run := ss.TrainEnv.Run.Cur // this is NOT triggered by increment yet -- use Cur
	row := dt.Rows
	dt.SetNumRows(row + 1)

	epclog := ss.TrnEpcLog
	epcix := etable.NewIdxView(epclog)
	// compute mean over last N epochs for run level
	nlast := 10
	if nlast > epcix.Len()-1 {
		nlast = epcix.Len() - 1
	}
	epcix.Idxs = epcix.Idxs[epcix.Len()-nlast-1:]

	simparams := ss.RunName() // includes tag

	dt.SetCellFloat("Run", row, float64(run))
	dt.SetCellString("Params", row, simparams)

	// note: essential to use Go version of update when called from another goroutine
	ss.RunPlot.GoUpdate()
	if ss.saveRunLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.RunFile.Header == true && ss.RunFile.HeaderWritten == false {
			dt.WriteCSVHeaders(ss.RunFile.File, etable.Tab)
			ss.RunFile.HeaderWritten = true
		}
		err := dt.WriteCSVRow(ss.RunFile.File, row, etable.Tab)
		if err != nil {
			log.Println("Error writing log: ", ss.RunFile.Name)
		}
	}
}

func (ss *Sim) ConfigRunLog(dt *etable.Table) {
	dt.SetMetaData("name", "RunLog")
	dt.SetMetaData("desc", "Record of performance at end of training")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	dt.SetFromSchema(etable.Schema{
		{"Run", etensor.INT64, nil, nil},
		{"Params", etensor.STRING, nil, nil},
		//{"CosDiff", etensor.FLOAT64, nil, nil},
	}, 0)
}

func (ss *Sim) ConfigRunPlot(plt *eplot.Plot2D, dt *etable.Table) *eplot.Plot2D {
	plt.Params.Title = "Sensitivity to Sequential Probabilities Run Plot"
	plt.Params.XAxisCol = "Run"
	plt.Params.Scale = 3
	plt.Params.LineWidth = 1.5
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Run", false, true, 0, false, 0)
	//plt.SetColParams("CosDiff", false, true, 0, true, 1)
	return plt
}

////////////////////////////////////////////////////////////////////////////////////////////
// 		Gui

func (ss *Sim) ConfigNetView(nv *netview.NetView) {
	nv.ViewDefaults()
	nv.Scene().Camera.Pose.Pos.Set(0, 2.5, 3.0)
	nv.Scene().Camera.LookAt(mat32.Vec3{0, 0, 0}, mat32.Vec3{0, 1, 0})
}

// ConfigGui configures the GoGi gui interface for this simulation,
func (ss *Sim) ConfigGui() *gi.Window {
	width := 1600
	height := 1200

	gi.SetAppName("SensitIvity to Sequential Probabilities")
	gi.SetAppAbout(`A model that learns to predict the next syllable </p>`)

	//plot.DefaultFont = "Helvetica"

	win := gi.NewMainWindow("WordSeg", "Sensitivity to Sequential Probabilities", width, height)
	ss.Win = win

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tbar := gi.AddNewToolBar(mfr, "tbar")
	tbar.SetStretchMaxWidth()
	ss.ToolBar = tbar

	splt := gi.AddNewSplitView(mfr, "split")
	splt.Dim = gi.X
	splt.SetStretchMax()

	sv := giv.AddNewStructView(splt, "sv")
	sv.SetStruct(ss)
	ss.StructView = sv

	tv := gi.AddNewTabView(splt, "tv")
	splt.SetSplits(.2, .8)

	nv := tv.AddNewTab(netview.KiT_NetView, "NetView").(*netview.NetView)
	nv.Params.Defaults()
	nv.Params.LayNmSize = 0.03
	nv.Var = "Act"
	nv.SetNet(ss.Net.Net)
	ss.NetView = nv
	ss.ConfigNetView(nv) // sg has this line

	plt := tv.AddNewTab(eplot.KiT_Plot2D, "TrnTrlPlot").(*eplot.Plot2D)
	plt.Params.XAxisCol = "Trial"
	plt.SetTable(ss.TrnTrlLog)
	ss.TrnTrlPlot = plt
	ss.ConfigTrnTrlPlot(plt, ss.TrnTrlLog)

	plt = tv.AddNewTab(eplot.KiT_Plot2D, "TrnEpcPlot").(*eplot.Plot2D)
	plt.Params.XAxisCol = "Epoch"
	plt.SetTable(ss.TrnEpcLog)
	ss.TrnEpcPlot = plt
	ss.ConfigTrnEpcPlot(plt, ss.TrnEpcLog)

	plt = tv.AddNewTab(eplot.KiT_Plot2D, "TstTrlPlot").(*eplot.Plot2D)
	plt.Params.XAxisCol = "Trial"
	plt.SetTable(ss.TstTrlLog)
	ss.TstTrlPlot = plt
	ss.ConfigTstTrlPlot(plt, ss.TstTrlLog)

	plt = tv.AddNewTab(eplot.KiT_Plot2D, "TstEpcPlot").(*eplot.Plot2D)
	plt.Params.XAxisCol = "Epoch"
	plt.SetTable(ss.TstEpcLog)
	ss.TstEpcPlot = plt
	ss.ConfigTstEpcPlot(plt, ss.TstEpcLog)

	plt = tv.AddNewTab(eplot.KiT_Plot2D, "RunPlot").(*eplot.Plot2D)
	plt.Params.XAxisCol = "Run"
	plt.SetTable(ss.RunLog)
	ss.RunPlot = plt
	ss.ConfigRunPlot(plt, ss.RunLog)

	// sound short
	tgS := tv.AddNewTab(etview.KiT_TensorGrid, "MelFBank").(*etview.TensorGrid)
	tgS.SetStretchMax()
	ss.MelFBankGridS = tgS
	tgS.SetTensor(&ss.TrainEnv.SndShort.MelFBankSegment)

	// sound long
	tgL := tv.AddNewTab(etview.KiT_TensorGrid, "MelFBank").(*etview.TensorGrid)
	tgL.SetStretchMax()
	ss.MelFBankGridL = tgL
	tgL.SetTensor(&ss.TrainEnv.SndLong.MelFBankSegment)

	// sound short
	tgS = tv.AddNewTab(etview.KiT_TensorGrid, "Power").(*etview.TensorGrid)
	tgS.SetStretchMax()
	ss.PowerGridS = tgS
	ss.TrainEnv.SndShort.LogPowerSegment.SetMetaData("grid-min", "10")
	tgS.SetTensor(&ss.TrainEnv.SndShort.LogPowerSegment)

	// sound long
	tgL = tv.AddNewTab(etview.KiT_TensorGrid, "Power").(*etview.TensorGrid)
	tgL.SetStretchMax()
	ss.PowerGridL = tgL
	ss.TrainEnv.SndLong.LogPowerSegment.SetMetaData("grid-min", "10")
	tgL.SetTensor(&ss.TrainEnv.SndLong.LogPowerSegment)

	tbar.AddAction(gi.ActOpts{Label: "Init", Icon: "update", Tooltip: "Initialize everything including network weights, and start over.  Also applies current params.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		ss.Init()
		vp.SetNeedsFullRender()
	})

	tbar.AddAction(gi.ActOpts{Label: "Train", Icon: "run", Tooltip: "Starts the network training, picking up from wherever it may have left off.  If not stopped, training will complete the specified number of Runs through the full number of Epochs of training, with testing automatically occuring at the specified interval.",
		UpdateFunc: func(act *gi.Action) {
			act.SetActiveStateUpdt(!ss.IsRunning)
		}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			tbar.UpdateActions()
			go ss.Train()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Pre-Train", Icon: "run", Tooltip: "Starts the network pre training, picking up from wherever it may have left off.  Pre training is always 1 run of max pre training epochs.",
		UpdateFunc: func(act *gi.Action) {
			act.SetActiveStateUpdt(!ss.IsRunning)
		}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			tbar.UpdateActions()
			go ss.PreTrain()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Stop", Icon: "stop", Tooltip: "Interrupts running.  Hitting Train again will pick back up where it left off.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		ss.Stop()
	})

	tbar.AddAction(gi.ActOpts{Label: "Step Trial", Icon: "step-fwd", Tooltip: "Advances one training trial at a time.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			ss.TrainTrial()
			ss.IsRunning = false
			vp.SetNeedsFullRender()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Step Sequence", Icon: "step-fwd", Tooltip: "Advances one sequence of trials at a time.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			tbar.UpdateActions()
			go ss.TrainSequence()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Step Epoch", Icon: "fast-fwd", Tooltip: "Advances one epoch (complete set of training patterns) at a time.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			ss.IsRunning = true
			tbar.UpdateActions()
			go ss.TrainEpoch()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Step Run", Icon: "fast-fwd", Tooltip: "Advances one full training Run at a time.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning && ss.Env == &ss.TrainEnv { // only one run for pretraining
			ss.IsRunning = true
			tbar.UpdateActions()
			go ss.TrainRun()
		}
	})

	tbar.AddSeparator("test")

	// ToDo: should have a dialog to ask if Sequence or Part/Whole testing if not already set
	tbar.AddAction(gi.ActOpts{Label: "Test Trial", Icon: "step-fwd", Tooltip: "Runs the next testing trial.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning && ss.Env != &ss.PreTrainEnv { // no testing during pretraining
			ss.IsRunning = true
			ss.TestTrial(ss.TstEpcFile, ss.TstEpcTidyFile) // don't return on change -- wrap
			ss.IsRunning = false
			vp.SetNeedsFullRender()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Test Sequences", Icon: "fast-fwd", Tooltip: "Tests all of the testing trials.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning && ss.Env != &ss.PreTrainEnv { // no testing during pretraining
			ss.IsRunning = true
			tbar.UpdateActions()
			ss.TestType = SequenceTesting
			go ss.RunTestAll()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Test Part/Whole", Icon: "fast-fwd", Tooltip: "Tests all part words and whole words.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning && ss.Env != &ss.PreTrainEnv { // no testing during pretraining
			ss.IsRunning = true
			tbar.UpdateActions()
			ss.TestType = PartWholeTesting
			go ss.RunTestAll()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Reset testing set", Icon: "fast-fwd", Tooltip: "Reset the testing set", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			tbar.UpdateActions()
			go ss.ResetTestSet()
		}
	})

	tbar.AddAction(gi.ActOpts{Label: "Run RSA on CatLayActs", Icon: "fast-fwd", Tooltip: "Some summary statistics of data in the tstresults table.", UpdateFunc: func(act *gi.Action) {
		act.SetActiveStateUpdt(!ss.IsRunning)
	}}, win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if !ss.IsRunning {
			tbar.UpdateActions()
			go ss.RSAAnal(ss.CatLayActs, ss.Net.SuperLays)
		}
	})

	tbar.AddSeparator("misc")

	tbar.AddAction(gi.ActOpts{Label: "New Seed", Icon: "new", Tooltip: "Generate a new initial random seed to get different results.  By default, Init re-establishes the same initial seed every time."}, win.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			ss.NewRndSeed()
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

	win.SetCloseCleanFunc(func(w *gi.Window) {
		go gi.Quit() // once main window is closed, quit
	})

	win.MainMenuUpdated()
	return win
}

// These props register Save methods so they can be used
var SimProps = ki.Props{
	"CallMethods": ki.PropSlice{
		{"SaveWeights", ki.Props{
			"desc": "save network weights to file",
			"icon": "file-save",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".wts,.wts.gz",
				}},
			},
		},
		}},
}

func (ss *Sim) CmdArgs() {
	ss.NoGui = true
	var nogui bool
	var note string
	saveNetData := false

	flag.BoolVar(&nogui, "nogui", true, "if not passing any other args and want to run nogui, use nogui")
	flag.StringVar(&ss.Net.ParamSet, "params", "", "ParamSet name to use -- must be valid name as listed in compiled-in params or loaded params")
	flag.BoolVar(&ss.Net.LogSetParams, "setparams", false, "if true, print a record of each parameter that is set")
	flag.StringVar(&ss.TrnList, "trnlist", "", "identifies the list of sound stimuli for train environment")
	flag.StringVar(&ss.TstList, "tstlist", "", "identifies the list of sound stimuli for test environment")
	flag.StringVar(&ss.PreTrnList, "prelist", "", "identifies the list of sound stimuli for pretrain environment")
	flag.StringVar(&ss.PreTstList, "pretstlist", "", "identifies the list of sound stimuli for test environment")
	flag.StringVar(&ss.Tag, "tag", "", "extra tag to add to file names saved from this run")
	flag.StringVar(&ss.HoldoutID, "holdoutid", "", "unique id for testing holdout file name")
	flag.StringVar(&note, "note", "", "user note -- not used")
	flag.IntVar(&ss.StartRun, "run", 0, "starting run number -- determines the random seed -- runs counts from there -- can do all runs in parallel by launching separate jobs with each run, runs = 1")
	flag.IntVar(&ss.MaxRuns, "runs", 1, "number of runs to do")
	flag.IntVar(&ss.MaxEpcs, "epochs", 2, "number of epochs to do")
	flag.IntVar(&ss.MaxSeqs, "seqs", 2, "number of sound sequences to do")
	flag.IntVar(&ss.MaxPreEpcs, "epochspre", 2, "number of epochs to do")
	flag.IntVar(&ss.MaxPreSeqs, "seqspre", 2, "number of sound sequences to do")
	flag.BoolVar(&ss.OpenWts, "openwts", false, "if true, use the wts specified by ss.wts")
	flag.StringVar(&ss.WtsPath, "wtspath", "", "path to weights file")
	flag.StringVar(&ss.Wts, "wts", "", "weights from prior training/pretraining")
	flag.BoolVar(&ss.SaveWts, "savewts", false, "if true, save final weights after each run")
	flag.IntVar(&ss.WtsInterval, "wtsinterval", 100, "save wts every N epochs")
	flag.IntVar(&ss.TestInterval, "tstinterval", -1, "test every N epochs, must be set on command line - must be less than epochs")
	flag.IntVar(&ss.PreTestInterval, "pretstinterval", -1, "test every N epochs, must be set on command line - must be less than epochs")
	flag.IntVar(&ss.RSA.Interval, "rsainterval", -1, "test every N epochs, must be set on command line - must be less than epochs")
	flag.BoolVar(&ss.SaveSimMat, "simmat", false, "if true, save the similarity matrix")
	flag.BoolVar(&ss.SaveActs, "acts", false, "if true, save activations after each run")
	flag.BoolVar(&ss.saveProcLog, "proclog", false, "if true, save log files separately for each processor (for debugging)")
	flag.BoolVar(&ss.saveRunLog, "runlog", false, "if true, save run epoch log to file")
	flag.BoolVar(&ss.saveTrnTrlLog, "trntrllog", false, "if true, save train epoch log to file")
	flag.BoolVar(&ss.saveTstTrlLog, "tsttrllog", false, "if true, save trl results log to file")
	flag.BoolVar(&ss.saveTrnEpcLog, "trnepclog", true, "if true, save train epoch log to file")
	flag.BoolVar(&ss.savePreTrnEpcLog, "pretrnepclog", true, "if true, save train epoch log to file")
	flag.BoolVar(&ss.saveTstEpcLog, "tstepclog", true, "if true, save train epoch log to file")
	flag.BoolVar(&ss.saveTstEpcTidy, "tstepctidy", false, "if true, save train epoch log to file")
	flag.BoolVar(&ss.savePreTstEpcLog, "pretstepclog", true, "if true, save train epoch log to file")
	flag.BoolVar(&ss.savePreTstEpcTidy, "pretstepctidy", false, "if true, save train epoch log to file")
	flag.BoolVar(&ss.saveTrnCondEpcLog, "trncondepclog", false, "if true, save train epoch log to file")
	flag.BoolVar(&ss.saveTstCondEpcLog, "tstcondepclog", false, "if true, save train epoch log to file")
	flag.BoolVar(&ss.saveActsLog, "actslog", false, "if true, save the activations to a file")
	flag.BoolVar(&ss.UseMPI, "mpi", false, "if set, use MPI for distributed computation")
	flag.BoolVar(&ss.UseRateSched, "ratesched", false, "if true use the coded rate schedule")
	flag.BoolVar(&ss.Pretrain, "pretrain", false, "if set run pretraining only")
	flag.BoolVar(&ss.TestRun, "test", false, "true for test instead of train")
	flag.BoolVar(&ss.CalcBtwWthin, "calcbtw", true, "calculates cos diff for between and within trials separately")
	flag.BoolVar(&ss.CalcCosDiff, "calccosdif", true, "calculates cos diff across all trials")
	flag.BoolVar(&saveNetData, "netdata", false, "if true, save network activation etc data from testing trials, for later viewing in netview")
	flag.IntVar(&ss.HoldoutPct, "holdoutpct", 34, "percentage of items to holdout from train set for testing")
	flag.Parse()

	if ss.UseMPI {
		fmt.Println("use mpi")
		ss.MPIInit()
		if ss.MaxSeqs%mpi.WorldSize() != 0 {
			log.Printf("MaxSeqs is %d, not an even multiple of the number of MPI procs: %d -- should be!\n", ss.MaxSeqs, mpi.WorldSize())
		}
		ss.MaxSeqs = ss.MaxSeqs / mpi.WorldSize()
	}

	ss.Init()
	ss.Config()

	if note != "" {
		mpi.Printf("note: %s\n", note)
	}
	if ss.Net.ParamSet != "" {
		mpi.Printf("Using ParamSet: %s\n", ss.Net.ParamSet)
	}

	if ss.saveTrnEpcLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.TrnEpcFile.File == nil {
			ss.TrnEpcFile.File = ss.CreateLogFile(*ss.TrnEpcFile)
			if ss.TrnEpcFile.File != nil {
				defer ss.TrnEpcFile.File.Close()
			}
		}
	}

	if ss.saveTstEpcLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.TstEpcFile.File == nil {
			ss.TstEpcFile.File = ss.CreateLogFile(*ss.TstEpcFile)
			if ss.TstEpcFile.File != nil {
				defer ss.TstEpcFile.File.Close()
			}
		}
	}

	if ss.saveTstEpcTidy == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.TstEpcTidyFile.File == nil {
			ss.TstEpcTidyFile.File = ss.CreateLogFile(*ss.TstEpcTidyFile)
			if ss.TstEpcTidyFile.File != nil {
				defer ss.TstEpcTidyFile.File.Close()
			}
		}
	}

	if ss.savePreTrnEpcLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.PreTrnEpcFile.File == nil {
			ss.PreTrnEpcFile.File = ss.CreateLogFile(*ss.PreTrnEpcFile)
			if ss.PreTrnEpcFile.File != nil {
				defer ss.PreTrnEpcFile.File.Close()
			}
		}
	}

	if ss.savePreTstEpcLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.PreTstEpcFile.File == nil {
			ss.PreTstEpcFile.File = ss.CreateLogFile(*ss.PreTstEpcFile)
			if ss.PreTstEpcFile.File != nil {
				defer ss.PreTstEpcFile.File.Close()
			}
		}
	}

	if ss.savePreTstEpcTidy == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.PreTstEpcTidyFile.File == nil {
			ss.PreTstEpcTidyFile.File = ss.CreateLogFile(*ss.PreTstEpcTidyFile)
			if ss.PreTstEpcTidyFile.File != nil {
				defer ss.PreTstEpcTidyFile.File.Close()
			}
		}
	}

	if ss.saveTrnTrlLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.TrnTrlFile.File == nil {
			ss.TrnTrlFile.File = ss.CreateLogFile(*ss.TrnTrlFile)
			if ss.TrnTrlFile.File != nil {
				defer ss.TrnTrlFile.File.Close()
			}
		}
	}

	if ss.saveTstTrlLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.TstTrlFile.File == nil {
			ss.TstTrlFile.File = ss.CreateLogFile(*ss.TstTrlFile)
			if ss.TstTrlFile.File != nil {
				defer ss.TstTrlFile.File.Close()
			}
		}
	}
	if ss.saveTrnCondEpcLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.TrnCondEpcFile.File == nil {
			ss.TrnCondEpcFile.File = ss.CreateLogFile(*ss.TrnCondEpcFile)
			if ss.TrnCondEpcFile.File != nil {
				defer ss.TrnCondEpcFile.File.Close()
			}
		}
	}
	if ss.saveTstCondEpcLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.TstCondEpcFile.File == nil {
			ss.TstCondEpcFile.File = ss.CreateLogFile(*ss.TstCondEpcFile)
			if ss.TstCondEpcFile.File != nil {
				defer ss.TstCondEpcFile.File.Close()
			}
		}
	}
	if ss.saveRunLog == true && (ss.saveProcLog || mpi.WorldRank() == 0) {
		if ss.RunFile.File == nil {
			ss.RunFile.File = ss.CreateLogFile(*ss.RunFile)
			if ss.RunFile.File != nil {
				defer ss.RunFile.File.Close()
			}
		}
	}
	if ss.SaveWts {
		if mpi.WorldRank() != 0 {
			ss.SaveWts = false
		}
		mpi.Printf("Saving final weights per run\n")
	}

	if saveNetData {
		ss.NetData = &netview.NetData{}
		ss.NetData.Init(ss.Net.Net, 100) // max is max trials to save
	}

	mpi.Printf("Running %d Runs\n", ss.MaxRuns)
	if ss.Pretrain {
		ss.Env = &ss.PreTrainEnv
		ss.MaxRuns = 1 // ToDo: somehow this is getting changed - reset here until the problem is located
		fmt.Printf("Running %d Runs starting at %d\n", ss.MaxRuns, ss.StartRun)
		ss.TrainEnv.Run.Set(ss.StartRun)
		ss.TrainEnv.Run.Max = ss.StartRun + ss.MaxRuns
		ss.NewRun()
		ss.PreTrain()
		ss.Train()
	} else if ss.TestRun {
		ss.Env = &ss.TestEnv
		ss.TestAll(&ss.TestEnv)
	} else {
		ss.Env = &ss.TrainEnv
		fmt.Printf("Running %d Runs starting at %d\n", ss.MaxRuns, ss.StartRun)
		ss.TrainEnv.Run.Set(ss.StartRun)
		ss.TrainEnv.Run.Max = ss.StartRun + ss.MaxRuns
		ss.NewRun()
		ss.Train()

		if saveNetData {
			ndfn := ss.Net.Net.Nm + "_" + ss.RunName() + ".netdata.gz"
			ss.NetData.SaveJSON(gi.FileName(ndfn))
		}
	}
	ss.MPIFinalize()
}

////////////////////////////////////////////////////////////////////
//  MPI code

// MPIInit initializes MPI
func (ss *Sim) MPIInit() {
	mpi.Init()
	var err error
	ss.Comm, err = mpi.NewComm(nil) // use all procs
	if err != nil {
		log.Println(err)
		ss.UseMPI = false
	} else {
		mpi.Printf("MPI running on %d procs\n", mpi.WorldSize())
	}
}

// MPIFinalize finalizes MPI
func (ss *Sim) MPIFinalize() {
	if ss.UseMPI {
		mpi.Printf("MPI finalize\n")
		mpi.Finalize()
	}
}

// CollectDWts collects the weight changes from all synapses into AllDWts
func (ss *Sim) CollectDWts(net *leabra.Network) {
	made := net.CollectDWts(&ss.AllDWts, 21488282) // plug in number from printout below, to avoid realloc
	if made {
		mpi.Printf("MPI: AllDWts len: %d\n", len(ss.AllDWts)) // put this number in above make
	}
}

// MPIWtFmDWt updates weights from weight changes, using MPI to integrate
// DWt changes across parallel nodes, each of which are learning on different
// sequences of inputs.
func (ss *Sim) MPIWtFmDWt() {
	net := ss.Net.Net

	if ss.UseMPI {
		ss.CollectDWts(&net.Network)
		ndw := len(ss.AllDWts)
		if len(ss.SumDWts) != ndw {
			ss.SumDWts = make([]float32, ndw)
		}
		ss.Comm.AllReduceF32(mpi.OpSum, ss.SumDWts, ss.AllDWts)
		net.SetDWts(ss.SumDWts)
	}
	net.WtFmDWt()
}

// SetTrainingFiles
func (ss *Sim) SetTrainingFiles(key string) {
	// Train Train Train Train
	ss.TrainEnv.SndPath = "ccn_images/word_seg_snd_files/"

	train := true
	switch ss.TrnList {
	case "CVs_I":
		ss.TrainEnv.SeqsPath = "CV_I_NoCoAr_Seqs/"
		ss.TrainEnv.WavsPath = "CV_I_NoCoAr_Wavs/"
		ss.TrainEnv.TimesPath = "CV_I_NoCoAr_Times/"
		ss.TrainEnv.CVsPerWord = 3 // The saffran, et al experiment used 3 syllable words
		ss.TrainEnv.CVsPerPos = 4  // The saffran, et al experiment had 4 cv possibilities per syllable position

	case "CVs_III", "CVs_IV", "CVs_V", "CVs_VI":
		ss.TrainEnv.SeqsPath = "CV_III_IV_V_VI_NoCoAr_Seqs/"
		ss.TrainEnv.WavsPath = "CV_III_IV_V_VI_NoCoAr_Wavs/"
		ss.TrainEnv.TimesPath = "CV_III_IV_V_VI_NoCoAr_Times/"
		ss.TrainEnv.CVsPerWord = 3 // The saffran, et al experiment used 3 syllable words, Graf Estes 2
		ss.TrainEnv.CVsPerPos = 4  // The saffran, et al experiment had 4 cv possibilities per syllable position

	case "MT_IABC":
		// for graf-estes simulation
		ss.TrainEnv.SeqsPath = "MT_IABC_Seqs/"
		ss.TrainEnv.WavsPath = "MT_IABC_Wavs/"
		ss.TrainEnv.TimesPath = "MT_IABC_Times/"
		ss.TrainEnv.CVsPerWord = 2 // The graf-estes experiment used 2 syllable words
		ss.TrainEnv.CVsPerPos = 4  // The graf-estes experiment had 4 cv possibilities per syllable position

	case "TIMIT_ALL_SX_F":
		ss.TrainEnv.SndPath = "ccn_images/sound/timit/" // this one is different!
		ss.TrainEnv.WavsPath = "TIMIT/TRAIN/"
		ss.TrainEnv.SeqsPath = "TIMIT/TRAIN/"
		ss.TrainEnv.TimesPath = "TIMIT/TRAIN/"
		ss.TrainEnv.SndTimit = true
		ss.CalcPartWhole = false
		ss.CalcBtwWthin = true
		ss.SaveActs = false

	default:
		fmt.Println("No matching sound set for training - ok if doing pretrain or testing")
		train = false
	}

	if train {
		switch ss.TrnList {
		case "CVs_I":
			ss.TrainEnv.SndList = "CV_I_NoCoAr_Train.txt"
			ss.TrainEnv.CVs = CVs_I
		case "CVs_III":
			ss.TrainEnv.SndList = "CV_III_NoCoAr_Train.txt"
			ss.TrainEnv.CVs = CVs_III
		case "CVs_IV":
			ss.TrainEnv.SndList = "CV_IV_NoCoAr_Train.txt"
			ss.TrainEnv.CVs = CVs_IV
		case "CVs_V":
			ss.TrainEnv.SndList = "CV_V_NoCoAr_Train.txt"
			ss.TrainEnv.CVs = CVs_V
		case "CVs_VI":
			ss.TrainEnv.SndList = "CV_VI_NoCoAr_Train.txt"
			ss.TrainEnv.CVs = CVs_VI
		case "MT_IABC":
			ss.TrainEnv.SndList = "MT_IABC_Train.txt"
			ss.TrainEnv.CVs = CVs_MT
			ss.TrainEnv.Silence = true
			ss.TestWordsPart = []string{"ku ga", "pi mo"}
			ss.TestWordsWhole = []string{"do bu", "ti may"}
		case "TIMIT_ALL_SX_F":
			ss.TrainEnv.SndList = "trainAllFemaleSX.txt"
			ss.TrainEnv.CVs = []string{}
			ss.TrainEnv.Silence = false // don't add silence at start
		}

		// Sanity check
		if ss.TrainEnv.SndTimit == false { // i.e. check if training consonant vowels
			if ss.TrainEnv.CVsPerWord*ss.TrainEnv.CVsPerPos != len(ss.TrainEnv.CVs) {
				fmt.Println("ERROR! - len of cv list not equal to CVsPerWord * CVsPerPos")
			}
		}

		// always add the silence CV to the list!
		ss.TrainEnv.CVs = append(ss.TrainEnv.CVs, "ss")
		if len(ss.TrainEnv.CVs) >= 13 {
			ss.TrainEnv.ThirdCVs = ss.TrainEnv.CVs[8:12]
		}
		if len(ss.TrainEnv.CVs) >= 9 {
			ss.TrainEnv.SecondCVs = ss.TrainEnv.CVs[4:8]
		}
		if len(ss.TrainEnv.CVs) >= 5 {
			ss.TrainEnv.FirstCVs = ss.TrainEnv.CVs[0:4]
		}
	}
}

// IsTestWordPart compares last/cur to see if they match a 2 syllable word in "part word" test list.
// An empty list is the same as a list of all possible "part words"
// Assumes that "part wordness" has already been validated by IsPredictable() returning "Partially"
func (ss *Sim) IsTestWordPart(last, cur string) (testWord bool) {
	if len(ss.TestWordsPart) == 0 {
		return true
	}

	testWord = false
	w := last + " " + cur
	for _, pw := range ss.TestWordsPart {
		if w == pw {
			testWord = true
			break
		}
	}
	return
}

// IsTestWordWhole compares last/cur to see if they match a 2 syllable word in "whole word" test list.
// An empty list is the same as a list of all possible "whole words"
// Assumes that "whole wordness" has already been validated by IsPredictable() returning "Fully"
func (ss *Sim) IsTestWordWhole(last, cur string) (testWord bool) {
	if len(ss.TestWordsWhole) == 0 {
		return true
	}

	testWord = false
	w := last + " " + cur
	for _, ww := range ss.TestWordsWhole {
		if w == ww {
			testWord = true
			break
		}
	}
	return
}

// SetTestingFiles
func (ss *Sim) SetTestingFiles(key string) {
	// Test Test Test Test
	ss.TestEnv.SndPath = "ccn_images/word_seg_snd_files/"
	test := true

	switch ss.TstList {
	// full sequence testing
	case "Holdouts": // holdouts are a subset pulled out of training list
		ss.TestType = SequenceTesting
		ss.TestEnv.SeqsPath = ss.TrainEnv.SeqsPath
		ss.TestEnv.WavsPath = ss.TrainEnv.WavsPath
		ss.TestEnv.TimesPath = ss.TrainEnv.TimesPath
		ss.TestEnv.CVsPerWord = ss.TrainEnv.CVsPerWord
		ss.TestEnv.CVsPerPos = ss.TrainEnv.CVsPerWord
		ss.CalcPartWhole = false
		ss.CalcBtwWthin = true

	case "CVs_I":
		ss.TestType = SequenceTesting
		ss.TestEnv.SeqsPath = "CV_I_NoCoAr_Seqs/"
		ss.TestEnv.WavsPath = "CV_I_NoCoAr_Wavs/"
		ss.TestEnv.TimesPath = "CV_I_NoCoAr_Times/"
		ss.TestEnv.CVsPerWord = 3 // The saffran, et al experiment used 3 syllable words
		ss.TestEnv.CVsPerPos = 4  // The saffran, et al experiment had 4 cv possibilities per syllable position
		ss.CalcPartWhole = false
		ss.CalcBtwWthin = true

	// Part word / whole word testing
	case "CVs_I_PWWW":
		ss.TestType = PartWholeTesting
		ss.TestEnv.SeqsPath = "CV_I_NoCoAr_PWWW_Seqs/"
		ss.TestEnv.WavsPath = "CV_I_NoCoAr_PWWW_Wavs/"
		ss.TestEnv.TimesPath = "CV_I_NoCoAr_PWWW_Times/"
		ss.TestEnv.CVsPerWord = 3 // The saffran, et al experiment used 3 syllable words
		ss.TestEnv.CVsPerPos = 4  // The saffran, et al experiment had 4 cv possibilities per syllable position
		ss.CalcPartWhole = true
		ss.CalcBtwWthin = false

	case "CVs_III_PWWW", "CVs_IV_PWWW", "CVs_V_PWWW", "CVs_VI_PWWW": // part-word / whole-word testing
		ss.TestType = PartWholeTesting
		ss.TestEnv.SeqsPath = "CV_III_IV_V_VI_NoCoAr_PWWW_Seqs/"
		ss.TestEnv.WavsPath = "CV_III_IV_V_VI_NoCoAr_PWWW_Wavs/"
		ss.TestEnv.TimesPath = "CV_III_IV_V_VI_NoCoAr_PWWW_Times/"
		ss.TestEnv.CVsPerWord = 3 // The saffran, et al experiment used 3 syllable words
		ss.TestEnv.CVsPerPos = 4  // The saffran, et al experiment had 4 cv possibilities per syllable position

	case "MT_I_All_PWWW": // part-word / whole-word testing - taken from second minute of language I
		// for graf-estes simulation
		ss.TestType = PartWholeTesting
		ss.TestEnv.SeqsPath = "MT_IB_PWWW_Seqs/"
		ss.TestEnv.WavsPath = "MT_IB_PWWW_Wavs/"
		ss.TestEnv.TimesPath = "MT_IB_PWWW_Times/"
		ss.TestEnv.CVsPerWord = 2 // The graf-estes experiment used 2 syllable words
		ss.TestEnv.CVsPerPos = 4  // The graf-estes experiment had 4 cv possibilities per syllable position

	case "MT_IABC":
		ss.TestType = SequenceTesting
		ss.TestEnv.SeqsPath = "MT_IABC_Seqs/"
		ss.TestEnv.WavsPath = "MT_IABC_Wavs/"
		ss.TestEnv.TimesPath = "MT_IABC_Times/"
		ss.TestEnv.CVsPerWord = 2 // The graf-estes experiment used 2 syllable words
		ss.TestEnv.CVsPerPos = 4  // The graf-estes experiment had 4 cv possibilities per syllable position

	default:
		fmt.Println("No matching sound set for testing - ok if not testing or doing holdout train/test ")
		test = false
		if ss.TestRun == true {
			os.Exit(99)
		}
	}

	if test {
		switch ss.TstList {
		// full sequence testing
		case "Holdouts":
			ss.TestEnv.SndList = "" // gets set when splitting TrainEnv.SndList
			ss.TestEnv.CVs = ss.TrainEnv.CVs
			ss.Holdout = true
			ss.TestEnv.Silence = ss.TrainEnv.Silence
		case "CVs_I":
			ss.TestEnv.SndList = "CV_I_NoCoAr_Test.txt"
			ss.TestEnv.CVs = CVs_I

			// Part word / whole word testing
		case "CVs_I_PWWW":
			ss.TestEnv.SndList = "CV_I_NoCoAr_PWWW.txt"
			ss.TestEnv.CVs = CVs_I
			ss.TestWordsPart = []string{"do da", "do go", "do pa", "ku da", "ku go", "ku ti", "pi go", "pi pa", "pi ti", "tu da", "tu pa", "tu ti"}
			ss.TestWordsWhole = []string{"da ro", "go la", "pa bi", "ti bu"}
		case "CVs_III_PWWW":
			ss.TestEnv.SndList = "CV_III_NoCoAr_PWWW.txt"
			ss.TestEnv.CVs = CVs_III
			ss.TestWordsPart = []string{"hi ho", "ra pa", "di ro", "sa su", "hi pa", "ra ho", "di su", "sa ro", "hi ro", "ra su", "di ho", "sa pa"}
			ss.TestWordsWhole = []string{"su ba", "ro lu", "pa go", "ho li"}
		case "CVs_IV_PWWW":
			ss.TestEnv.SndList = "CV_IV_NoCoAr_PWWW.txt"
			ss.TestEnv.CVs = CVs_IV
			ss.TestWordsPart = []string{"ru ki", "si hu", "ta ki", "po na", "ru hu", "si do", "ta na", "po do", "ru na", "si ki", "ta do", "po hu"}
			ss.TestWordsWhole = []string{"do ka", "na to", "hu mo", "ki mu"}
		case "CVs_V_PWWW":
			ss.TestEnv.SndList = "CV_V_NoCoAr_PWWW.txt"
			ss.TestEnv.CVs = CVs_V
			ss.TestWordsPart = []string{"bo ma", "ga gu", "ha bi", "so bu", "bo gu", "ga ma", "ha bu", "so bi", "bo bi", "ga bu", "ha ma", "so gu"}
			ss.TestWordsWhole = []string{"gu ri", "ma gi", "bi tu", "bu ni"}
		case "CVs_VI_PWWW":
			ss.TestEnv.SndList = "CV_VI_NoCoAr_PWWW.txt"
			ss.TestEnv.CVs = CVs_VI
			ss.TestWordsPart = []string{"mi lo", "pu nu", "ko ti", "la da", "mi nu", "pu da", "ko lo", "la ti", "mi ti", "pu lo", "ko da", "la nu"}
			ss.TestWordsWhole = []string{"da ku", "ti no", "nu pi", "lo du"}
		case "MT_I_PWWW":
			ss.TestEnv.SndList = "MT_IB_PWWW_List.txt"
			ss.TestEnv.CVs = []string{"ti", "do", "ga", "mo", "may", "bu", "pi", "ku"}
			ss.TestWordsPart = []string{"pi mo", "ku ga", "bu ga", "ku do", "may mo", "pi ti", "bu mo", "ku ti", "may ga", "pi do"}
			ss.TestWordsWhole = []string{"mo ku", "ga pi", "do bu", "ti may"}
		case "MT_IABC":
			ss.TestEnv.SndList = "MT_IABC_Test.txt"
			ss.TestEnv.CVs = CVs_MT
			ss.TestEnv.Silence = true
			ss.TestWordsPart = []string{"ku ga", "pi mo"}
			ss.TestWordsWhole = []string{"do bu", "ti may"}

		default:
			log.Println("No matching case for test sndfiles switch")
		}

		// add the silence CV to the list!
		if ss.Holdout == false { // if true test = train so "ss" has already been added
			ss.TestEnv.CVs = append(ss.TestEnv.CVs, "ss")
		}

		// always set these
		if len(ss.TrainEnv.CVs) >= 13 {
			ss.TestEnv.ThirdCVs = ss.TrainEnv.CVs[8:12]
		}
		if len(ss.TrainEnv.CVs) >= 9 {
			ss.TestEnv.SecondCVs = ss.TrainEnv.CVs[4:8]
		}
		if len(ss.TrainEnv.CVs) >= 5 {
			ss.TestEnv.FirstCVs = ss.TrainEnv.CVs[0:4]
		}
	}
}

// SetTrainingFiles
func (ss *Sim) SetPretrainingFiles(key string) {
	pretrain := true

	switch ss.PreTrnList {
	case "TIMIT_ALL_SX_F":
		ss.PreTrainEnv.SndPath = "ccn_images/sound/timit/"
		ss.PreTrainEnv.WavsPath = "TIMIT/TRAIN/"
		ss.PreTrainEnv.SeqsPath = "TIMIT/TRAIN/"
		ss.PreTrainEnv.TimesPath = "TIMIT/TRAIN/"
		ss.PreTrainEnv.SndTimit = true
		ss.CalcPartWhole = false
		ss.CalcBtwWthin = true
		ss.SaveActs = false

	default:
		fmt.Println("No matching sound set for pretraining - ok if doing train or test")
		pretrain = false
	}

	if pretrain {
		switch ss.PreTrnList {
		case "TIMIT_ALL_SX_F":
			ss.PreTrainEnv.SndList = "trainAllFemaleSX.txt"
			ss.PreTrainEnv.CVs = []string{}
			ss.PreTrainEnv.Silence = false // don't add silence at start		}
		}
	}
}

// SetTestingFiles
func (ss *Sim) SetPretestingFiles(key string) {
	// Test Test Test Test
	ss.PreTestEnv.SndPath = "ccn_images/word_seg_snd_files/"
	test := true

	switch ss.PreTstList {
	// full sequence testing
	case "Holdouts": // holdouts are a subset pulled out of training list
		ss.TestType = SequenceTesting
		ss.PreTestEnv.SeqsPath = ss.TrainEnv.SeqsPath
		ss.PreTestEnv.WavsPath = ss.TrainEnv.WavsPath
		ss.PreTestEnv.TimesPath = ss.TrainEnv.TimesPath
		ss.PreTestEnv.CVsPerWord = ss.TrainEnv.CVsPerWord
		ss.PreTestEnv.CVsPerPos = ss.TrainEnv.CVsPerPos
		ss.CalcPartWhole = false
		ss.CalcBtwWthin = true

	case "CVs_I":
		// for saffran simulation
		ss.TestType = SequenceTesting
		ss.PreTestEnv.SeqsPath = "CV_I_NoCoAr_Seqs/"
		ss.PreTestEnv.WavsPath = "CV_I_NoCoAr_Wavs/"
		ss.PreTestEnv.TimesPath = "CV_I_NoCoAr_Times/"
		ss.PreTestEnv.CVsPerWord = 3 // The saffran, et al experiment used 3 syllable words
		ss.PreTestEnv.CVsPerPos = 4  // The saffran, et al experiment had 4 cv possibilities per syllable position
		ss.CalcPartWhole = false
		ss.CalcBtwWthin = true

	case "MT_IABC":
		// for graf-estes simulation
		ss.TestType = SequenceTesting
		ss.PreTestEnv.SeqsPath = "MT_IABC_Seqs/"
		ss.PreTestEnv.WavsPath = "MT_IABC_Wavs/"
		ss.PreTestEnv.TimesPath = "MT_IABC_Times/"
		ss.PreTestEnv.CVsPerWord = 2 // The graf-estes experiment used 2 syllable words
		ss.PreTestEnv.CVsPerPos = 4  // The graf-estes experiment had 4 cv possibilities per syllable position

	default:
		fmt.Println("No matching sound set for testing - ok if not testing or doing holdout train/test ")
		test = false
		if ss.TestRun == true {
			os.Exit(99)
		}
	}

	if test {
		switch ss.PreTstList {
		// full sequence testing
		case "Holdouts":
			ss.PreTestEnv.SndList = "" // gets set when splitting TrainEnv.SndList
			ss.PreTestEnv.CVs = ss.TrainEnv.CVs
			ss.Holdout = true
			ss.TestEnv.Silence = ss.TrainEnv.Silence

		case "CVs_I":
			ss.PreTestEnv.SndList = "CV_I_NoCoAr_Test.txt"
			ss.PreTestEnv.CVs = CVs_I

		case "MT_IABC":
			ss.PreTestEnv.SndList = "MT_IABC_Test.txt"
			ss.PreTestEnv.CVs = CVs_MT
			ss.PreTestEnv.Silence = true
			ss.TestWordsPart = []string{"ku ga", "pi mo"}
			ss.TestWordsWhole = []string{"do bu", "ti may"}

		default:
			log.Println("No matching case for pretest sndfiles switch")
		}

		// add the silence CV to the list!
		if ss.Holdout == false { // if true test = train so "ss" has already been added
			ss.PreTestEnv.CVs = append(ss.PreTestEnv.CVs, "ss")
		}

		// always set these
		if len(ss.TrainEnv.CVs) >= 13 {
			ss.PreTestEnv.ThirdCVs = ss.TrainEnv.CVs[8:12]
		}
		if len(ss.TrainEnv.CVs) >= 9 {
			ss.PreTestEnv.SecondCVs = ss.TrainEnv.CVs[4:8]
		}
		if len(ss.TrainEnv.CVs) >= 5 {
			ss.PreTestEnv.FirstCVs = ss.TrainEnv.CVs[0:4]
		}
	}
}
