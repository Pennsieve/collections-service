package test

import (
	"fmt"
	"github.com/google/uuid"
)

const DOIPrefix = "10.9999.9"

func NewDOI() string {
	return fmt.Sprintf("%s/%s", DOIPrefix, uuid.NewString())
}
