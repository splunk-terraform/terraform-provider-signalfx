package util

import (
	"encoding/json"
	"strconv"
)

// StringOrInteger is scalar value that might be a string or an integer.
type StringOrInteger string

// UnmarshalJSON handles the decision of this being a string or integer
// and unmarshaling accordingly.
func (sos *StringOrInteger) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*sos = StringOrInteger(s)
	} else {
		var num int
		err := json.Unmarshal(b, &num)
		if err != nil {
			return err
		}
		*sos = StringOrInteger(strconv.Itoa(num))
	}
	return nil
}
