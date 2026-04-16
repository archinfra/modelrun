package deploy

import (
	"reflect"
	"testing"

	"modelrun/backend/internal/domain"
)

func TestHydrateTasksMergesRuntimeStepLogs(t *testing.T) {
	executor := &Executor{
		active:  map[string]map[string]func(){},
		runtime: map[string]*runtimeStepOutput{},
	}
	executor.appendRuntimeStepLogs("deployment-1", "server-1", "step-1", []string{"line-1", "line-2"})

	tasks := []domain.DeploymentTask{
		{
			ID:           "task-1",
			DeploymentID: "deployment-1",
			ServerID:     "server-1",
			Steps: []domain.DeploymentStep{
				{ID: "step-1", Logs: []string{"persisted"}},
				{ID: "step-2", Logs: []string{"other"}},
			},
		},
	}

	got := executor.HydrateTasks(tasks)
	want := []string{"persisted", "line-1", "line-2"}
	if !reflect.DeepEqual(got[0].Steps[0].Logs, want) {
		t.Fatalf("HydrateTasks() step logs = %#v, want %#v", got[0].Steps[0].Logs, want)
	}
	if !reflect.DeepEqual(tasks[0].Steps[0].Logs, []string{"persisted"}) {
		t.Fatalf("HydrateTasks() should not mutate input tasks, got %#v", tasks[0].Steps[0].Logs)
	}
}

func TestDrainRuntimeStepLogsRemovesBufferedLines(t *testing.T) {
	executor := &Executor{
		active:  map[string]map[string]func(){},
		runtime: map[string]*runtimeStepOutput{},
	}
	executor.appendRuntimeStepLogs("deployment-1", "server-1", "step-1", []string{"line-1"})

	drained := executor.drainRuntimeStepLogs("deployment-1", "server-1", "step-1")
	if !reflect.DeepEqual(drained, []string{"line-1"}) {
		t.Fatalf("drainRuntimeStepLogs() = %#v, want %#v", drained, []string{"line-1"})
	}

	got := executor.HydrateTasks([]domain.DeploymentTask{{
		ID:           "task-1",
		DeploymentID: "deployment-1",
		ServerID:     "server-1",
		Steps:        []domain.DeploymentStep{{ID: "step-1"}},
	}})
	if len(got[0].Steps[0].Logs) != 0 {
		t.Fatalf("expected drained logs to be removed from runtime cache, got %#v", got[0].Steps[0].Logs)
	}
}
