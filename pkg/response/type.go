package response

import (
	"encoding/json"
	"time"
)

// Resp is the standard JSON response body.
type Resp struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Errors    any    `json:"errors,omitempty"`
}

// Date is a date that marshals as DateFormat.
type Date time.Time

// MarshalJSON implements json.Marshaler for Date.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).Local().Format(DateFormat))
}

// DateTime is a datetime that marshals as DateTimeFormat.
type DateTime time.Time

// MarshalJSON implements json.Marshaler for DateTime.
func (d DateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).Local().Format(DateTimeFormat))
}
