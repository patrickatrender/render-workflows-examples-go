package render

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/render-oss/sdk/go/pkg/render/internal/client"
	workflows "github.com/render-oss/sdk/go/pkg/render/internal/client/workflows"
)

// WorkflowsService handles workflow-related API operations
type WorkflowsService struct {
	client *Client
}

// PositionalInput builds task input from positional arguments. Each argument is
// decoded into the matching parameter of the task function.
//
//	input, err := render.PositionalInput(4)
//	run, err := client.Workflows.RunTask("my-task", input)
func PositionalInput(args ...any) (TaskData, error) {
	var data TaskData
	if args == nil {
		args = []any{}
	}
	if err := data.FromTaskData0(args); err != nil {
		return data, fmt.Errorf("failed to encode task input: %w", err)
	}
	return data, nil
}

// ObjectInput builds object-shaped task input from a struct or map. The task
// must declare exactly one parameter, and the object is decoded into it, so the
// parameter's JSON tags act as the named parameters.
//
//	input, err := render.ObjectInput(MyArgs{Service: "srv-123"})
//	run, err := client.Workflows.RunTask("my-task", input)
func ObjectInput(value any) (TaskData, error) {
	var data TaskData

	encoded, err := json.Marshal(value)
	if err != nil {
		return data, fmt.Errorf("failed to encode task input: %w", err)
	}
	if trimmed := bytes.TrimSpace(encoded); len(trimmed) == 0 || trimmed[0] != '{' {
		return data, fmt.Errorf("task input must marshal to a JSON object, got %s", encoded)
	}

	if err := data.UnmarshalJSON(encoded); err != nil {
		return data, fmt.Errorf("failed to encode task input: %w", err)
	}
	return data, nil
}

// RunTask executes a task using the workflows API
// POST /task-runs
func (w *WorkflowsService) RunTask(taskSlug TaskSlug, input TaskData) (*TaskRunWithGet, error) {
	runTask := workflows.RunTask{
		Task:  workflows.TaskSlug(taskSlug),
		Input: workflows.TaskData(input),
	}

	resp, err := w.client.internal.CreateTaskWithResponse(context.Background(), runTask)
	if err != nil {
		return nil, fmt.Errorf("failed to make run task request: %w", err)
	}

	if resp.StatusCode() != 202 {
		return nil, fmt.Errorf("run task failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if resp.JSON202 == nil {
		return nil, fmt.Errorf("unexpected response format")
	}

	taskRun := (*TaskRun)(resp.JSON202)
	return &TaskRunWithGet{
		TaskRun: taskRun,
		client:  w,
	}, nil
}

// GetTaskRun gets details about a specific task run
// GET /tasks-runs/{taskRunId}
func (w *WorkflowsService) GetTaskRun(taskRunID string) (*TaskRunDetails, error) {
	resp, err := w.client.internal.GetTaskRunWithResponse(context.Background(), taskRunID)
	if err != nil {
		return nil, fmt.Errorf("failed to make get task run request: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("get task run failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response format")
	}

	return (*TaskRunDetails)(resp.JSON200), nil
}

// CancelTaskRun cancels a running task
// POST /task-runs/{taskRunId}/cancel
func (w *WorkflowsService) CancelTaskRun(taskRunID string) error {
	resp, err := w.client.internal.CancelTaskRunWithResponse(context.Background(), taskRunID)
	if err != nil {
		return fmt.Errorf("failed to make cancel task run request: %w", err)
	}

	if resp.StatusCode() != 204 {
		return fmt.Errorf("cancel task run failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	return nil
}

// ListTaskRuns lists task runs with optional filtering
// GET /task-runs
func (w *WorkflowsService) ListTaskRuns(params *ListTaskRunsParams) ([]TaskRun, error) {
	resp, err := w.client.internal.ListTaskRunsWithResponse(
		context.Background(),
		(*client.ListTaskRunsParams)(params),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to make list task runs request: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("list task runs failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response format")
	}

	result := make([]TaskRun, len(*resp.JSON200))
	for i, item := range *resp.JSON200 {
		result[i] = TaskRun(item.TaskRun)
	}

	return result, nil
}
