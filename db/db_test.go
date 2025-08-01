package db

import (
	"context"
	"database/sql"

	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) Query(ctx context.Context, query string, args ...any) error {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Error(0)
}

func (m *MockDatabase) QueryRows(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(*sql.Rows), callArgs.Error(1)
}

func (m *MockDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}
