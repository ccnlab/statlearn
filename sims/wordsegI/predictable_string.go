// Code generated by "stringer -type=Predictable"; DO NOT EDIT.

package main

import (
	"errors"
	"strconv"
)

var _ = errors.New("dummy error")

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Ignore-0]
	_ = x[Fully-1]
	_ = x[Partially-2]
	_ = x[Unpredictable-3]
	_ = x[PredictableN-4]
}

const _Predictable_name = "IgnoreFullyPartiallyUnpredictablePredictableN"

var _Predictable_index = [...]uint8{0, 6, 11, 20, 33, 45}

func (i Predictable) String() string {
	if i < 0 || i >= Predictable(len(_Predictable_index)-1) {
		return "Predictable(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Predictable_name[_Predictable_index[i]:_Predictable_index[i+1]]
}

func (i *Predictable) FromString(s string) error {
	for j := 0; j < len(_Predictable_index)-1; j++ {
		if s == _Predictable_name[_Predictable_index[j]:_Predictable_index[j+1]] {
			*i = Predictable(j)
			return nil
		}
	}
	return errors.New("String: " + s + " is not a valid option for type: Predictable")
}
