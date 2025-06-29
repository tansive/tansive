package jsruntime

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/rs/zerolog/log"
)

func bindConsole(ctx context.Context, vm *goja.Runtime) {
	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		args := make([]any, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		msg := fmt.Sprintf("%v", args)
		log.Ctx(ctx).Info().Msg(msg)
		return goja.Undefined()
	})

	// console.error(...)
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		args := make([]any, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		msg := fmt.Sprintf("%v", args)
		log.Ctx(ctx).Error().Msg(msg)
		return goja.Undefined()
	})
	_ = vm.Set("console", console)
}
