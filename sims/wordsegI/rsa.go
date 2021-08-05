// Copyright (c) 2020, The CCNLab Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"

	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"github.com/emer/etable/metric"
	"github.com/emer/etable/simat"
)

var Debug = false

// Object categories
var Objs = []string{
	"b", // ba, bi, bo, bu
	"d", // ...
	"g",
	"h",
	"k",
	"l",
	"m",
	"n",
	"p",
	"r",
	"s",
	"t",
}
var ObjIdxs map[string]int

// MannerCat is a categorization by manner of articulation
var MannerCats = map[string]string{
	"b":   "stop",
	"d":   "stop",
	"g":   "stop",
	"p":   "stop",
	"t":   "stop",
	"k":   "stop",
	"dx":  "stop",
	"q":   "stop",
	"bcl": "closure", // stop closures
	"dcl": "closure",
	"gcl": "closure",
	"pcl": "closure",
	"tck": "closure",
	"kcl": "closure",

	"jh": "affricative",
	"ch": "affricative",
	//"dcl": "closure", // same closure as 'd'
	"tcl": "closure",

	"s":  "fricative",
	"sh": "fricative",
	"z":  "fricative",
	"zh": "fricative",
	"f":  "fricative",
	"th": "fricative",
	"v":  "fricative",
	"dh": "fricative",

	"m":   "nasal",
	"n":   "nasal",
	"ng":  "nasal",
	"em":  "nasal",
	"en":  "nasal",
	"eng": "nasal",
	"nx":  "nasal",

	//"h": "glottal-fricative",
	"l":  "glide",
	"r":  "glide",
	"w":  "glide",
	"y":  "glide",
	"hh": "glide",
	"hv": "glide",
	"el": "glide",

	"iy":   "vowel",
	"ih":   "vowel",
	"eh":   "vowel",
	"ey":   "vowel",
	"ae":   "vowel",
	"aa":   "vowel",
	"aw":   "vowel",
	"ay":   "vowel",
	"ah":   "vowel",
	"ao":   "vowel",
	"oy":   "vowel",
	"ow":   "vowel",
	"uh":   "vowel",
	"uw":   "vowel",
	"ux":   "vowel",
	"er":   "vowel",
	"ax":   "vowel",
	"ix":   "vowel",
	"axr":  "vowel",
	"ax-h": "vowel",
}

// PlaceCat is a categorization by place of articulation
var PlaceCats = map[string]string{
	"b": "bilabial",
	"d": "alveolar",
	"g": "velar",
	"k": "velar",
	"p": "bilabial",
	"t": "alveolar",
	"m": "bilabial",
	"n": "alveolar",
	"s": "alveolar",
	"h": "glottal",
	"l": "alveolar",
	"r": "alveolar",
}

var CatsBlanks []string // cats with repeats all blank -- for labels

// RSA handles representational similarity analysis
type RSA struct {
	Interval int                      `desc:"how often to run RSA analyses over epochs"`
	Cats     []string                 `desc:"category names for each row of simmat / activation table -- call SetCats"`
	Sims     map[string]*simat.SimMat `desc:"similarity matricies for each layer"`
	V1Sims   []float64                `desc:"similarity for each layer relative to V1"`
	//CatDists   []float64                `desc:"AvgContrastDist for each layer under MannerCats centroid meta categories"`
	//BasicDists []float64                `desc:"AvgBasicDist for each layer -- basic-level distances"`
	//ExptDists  []float64                `desc:"AvgExptDist for each layer -- distances from expt data"`
	//MannerSims map[string]*simat.SimMat `desc:"similarity matricies for each layer, organized into MannerCats and sorted"`
	//MannerObjs map[string]*[]string     `desc:"corresponding ordering of objects in sorted Cat5Sims lists"`
	//PermNCats  map[string]int           `desc:"number of categories remaining after permutation from LbaCat"`
	//PermDists  map[string]float64       `desc:"avg contrast dist for permutation"`
}

