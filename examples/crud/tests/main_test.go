//go:build integration

/*
Package tests holds the controller (HTTP) tests for the users CRUD example.

They live outside the packages they exercise, so they see only what a real
client sees: requests in, envelopes out.

This file is built only under -tags=integration, like every test here, because
each request runs through to Postgres.
*/
package tests

import (
	"testing"

	"github.com/Binh-2060/go-application-template/internal/testsupport"
)

func TestMain(m *testing.M) { testsupport.Main(m) }

// A unique name prefix for this test; its users rows are deleted afterwards.
func marker(t *testing.T) string { return testsupport.Marker(t, "users", "name") }
