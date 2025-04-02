package test

type SeedUser struct {
	ID           int64
	NodeID       string
	IsSuperAdmin bool
}

// These users are already present in the Pennsieve seed DB Docker container used for tests

var User = SeedUser{
	ID:           1,
	NodeID:       "N:user:99f02be5-009c-4ecd-9006-f016d48628bf",
	IsSuperAdmin: false,
}

var User2 = SeedUser{
	ID:           2,
	NodeID:       "N:user:29cb5354-b471-4a72-adae-6fcb262447d9",
	IsSuperAdmin: false,
}

var SuperUser = SeedUser{
	ID:           3,
	NodeID:       "N:user:4e8c459b-bffb-49e1-8c6a-de2d8190d84e",
	IsSuperAdmin: true,
}