// Init initializes maps etc if not done yet
func (rs *RSA) Init(lays []string) { // are we doing RSA on consonant vowel sequences or phones (timit data)
	if rs.Sims != nil {
		return
	}
	nc := len(lays)
	rs.Sims = make(map[string]*simat.SimMat, nc)
	//rs.MannerSims = make(map[string]*simat.SimMat, nc)
	//rs.MannerObjs = make(map[string]*[]string, nc)
	rs.V1Sims = make([]float64, nc)
	//rs.CatDists = make([]float64, nc)
	//rs.BasicDists = make([]float64, nc)
	//rs.ExptDists = make([]float64, nc)
	//rs.PermNCats = make(map[string]int)
	//rs.PermDists = make(map[string]float64)

	if ObjIdxs == nil {
		no := len(Objs)
		ObjIdxs = make(map[string]int, no)
		CatsBlanks = make([]string, no)
		lstcat := ""
		for i, o := range Objs {
			ObjIdxs[o] = i
			cat := MannerCats[o]
			if cat != lstcat {
				CatsBlanks[i] = cat
				lstcat = cat
			}
		}
		//rs.OpenExptMat()
	}
}

// SetCats sets the categories from given list of category/object_file names
func (rs *RSA) SetCats(objs []string) {
	l := len(objs)
	rs.Cats = make([]string, 0, l*l)
	for _, ob := range objs {
		cat := strings.Split(ob, "/")[0]
		rs.Cats = append(rs.Cats, cat)
	}
}

func (rs *RSA) SimByName(cn string) *simat.SimMat {
	sm, ok := rs.Sims[cn]
	if !ok || sm == nil {
		sm = &simat.SimMat{}
		rs.Sims[cn] = sm
	}
	return sm
}

//func (rs *RSA) Cat5SimByName(cn string) *simat.SimMat {
//	sm, ok := rs.Cat5Sims[cn]
//	if !ok || sm == nil {
//		sm = &simat.SimMat{}
//		rs.Cat5Sims[cn] = sm
//	}
//	return sm
//}

//func (rs *RSA) Cat5ObjByName(cn string) *[]string {
//	sm, ok := rs.Cat5Objs[cn]
//	if !ok || sm == nil {
//		nsm := sliceclone.String(rs.Cats)
//		sm = &nsm
//		rs.Cat5Objs[cn] = sm
//	}
//	return sm
//}

// StatsFmActs computes RSA stats from given acts table, for given columns (layer names)
func (rs *RSA) StatsFmActs(acts *etable.Table, layNms []string) {
	//segment := 0 // use the first segment of phoneme
	tix := etable.NewIdxView(acts)
	//tix.Filter(func(et *etable.Table, row int) bool { // if we want to filter by segment
	//	tck := int(et.CellFloat("Segment", row))
	//	return tck == segment
	//})

	//tix.SortCol(acts.ColIdx("Cons"), true)
	//for _, cn := range layNms {
	//	sm := rs.SimByName(cn + "_Cons")
	//	rs.SimMatFmActs(sm, tix, cn, "Cons")
	//}

	tix.SortCol(acts.ColIdx("MannerCat"), true)
	for _, cn := range layNms {
		sm := rs.SimByName(cn + "_Manner")
		rs.SimMatFmActs(sm, tix, cn, "MannerCat")
	}

	tix.SortCol(acts.ColIdx("PlaceCat"), true)
	for _, cn := range layNms {
		sm := rs.SimByName(cn + "_Place")
		rs.SimMatFmActs(sm, tix, cn, "PlaceCat")
	}

	//osm := rs.SimByName(cn + "_Obj")
	//rs.ObjSimMat(osm, sm, rs.Cats)
	//
	//dist := metric.CrossEntropy64(osm.Mat.(*etensor.Float64).Values, expt.Mat.(*etensor.Float64).Values)
	//rs.ExptDists[i] = dist

	//v1sm := rs.Sims["V1m"]
	//v1sm64 := v1sm.Mat.(*etensor.Float64)
	//for i, cn := range layNms {
	//	osm := rs.SimByName(cn)
	//
	//	rs.CatDists[i] = -rs.AvgContrastDist(osm, rs.Cats, MannerCats)
	//	rs.BasicDists[i] = rs.AvgBasicDist(osm, rs.Cats)
	//
	//	if v1sm == osm {
	//		rs.V1Sims[i] = 1
	//		continue
	//	}
	//	osm64 := osm.Mat.(*etensor.Float64)
	//	rs.V1Sims[i] = metric.Correlation64(osm64.Values, v1sm64.Values)
	//}
	//cat5s := []string{"TE"}
	//for _, cn := range cat5s {
	//	rs.StatsSortPermuteCat5(cn)
	//}
}

