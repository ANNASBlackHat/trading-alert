package mcp

import (
	"context"
	"errors"
)

// startInMemoryMongo returns an in-memory Mongo server for tests.
//
// It is wired here (rather than importing mongomem directly) so the package
// builds without that optional dependency: when the dependency is present,
// build with -tags mongomem to enable TestInMemoryToolHandlers; otherwise
// the test skips gracefully.
var startInMemoryMongo func(ctx context.Context) (inMemoryMongo, error)

// inMemoryMongo is the narrow interface TestInMemoryToolHandlers needs.
type inMemoryMongo interface {
	Stop()
	URI() string
}

// SetStartInMemoryMongo lets a build-tagged file inject the real factory.
func SetStartInMemoryMongo(f func(ctx context.Context) (inMemoryMongo, error)) {
	startInMemoryMongo = f
}

// errNoInMemoryMongo is returned when the optional dependency is absent.
var errNoInMemoryMongo = errors.New("mongomem tag not set; in-memory mongo unavailable")

// defaultStartInMemoryMongo is the no-op used until a build tag overrides it.
func defaultStartInMemoryMongo(ctx context.Context) (inMemoryMongo, error) {
	return nil, errNoInMemoryMongo
}

func init() {
	startInMemoryMongo = defaultStartInMemoryMongo
}
