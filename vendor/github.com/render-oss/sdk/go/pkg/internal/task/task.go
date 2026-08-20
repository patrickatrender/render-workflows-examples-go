package task

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// Retry contains retry configuration for a task
type Retry struct {
	MaxRetries   int           `json:"max_retries"`
	WaitDuration time.Duration `json:"wait_duration"`
	Factor       float32       `json:"factor"`
}

// Options contains configuration options for a task
type Options struct {
	Retry *Retry `json:"retry,omitempty"`
}

// TaskInfo contains a task function and its options
type TaskInfo struct {
	Task    Task     `json:"-"`
	Options *Options `json:"options,omitempty"`
}

type Tasks struct {
	Tasks map[string]*TaskInfo
}

type Task interface{}

var jsonNull = []byte("null")

// Input holds the raw JSON arguments of a task invocation.
//
// Task input arrives on the wire as either a JSON array (positional arguments)
// or a JSON object (named parameters). At most one of Args and Object is set.
type Input struct {
	// Args holds one raw JSON value per positional argument.
	Args []json.RawMessage

	// Object holds the entire payload when the input was a JSON object.
	Object json.RawMessage
}

// IsObject reports whether the input was a JSON object rather than an array.
func (i Input) IsObject() bool {
	return i.Object != nil
}

// ParseInput decodes a raw task input payload. The payload is either a JSON
// array of positional arguments or a JSON object of named parameters. An empty
// or null payload is treated as "no arguments".
func ParseInput(raw []byte) (Input, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, jsonNull) {
		return Input{}, nil
	}

	switch trimmed[0] {
	case '{':
		if !json.Valid(trimmed) {
			return Input{}, fmt.Errorf("failed to unmarshal input: input is not valid JSON")
		}
		object := make(json.RawMessage, len(trimmed))
		copy(object, trimmed)
		return Input{Object: object}, nil
	case '[':
		var args []json.RawMessage
		if err := json.Unmarshal(trimmed, &args); err != nil {
			return Input{}, fmt.Errorf("failed to unmarshal input: %w", err)
		}
		return Input{Args: args}, nil
	default:
		return Input{}, fmt.Errorf("task input must be a JSON array or a JSON object")
	}
}

// InputFromValues builds an Input from Go values, as if they had been sent as a
// JSON array of positional arguments.
func InputFromValues(values ...interface{}) (Input, error) {
	if values == nil {
		values = []interface{}{}
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return Input{}, fmt.Errorf("failed to marshal input values: %w", err)
	}
	return ParseInput(raw)
}

// InputFromObject builds an Input from a Go value that marshals to a JSON
// object, as if it had been sent as named parameters.
func InputFromObject(value interface{}) (Input, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return Input{}, fmt.Errorf("failed to marshal input object: %w", err)
	}
	if trimmed := bytes.TrimSpace(raw); len(trimmed) == 0 || trimmed[0] != '{' {
		return Input{}, fmt.Errorf("input object must marshal to a JSON object, got %s", raw)
	}
	return ParseInput(raw)
}

func VerifySignature(t Task) error {
	// Step 1: Check that t is a function
	tType := reflect.TypeOf(t)
	if tType.Kind() != reflect.Func {
		return fmt.Errorf("task must be a function")
	}

	// Step 2: Check the function has at least one input
	if tType.NumIn() == 0 {
		return fmt.Errorf("task function must have at least one input parameter")
	}

	// Step 3: Check that the first argument implements TaskContext
	firstParam := tType.In(0)
	taskCtxType := reflect.TypeOf((*TaskContext)(nil)).Elem()

	if !firstParam.Implements(taskCtxType) {
		return fmt.Errorf("first argument must implement TaskContext interface")
	}

	return nil
}

// GetFunctionName returns the short name of the given Task (function).
func GetFunctionName(t Task) (string, error) {
	v := reflect.ValueOf(t)
	if v.Kind() != reflect.Func {
		return "", fmt.Errorf("input is not a function")
	}

	// Get the function pointer
	fn := runtime.FuncForPC(v.Pointer())
	if fn == nil {
		return "", fmt.Errorf("unable to get function information")
	}

	// Get the full name and extract the short name
	fullName := fn.Name() // e.g., "github.com/foo/bar.MyFunction"
	parts := strings.Split(fullName, ".")
	shortName := parts[len(parts)-1] // e.g., "MyFunction"

	// Remove any closure suffix if present, like "-fm" or ".func1"
	shortName = strings.SplitN(shortName, "-", 2)[0]
	shortName = strings.SplitN(shortName, ".", 2)[0]

	return shortName, nil
}

// isJSONNull reports whether raw is the JSON null literal.
func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), jsonNull)
}

// canBeNil reports whether a value of the given type can represent nil.
func canBeNil(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice:
		return true
	default:
		return false
	}
}

// decodeJSONValue unmarshals a raw JSON value into a newly allocated value of
// the given type. desc describes the value being decoded, for error messages.
//
// A top-level null is rejected for types that cannot represent nil: silently
// producing a zero value would turn a caller bug into a mysterious 0 or "".
// Nulls nested inside a struct follow ordinary encoding/json semantics.
func decodeJSONValue(raw json.RawMessage, t reflect.Type, desc string) (reflect.Value, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return reflect.Value{}, fmt.Errorf("%s is empty, expected a JSON value of type %s", desc, t)
	}

	if isJSONNull(raw) && !canBeNil(t) {
		return reflect.Value{}, fmt.Errorf("%s is null, but type %s cannot represent null", desc, t)
	}

	value := reflect.New(t)
	if err := json.Unmarshal(raw, value.Interface()); err != nil {
		return reflect.Value{}, fmt.Errorf("%s: cannot decode into %s: %w", desc, t, err)
	}

	return value.Elem(), nil
}

