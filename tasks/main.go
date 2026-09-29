package main

import (
	"fmt"
	"log"
	"time"

	"github.com/render-oss/sdk/go/pkg/tasks"
)

func square(ctx tasks.TaskContext, a int) int {
	return a * a
}

func addSquares(ctx tasks.TaskContext, a int, b int) int {
	log.Printf("addSquares: %d, %d", a, b)
	var result1 int
	var result2 int

	log.Printf("Executing square: %d", a)
	err := ctx.ExecuteTask(square, a).Get(&result1)
	if err != nil {
		log.Printf("Error executing square: %d", a)
		panic(err)
	}
	log.Printf("Executing square: %d", b)
	err = ctx.ExecuteTask(square, b).Get(&result2)
	if err != nil {
		log.Printf("Error executing square: %d", b)
		panic(err)
	}
	return result1 + result2
}

// summarizeDeploy takes a struct containing a time.Time, which the SDK decodes
// from either ["{...}"] positional input or a {...} object input.
func summarizeDeploy(ctx tasks.TaskContext, report DeployReport) string {
	return fmt.Sprintf("%s failed %d time(s) as of %s (labels: %v)",
		report.Service, report.Attempts, report.CreatedAt.Format(time.RFC3339), report.Labels)
}

// summarizeDeploy takes a struct containing a time.Time, which the SDK decodes
// from either ["{...}"] positional input or a {...} object input.
func summarizeDeployMap(ctx tasks.TaskContext, report map[string]any) string {
	createdAt, _ := time.Parse(time.RFC3339, report["created_at"].(string))
	return fmt.Sprintf("%s failed %d time(s) as of %s (labels: %v)",
		report["service"], report["attempts"], createdAt.Format(time.RFC3339), report["labels"])
}

func processWorkflowPayload(ctx tasks.TaskContext, payload WorkflowPayload) string {
	return fmt.Sprintf("Processing workflow payload for channel: %s", payload.ChannelID)
}

// DeployReport is decoded straight into the task parameter, so its JSON tags
// double as the named parameters when the task is triggered with object input.
type DeployReport struct {
	Service   string            `json:"service"`
	CreatedAt time.Time         `json:"created_at"`
	Attempts  int               `json:"attempts"`
	Labels    map[string]string `json:"labels"`
}

type WorkflowPayload struct {
	ChannelID          string
	RenderOwnerID      string
	BotID              string
	EncAccessToken     string
	AccessTokenKeyID   string
	EncRefreshToken    string
	RefreshTokenKeyID  string
	Intent             string
	StandaloneQuery    string
	RequiresLiveData   bool
	IsTroubleshooting  bool
	IsResourceSpecific bool
	ResourceType       string
}

func main() {
	tasks.MustRegister(square)
	tasks.MustRegister(addSquares)

	tasks.MustRegister(summarizeDeploy)
	tasks.MustRegister(summarizeDeployMap)

	tasks.MustRegister(processWorkflowPayload)
	tasks.Start()
}
