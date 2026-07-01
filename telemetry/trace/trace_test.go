package trace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeProtocol(t *testing.T) {
	assert.Equal(t, "grpc", normalizeProtocol(""))
	assert.Equal(t, "grpc", normalizeProtocol("GRPC"))
	assert.Equal(t, "http", normalizeProtocol("http"))
}

func TestNormalizeGRPCEndpoint(t *testing.T) {
	assert.Equal(t, "localhost:4317", normalizeGRPCEndpoint(""))
	assert.Equal(t, "collector:4317", normalizeGRPCEndpoint("grpc://collector:4317"))
	assert.Equal(t, "collector:4317", normalizeGRPCEndpoint("http://collector:4317"))
}

func TestNormalizeHTTPEndpoint(t *testing.T) {
	assert.Equal(t, "localhost:4318", normalizeHTTPEndpoint(""))
	assert.Equal(t, "collector:4318", normalizeHTTPEndpoint("http://collector:4318"))
}
