package test

import (
	"fmt"
	"github.com/google/uuid"
)

const ExternalDOIPrefix = "10.9999.9"

func NewExternalDOI() string {
	return fmt.Sprintf("%s/%s", ExternalDOIPrefix, uuid.NewString())
}

const PennsieveDOIPrefix = "10.1111"

func NewPennsieveDOI() string {
	return NewDOIWithPrefix(PennsieveDOIPrefix)
}

func NewDOIWithPrefix(prefix string) string {
	return fmt.Sprintf("%s/%s", prefix, uuid.NewString())
}
