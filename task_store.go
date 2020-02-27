package taskStore

import (
	"context"
	"fmt"
)

type TaskStatus string

const TaskStatusRunning TaskStatus = "Running"
const TaskStatusCompleted TaskStatus = "Completed"
const TaskStatusFailed TaskStatus = "Failed"

type TaskResult struct {
	context.Context
	Status TaskStatus
	Result interface{}
	Error  error
}

type Interface interface {
	RegistTask(key string, task func(context.Context) (interface{}, error))
	GetTask(key string) *TaskResult
}

func New() Interface {
	return &storeImpl{
		TaskMap: make(map[string]*TaskResult),
	}
}

type storeImpl struct {
	TaskMap map[string]*TaskResult
}

func (store *storeImpl) RegistTask(key string, task func(context.Context) (interface{}, error)) {
	ctx := context.Background()
	record := &TaskResult{
		Context: ctx,
		Status:  TaskStatusRunning,
		Result:  nil,
	}
	store.TaskMap[key] = record
	go func() {
		fmt.Printf("⬇️⬇️⬇️ starting task %s\n", key)
		result, err := task(ctx)
		fmt.Printf("⬆️⬆️⬆️ finishing task %s\n", key)
		if err != nil {
			record.Status = TaskStatusFailed
			record.Error = err
		} else {
			record.Status = TaskStatusCompleted
			record.Result = result
		}
	}()
}

func (store *storeImpl) GetTask(key string) *TaskResult {
	if value, exists := store.TaskMap[key]; exists {
		//delete(store.TaskMap, key)
		return value
	}
	return nil
}