//func (rs *RSA) StatsSortPermuteCat5(laynm string) {
//	sm := rs.SimByName(laynm)
//	if len(sm.Rows) == 0 {
//		return
//	}
//	sm5 := rs.Cat5SimByName(laynm)
//	obj := rs.CatSortSimMat(sm, sm5, rs.Cats, MannerCats, true, laynm+"_LbaCat")
//	obj5 := rs.Cat5ObjByName(laynm)
//	copy(*obj5, obj)
//	pnm := laynm + "perm"
//	pcats, ncat, pdist := rs.PermuteCatTest(sm, rs.Cats, MannerCats, pnm)
//	sm5p := rs.Cat5SimByName(pnm)
//	objp := rs.CatSortSimMat(sm, sm5p, rs.Cats, pcats, true, pnm)
//	obj5p := rs.Cat5ObjByName(pnm)
//	copy(*obj5p, objp)
//	rs.PermNCats[laynm] = ncat
//	rs.PermDists[laynm] = pdist
//}

// ConfigSimMat sets meta data
func (rs *RSA) ConfigSimMat(sm *simat.SimMat) {
	smat := sm.Mat.(*etensor.Float64)
	smat.SetMetaData("max", "2")
	smat.SetMetaData("min", "0")
	smat.SetMetaData("colormap", "Viridis")
	smat.SetMetaData("grid-fill", "1")
	smat.SetMetaData("dim-extra", "0.5")
	smat.SetMetaData("grid-min", "1")
}

// SimMatFmActs computes the given SimMat from given acts table (IdxView),
// for given column name.
func (rs *RSA) SimMatFmActs(sm *simat.SimMat, acts *etable.IdxView, colnm string, varNm string) {
	sm.Init()
	rs.ConfigSimMat(sm)

	n := acts.Table.Rows
	smat := sm.Mat.(*etensor.Float64)
	smat.SetShape([]int{n, n}, nil, nil)

	sm.Rows = make([]string, n)
	for r := 0; r < n; r++ {
		sm.Rows[r] = acts.Table.CellString(varNm, r)
	}
	sm.Cols = sm.Rows
	smat.SetMetaData("max", "1")
	smat.SetMetaData("min", "0")
	smat.SetMetaData("colormap", "Viridis")
	smat.SetMetaData("grid-fill", "1")
	smat.SetMetaData("dim-extra", "0.15")
	smat.SetMetaData("grid-min", "1")

	sm.TableCol(acts, colnm, varNm, true, metric.Correlation64)
}

// OpenSimMat opens a saved sim mat for given layer name,
// using given cat strings per row of sim mat
//func (rs *RSA) OpenSimMat(laynm string, fname gi.FileName) {
//	sm := rs.SimByName(laynm)
//	no := len(rs.Cats)
//	sm.Init()
//	rs.ConfigSimMat(sm)
//	smat := sm.Mat.(*etensor.Float64)
//	smat.SetShape([]int{no, no}, nil, nil)
//	err := etensor.OpenCSV(smat, fname, etable.Tab.Rune())
//	if err != nil {
//		log.Println(err)
//		return
//	}
//	sm.Rows = simat.BlankRepeat(rs.Cats)
//	sm.Cols = sm.Rows
//	rs.StatsSortPermuteCat5(laynm)
//	rs.PermDists[laynm+"_BasicDist"] = rs.AvgBasicDist(sm, rs.Cats)
//
//	expt := rs.SimByName("Expt1")
//
//	osm := rs.SimByName(laynm + "_Obj")
//	rs.ObjSimMat(osm, sm, rs.Cats)
//	dist := metric.CrossEntropy64(osm.Mat.(*etensor.Float64).Values, expt.Mat.(*etensor.Float64).Values)
//	rs.PermDists[laynm+"_ExptDist"] = dist
//
//}

