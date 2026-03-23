package runfx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// KeyReader handles reading keyboard input from a terminal and converting it to Key events.
type KeyReader struct {
	reader *bufio.Reader
	input  io.Reader
}

// NewKeyReader creates a new keyboard input reader.
func NewKeyReader(input io.Reader) *KeyReader {
	if input == nil {
		input = os.Stdin
	}
	return &KeyReader{
		reader: bufio.NewReader(input),
		input:  input,
	}
}

// ReadKey reads the next keyboard input and returns the corresponding Key.
func (kr *KeyReader) ReadKey(ctx context.Context) (Key, error) {
	keyCh := make(chan Key, 1)
	errCh := make(chan error, 1)

	go func() {
		key, err := kr.readKeyBlocking()
		if err != nil {
			errCh <- err
			return
		}
		keyCh <- key
	}()

	select {
	case key := <-keyCh:
		return key, nil
	case err := <-errCh:
		return Key{Code: KeyUnknown}, err
	case <-ctx.Done():
		return Key{Code: KeyUnknown}, ctx.Err()
	}
}

func (kr *KeyReader) readKeyBlocking() (Key, error) {
	r, _, err := kr.reader.ReadRune()
	if err != nil {
		return Key{Code: KeyUnknown}, err
	}

	if r == 27 {
		next, err := kr.reader.Peek(1)
		if errors.Is(err, io.EOF) {
			err = nil
		}
		if err != nil {
			return Key{Code: KeyUnknown}, fmt.Errorf("peek escape sequence: %w", err)
		}
		if len(next) == 0 {
			return Key{Code: KeyEscape}, nil
		}

		if next[0] == '[' {
			_, _ = kr.reader.ReadByte() // consume '['
			return kr.parseCSISequence()
		}
		return Key{Code: KeyEscape}, nil
	}

	return kr.parseRegularRune(r), nil
}

func (kr *KeyReader) parseCSISequence() (Key, error) {
	seq := []byte{}

	for {
		b, err := kr.reader.ReadByte()
		if err != nil {
			return Key{Code: KeyUnknown}, err
		}
		seq = append(seq, b)
		if (b >= 'A' && b <= 'Z') || b == '~' {
			break
		}
	}

	return kr.decodeCSI(seq)
}

func (kr *KeyReader) decodeCSI(seq []byte) (Key, error) {
	s := string(seq)

	switch s {
	case "A":
		return Key{Code: KeyArrowUp}, nil
	case "B":
		return Key{Code: KeyArrowDown}, nil
	case "C":
		return Key{Code: KeyArrowRight}, nil
	case "D":
		return Key{Code: KeyArrowLeft}, nil
	case "3~":
		return Key{Code: KeyDelete}, nil
	}

	if strings.Contains(s, ";") {
		parts := strings.Split(s, ";")
		if len(parts) != 2 {
			return Key{Code: KeyUnknown}, nil
		}
		modNum, _ := strconv.Atoi(parts[1][:1])
		var mod Modifier
		switch modNum {
		case 2:
			mod = ModShift
		case 3:
			mod = ModAlt
		case 4:
			mod = ModShift | ModAlt
		case 5:
			mod = ModCtrl
		case 6:
			mod = ModCtrl | ModShift
		case 7:
			mod = ModCtrl | ModAlt
		case 8:
			mod = ModCtrl | ModAlt | ModShift
		default:
			mod = ModNone
		}
		last := parts[1][1:]
		switch last {
		case "A":
			return Key{Code: KeyArrowUp, Modifier: mod}, nil
		case "B":
			return Key{Code: KeyArrowDown, Modifier: mod}, nil
		case "C":
			return Key{Code: KeyArrowRight, Modifier: mod}, nil
		case "D":
			return Key{Code: KeyArrowLeft, Modifier: mod}, nil
		}
	}

	return Key{Code: KeyUnknown}, nil
}

func (kr *KeyReader) parseRegularRune(r rune) Key {
	switch r {
	case '\r', '\n':
		return Key{Code: KeyEnter}
	case '\t':
		return Key{Code: KeyTab}
	case ' ':
		return Key{Code: KeySpace, Rune: ' '}
	case 127, 8:
		return Key{Code: KeyBackspace}
	case 3:
		return Key{Code: KeyCtrlC, Modifier: ModCtrl}
	case 4:
		return Key{Code: KeyCtrlD, Modifier: ModCtrl}
	case 26:
		return Key{Code: KeyCtrlZ, Modifier: ModCtrl}
	default:
		if r >= '0' && r <= '9' {
			return Key{Code: KeyCode(int(Key0) + int(r-'0')), Rune: r}
		}
		if r >= 'a' && r <= 'z' {
			return Key{Code: KeyCode(int(KeyA) + int(r-'a')), Rune: r}
		}
		if r >= 'A' && r <= 'Z' {
			return Key{Code: KeyCode(int(KeyA) + int(r-'A')), Rune: r, Modifier: ModShift}
		}
		return Key{Code: KeyUnknown, Rune: r}
	}
}
