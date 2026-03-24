package share

import "fmt"

// MustOverload provides strict one-argument coercion into T and panics on invalid input.
//
// Prefer TryOverload when args may be user-provided or otherwise fallible.
func MustOverload[T any](have []any, fallback T) T {
	value, err := TryOverload(have, fallback)
	if err != nil {
		panic(err)
	}
	return value
}

// Overload is a compatibility alias for MustOverload.
//
// Prefer TryOverload for fallible multipath parsing or MustOverload when panic
// semantics should be explicit at the callsite.
func Overload[T any](have []any, fallback T) T {
	return MustOverload(have, fallback)
}

// TryOverload provides the same strict dispatch as Overload without panicking.
func TryOverload[T any](have []any, fallback T) (T, error) {
	if len(have) == 0 {
		return fallback, nil
	}
	if len(have) > 1 {
		return fallback, fmt.Errorf("overload: expected 0 or 1 argument")
	}

	arg := have[0]
	var zero T // we only use this to format the error message

	switch v := arg.(type) {
	case *T:
		return *v, nil
	case T:
		return v, nil
	default:
		return fallback, fmt.Errorf(
			"overload: expected type %T or *%T, got %T",
			zero, &zero, arg,
		)
	}
}

// MustOverloadWithOptions handles multipath parsing and panics on invalid input.
//
// Prefer TryOverloadWithOptions when args may be user-provided or otherwise fallible.
func MustOverloadWithOptions[T any](args []any, fallback T) T {
	value, err := TryOverloadWithOptions(args, fallback)
	if err != nil {
		panic(err)
	}
	return value
}

// OverloadWithOptions is a compatibility alias for MustOverloadWithOptions.
//
// Prefer TryOverloadWithOptions for fallible multipath parsing or
// MustOverloadWithOptions when panic semantics should be explicit at the callsite.
func OverloadWithOptions[T any](args []any, fallback T) T {
	return MustOverloadWithOptions(args, fallback)
}

// TryOverloadWithOptions handles multipath parsing without panicking.
func TryOverloadWithOptions[T any](args []any, fallback T) (T, error) {
	if len(args) == 0 {
		return fallback, nil
	}

	var holder *T
	var options []Option[T]

	for _, arg := range args {
		switch v := arg.(type) {
		case Option[T]:
			options = append(options, v)
		case *T:
			if holder != nil {
				return fallback, fmt.Errorf(
					"OverloadWithOptions: multiple config structs not allowed",
				)
			}
			holder = v
		case T:
			if holder != nil {
				return fallback, fmt.Errorf(
					"OverloadWithOptions: multiple config structs not allowed",
				)
			}
			temp := v
			holder = &temp
		default:
			return fallback, fmt.Errorf(
				"OverloadWithOptions: expected type %T, *%T, or Option[%T], got %T",
				fallback, &fallback, fallback, arg,
			)
		}
	}

	// Determine the base instance
	var instance T
	if holder != nil {
		instance = *holder
	} else {
		instance = fallback
	}

	// Apply all functional options
	ApplyOptions(&instance, options...)
	return instance, nil
}
