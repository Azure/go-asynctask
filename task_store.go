package taskStore

import (
	"context"
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
	RegistTask(ctx context.Context, key string, task func(context.Context) (interface{}, error))
	GetTask(key string) *TaskResult
	WaitTask(key string) *TaskResult
}

func New() Interface {
	return &storeImpl{
		TaskMap: make(map[string]*TaskResult),
	}
}

type storeImpl struct {
	TaskMap map[string]*TaskResult
}

func (store *storeImpl) RegistTask(ctx context.Context, key string, task func(context.Context) (interface{}, error)) {
	record := &TaskResult{
		Context: ctx,
		Status:  TaskStatusRunning,
		Result:  nil,
	}
	store.TaskMap[key] = record
	go func() {
		result, err := task(ctx)
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

func (store *storeImpl) WaitTask(key string) *TaskResult {
	// TODO: Implement
	if value, exists := store.TaskMap[key]; exists {
		//delete(store.TaskMap, key)
		return value
	}
	return nil
}
