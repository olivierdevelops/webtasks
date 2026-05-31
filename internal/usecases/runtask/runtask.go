// Package runtask declares the RunRegisteredTask contract and the dep shapes
// the use case needs from the layers below. The wired-up implementation lives
// in orchestrator/usecases (per VHCO).
package runtask

import (
	"context"

	"webtasks/internal/domain"
	"webtasks/internal/features"
)

// RunRegisteredTask is the use-case contract.
type RunRegisteredTask interface {
	Execute(ctx context.Context, taskName string, vals domain.InputValues, events features.EventPublisher) (domain.Output, error)
}
