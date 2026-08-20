package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/render-oss/sdk/go/pkg/render"
)

// DeployReport is decoded straight into the task parameter, so its JSON tags
// double as the named parameters when the task is triggered with object input.
type DeployReport struct {
	Service   string            `json:"service"`
	CreatedAt time.Time         `json:"created_at"`
	Attempts  int               `json:"attempts"`
	Labels    map[string]string `json:"labels"`
}

func main() {
	// Example usage of the render client
	client, err := render.NewClient()
	if err != nil {
		log.Fatalf("Failed to create render client: %v", err)
	}

	otherTaskSlug := render.TaskSlug("render-workflows-examples-go/summarizeDeploy")
	var otherInput render.TaskData
	deployReport := DeployReport{
		Attempts:  2,
		CreatedAt: time.Date(2026, 8, 19, 12, 30, 0, 0, time.UTC),
		Labels: map[string]string{
			"env": "prod",
		},
		Service: "srv-123",
	}
	_ = otherInput.FromTaskData0([]interface{}{deployReport})

	otherTaskRun, err := client.Workflows.RunTask(otherTaskSlug, otherInput)
	if err != nil {
		log.Fatalf("Failed to run other task: %v", err)
	}

	fmt.Printf("Other task run created with ID: %s, Status: %s\n", otherTaskRun.Id, otherTaskRun.Status)

	otherResult, err := otherTaskRun.Get(context.Background())
	if err != nil {
		log.Fatalf("Failed to get other task run details: %v", err)
	}

	fmt.Printf("Other task run details: ID=%s, Status=%s, Results=%v\n",
		otherResult.Id, otherResult.Status, otherResult.Results)

	// Example: Run a task
	taskSlug := render.TaskSlug("render-workflows-examples-go/square")
	var input render.TaskData
	_ = input.FromTaskData0([]interface{}{4})

	taskRun, err := client.Workflows.RunTask(taskSlug, input)
	if err != nil {
		log.Fatalf("Failed to run task: %v", err)
	}

	fmt.Printf("Task run created with ID: %s, Status: %s\n", taskRun.Id, taskRun.Status)

	result, err := taskRun.Get(context.Background())
	if err != nil {
		log.Fatalf("Failed to get task run details: %v", err)
	}

	fmt.Printf("Task run details: ID=%s, Status=%s, Results=%v\n",
		result.Id, result.Status, result.Results)
	// Example: Get task run details
	details, err := client.Workflows.GetTaskRun(taskRun.Id)
	if err != nil {
		log.Fatalf("Failed to get task run details: %v", err)
	}

	fmt.Printf("Task run details: ID=%s, Status=%s, Results=%v\n",
		details.Id, details.Status, details.Results)

	// Example: List task runs
	params := &render.ListTaskRunsParams{
		Limit: func() *int { i := 10; return &i }(),
	}

	taskRuns, err := client.Workflows.ListTaskRuns(params)
	if err != nil {
		log.Fatalf("Failed to list task runs: %v", err)
	}

	fmt.Printf("Found %d task runs\n", len(taskRuns))
}
