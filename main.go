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

// DeployReport is decoded straight into the task parameter, so its JSON tags
// double as the named parameters when the task is triggered with object input.
type DeployReport struct {
	Service   string            `json:"service"`
	CreatedAt time.Time         `json:"created_at"`
	Attempts  int               `json:"attempts"`
	Labels    map[string]string `json:"labels"`
}

func main() {
	tasks.MustRegister(square)
	tasks.MustRegister(addSquares)

	tasks.MustRegister(summarizeDeploy)
	tasks.Start()
}
