package jsruntime

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/tansive/tansive/internal/common/apperrors"
)

type JSFunction struct {
	code     string
	function goja.Callable
}

// Options for controlling execution
type Options struct {
	Timeout      time.Duration // max execution time
	SkillInvoker SkillInvoker
}

type SkillInvoker func(skillName string, inputArgs map[string]any) ([]byte, apperrors.Error)

// New creates a JSFunction from a JS function source string.
func New(ctx context.Context, jsCode string) (*JSFunction, apperrors.Error) {
	vm := goja.New()
	bindConsole(ctx, vm)
	wrapped := fmt.Sprintf("(%s)", jsCode)
	v, err := vm.RunString(wrapped)
	if err != nil {
		return nil, ErrInvalidJSFunction.Err(err)
	}

	fn, ok := goja.AssertFunction(v)
	if !ok {
		return nil, ErrInvalidJSFunction.Msg("script is not a function")
	}

	return &JSFunction{
		code:     jsCode,
		function: fn,
	}, nil
}

// Run executes the function with two JSON arguments, respecting timeout and returning JSON output.
func (j *JSFunction) Run(ctx context.Context, sessionArgs, inputArgs map[string]any, opts Options) (map[string]any, apperrors.Error) {
	// New VM per run to isolate memory
	vm := goja.New()
	bindConsole(ctx, vm)
	if opts.SkillInvoker != nil {
		bindSkillService(ctx, vm, opts.SkillInvoker)
	}

	// Recompile function
	wrapped := fmt.Sprintf("(%s)", j.code)
	v, err := vm.RunString(wrapped)
	if err != nil {
		return nil, ErrJSExecutionError.Err(err)
	}
	fn, _ := goja.AssertFunction(v)

	// Use context with timeout
	if opts.Timeout == 0 {
		opts.Timeout = 500 * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	done := make(chan struct{})
	var result goja.Value
	var callErr error

	go func() {
		defer func() {
			if r := recover(); r != nil {
				callErr = fmt.Errorf("panic: %v", r)
			}
			close(done)
		}()

		val1 := vm.ToValue(sessionArgs)
		val2 := vm.ToValue(inputArgs)
		result, callErr = fn(goja.Undefined(), val1, val2)
	}()

	select {
	case <-ctx.Done():
		return nil, ErrJSRuntimeTimeout
	case <-done:
		if callErr != nil {
			if jsErr, ok := callErr.(*goja.Exception); ok {
				return nil, ErrJSRuntimeError.Msg(jsErr.Value().String())
			}
			return nil, ErrJSExecutionError.Err(callErr)
		}
	}

	exported := result.Export()
	resMap, ok := exported.(map[string]any)
	if !ok {
		msg := fmt.Sprintf("expected function to return object, got %T", exported)
		return nil, ErrJSExecutionError.Msg(msg)
	}

	return resMap, nil
}
