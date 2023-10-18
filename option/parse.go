package option

import (
	"reflect"
)

// FlagParser parses flags in an option.
type FlagParser interface {
	Parse() error
}

// Parse parses applicable fields of the passed-in option pointer and returns
// error during parsing.
func Parse(optsPtr interface{}) error {
	return rangeFields(optsPtr, func(fp FlagParser) error {
		return fp.Parse()
	})
}

// rangeFields goes through all fields of ptr, optionally run fn if a field is
// public AND typed T.
func rangeFields[T any](ptr any, fn func(T) error) error {
	v := reflect.ValueOf(ptr).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.CanSet() {
			iface := f.Addr().Interface()
			if opts, ok := iface.(T); ok {
				if err := fn(opts); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
