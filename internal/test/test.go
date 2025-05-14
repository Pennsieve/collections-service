package test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Helper(t require.TestingT) {
	if tt, hasHelper := t.(*testing.T); hasHelper {
		tt.Helper()
	}
}
