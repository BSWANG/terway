// Code generated by "stringer -type=ConditionType -trimprefix=Condition"; DO NOT EDIT.

package eni

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Full-0]
	_ = x[ResourceTypeMismatch-1]
	_ = x[NetworkInterfaceMismatch-2]
	_ = x[InsufficientVSwitchIP-3]
}

const _ConditionType_name = "FullResourceTypeMismatchNetworkInterfaceMismatchInsufficientVSwitchIP"

var _ConditionType_index = [...]uint8{0, 4, 24, 48, 69}

func (i ConditionType) String() string {
	if i < 0 || i >= ConditionType(len(_ConditionType_index)-1) {
		return "ConditionType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ConditionType_name[_ConditionType_index[i]:_ConditionType_index[i+1]]
}
