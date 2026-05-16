package runtime

import (
	"context"
	"time"
)

type Runtime interface {
	PrepareDisk(ctx context.Context, plan MachinePlan) (DiskPlan, error)
	Start(ctx context.Context, plan MachinePlan) (RuntimeInstance, error)
	Stop(ctx context.Context, instance RuntimeInstance, timeout time.Duration) error
	Inspect(ctx context.Context, instance RuntimeInstance) (ProcessInfo, error)
	Destroy(ctx context.Context, instance RuntimeInstance) error
}
