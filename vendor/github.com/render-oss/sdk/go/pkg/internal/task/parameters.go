package task

import (
	"fmt"
	"reflect"
)

// Parameter describes a single parameter of a task function, so that callers
// triggering the task know what input it expects.
type Parameter struct {
	// Name is the positional name of the parameter. Go does not keep parameter
	// names in the compiled binary, so they cannot be recovered by reflection.
	Name string

	// Type is the Go type of the parameter.
	Type string
}

// Parameters describes the parameters of a task function, excluding the leading
// TaskContext.
//
// The parameters are positional, because that is the signature the task is
// called with. A task with one struct parameter can also be triggered with
// object input, but that object is one argument, not one argument per field.
func Parameters(t Task) []Parameter {
	tType := reflect.TypeOf(t)
	if tType == nil || tType.Kind() != reflect.Func || tType.NumIn() == 0 {
		return nil
	}

	// The first argument is always the TaskContext.
	numParams := tType.NumIn() - 1

	params := make([]Parameter, 0, numParams)
	for i := 0; i < numParams; i++ {
		params = append(params, Parameter{
			Name: fmt.Sprintf("arg%d", i),
			Type: tType.In(i + 1).String(),
		})
	}
	return params
}
