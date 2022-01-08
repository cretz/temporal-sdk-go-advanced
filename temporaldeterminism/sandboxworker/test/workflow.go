package test

import (
	"context"
	"fmt"

	_ "github.com/cretz/temporal-sdk-go-advanced/temporaldeterminism/sandboxworker"
	"go.temporal.io/sdk/workflow"
)

func HelloWorkflow(ctx workflow.Context, name string) (string, error) {
	var str string
	err := workflow.ExecuteActivity(ctx, HelloActivity, "World").Get(ctx, &str)
	return str, err
}

func HelloActivity(ctx context.Context, name string) (string, error) {
	return fmt.Sprintf("Hello, %v!", name), nil
}
