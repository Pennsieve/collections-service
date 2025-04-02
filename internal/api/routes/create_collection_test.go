package routes

import (
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeduplicateDOIs(t *testing.T) {
	doi1 := test.NewDOI()
	doi2 := test.NewDOI()
	doi3 := test.NewDOI()
	type args struct {
		createRequest *dto.CreateCollectionRequest
		expectedDOIs  []string
	}
	tests := []struct {
		name string
		args args
	}{
		{"no dois",
			args{&dto.CreateCollectionRequest{Name: uuid.NewString()}, nil},
		},
		{"no dups",
			args{createRequest: &dto.CreateCollectionRequest{Name: uuid.NewString(),
				DOIs: []string{doi1, doi2, doi3}},
				expectedDOIs: []string{doi1, doi2, doi3}},
		},
		{"some dups",
			args{createRequest: &dto.CreateCollectionRequest{Name: uuid.NewString(),
				DOIs: []string{doi3, doi1, doi2, doi3, doi2}},
				expectedDOIs: []string{doi3, doi1, doi2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DeduplicateDOIs(tt.args.createRequest)
			assert.Equal(t, tt.args.expectedDOIs, tt.args.createRequest.DOIs)
		})
	}
}
