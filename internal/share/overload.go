package share

import "fmt"

// Overload[T] provides strict one-argument coercion into T.
//
// - If no value is passed, fallback is used.
// - If one value is passed, it MUST be T or *T.
// - Any other type (including nil, map, or unrelated struct) triggers panic.
//
// This is NOT a soft cast. It’s a strict dispatch helper for multipath entry.
func Overload[T any](have []any, fallback T) T {
	value, err := TryOverload(have, fallback)
	if err != nil {
		panic(err)
	}
	return value
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

// OverloadWithOptions[T] handles multipath with strict rules:
//
// - If no args: use fallback
// - Can mix: config struct (T or *T) + functional options (Option[T])
// - Only one config struct allowed
// - All types must match T
func OverloadWithOptions[T any](args []any, fallback T) T {
	value, err := TryOverloadWithOptions(args, fallback)
	if err != nil {
		panic(err)
	}
	return value
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
