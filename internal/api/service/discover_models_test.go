package service

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
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

func TestInternalContributorBuilder_Build(t *testing.T) {

	t.Run("correct values are set", func(t *testing.T) {
		first := uuid.NewString()
		last := uuid.NewString()
		middle := uuid.NewString()
		orcid := uuid.NewString()
		degree := uuid.NewString()
		userID := rand.Int64()

		c := NewInternalContributorBuilder().
			WithFirstName(first).
			WithLastName(last).
			WithMiddleInitial(middle).
			WithORCID(orcid).
			WithDegree(degree).
			WithUserID(userID).
			Build()

		assert.Positive(t, c.ID)
		assert.Equal(t, first, c.FirstName)
		assert.Equal(t, last, c.LastName)
		assert.Equal(t, middle, c.MiddleInitial)
		assert.Equal(t, orcid, c.ORCID)
		assert.Equal(t, degree, c.Degree)
		assert.Equal(t, userID, c.UserID)
	})

	t.Run("same values => same id, different values => different id", func(t *testing.T) {
		value1 := uuid.NewString()
		value2 := uuid.NewString()
		c1 := NewInternalContributorBuilder().
			WithFirstName(value1).
			WithLastName(value2).
			Build()
		require.Positive(t, c1.ID)

		c2 := NewInternalContributorBuilder().
			WithFirstName(value1).
			WithLastName(value2).
			Build()
		assert.Equal(t, c1.ID, c2.ID)

		d := NewInternalContributorBuilder().
			WithFirstName(value2).
			WithLastName(value1).
			Build()
		assert.NotEqual(t, c1.ID, d.ID)

	})

	t.Run("same value in different fields => different id", func(t *testing.T) {
		value := uuid.NewString()
		c := NewInternalContributorBuilder().WithFirstName(value).Build()
		d := NewInternalContributorBuilder().WithLastName(value).Build()
		assert.NotEqual(t, c.ID, d.ID)
	})
}
