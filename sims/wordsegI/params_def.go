// Copyright (c) 2020, The CCNLab Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// params from job 2150

package main

import "github.com/emer/emergent/params"

// ParamSets is the default set of parameters -- Base is always applied, and others can be optionally
// selected to apply on top of that
var ParamSets = params.Sets{
	{Name: "Base", Desc: "these are the best params", Sheets: params.Sheets{
		"Network": &params.Sheet{
			// layer classes, specifics
			{Sel: "Layer", Desc: "",
				Params: params.Params{
					"Layer.Learn.AvgL.Gain":   "2.0", // 2 better than 3, 1 no better than 2
					"Layer.Act.Gbar.L":        "0.2", // .1 or .3 are worse (lower cos diff)
					"Layer.Act.Init.Decay":    "0.0", // this used to be default for deep, now set manually
					"Layer.Inhib.Layer.FBTau": "1.4", // smoother = faster? but worse?
					"Layer.Inhib.Pool.FBTau":  "1.4", // smoother = faster?
					// noise no help
				}},
			{Sel: "TRCLayer", Desc: "avg mix param",
				Params: params.Params{
					"Layer.TRC.AvgMix":   "0.5", // .3 > .2 > .1 for cb/rb cos diff -- higher def better
					"Layer.TRC.MaxInhib": ".01", // .6 is default
				}},
			{Sel: ".A1", Desc: "A1 uses pool inhib", // A1 is caudal
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.8",
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.8",
				}},
			{Sel: ".R", Desc: "A1 uses pool inhib", // R is rostral
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.8",
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.8",
				}},
			{Sel: ".CB", Desc: "CB uses pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6",
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.7",
					//"Layer.Inhib.ActAvg.Init": "0.06",
				}},
			{Sel: ".CPB", Desc: "CB uses pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6",
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.8",
					//"Layer.Inhib.ActAvg.Init": "0.06",
				}},
			{Sel: ".CPBCT", Desc: "CB uses pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6", // reducing to this point helped
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.8",
					//"Layer.Inhib.ActAvg.Init": "0.06",
				}},
			{Sel: ".CPBTH", Desc: "CB uses pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6",
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.8", // reducing more no help
					//"Layer.Inhib.ActAvg.Init": "0.06",
				}},
			{Sel: ".RB", Desc: "RB uses pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6",
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.7",
					//"Layer.Inhib.ActAvg.Init": "0.06",
				}},
			{Sel: ".RPB", Desc: "RB uses pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6",
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.8",
					//"Layer.Inhib.ActAvg.Init": "0.06",
				}},
			{Sel: ".RPBCT", Desc: "RB uses pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6", // reducing to this point helped
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.8",
					//"Layer.Inhib.ActAvg.Init": "0.06",
				}},
			{Sel: ".RPBTH", Desc: "RB uses pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6",
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.8", // reducing more no help
					//"Layer.Inhib.ActAvg.Init": "0.06",
				}},
			{Sel: ".STS", Desc: "STS does not use pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6",
					"Layer.Inhib.Pool.On":  "true", // false definitely worse
					"Layer.Inhib.Pool.Gi":  "1.7",
					//"Layer.Learn.AvgL.Gain":   "2.0",
					//"Layer.Inhib.ActAvg.Init": "0.12",
				}},
			{Sel: ".STSCT", Desc: "STS does not use pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6",
					"Layer.Inhib.Pool.On":  "true", // false definitely worse
					"Layer.Inhib.Pool.Gi":  "1.8",
					//"Layer.Learn.AvgL.Gain":   "2.0",
					//"Layer.Inhib.ActAvg.Init": "0.12",
				}},
			{Sel: ".STSTH", Desc: "STS uses pool inhib",
				Params: params.Params{
					"Layer.Inhib.Layer.Gi": "1.6",
					"Layer.Inhib.Pool.On":  "true",
					"Layer.Inhib.Pool.Gi":  "1.8", // reducing more no help
					//"Layer.Inhib.ActAvg.Init": "0.06",
				}},

			// prjn classes, specifics
			{Sel: "Prjn", Desc: "norm and momentum on works better, but wt bal is not better for smaller nets",
				Params: params.Params{
					"Prjn.Learn.Lrate":         "0.01",  // .02 seems good, .01 used for paper, lower increases slope for pretrain but not for train
					"Prjn.Learn.Norm.On":       "true",  // false a bit worse with or without momentum (8/20/20)
					"Prjn.Learn.Momentum.On":   "false", // on significantly increasing hogging
					"Prjn.Learn.WtBal.On":      "true",  // false worse
					"Prjn.Learn.Momentum.MTau": "5",     // if Momentum.On == true
				}},
			{Sel: ".A1ToRB", Desc: "caudal feeding rostral at next higher area",
				Params: params.Params{
					"Prjn.WtScale.Rel": ".5",
				}},
			{Sel: ".CBToRPB", Desc: "caudal feeding rostral at next higher area",
				Params: params.Params{
					"Prjn.WtScale.Rel": ".2",
				}},
			{Sel: ".FwdStd", Desc: "standard feedforward",
				Params: params.Params{
					"Prjn.WtScale.Rel": "1.0",
				}},
			{Sel: ".FwdWeak", Desc: "weak feedforward",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.05",
				}},
			{Sel: ".FwdMedium", Desc: "medium feedforward",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.1",
				}},
			{Sel: ".FwdStrong", Desc: "strong feedforward",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.2",
				}},
			{Sel: ".FwdMax", Desc: "max feedforward",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.5",
				}},
			{Sel: ".BackWeak", Desc: "weak .05",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.05",
				}},
			{Sel: ".Back", Desc: "top-down back-projections MUST have lower relative weight scale, otherwise network hallucinates -- smaller as network gets bigger",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.1",
				}},
			{Sel: ".BackStrong", Desc: "strong .2",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.2",
				}},
			{Sel: ".BackMax", Desc: "strongest",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.5", // .1 > .2, orig .5 -- see BackStrong
				}},
			{Sel: ".FwdToPulv", Desc: "feedforward to pulvinar directly",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.2",
				}},
			{Sel: ".BackToPulv", Desc: "top-down to pulvinar directly",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.4",
				}},
			{Sel: ".FmPulv", Desc: "default for pulvinar",
				Params: params.Params{
					"Prjn.WtScale.Rel": ".1", // .2 > .1 > .05 still true - well perhaps
				}},
			{Sel: ".Lateral", Desc: "default for lateral",
				Params: params.Params{
					"Prjn.WtInit.Sym":  "false",
					"Prjn.WtScale.Rel": "0.02",
					"Prjn.WtInit.Mean": "0.5",
					"Prjn.WtInit.Var":  "0",
				}},
			{Sel: "CTCtxtPrjn", Desc: "defaults for CT Ctxt prjns",
				Params: params.Params{
					"Prjn.WtScale.Rel": "1",
				}},
			{Sel: ".CTSelfBelt", Desc: "CT to CT for belt layers",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.5",
				}},
			{Sel: ".CTSelfParaBelt", Desc: "CT to CT for parabelt layers",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.5", // lower not better
				}},
			{Sel: ".CTSelfSTS", Desc: "CT to CT for sts layer",
				Params: params.Params{
					"Prjn.WtScale.Rel": "0.5", // lower not better
				}},
			{Sel: ".ToCT1to1", Desc: "1to1 has no weight var... fixed?",
				Params: params.Params{
					"Prjn.WtInit.Mean": "0.5",
					"Prjn.WtInit.Var":  "0",
				}},
		},
	}},
}
