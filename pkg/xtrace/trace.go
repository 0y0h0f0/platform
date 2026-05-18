package xtrace

import "context"

func Init(context.Context, string) (func(context.Context) error, error) {
	return func(context.Context) error { return nil }, nil
}
