package shell

import (
	"context"
)

type Executor interface {
	Execute(ctx context.Context, name string, args []string, io IOBindings) (int, error)
}

type Parser interface {
	Parse(line string) ([]string, error)
}
