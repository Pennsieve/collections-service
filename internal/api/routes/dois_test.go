package routes

import (
	"github.com/pennsieve/collections-service/internal/test/apitest"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCategorizeDOIs(t *testing.T) {
	pennsieveDOI1 := apitest.NewPennsieveDOI().Value
	pennsieveDOI2 := apitest.NewPennsieveDOI().Value
	pennsieveDOI3 := apitest.NewPennsieveDOI().Value

	externalDOI1 := apitest.NewExternalDOI().Value
	externalDOI2 := apitest.NewExternalDOI().Value

	type args struct {
		inputDOIs             []string
		expectedPennsieveDOIs []string
		expectedExternalDOIs  []string
	}
	tests := []struct {
		name string
		args args
	}{
		{"no DOIs",
			args{nil, nil, nil},
		},
		{"no dups",
			args{
				inputDOIs:             []string{pennsieveDOI1, pennsieveDOI2, externalDOI1, pennsieveDOI3, externalDOI2},
				expectedPennsieveDOIs: []string{pennsieveDOI1, pennsieveDOI2, pennsieveDOI3},
				expectedExternalDOIs:  []string{externalDOI1, externalDOI2}},
		},
		{"some dups",
			args{inputDOIs: []string{pennsieveDOI3, pennsieveDOI1, pennsieveDOI2, pennsieveDOI3, pennsieveDOI2},
				expectedPennsieveDOIs: []string{pennsieveDOI3, pennsieveDOI1, pennsieveDOI2},
				expectedExternalDOIs:  nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPennsieve, actualExternal := CategorizeDOIs(apitest.PennsieveDOIPrefix, tt.args.inputDOIs)
			assert.Equal(t, tt.args.expectedPennsieveDOIs, actualPennsieve)
			assert.Equal(t, tt.args.expectedExternalDOIs, actualExternal)
		})
	}

}
