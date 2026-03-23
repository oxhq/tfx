package flowfx

import (
	"maps"

	"github.com/oxhq/tfx/internal/share"
)

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func tryConstruct[C any, T any](args []any, defaults C, ctor func(C) T) (T, error) {
	cfg, err := share.TryOverload(args, defaults)
	if err != nil {
		var zero T
		return zero, err
	}
	return ctor(cfg), nil
}

func cloneSlice[T any](src []T) []T {
	if src == nil {
		return nil
	}
	dst := make([]T, len(src))
	copy(dst, src)
	return dst
}

func cloneMap[K comparable, V any](src map[K]V) map[K]V {
	if src == nil {
		return nil
	}
	dst := make(map[K]V, len(src))
	maps.Copy(dst, src)
	return dst
}

func appendUnique(order []string, key string) []string {
	for _, existing := range order {
		if existing == key {
			return order
		}
	}
	return append(order, key)
}

func dedupeStrings(src []string) []string {
	if src == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(src))
	deduped := make([]string, 0, len(src))
	for _, value := range src {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	return deduped
}
