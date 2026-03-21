package grpc_test

import (
	"testing"

	pkggrpc "github.com/ssoeasy-dev/pkg/grpc"
	"github.com/stretchr/testify/assert"
)

func TestNewServer_ReturnsNonNil(t *testing.T) {
	srv := pkggrpc.NewServer("127.0.0.1:0", testLogger(), nil)
	assert.NotNil(t, srv)
}

func TestServer_GetGRPCServer_ReturnsNonNil(t *testing.T) {
	srv := pkggrpc.NewServer("127.0.0.1:0", testLogger(), nil)
	assert.NotNil(t, srv.GetGRPCServer())
}

func TestServer_RegisterReflection_DoesNotPanic(t *testing.T) {
	srv := pkggrpc.NewServer("127.0.0.1:0", testLogger(), nil)
	assert.NotPanics(t, srv.RegisterReflection)
}
