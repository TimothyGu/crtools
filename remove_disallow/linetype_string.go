// Code generated by "stringer -type LineType"; DO NOT EDIT.

package main

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Unrelated-0]
	_ = x[Empty-1]
	_ = x[PreprocessorDirective-2]
	_ = x[DisallowCopy-3]
	_ = x[DisallowAssign-4]
	_ = x[DisallowCopyAndAssign-5]
	_ = x[DisallowImplicitConstructors-6]
	_ = x[ClassDecl-7]
	_ = x[Label-8]
}

const _LineType_name = "UnrelatedEmptyPreprocessorDirectiveDisallowCopyDisallowAssignDisallowCopyAndAssignDisallowImplicitConstructorsClassDeclLabel"

var _LineType_index = [...]uint8{0, 9, 14, 35, 47, 61, 82, 110, 119, 124}

func (i LineType) String() string {
	if i < 0 || i >= LineType(len(_LineType_index)-1) {
		return "LineType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _LineType_name[_LineType_index[i]:_LineType_index[i+1]]
}