// CallTask invokes the given Task (a function) with the provided raw input,
// unmarshalling each argument directly into the declared parameter type.
//
// name is the registered task name, used in error messages. If it is empty, the
// name is derived from the function itself.
func CallTask(t Task, name string, tctx TaskContext, input Input) ([]interface{}, error) {
	v := reflect.ValueOf(t)
	if v.Kind() != reflect.Func {
		return nil, fmt.Errorf("task is not a function")
	}

	if name == "" {
		fnName, err := GetFunctionName(t)
		if err != nil {
			return nil, err
		}
		name = fnName
	}

	tType := v.Type()
	if tType.NumIn() == 0 {
		return nil, fmt.Errorf("task %s must accept a TaskContext as its first argument", name)
	}

	// The first argument is always the TaskContext, so the remaining inputs are
	// the user-defined parameters.
	numParams := tType.NumIn() - 1

	args := make([]reflect.Value, tType.NumIn())

	ctxType := tType.In(0)
	if tctx == nil {
		args[0] = reflect.Zero(ctxType)
	} else {
		ctxValue := reflect.ValueOf(tctx)
		if !ctxValue.Type().AssignableTo(ctxType) {
			return nil, fmt.Errorf("task %s: task context of type %s is not assignable to %s", name, ctxValue.Type(), ctxType)
		}
		args[0] = ctxValue
	}

	if input.IsObject() {
		// Go reflection cannot recover parameter names, so object input maps
		// onto a single parameter whose JSON tags act as the named parameters.
		if numParams != 1 {
			return nil, fmt.Errorf("task %s received object input but declares %d parameter(s); object input requires exactly one parameter", name, numParams)
		}

		value, err := decodeJSONValue(input.Object, tType.In(1), "input object")
		if err != nil {
			return nil, fmt.Errorf("task %s: %w", name, err)
		}
		args[1] = value
	} else {
		if len(input.Args) != numParams {
			return nil, fmt.Errorf("task %s expected %d argument(s), got %d", name, numParams, len(input.Args))
		}

		for i, raw := range input.Args {
			value, err := decodeJSONValue(raw, tType.In(i+1), fmt.Sprintf("argument %d", i))
			if err != nil {
				return nil, fmt.Errorf("task %s: %w", name, err)
			}
			args[i+1] = value
		}
	}

	// Call the function
	out := v.Call(args)

	// Convert results to []interface{}
	results := make([]interface{}, len(out))
	for i, val := range out {
		results[i] = val.Interface()
	}

	// Check if the last return value is an error type
	if len(out) > 0 {
		lastValue := out[len(out)-1]
		if lastValue.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			// The last return value implements the error interface
			if !lastValue.IsNil() {
				// Return the error and the results (excluding the error)
				errorResults := results[:len(results)-1]
				return errorResults, lastValue.Interface().(error)
			}
			// Error is nil, so exclude it from results
			results = results[:len(results)-1]
		}
	}

	return results, nil
}

// TaskResult holds the raw JSON output of a task run, or the error it failed
// with. Use Get to decode the output into typed values.
type TaskResult struct {
	Result []json.RawMessage
	Error  error
}

// Get decodes the task output into the given pointers, unmarshalling each
// returned value directly into the type it points at.
func (t *TaskResult) Get(output ...interface{}) error {
	if t.Error != nil {
		return t.Error
	}

	if output == nil {
		return fmt.Errorf("output cannot be nil")
	}

	if len(output) != len(t.Result) {
		return fmt.Errorf("expected %d output arguments, got %d", len(t.Result), len(output))
	}

	for i, arg := range output {
		if arg == nil {
			return fmt.Errorf("output argument %d is nil", i)
		}

		argVal := reflect.ValueOf(arg)
		if argVal.Kind() != reflect.Ptr || argVal.IsNil() {
			return fmt.Errorf("output argument %d must be a non-nil pointer", i)
		}

		targetElem := argVal.Elem()
		value, err := decodeJSONValue(t.Result[i], targetElem.Type(), fmt.Sprintf("result value at index %d", i))
		if err != nil {
			return err
		}

		targetElem.Set(value)
	}

	return nil
}

type TaskContext interface {
	ExecuteTask(task Task, input ...interface{}) *TaskResult
}

func NewTasks() *Tasks {
	return &Tasks{
		Tasks: make(map[string]*TaskInfo),
	}
}

func (t *Tasks) RegisterTask(task Task) error {
	return t.RegisterTaskWithOptions(task, nil)
}

func (t *Tasks) RegisterTaskWithOptions(task Task, options *Options) error {
	err := VerifySignature(task)
	if err != nil {
		return err
	}
	name, err := GetFunctionName(task)
	if err != nil {
		return err
	}
	t.Tasks[name] = &TaskInfo{
		Task:    task,
		Options: options,
	}
	return nil
}

func (t *Tasks) GetTaskByName(name string) (Task, error) {
	taskInfo, ok := t.Tasks[name]
	if !ok {
		return nil, fmt.Errorf("task %s not found", name)
	}
	return taskInfo.Task, nil
}

func (t *Tasks) GetTaskInfoByName(name string) (*TaskInfo, error) {
	taskInfo, ok := t.Tasks[name]
	if !ok {
		return nil, fmt.Errorf("task %s not found", name)
	}
	return taskInfo, nil
}

func (t *Tasks) GetTaskNames() []string {
	names := make([]string, 0, len(t.Tasks))
	for name := range t.Tasks {
		names = append(names, name)
	}
	return names
}

func (t *Tasks) ExecuteTaskByName(name string, tctx TaskContext, input Input) ([]interface{}, error) {
	task, err := t.GetTaskByName(name)
	if err != nil {
		return nil, err
	}

	return CallTask(task, name, tctx, input)
}
