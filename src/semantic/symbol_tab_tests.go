package semantic

import (
	"reflect"
	"testing"

	"dash-lang.io/src/internal"
	"dash-lang.io/src/types"
)

func TestFunctionTable(t *testing.T) {
	tests := []struct {
		name     string
		fn       string
		info     FnInfo
		wantInfo FnInfo
		wantOk   bool
	}{
		{
			name: "add and retrieve",
			fn:   "TestFunc",
			info: FnInfo{Type: &types.Function{
				Arg: []types.TypeSpec{&types.ConstI64, &types.ConstString},
				Ret: []types.TypeSpec{},
			},
			},
			wantInfo: FnInfo{Type: &types.Function{
				Arg: []types.TypeSpec{&types.ConstI64, &types.ConstString},
				Ret: []types.TypeSpec{},
			},
			},
			wantOk: true,
		},
		{
			name: "overwrite func",
			fn:   "TestFunc",
			info: FnInfo{Type: &types.Function{
				Arg: []types.TypeSpec{&types.ConstBool},
				Ret: []types.TypeSpec{&types.ConstI64},
			},
			},
			wantInfo: FnInfo{Type: &types.Function{
				Arg: []types.TypeSpec{&types.ConstBool},
				Ret: []types.TypeSpec{&types.ConstI64},
			},
			},
			wantOk: true,
		},
		{
			name:   "retrieve non-existing",
			fn:     "NonExistentFunc",
			wantOk: false,
		},
	}

	ft := internal.NewStackedSymbolTable[FnInfo]()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantOk {
				ft.Set(tt.fn, tt.info)
			}
			gotInfo, gotOk := ft.Get(tt.fn)

			if gotOk != tt.wantOk && !reflect.DeepEqual(gotInfo, tt.wantInfo) {
				t.Errorf("FunctionTable.Get() = %v, want %v", gotInfo, tt.wantInfo)
			}
		})
	}
}

func TestVariableTable(t *testing.T) {
	tests := []struct {
		name     string
		vr       string
		info     VarInfo
		wantInfo VarInfo
		wantOk   bool
	}{
		{
			name: "add and retrieve",
			vr:   "var1",
			info: VarInfo{
				Type: &types.ConstI64,
			},
			wantInfo: VarInfo{
				Type: &types.ConstI64,
			},
			wantOk: true,
		},
		{
			name: "overwrite in scope",
			vr:   "var1",
			info: VarInfo{
				Type: &types.ConstString,
			},
			wantInfo: VarInfo{
				Type: &types.ConstString,
			},
			wantOk: true,
		},
		{
			name:     "retrive non existing",
			vr:       "nonExistentVar",
			wantInfo: VarInfo{},
			wantOk:   false,
		},
	}

	vt := internal.NewStackedSymbolTable[VarInfo]()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt.Scope()
			if tt.wantOk {
				vt.Set(tt.vr, tt.info)
			}
			gotInfo, gotOk := vt.Get(tt.vr)
			if gotOk != tt.wantOk && !reflect.DeepEqual(gotInfo, tt.wantInfo) {
				t.Errorf("VariableTable.Get() got = %+v, want %+v, gotOk = %v, wantOk = %v", gotInfo, tt.wantInfo, gotOk, tt.wantOk)
			}
		})
	}
}
