package app

import "time"

type EventName string

const (
	EventProjectLoaded      EventName = "project.loaded"
	EventConfigValidated    EventName = "config.validated"
	EventImagePulling       EventName = "image.pulling"
	EventImageReady         EventName = "image.ready"
	EventDiskReady          EventName = "disk.ready"
	EventCloudInitGenerated EventName = "cloud_init.generated"
	EventVMStarting         EventName = "vm.starting"
	EventSSHWaiting         EventName = "ssh.waiting"
	EventSSHReady           EventName = "ssh.ready"
	EventProvisionStarted   EventName = "provision.started"
	EventProvisionFinished  EventName = "provision.finished"
	EventProvisionSkipped   EventName = "provision.skipped"
	EventSnapshotCreated    EventName = "snapshot.created"
	EventRestoreStarted     EventName = "restore.started"
	EventRestoreFinished    EventName = "restore.finished"
	EventInstanceReady      EventName = "instance.ready"
	EventInstanceStopped    EventName = "instance.stopped"
	EventInstanceDestroyed  EventName = "instance.destroyed"
	EventWorkflowCompleted  EventName = "workflow.completed"
	EventWorkflowFailed     EventName = "workflow.failed"
)

type Event struct {
	SchemaVersion string         `json:"schema_version"`
	Type          string         `json:"type"`
	Name          EventName      `json:"name"`
	Command       string         `json:"command"`
	ProjectID     string         `json:"project_id,omitempty"`
	Instance      string         `json:"instance,omitempty"`
	Message       string         `json:"message,omitempty"`
	Time          time.Time      `json:"time"`
	Data          map[string]any `json:"data,omitempty"`
}

type EventSink func(Event)

func emitEvent(sink EventSink, command string, name EventName, options EventOptions) {
	if sink == nil {
		return
	}
	sink(NewEvent(command, name, options))
}

func NewEvent(command string, name EventName, options EventOptions) Event {
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return Event{
		SchemaVersion: "yeast.v1",
		Type:          "event",
		Name:          name,
		Command:       command,
		ProjectID:     options.ProjectID,
		Instance:      options.Instance,
		Message:       options.Message,
		Time:          now,
		Data:          options.Data,
	}
}

type EventOptions struct {
	ProjectID string
	Instance  string
	Message   string
	Now       time.Time
	Data      map[string]any
}
