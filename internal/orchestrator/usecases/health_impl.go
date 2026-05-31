package usecases

import (
	"webtasks/internal/features"
	"webtasks/internal/usecases/health"
)

type healthImpl struct {
	reg   features.TaskRegistry
	lease features.WindowLease
}

func NewHealth(reg features.TaskRegistry, lease features.WindowLease) health.Health {
	return &healthImpl{reg, lease}
}

func (h *healthImpl) Execute() map[string]any {
	return map[string]any{
		"ok":        true,
		"taskCount": len(h.reg.List()),
		"pools":     h.lease.Status(),
	}
}
