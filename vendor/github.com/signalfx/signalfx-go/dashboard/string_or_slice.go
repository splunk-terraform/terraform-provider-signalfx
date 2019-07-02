package dashboard

import "encoding/json"

// StringOrSlice is a slice of strings that might be just a single
// string in JSON. e.g. if the value is "foo" we want to make it ["foo"]
type StringOrSlice []string

// UnmarshalJSON handles the decision of this being a string or slice
// and unmarshaling accordingly.
func (sos *StringOrSlice) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*sos = StringOrSlice([]string{s})
	} else {
		return json.Unmarshal(b, (*[]string)(sos))
	}
	return nil
}
