package runfx

import "testing"

type orderedVisual struct {
	id      string
	resized []string
}

func (v *orderedVisual) Render() []byte {
	return []byte(v.id)
}

func (v *orderedVisual) OnResize(cols, rows int) {
	v.resized = append(v.resized, v.id)
}

func TestMultiplexerPreservesMountOrder(t *testing.T) {
	mux := NewMultiplexer()
	first := &orderedVisual{id: "A"}
	second := &orderedVisual{id: "B"}

	firstID := mux.Mount(first)
	secondID := mux.Mount(second)

	if got := string(mux.Render()); got != "AB" {
		t.Fatalf("expected render order AB, got %q", got)
	}

	ids := mux.ListVisuals()
	if len(ids) != 2 || ids[0] != firstID || ids[1] != secondID {
		t.Fatalf("expected stable mount order, got %#v", ids)
	}

	mux.OnResize(80, 24)
	if len(first.resized) != 1 || len(second.resized) != 1 {
		t.Fatal("expected resize to reach both visuals")
	}

	mux.Unmount(firstID)
	if got := string(mux.Render()); got != "B" {
		t.Fatalf("expected render order B after unmount, got %q", got)
	}
}