// CatSortSimMat takes an input sim matrix and categorizes the items according to given cats
// and then sorts items within that according to their average within - between cat similarity.
// contrast = use within - between metric, otherwise just within
// returns the new ordering of objects (like nms but sorted according to new sort)
//func (rs *RSA) CatSortSimMat(insm *simat.SimMat, osm *simat.SimMat, nms []string, catmap map[string]string, contrast bool, name string) []string {
//	no := len(insm.Rows)
//	sch := etable.Schema{
//		{"Cat", etensor.STRING, nil, nil},
//		{"Dist", etensor.FLOAT64, nil, nil},
//		{"Obj", etensor.STRING, nil, nil},
//	}
//	dt := &etable.Table{}
//	dt.SetFromSchema(sch, no)
//	cats := dt.Cols[0].(*etensor.String).Values
//	dists := dt.Cols[1].(*etensor.Float64).Values
//	objs := dt.Cols[2].(*etensor.String).Values
//	for i, nm := range nms {
//		cats[i] = catmap[nm]
//		objs[i] = nm
//	}
//	smatv := insm.Mat.(*etensor.Float64).Values
//	avgCtrstDist := 0.0
//	for ri := 0; ri < no; ri++ {
//		roff := ri * no
//		aid := 0.0
//		ain := 0
//		abd := 0.0
//		abn := 0
//		rc := cats[ri]
//		for ci := 0; ci < no; ci++ {
//			if ri == ci {
//				continue
//			}
//			cc := cats[ci]
//			d := smatv[roff+ci]
//			if cc == rc {
//				aid += d
//				ain++
//			} else {
//				abd += d
//				abn++
//			}
//		}
//		if ain > 0 {
//			aid /= float64(ain)
//		}
//		if abn > 0 {
//			abd /= float64(abn)
//		}
//		dval := aid
//		if contrast {
//			dval -= abd
//		}
//		dists[ri] = dval
//		avgCtrstDist += (1 - aid) - (1 - abd)
//	}
//	avgCtrstDist /= float64(no)
//	ix := etable.NewIdxView(dt)
//	ix.SortColNames([]string{"Cat", "Dist"}, true) // ascending
//	osm.Init()
//	osm.Mat.CopyShapeFrom(insm.Mat)
//	osm.Mat.CopyMetaData(insm.Mat)
//	rs.ConfigSimMat(osm)
//	omatv := osm.Mat.(*etensor.Float64).Values
//	bcols := make([]string, no)
//	last := ""
//	for sri := 0; sri < no; sri++ {
//		sroff := sri * no
//		ri := ix.Idxs[sri]
//		roff := ri * no
//		cat := cats[ri]
//		if cat != last {
//			bcols[sri] = cat
//			last = cat
//		}
//		// bcols[sri] = nms[ri] // uncomment this to see all the names
//		for sci := 0; sci < no; sci++ {
//			ci := ix.Idxs[sci]
//			d := smatv[roff+ci]
//			omatv[sroff+sci] = d
//		}
//	}
//	osm.Rows = bcols
//	osm.Cols = bcols
//	if Debug {
//		fmt.Printf("%v  avg contrast dist: %.4f\n", name, avgCtrstDist)
//	}
//	sobjs := make([]string, no)
//	for i := 0; i < no; i++ {
//		nm := nms[ix.Idxs[i]]
//		sobjs[i] = catmap[nm] + ": " + nm
//	}
//	return sobjs
//}

