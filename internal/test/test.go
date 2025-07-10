package test

import (
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func Helper(t require.TestingT) {
	if tt, hasHelper := t.(*testing.T); hasHelper {
		tt.Helper()
	}
}

func Close(t require.TestingT, closer io.Closer) {
	require.NoError(t, closer.Close())
}
