package usecases

import (
	"webtasks/internal/domain"
	"webtasks/internal/features"
	"webtasks/internal/usecases/listtasks"
)

type listImpl struct{ reg features.TaskRegistry }

func NewListTasks(reg features.TaskRegistry) listtasks.ListTasks { return &listImpl{reg} }
func (l *listImpl) Execute() []domain.TaskDef                    { return l.reg.List() }