// AvgContrastDist computes average contrast dist over given cat map
// nms gives the base category names for each row in the simat, which is
// then used to lookup the meta category in the catmap, which is used
// for determining the within vs. between category status.
//func (rs *RSA) AvgContrastDist(insm *simat.SimMat, nms []string, catmap map[string]string) float64 {
//	no := len(insm.Rows)
//	smatv := insm.Mat.(*etensor.Float64).Values
//	avgd := 0.0
//	for ri := 0; ri < no; ri++ {
//		roff := ri * no
//		aid := 0.0
//		ain := 0
//		abd := 0.0
//		abn := 0
//		rnm := nms[ri]
//		rc := catmap[rnm]
//		for ci := 0; ci < no; ci++ {
//			if ri == ci {
//				continue
//			}
//			cnm := nms[ci]
//			cc := catmap[cnm]
//			d := smatv[roff+ci]
//			if cc == rc {
//				aid += d
//				ain++
//			} else {
//				abd += d
//				abn++
//			}
//		}
//		if ain > 0 {
//			aid /= float64(ain)
//		}
//		if abn > 0 {
//			abd /= float64(abn)
//		}
//		avgd += aid - abd
//	}
//	avgd /= float64(no)
//	return avgd
//}

// AvgBasicDist computes average distance within basic-level categories given by nms
//func (rs *RSA) AvgBasicDist(insm *simat.SimMat, nms []string) float64 {
//	no := len(insm.Rows)
//	smatv := insm.Mat.(*etensor.Float64).Values
//	avgd := 0.0
//	ain := 0
//	for ri := 0; ri < no; ri++ {
//		roff := ri * no
//		rnm := nms[ri]
//		for ci := 0; ci < ri; ci++ {
//			cnm := nms[ci]
//			d := smatv[roff+ci]
//			if rnm == cnm {
//				avgd += d
//				ain++
//			}
//		}
//	}
//	if ain > 0 {
//		avgd /= float64(ain)
//	}
//	return avgd
//}

// PermuteCatTest takes an input sim matrix and tries all one-off permutations relative to given
// initial set of categories, and computes overall average constrast distance for each
// selects categs with lowest dist and iterates until no better permutation can be found.
// returns new map, number of categories used in new map, and the avg contrast distance for it
//func (rs *RSA) PermuteCatTest(insm *simat.SimMat, nms []string, catmap map[string]string, desc string) (map[string]string, int, float64) {
//	if Debug {
//		fmt.Printf("\n#########\n%v\n", desc)
//	}
//	catm := map[string]int{} // list of categories and index into catnms
//	catnms := []string{}
//	for _, nm := range nms {
//		cat := catmap[nm]
//		if _, has := catm[cat]; !has {
//			catm[cat] = len(catnms)
//			catnms = append(catnms, cat)
//		}
//	}
//	ncats := len(catnms)
//
//	itrmap := make(map[string]string)
//	for k, v := range catmap {
//		itrmap[k] = v
//	}
//
//	std := rs.AvgContrastDist(insm, nms, catmap)
//	if Debug {
//		fmt.Printf("std: %.4f  starting\n", std)
//	}
//
//	for itr := 0; itr < 100; itr++ {
//		std = rs.AvgContrastDist(insm, nms, itrmap)
//
//		effmap := make(map[string]string)
//		mind := 100.0
//		mindnm := ""
//		mindcat := ""
//		for _, nm := range nms { // go over each item
//			cat := itrmap[nm]
//			for oc := 0; oc < ncats; oc++ { // go over alternative categories
//				ocat := catnms[oc]
//				if ocat == cat {
//					continue
//				}
//				for k, v := range itrmap {
//					if k == nm {
//						effmap[k] = ocat // switch
//					} else {
//						effmap[k] = v
//					}
//				}
//				avgd := rs.AvgContrastDist(insm, nms, effmap)
//				if avgd < mind {
//					mind = avgd
//					mindnm = nm
//					mindcat = ocat
//				}
//				// if avgd < std {
//				// 	fmt.Printf("Permute test better than std dist: %v  min dist: %v  for name: %v  in cat: %v\n", std, avgd, nm, ocat)
//				// }
//			}
//		}
//		if mind >= std {
//			break
//		}
//		if Debug {
//			fmt.Printf("itr %v std: %.4f  min: %.4f  name: %v  cat: %v\n", itr, std, mind, mindnm, mindcat)
//		}
//		itrmap[mindnm] = mindcat // make the switch
//	}
//	if Debug {
//		fmt.Printf("std: %.4f  final\n", std)
//	}
//
//	nCatUsed := 0
//	for oc := 0; oc < ncats; oc++ {
//		cat := catnms[oc]
//		if Debug {
//			fmt.Printf("%v\n", cat)
//		}
//		nin := 0
//		for _, nm := range Objs {
//			ct := itrmap[nm]
//			if ct == cat {
//				nin++
//				if Debug {
//					fmt.Printf("\t%v\n", nm)
//				}
//			}
//		}
//		if nin > 0 {
//			nCatUsed++
//		}
//	}
//	return itrmap, nCatUsed, -std
//}

