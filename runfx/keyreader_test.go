package runfx

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestKeyReaderReadsUTF8Rune(t *testing.T) {
	kr := NewKeyReader(bytes.NewBufferString("é"))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	key, err := kr.ReadKey(ctx)
	if err != nil {
		t.Fatalf("ReadKey returned error: %v", err)
	}

	if key.Rune != 'é' {
		t.Fatalf("expected UTF-8 rune é, got %q", key.Rune)
	}

	if key.Code != KeyUnknown {
		t.Fatalf("expected unknown code for non-ASCII rune, got %v", key.Code)
	}
}

func TestKeyReaderReadsCtrlZ(t *testing.T) {
	kr := NewKeyReader(bytes.NewReader([]byte{26}))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	key, err := kr.ReadKey(ctx)
	if err != nil {
		t.Fatalf("ReadKey returned error: %v", err)
	}

	if key.Code != KeyCtrlZ {
		t.Fatalf("expected Ctrl+Z, got %v", key.Code)
	}

	if key.Modifier != ModCtrl {
		t.Fatalf("expected Ctrl modifier, got %v", key.Modifier)
	}
}
