package runfx

import "testing"

func TestKeyIsPrintableIncludesWASDAndSpace(t *testing.T) {
	cases := []Key{
		{Code: KeyW, Rune: 'w'},
		{Code: KeyA, Rune: 'a'},
		{Code: KeyS, Rune: 's'},
		{Code: KeyD, Rune: 'd'},
		{Code: KeySpace, Rune: ' '},
	}

	for _, key := range cases {
		if !key.IsPrintable() {
			t.Fatalf("expected %v to be printable", key.Code)
		}
	}
}

func TestKeyIsCancelIncludesCtrlZ(t *testing.T) {
	if !(Key{Code: KeyCtrlZ}).IsCancel() {
		t.Fatal("expected Ctrl+Z to be a cancel key")
	}
}
