// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// network from job 2150
package main

import (
	"fmt"
	//"github.com/emer/leabra/leabra"
	"log"

	"github.com/emer/emergent/emer"
	"github.com/emer/emergent/params"
	"github.com/emer/emergent/prjn"
	"github.com/emer/emergent/relpos"
	"github.com/emer/etable/etable"
	"github.com/emer/leabra/deep"
)

// WordNet encapsulates the network configuration
type WordNet struct {
	Net          *deep.Network `view:"no-inline" desc:"the network -- click to view / edit parameters for layers, prjns, etc"`
	Pats         *etable.Table `view:"no-inline" desc:"the training patterns to use"`
	Params       params.Sets   `view:"no-inline" desc:"full collection of param sets"`
	ParamSet     string        `desc:"which set of *additional* parameters to use -- always applies Base and optionaly this next if set"`
	LogSetParams bool          `view:"-" desc:"if true, print message for all params that are set"`
	SuperLays    []string      `interactive:"-" desc:"superficial layer names"`
	HidLays      []string      `interactive:"-" desc:"superficial layer names"`
	TRCLays      []string      `interactive:"+" desc:"TRC layer names"`

	// Projections
	Topo22Skp11Prjn      *prjn.PoolTile `view:"-" desc:"feedforward topo prjn 2x by 2y, skip 1x, skip 1y"`
	Topo22Skp11PrjnRecip *prjn.PoolTile `view:"-" desc:"topo reciprocal projection"`

	Topo23Skp12Prjn      *prjn.PoolTile `view:"-" desc:"feedforward topo prjn 2x by 3y, skip 1x, skip 2y"`
	Topo23Skp12PrjnRecip *prjn.PoolTile `view:"-" desc:"topo reciprocal projection"`

	Topo33Skp22Prjn      *prjn.PoolTile `view:"-" desc:"feedforward topo prjn 3x by 3y, skip 2x 2y"`
	Topo33Skp22PrjnRecip *prjn.PoolTile `view:"-" desc:"topo reciprocal projection"`

	Topo33Skp21Prjn      *prjn.PoolTile `view:"-" desc:"feedforward topo prjn 3x by 3y, skip 2x 1y"`
	Topo33Skp21PrjnRecip *prjn.PoolTile `view:"-" desc:"topo reciprocal projection"`

	Topo33Skp12Prjn      *prjn.PoolTile `view:"-" desc:"feedforward topo prjn 3x by 2y, skip 1x 2y"`
	Topo33Skp12PrjnRecip *prjn.PoolTile `view:"-" desc:"topo reciprocal projection"`

	Topo32Skp21Prjn      *prjn.PoolTile `view:"-" desc:"feedforward topo prjn 3x by 2y, skip 2x 1y"`
	Topo32Skp21PrjnRecip *prjn.PoolTile `view:"-" desc:"topo reciprocal projection"`

	Topo32Skp11Prjn      *prjn.PoolTile `view:"-" desc:"feedforward topo prjn 3x by 2y, skip 1x 1y"`
	Topo32Skp11PrjnRecip *prjn.PoolTile `view:"-" desc:"topo reciprocal projection"`

	Topo32Skp01Prjn      *prjn.PoolTile `view:"-" desc:"feedforward topo prjn 3x by 1y, skip 0x 1y"`
	Topo32Skp01PrjnRecip *prjn.PoolTile `view:"-" desc:"topo reciprocal projection"`
}

// New creates new blank elements and initializes defaults
func NewWordNet() *WordNet {
	wn := WordNet{}
	wn.Net = &deep.Network{}
	wn.NewPrjns()
	wn.Params = ParamSets
	return &wn
}

