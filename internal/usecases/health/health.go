// Package health declares the Health contract.
package health

type Health interface {
	Execute() map[string]any
}
