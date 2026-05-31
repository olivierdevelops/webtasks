// Package listtasks declares the ListTasks contract.
package listtasks

import "webtasks/internal/domain"

type ListTasks interface {
	Execute() []domain.TaskDef
}
