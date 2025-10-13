package rest

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// nameGenerator provides unique name generation for resources
// This is a simple implementation - can be replaced with a more sophisticated one
var nameGenerator = &simpleNameGenerator{}

type simpleNameGenerator struct {
	counter int
}

func (g *simpleNameGenerator) GenerateName(base string) string {
	g.counter++
	return fmt.Sprintf("%s%d", base, g.counter)
}

// validateFieldErrors converts field.ErrorList to regular error
func validateFieldErrors(errs field.ErrorList) error {
	if len(errs) == 0 {
		return nil
	}
	return errs.ToAggregate()
}