// ObjSimMat compresses full simat into a much smaller per-object sim mat
//func (rs *RSA) ObjSimMat(osm *simat.SimMat, fsm *simat.SimMat, nms []string) {
//	fsmat := fsm.Mat.(*etensor.Float64)
//
//	ono := len(Objs)
//	osm.Init()
//	osmat := osm.Mat.(*etensor.Float64)
//	osmat.SetShape([]int{ono, ono}, nil, nil)
//	osm.Rows = CatsBlanks
//	osm.Cols = CatsBlanks
//	osmat.SetMetaData("max", "1")
//	osmat.SetMetaData("min", "0")
//	osmat.SetMetaData("colormap", "Viridis")
//	osmat.SetMetaData("grid-fill", "1")
//	osmat.SetMetaData("dim-extra", "0.15")
//
//	nmat := &etensor.Float64{}
//	nmat.SetShape([]int{ono, ono}, nil, nil)
//
//	nf := len(nms)
//	for ri := 0; ri < nf; ri++ {
//		roi := ObjIdxs[nms[ri]]
//		for ci := 0; ci < nf; ci++ {
//			sidx := ri*nf + ci
//			sval := fsmat.Values[sidx]
//			coi := ObjIdxs[nms[ci]]
//			oidx := roi*ono + coi
//			if ri == ci {
//				osmat.Values[oidx] = 0
//			} else {
//				osmat.Values[oidx] += sval
//			}
//			nmat.Values[oidx] += 1
//		}
//	}
//	for ri := 0; ri < ono; ri++ {
//		for ci := 0; ci < ono; ci++ {
//			oidx := ri*ono + ci
//			osmat.Values[oidx] /= nmat.Values[oidx]
//		}
//	}
//	norm.DivNorm64(osmat.Values, norm.Max64)
//}

//func (rs *RSA) OpenExptMat() {
//	no := len(Objs)
//	sm := rs.SimByName("Expt1")
//	sm.Init()
//	smat := sm.Mat.(*etensor.Float64)
//	smat.SetShape([]int{no, no}, nil, nil)
//	err := etensor.OpenCSV(smat, gi.FileName("expt1_simat.csv"), etable.Comma.Rune())
//	if err != nil {
//		log.Println(err)
//		return
//	}
//	norm.DivNorm64(smat.Values, norm.Max64)
//	sm.Rows = CatsBlanks
//	sm.Cols = CatsBlanks
//	smat.SetMetaData("max", "1")
//	smat.SetMetaData("min", "0")
//	smat.SetMetaData("colormap", "Viridis")
//	smat.SetMetaData("grid-fill", "1")
//	smat.SetMetaData("dim-extra", "0.15")
//}
