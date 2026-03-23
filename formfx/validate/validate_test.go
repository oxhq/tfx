package validate

import (
	"regexp"
	"testing"
)

func TestNonEmpty(t *testing.T) {
	t.Parallel()

	if err := NonEmpty()(""); err == nil {
		t.Fatal("expected error for empty input")
	}
	if err := NonEmpty()("ok"); err != nil {
		t.Fatalf("unexpected error for non-empty input: %v", err)
	}
}

func TestMatches(t *testing.T) {
	t.Parallel()

	rx := regexp.MustCompile(`^[a-z]+$`)
	if err := Matches(rx)("abc"); err != nil {
		t.Fatalf("unexpected regex error: %v", err)
	}
	if err := Matches(rx)("123"); err == nil {
		t.Fatal("expected regex mismatch error")
	}
}

func TestInMinMaxAndAll(t *testing.T) {
	t.Parallel()

	if err := In("red", "blue")("green"); err == nil {
		t.Fatal("expected membership validation error")
	}
	if err := MinLen(3)("go"); err == nil {
		t.Fatal("expected min length error")
	}
	if err := MaxLen(3)("gopher"); err == nil {
		t.Fatal("expected max length error")
	}

	validator := All(NonEmpty(), MinLen(3), MaxLen(5), In("go1", "go2", "go3"))
	if err := validator("go2"); err != nil {
		t.Fatalf("unexpected composed validation error: %v", err)
	}
	if err := validator(""); err == nil {
		t.Fatal("expected first validator failure")
	}
}
