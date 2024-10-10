// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfext

import "time"

// LogFields is an extension to logging fields parameter
// to help as a convience and provides some level of standards.
type LogFields map[string]any

// NewLogFields creates an empty [LogFields] that be used
// as part of the `tflog` fields parameter.
func NewLogFields() LogFields {
	return make(LogFields)
}

// ErrorLogFields is a convience function that allows
// for a [LogFields] to be created with the error entry set.
func ErrorLogFields(err error) LogFields {
	return NewLogFields().Error(err)
}

// Error appends the field name `error` if the error value is not nil
func (lf LogFields) Error(err error) LogFields {
	if err != nil {
		lf["error"] = err.Error()
	}
	return lf
}

// Duration is a convience function to ensure the key is correctly set.
func (lf LogFields) Duration(key string, val time.Duration) LogFields {
	return lf.Field(key, val.String())
}

// Field appends any type to be set as the key's value.
// if the field already exists, it is overwritten.
func (lf LogFields) Field(key string, val any) LogFields {
	lf[key] = val
	return lf
}
