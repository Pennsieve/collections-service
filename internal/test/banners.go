package test

import (
	"fmt"
	"github.com/google/uuid"
)

func NewBanner() *string {
	banner := fmt.Sprintf("https://example.com/%s.png", uuid.NewString())
	return &banner
}
