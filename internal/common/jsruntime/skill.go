package jsruntime

import (
	"context"
	"encoding/json"
	"unicode/utf8"

	"github.com/dop251/goja"
	"github.com/rs/zerolog/log"
)

func toGojaValue(vm *goja.Runtime, data []byte) goja.Value {
	var retValMap map[string]any
	if err := json.Unmarshal(data, &retValMap); err == nil {
		return vm.ToValue(retValMap)
	}
	var retValQuotedString string
	if err := json.Unmarshal(data, &retValQuotedString); err == nil {
		return vm.ToValue(retValQuotedString)
	}
	var retValAny any
	if err := json.Unmarshal(data, &retValAny); err == nil {
		return vm.ToValue(retValAny)
	}
	if utf8.Valid(data) {
		return vm.ToValue(string(data))
	}
	return vm.ToValue(data)
}

func bindSkillService(ctx context.Context, vm *goja.Runtime, skillInvoker SkillInvoker) {
	skillService := vm.NewObject()
	_ = skillService.Set("invokeSkill", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("invalid arguments"))
		}
		skillName, ok := call.Arguments[0].Export().(string)
		if !ok {
			panic(vm.ToValue("invalid skill name"))
		}
		inputArgs, ok := call.Arguments[1].Export().(map[string]any)
		if !ok {
			panic(vm.ToValue("invalid input arguments"))
		}
		ret, err := skillInvoker(skillName, inputArgs)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("error invoking skill")
			panic(vm.ToValue(err.Error()))
		}
		return toGojaValue(vm, ret)
	})
	err := vm.Set("SkillService", skillService)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("error binding skill service")
		panic(err)
	}
}
