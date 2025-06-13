package apijson

import (
	"fmt"
	"time"
)

// Date is a time.Time for JSON that only looks at date portion of value. Needed for
// embargo release date JSON field and others which has no time info.
type Date time.Time

func (d Date) MarshalText() (text []byte, err error) {
	dateOnly := time.Time(d).Format(time.DateOnly)
	return []byte(dateOnly), nil
}

func (d *Date) UnmarshalText(data []byte) error {
	parsed, err := time.Parse(time.DateOnly, string(data))
	if err != nil {
		return fmt.Errorf("error parsing Date %s: %w", string(data), err)
	}
	*d = Date(parsed)
	return nil
}