// NewPrjns creates new projections
func (wn *WordNet) NewPrjns() {
	wn.Topo22Skp11Prjn = prjn.NewPoolTile()
	wn.Topo22Skp11Prjn.Wrap = false
	wn.Topo22Skp11Prjn.Size.Set(2, 2)
	wn.Topo22Skp11Prjn.Skip.Set(1, 1)
	wn.Topo22Skp11Prjn.Start.Set(0, 0)
	wn.Topo22Skp11Prjn.TopoRange.Min = 0.8
	wn.Topo22Skp11PrjnRecip = prjn.NewPoolTileRecip(wn.Topo22Skp11Prjn)
	wn.Topo22Skp11PrjnRecip.TopoRange.Min = 0.8

	wn.Topo23Skp12Prjn = prjn.NewPoolTile()
	wn.Topo23Skp12Prjn.Wrap = false
	wn.Topo23Skp12Prjn.Size.Set(2, 3)
	wn.Topo23Skp12Prjn.Skip.Set(1, 2)
	wn.Topo23Skp12Prjn.Start.Set(0, -1)
	wn.Topo23Skp12Prjn.TopoRange.Min = 0.8
	wn.Topo23Skp12PrjnRecip = prjn.NewPoolTileRecip(wn.Topo23Skp12Prjn)
	wn.Topo23Skp12PrjnRecip.TopoRange.Min = 0.8

	wn.Topo32Skp11Prjn = prjn.NewPoolTile()
	wn.Topo32Skp11Prjn.Wrap = false
	wn.Topo32Skp11Prjn.Size.Set(3, 2)
	wn.Topo32Skp11Prjn.Skip.Set(1, 1)
	wn.Topo32Skp11Prjn.Start.Set(0, 0)
	wn.Topo32Skp11Prjn.TopoRange.Min = 0.8
	wn.Topo32Skp11PrjnRecip = prjn.NewPoolTileRecip(wn.Topo32Skp11Prjn)
	wn.Topo32Skp11PrjnRecip.TopoRange.Min = 0.8

	wn.Topo32Skp21Prjn = prjn.NewPoolTile()
	wn.Topo32Skp21Prjn.Wrap = false
	wn.Topo32Skp21Prjn.Size.Set(3, 2)
	wn.Topo32Skp21Prjn.Skip.Set(2, 1)
	wn.Topo32Skp21Prjn.Start.Set(0, 0)
	wn.Topo32Skp21Prjn.TopoRange.Min = 0.8
	wn.Topo32Skp21PrjnRecip = prjn.NewPoolTileRecip(wn.Topo32Skp21Prjn)
	wn.Topo32Skp21PrjnRecip.TopoRange.Min = 0.8

	wn.Topo33Skp12Prjn = prjn.NewPoolTile()
	wn.Topo33Skp12Prjn.Wrap = false
	wn.Topo33Skp12Prjn.Size.Set(3, 3)
	wn.Topo33Skp12Prjn.Skip.Set(1, 2)
	wn.Topo33Skp12Prjn.Start.Set(0, 0)
	wn.Topo33Skp12Prjn.TopoRange.Min = 0.8
	wn.Topo33Skp12PrjnRecip = prjn.NewPoolTileRecip(wn.Topo33Skp12Prjn)
	wn.Topo33Skp12PrjnRecip.TopoRange.Min = 0.8

	wn.Topo33Skp21Prjn = prjn.NewPoolTile()
	wn.Topo33Skp21Prjn.Wrap = false
	wn.Topo33Skp21Prjn.Size.Set(3, 3)
	wn.Topo33Skp21Prjn.Skip.Set(2, 1)
	wn.Topo33Skp21Prjn.Start.Set(0, -1)
	wn.Topo33Skp21Prjn.TopoRange.Min = 0.8
	wn.Topo33Skp21PrjnRecip = prjn.NewPoolTileRecip(wn.Topo33Skp21Prjn)
	wn.Topo33Skp21PrjnRecip.TopoRange.Min = 0.8

	wn.Topo33Skp22Prjn = prjn.NewPoolTile()
	wn.Topo33Skp22Prjn.Wrap = false
	wn.Topo33Skp22Prjn.Size.Set(3, 3)
	wn.Topo33Skp22Prjn.Skip.Set(2, 2)
	wn.Topo33Skp22Prjn.Start.Set(0, 0)
	wn.Topo33Skp22Prjn.TopoRange.Min = 0.8
	wn.Topo33Skp22PrjnRecip = prjn.NewPoolTileRecip(wn.Topo33Skp22Prjn)
	wn.Topo33Skp22PrjnRecip.TopoRange.Min = 0.8

	wn.Topo32Skp01Prjn = prjn.NewPoolTile()
	wn.Topo32Skp01Prjn.Wrap = false
	wn.Topo32Skp01Prjn.Size.Set(3, 2)
	wn.Topo32Skp01Prjn.Skip.Set(0, 1)
	wn.Topo32Skp01Prjn.Start.Set(0, 0)
	wn.Topo32Skp01Prjn.TopoRange.Min = 0.8
	wn.Topo32Skp01PrjnRecip = prjn.NewPoolTileRecip(wn.Topo32Skp01Prjn)
	wn.Topo32Skp01PrjnRecip.TopoRange.Min = 0.8
}

