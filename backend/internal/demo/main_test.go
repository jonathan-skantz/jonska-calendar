package demo

import (
	"testing"

	"github.com/jonathan-skantz/calendar/backend/internal/testutil"
)

// One database container serves all of this package's tests.
func TestMain(m *testing.M) {
	testutil.Run(m)
}
