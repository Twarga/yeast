package app

import (
	"context"
	"time"

	"github.com/Twarga/yeast/internal/audit"
)

// actorKey carries the caller identity ("cli", "api", "mcp:<client>")
// through ctx so Service methods can audit with the right actor.
type actorKey struct{}

// WithActor returns a ctx stamped with the caller identity.
func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

// ActorFrom extracts the stamped actor; defaults to "cli".
func ActorFrom(ctx context.Context) string {
	if v, ok := ctx.Value(actorKey{}).(string); ok && v != "" {
		return v
	}
	return "cli"
}

// auditCall appends one record for a completed mutating operation.
// Invoked via defer inside Service methods after project root resolves;
// reads the method's named err return at fire time.
func (s *Service) auditCall(ctx context.Context, projectRoot, action string, args map[string]any, started time.Time, opErr error, snapshotIDs ...string) {
	e := audit.Entry{
		Actor:      ActorFrom(ctx),
		Action:     action,
		Args:       audit.Redact(args),
		DurationMs: time.Since(started).Milliseconds(),
	}
	if len(snapshotIDs) > 0 {
		e.SnapshotIDs = snapshotIDs
	}
	if opErr != nil {
		e.Error = opErr.Error()
	}
	audit.Append(projectRoot, e)
}

// AuditResult carries the audit log for `yeast audit`.
type AuditResult struct {
	Entries []audit.Entry `json:"entries"`
}
