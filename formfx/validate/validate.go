package validate

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
)

// errorFn is a type for validation functions that return an error.
type errorFn func(string) error

// NonEmpty returns an error if the input string is empty.
func NonEmpty() errorFn {
	return func(s string) error {
		if s == "" {
			return errors.New("input cannot be empty")
		}
		return nil
	}
}

// Matches returns an error if the input string does not match the given regex.
func Matches(rx *regexp.Regexp) errorFn {
	return func(s string) error {
		if !rx.MatchString(s) {
			return fmt.Errorf("input does not match required pattern: %s", rx.String())
		}
		return nil
	}
}

// In returns an error if the input string is not one of the allowed values.
func In(allowed ...string) errorFn {
	return func(s string) error {
		if slices.Contains(allowed, s) {
			return nil
		}
		return fmt.Errorf("input must be one of %v", allowed)
	}
}

// MinLen returns an error if the input string's length is less than n.
func MinLen(n int) errorFn {
	return func(s string) error {
		if len(s) < n {
			return fmt.Errorf("input must be at least %d characters long", n)
		}
		return nil
	}
}

// MaxLen returns an error if the input string's length is greater than n.
func MaxLen(n int) errorFn {
	return func(s string) error {
		if len(s) > n {
			return fmt.Errorf("input must be at most %d characters long", n)
		}
		return nil
	}
}

// All composes multiple validation functions into a single one.
// It returns the first error encountered.
func All(validators ...errorFn) errorFn {
	return func(s string) error {
		for _, validator := range validators {
			if err := validator(s); err != nil {
				return err
			}
		}
		return nil
	}
}
