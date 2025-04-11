package store

import (
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"time"
)

type Collection struct {
	ID          int64     `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	UpdatedAt   time.Time `db:"updated_at"`
	CreatedAt   time.Time `db:"created_at"`
	NodeID      string    `db:"node_id"`
}

type CollectionUser struct {
	CollectionID  int64             `db:"collection_id"`
	UserID        int64             `db:"user_id"`
	PermissionBit pgdb.DbPermission `db:"permission_bit"`
	CreatedAt     time.Time         `db:"created_at"`
	UpdatedAt     time.Time         `db:"updated_at"`
	Role          PgxRole           `db:"role"`
}

// PgxRole is a wrapper around role.Role that implements pgtype.TextScanner and pgtype.TextValuer
// so that we can scan into them and use them as query parameters
type PgxRole role.Role

func (r *PgxRole) ScanText(v pgtype.Text) error {
	if !v.Valid {
		return fmt.Errorf("invalid pgtype.Text: %s", v.String)
	}
	roleFromString, valid := role.RoleFromString(v.String)
	if !valid {
		return fmt.Errorf("invalid string for role.Role: %s", v.String)
	}
	*r = PgxRole(roleFromString)
	return nil
}

// TextValue needs a non-pointer receiver in order to work. (So that PgxRole values can be passed to the Query functions)
func (r PgxRole) TextValue() (pgtype.Text, error) {
	roleString := r.AsRole().String()
	return pgtype.Text{String: roleString, Valid: true}, nil
}

func (r *PgxRole) AsRole() role.Role {
	return role.Role(*r)
}

type CollectionDOI struct {
	ID           int64     `db:"id"`
	CollectionID int64     `db:"collection_id"`
	DOI          string    `db:"doi"`
	UpdatedAt    time.Time `db:"updated_at"`
	CreatedAt    time.Time `db:"created_at"`
}
