package extensions

import (
	"github.com/anssihalmeaho/funl/funl"
	"github.com/anssihalmeaho/funl/std"
	"github.com/anssihalmeaho/fuvaluez/fuvaluez"
)

func init() {
	funl.AddExtensionInitializer(initValuez)
}

func convGetter(inGetter func(string) fuvaluez.FZProc) func(string) std.StdFuncType {
	return func(name string) std.StdFuncType {
		return std.StdFuncType(inGetter(name))
	}
}

func initValuez(interpreter *funl.Interpreter) (err error) {
	stdModuleName := "valuez"
	topFrame := funl.NewTopFrameWithInterpreter(interpreter)
	stdFuncs := []std.StdFuncInfo{
		{
			Name:   "open",
			Getter: convGetter(fuvaluez.GetVZOpen),
		},
		{
			Name:   "new-col",
			Getter: convGetter(fuvaluez.GetVZNewCol),
		},
		{
			Name:   "get-col",
			Getter: convGetter(fuvaluez.GetVZGetCol),
		},
		{
			Name:   "get-col-names",
			Getter: convGetter(fuvaluez.GetVZGetColNames),
		},
		{
			Name:   "put-value",
			Getter: convGetter(fuvaluez.GetVZPutValue),
		},
		{
			Name:   "get-values",
			Getter: convGetter(fuvaluez.GetVZGetValues),
		},
		{
			Name:   "take-values",
			Getter: convGetter(fuvaluez.GetVZTakeValues),
		},
		{
			Name:   "update",
			Getter: convGetter(fuvaluez.GetVZUpdate),
		},
		{
			Name:   "trans",
			Getter: convGetter(fuvaluez.GetVZTrans),
		},
		{
			Name:   "view",
			Getter: convGetter(fuvaluez.GetVZView),
		},
		{
			Name:   "del-col",
			Getter: convGetter(fuvaluez.GetVZDelCol),
		},
		{
			Name:   "close",
			Getter: convGetter(fuvaluez.GetVZClose),
		},
	}
	err = std.SetSTDFunctions(topFrame, stdModuleName, stdFuncs, interpreter)
	return
}