func (wn *WordNet) Config() {
	net := wn.Net
	net.InitName(net, "WordSeg")

	// primary auditory
	a1s := net.AddLayer4D("A1", 12, 6, 2, 7, emer.Input)
	a1s.SetClass("A1")

	rs := net.AddLayer4D("R", 12, 6, 2, 7, emer.Input)
	rs.SetClass("R")

	one2one := prjn.NewOneToOne()
	pOne2One := prjn.NewPoolOneToOne()

	// belt (B)
	cbs, cbct, cbth := net.AddDeep4D("CB", 5, 4, 5, 5)
	cbth.Shape().SetShape([]int{5, 4, 2, 7}, nil, nil)
	cbth.(*deep.TRCLayer).Drivers.Add("A1")
	cbs.SetClass("CB")
	cbct.SetClass("CB")
	cbth.SetClass("CB")
	cbct.RecvPrjns().SendName("CB").SetPattern(one2one)
	cbct.RecvPrjns().SendName("CB").SetClass("ToCT1to1")
	cbth.SetName("CBTh")

	rbs, rbct, rbth := net.AddDeep4D("RB", 5, 4, 5, 5)
	rbth.Shape().SetShape([]int{5, 4, 2, 7}, nil, nil)
	rbth.(*deep.TRCLayer).Drivers.Add("R")
	rbs.SetClass("RB")
	rbct.SetClass("RB")
	rbth.SetClass("RB")
	rbct.RecvPrjns().SendName("RB").SetPattern(one2one)
	rbct.RecvPrjns().SendName("RB").SetClass("ToCT1to1")
	rbth.SetName("RBTh")

	// parabelt (PB)
	cpbs, cpbct, cpbth := net.AddDeep4D("CPB", 5, 3, 5, 5)
	cpbth.Shape().SetShape([]int{5, 3, 2, 7}, nil, nil)
	cpbth.(*deep.TRCLayer).Drivers.Add("A1")
	cpbs.SetClass("CPB")
	cpbct.SetClass("CPBCT")
	cpbth.SetClass("CPBTH")
	cpbct.RecvPrjns().SendName("CPB").SetPattern(one2one)
	cpbct.RecvPrjns().SendName("CPB").SetClass("ToCT1to1")
	cpbth.SetName("CPBTh")

	rpbs, rpbct, rpbth := net.AddDeep4D("RPB", 5, 3, 5, 5)
	rpbth.Shape().SetShape([]int{5, 3, 2, 7}, nil, nil)
	rpbth.(*deep.TRCLayer).Drivers.Add("R")
	rpbs.SetClass("RPB")
	rpbct.SetClass("RPBCT")
	rpbth.SetClass("RPBTH")
	rpbct.RecvPrjns().SendName("RPB").SetPattern(one2one)
	rpbct.RecvPrjns().SendName("RPB").SetClass("ToCT1to1")
	rpbth.SetName("RPBTh")

	// superior temporal
	stss, stsct, ststh := net.AddDeep4D("STS", 5, 3, 6, 6)
	ststh.Shape().SetShape([]int{5, 3, 4, 7}, nil, nil)
	ststh.(*deep.TRCLayer).Drivers.Add("A1", "R")
	stss.SetClass("STS")
	stsct.SetClass("STSCT")
	ststh.SetClass("STSTH")
	stsct.RecvPrjns().SendName("STS").SetPattern(one2one)
	stsct.RecvPrjns().SendName("STS").SetClass("ToCT1to1")
	ststh.SetName("STSTh")

	a1s.SetRelPos(relpos.Rel{Scale: 1.0})
	rs.SetRelPos(relpos.Rel{Rel: relpos.LeftOf, Other: "A1", XAlign: relpos.Left, Space: 10, Scale: 1.0})

	cbs.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "A1", XAlign: relpos.Left, Space: 50})
	cbct.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "CB", XAlign: relpos.Left, Space: 20})
	cbth.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "CBCT", XAlign: relpos.Left, Space: 20, Scale: 1})

	rbs.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "R", XAlign: relpos.Left, Space: 50})
	rbct.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "RB", XAlign: relpos.Left, Space: 20})
	rbth.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "RBCT", XAlign: relpos.Left, Space: 20, Scale: 1})

	cpbs.SetRelPos(relpos.Rel{Rel: relpos.Above, Other: "A1", XAlign: relpos.Left, YAlign: relpos.Front})
	cpbct.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "CPB", XAlign: relpos.Left, Space: 20, Scale: 1.0})
	cpbth.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "CPBCT", XAlign: relpos.Left, Space: 20, Scale: 1.0})

	rpbs.SetRelPos(relpos.Rel{Rel: relpos.LeftOf, Other: "CPB", XAlign: relpos.Left, Space: 10, Scale: 1.0})
	rpbct.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "RPB", XAlign: relpos.Left, Space: 20, Scale: 1.0})
	rpbth.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "RPBCT", XAlign: relpos.Left, Space: 20, Scale: 1.0})

	stss.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "RPBTh", XAlign: relpos.Left, Space: 20, Scale: 1.0})
	stsct.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "STS", XAlign: relpos.Left, Space: 20, Scale: 1.0})
	ststh.SetRelPos(relpos.Rel{Rel: relpos.Behind, Other: "STSCT", XAlign: relpos.Left, Space: 20, Scale: 1.0})

	// primary to belt
	net.ConnectLayers(a1s, cbs, wn.Topo33Skp12Prjn, emer.Forward).SetClass("FwdStd")
	net.ConnectLayers(rs, rbs, wn.Topo33Skp12Prjn, emer.Forward).SetClass("FwdStd")
	net.ConnectLayers(a1s, rbs, wn.Topo33Skp12Prjn, emer.Forward).SetClass("A1ToRB")

	// superficial belt to parabelt
	net.ConnectLayers(cbs, cpbs, wn.Topo22Skp11Prjn, emer.Forward).SetClass("FwdStd")
	net.ConnectLayers(rbs, rpbs, wn.Topo22Skp11Prjn, emer.Forward).SetClass("FwdStd")

	// superficial parabelt to belt
	net.ConnectLayers(cpbs, cbs, wn.Topo22Skp11PrjnRecip, emer.Back).SetClass("Back")
	net.ConnectLayers(rpbs, rbs, wn.Topo22Skp11PrjnRecip, emer.Back).SetClass("Back")

	// ct parabelt to ct belt
	net.ConnectLayers(cpbct, cbct, wn.Topo22Skp11PrjnRecip, emer.Back).SetClass("BackStrong")
	net.ConnectLayers(rpbct, rbct, wn.Topo22Skp11PrjnRecip, emer.Back).SetClass("BackStrong")

	// ct parabelt to sts thalamic
	net.ConnectLayers(rpbct, ststh, wn.Topo22Skp11Prjn, emer.Forward).SetClass("FwdToPulv")
	net.ConnectLayers(cpbct, ststh, wn.Topo22Skp11Prjn, emer.Forward).SetClass("FwdToPulv") // ct parabelt to thalamic belt

	net.ConnectLayers(cpbct, cbth, wn.Topo32Skp11PrjnRecip, emer.Back).SetClass("BackToPulv")
	net.ConnectLayers(rpbct, rbth, wn.Topo32Skp11PrjnRecip, emer.Back).SetClass("BackToPulv")

	//ct sts to ct pb
	net.ConnectLayers(stsct, cpbct, wn.Topo22Skp11PrjnRecip, emer.Back).SetClass("BackWeak")
	net.ConnectLayers(stsct, rpbct, wn.Topo22Skp11PrjnRecip, emer.Back).SetClass("BackWeak")

	// ct sts to thalamic parabelt
	net.ConnectLayers(stsct, cpbth, wn.Topo22Skp11PrjnRecip, emer.Back).SetClass("BackToPulv") // this helps
	net.ConnectLayers(stsct, rpbth, wn.Topo22Skp11PrjnRecip, emer.Back).SetClass("BackToPulv") // this helps

	// superficial parabelt to ct belt
	net.ConnectLayers(cpbs, cbct, wn.Topo22Skp11PrjnRecip, emer.Back).SetClass("BackMax")
	net.ConnectLayers(rpbs, rbct, wn.Topo22Skp11PrjnRecip, emer.Back).SetClass("BackMax")

	// caudal to rostral superficial
	net.ConnectLayers(cbs, rbs, wn.Topo32Skp01Prjn, emer.Forward).SetClass("FwdStrong")
	net.ConnectLayers(cpbs, rpbs, wn.Topo32Skp01Prjn, emer.Forward).SetClass("FwdStrong")

	// rostral to caudal superficial
	net.ConnectLayers(rbs, cbs, wn.Topo32Skp01Prjn, emer.Forward).SetClass("BackWeak")
	net.ConnectLayers(rpbs, cpbs, wn.Topo32Skp01Prjn, emer.Forward).SetClass("BackWeak")

	// superficial parabelt to superficial sts
	net.ConnectLayers(cpbs, stss, wn.Topo32Skp01Prjn, emer.Forward).SetClass("FwdMedium")
	net.ConnectLayers(rpbs, stss, wn.Topo32Skp01Prjn, emer.Forward).SetClass("FwdMedium")

	// superficial sts to superficial parabelt
	net.ConnectLayers(stss, cpbs, wn.Topo32Skp01Prjn, emer.Back).SetClass("BackWeak")
	net.ConnectLayers(stss, rpbs, wn.Topo32Skp01Prjn, emer.Back).SetClass("BackWeak")

	net.ConnectCtxtToCT(cbct, cbct, wn.Topo22Skp11PrjnRecip).SetClass("CTSelfBelt")
	net.ConnectCtxtToCT(cpbct, cpbct, wn.Topo22Skp11PrjnRecip).SetClass("CTSelfParaBelt")
	net.ConnectCtxtToCT(rbct, rbct, wn.Topo22Skp11PrjnRecip).SetClass("CTSelfBelt")
	net.ConnectCtxtToCT(rpbct, rpbct, wn.Topo22Skp11PrjnRecip).SetClass("CTSelfParaBelt")
	net.ConnectCtxtToCT(stsct, stsct, pOne2One).SetClass("CTSelfSTS")

	sameu := prjn.NewPoolSameUnit()
	sameu.SelfCon = false
	net.ConnectLayers(cbs, cbs, sameu, emer.Lateral)
	net.ConnectLayers(rbs, rbs, sameu, emer.Lateral)
	net.ConnectLayers(cpbs, cpbs, sameu, emer.Lateral)
	net.ConnectLayers(rpbs, rpbs, sameu, emer.Lateral)
	net.ConnectLayers(stss, stss, sameu, emer.Lateral)

	wn.TRCLays = make([]string, 0, 10)
	nl := wn.Net.NLayers()
	for li := 0; li < nl; li++ {
		ly := wn.Net.Layer(li)
		if ly.Type() == deep.TRC {
			wn.TRCLays = append(wn.TRCLays, ly.Name())
		}
	}

	net.Defaults()
	wn.SetParams("Network", wn.LogSetParams) // only set Network params

	err := net.Build()
	if err != nil {
		log.Println(err)
		return
	}

	//ar := net.ThreadAlloc(4) // must be done after build
	//ar := net.ThreadReport() // hand tuning now..
	//mpi.Printf("%s", ar)

	a1s.SetThread(0)

	cbs.SetThread(0)
	cbct.SetThread(0)
	cbth.SetThread(0)

	cpbs.SetThread(0)
	cpbct.SetThread(0)
	cpbth.SetThread(0)

	rs.SetThread(1)

	rbs.SetThread(1)
	rbct.SetThread(1)
	rbth.SetThread(1)

	rpbs.SetThread(1)
	rpbct.SetThread(1)
	rpbth.SetThread(1)

	stss.SetThread(0)
	stsct.SetThread(0) // split sts
	ststh.SetThread(0)

	net.InitTopoScales()
	wn.Net.InitWts()
}

