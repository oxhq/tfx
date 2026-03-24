package share

import "testing"

type overloadConfig struct {
	Name string
}

func TestTryOverloadReturnsErrorForInvalidArguments(t *testing.T) {
	_, err := TryOverload([]any{overloadConfig{}, overloadConfig{}}, overloadConfig{})
	if err == nil {
		t.Fatal("expected error for multiple arguments")
	}

	_, err = TryOverload([]any{"bad"}, overloadConfig{})
	if err == nil {
		t.Fatal("expected error for invalid argument type")
	}
}

func TestTryOverloadWithOptionsReturnsErrorForInvalidArguments(t *testing.T) {
	type cfg struct {
		Name string
	}

	_, err := TryOverloadWithOptions([]any{cfg{}, &cfg{}}, cfg{})
	if err == nil {
		t.Fatal("expected error for multiple config values")
	}

	_, err = TryOverloadWithOptions([]any{"bad"}, cfg{})
	if err == nil {
		t.Fatal("expected error for invalid argument type")
	}
}

func TestTryOverloadWithOptionsAppliesOptions(t *testing.T) {
	type cfg struct {
		Name string
	}

	got, err := TryOverloadWithOptions([]any{
		cfg{Name: "base"},
		Option[cfg](func(c *cfg) { c.Name = "updated" }),
	}, cfg{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "updated" {
		t.Fatalf("expected option to update config, got %q", got.Name)
	}
}

func TestOverloadStillPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = Overload([]any{"bad"}, overloadConfig{})
}

func TestOverloadWithOptionsStillPanics(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = OverloadWithOptions([]any{"bad"}, overloadConfig{})
}

func TestMustOverloadStillPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = MustOverload([]any{"bad"}, overloadConfig{})
}

func TestMustOverloadWithOptionsStillPanics(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic")
		}
	}()

	_ = MustOverloadWithOptions([]any{"bad"}, overloadConfig{})
}
