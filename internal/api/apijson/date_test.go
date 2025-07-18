package apijson

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDate_MarshalText(t *testing.T) {
	dateTimeObject := time.Now()
	dateString := dateTimeObject.Format(time.DateOnly)
	dateObject := Date(dateTimeObject)

	t.Run("as pointer member", func(t *testing.T) {
		hasDatePointer := struct {
			Date *Date
		}{
			Date: &dateObject,
		}

		asBytes, err := json.Marshal(hasDatePointer)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf(`{"Date":%q}`, dateString), string(asBytes))
	})

	t.Run("as non-pointer member", func(t *testing.T) {
		hasDate := struct {
			Date Date
		}{
			Date: dateObject,
		}

		asBytes, err := json.Marshal(hasDate)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf(`{"Date":%q}`, dateString), string(asBytes))
	})
}

func TestDate_UnmarshalText(t *testing.T) {
	dateTimeObject := time.Now()
	dateString := dateTimeObject.Format(time.DateOnly)

	jsonBytes := []byte(fmt.Sprintf(`{"Date":%q}`, dateString))

	t.Run("as pointer member", func(t *testing.T) {
		var hasDatePointer struct {
			Date *Date
		}

		require.NoError(t, json.Unmarshal(jsonBytes, &hasDatePointer))
		assert.NotZero(t, *hasDatePointer.Date)

		assert.True(t, hasDatePointer.Date.EqualToTime(dateTimeObject))
	})

	t.Run("as non-pointer member", func(t *testing.T) {
		var hasDate struct {
			Date Date
		}

		require.NoError(t, json.Unmarshal(jsonBytes, &hasDate))
		assert.NotZero(t, hasDate.Date)

		assert.True(t, hasDate.Date.EqualToTime(dateTimeObject))
	})
}