/////////////////////////////////////////////////////////////////////////
//   Params setting

// ParamsName returns name of current set of parameters
func (wn *WordNet) ParamsName() string {
	if wn.ParamSet == "" {
		return "Base"
	}
	return wn.ParamSet
}

// SetParams sets the params for "Base" and then current ParamSet.
// If sheet is empty, then it applies all avail sheets (e.g., Network, Sim)
// otherwise just the named sheet
// if setMsg = true then we output a message for each param that was set.
func (wn *WordNet) SetParams(sheet string, setMsg bool) error {
	if sheet == "" {
		// this is important for catching typos and ensuring that all sheets can be used
		wn.Params.ValidateSheets([]string{"Network", "Sim"})
	}
	err := wn.SetParamsSet("Base", sheet, setMsg)
	if wn.ParamSet != "" && wn.ParamSet != "Base" {
		err = wn.SetParamsSet(wn.ParamSet, sheet, setMsg)
		if err == nil {
			fmt.Printf("Using ParamSet: %s\n", wn.ParamSet)
		}
	}
	return err
}

// SetParamsSet sets the params for given params.Set name.
// If sheet is empty, then it applies all avail sheets (e.g., Network, Sim)
// otherwise just the named sheet
// if setMsg = true then we output a message for each param that was set.
func (wn *WordNet) SetParamsSet(setNm string, sheet string, setMsg bool) error {
	pset, err := wn.Params.SetByNameTry(setNm)
	if err != nil {
		return err
	}
	if sheet == "" || sheet == "Network" {
		netp, ok := pset.Sheets["Network"]
		if ok {
			wn.Net.ApplyParams(netp, setMsg)
		}
	}

	if sheet == "" || sheet == "Sim" {
		simp, ok := pset.Sheets["Sim"]
		if ok {
			simp.Apply(wn, setMsg)
		}
	}
	// note: if you have more complex environments with parameters, definitely add
	// sheets for them, e.g., "TrainEnv", "TestEnv" etc
	return err
}
