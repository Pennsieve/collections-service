package service

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPublishDOICollectionRequest_Marshal(t *testing.T) {

	request := PublishDOICollectionRequest{}

	reqBytes, err := json.Marshal(request)
	require.NoError(t, err)
	reqJSONString := string(reqBytes)

	// Test that required slice values are marshalled as empty arrays rather than nulls.
	requiredArrays := []string{"banners", "dois", "contributors", "tags"}

	for _, requiredArray := range requiredArrays {
		t.Run(fmt.Sprintf("%s is [] and not null", requiredArray), func(t *testing.T) {
			isNull := fmt.Sprintf(`%q:null`, requiredArray)
			isEmpty := fmt.Sprintf(`%q:[]`, requiredArray)
			assert.NotContains(t, reqJSONString, isNull)
			assert.Contains(t, reqJSONString, isEmpty)
		})

	}
}
