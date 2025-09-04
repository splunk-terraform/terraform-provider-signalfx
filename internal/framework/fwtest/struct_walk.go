// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"container/list"
	"fmt"
	"iter"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

type WalkedValue struct {
	v   any
	p   path.Path
	err error
}

func NewWalkedValue(p path.Path, v any) WalkedValue {
	return WalkedValue{
		v: v,
		p: p,
	}
}

func NewWalkedValueError(msg string, args ...any) WalkedValue {
	return NewWalkedValueErrorWithPath(path.Empty(), msg, args...)
}

func NewWalkedValueErrorWithPath(p path.Path, msg string, args ...any) WalkedValue {
	return WalkedValue{
		p:   p,
		err: fmt.Errorf(msg, args...),
	}
}

func (wv WalkedValue) Any() any {
	return wv.v
}

func (w WalkedValue) Attr() attr.Value {
	if cast, ok := w.v.(attr.Value); ok {
		return cast
	}
	return nil
}

func (w WalkedValue) Err() error {
	return w.err
}

func (w WalkedValue) Path() path.Path {
	return w.p
}

func (wv WalkedValue) String() string {
	return fmt.Sprintf("path:%s value:%v error:%v", wv.Path(), wv.Any(), wv.Err())
}

func WalkStruct(orig any) iter.Seq[WalkedValue] {
	return func(yield func(WalkedValue) bool) {
		v := reflect.ValueOf(orig)
		if v.Kind() == reflect.Pointer {
			v = v.Elem()
		}
		// Ensure the initial value is a struct so it can be explored
		if v.Kind() != reflect.Struct {
			yield(NewWalkedValueError("model must be a struct, provided: %T", orig))
			return
		}
		var (
			queue = list.New()
		)
		queue.PushBack(WalkedValue{p: path.Empty(), v: v.Interface()})
		for queue.Len() > 0 {
			var (
				elem = queue.Remove(queue.Front()).(WalkedValue)
				v    = reflect.ValueOf(elem.Any())
			)
			if v.Kind() == reflect.Pointer {
				v = v.Elem()
			}
			switch v.Kind() {
			case reflect.Struct:
				for i := range v.Type().NumField() {
					var (
						field = v.Type().Field(i)
						named = v.Type().Field(i).Tag.Get("tfsdk")
					)
					if named == "" {
						continue
					}
					if !field.IsExported() {
						queue.PushBack(NewWalkedValueErrorWithPath(
							elem.Path().AtName(named),
							"framework requires field exported %q", named),
						)
						continue
					}

					// If there is a value set within the model, then use
					// the provided value, otherwise use a zero value
					if value := v.Field(i); !value.IsZero() {
						queue.PushBack(NewWalkedValue(
							elem.Path().AtName(named),
							value.Interface()),
						)
						continue
					}

					zero := reflect.Zero(field.Type)
					if zero.Kind() == reflect.Pointer {
						zero = reflect.Zero(field.Type.Elem())
					}
					queue.PushBack(NewWalkedValue(
						elem.Path().AtName(named),
						zero.Interface()),
					)
				}
			case reflect.Slice, reflect.Array:
				zero := reflect.MakeSlice(v.Type(), 1, 1)
				vv := reflect.Zero(zero.Index(0).Type())
				if vv.Kind() == reflect.Pointer {
					vv = reflect.Zero(vv.Type().Elem())
				}
				queue.PushBack(NewWalkedValue(
					elem.Path().AtListIndex(0),
					vv.Interface()),
				)
				continue
			}

			// Ignore the parent value
			if !elem.Path().Equal(path.Empty()) && !yield(elem) {
				return
			}

		}
	}
}
