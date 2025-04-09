package store

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPgxRole_ScanText(t *testing.T) {
	var pgxRole = PgxRole(role.Guest)
	err := pgxRole.ScanText(pgtype.Text{
		String: "owner",
		Valid:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, role.Owner, pgxRole.AsRole())
}
