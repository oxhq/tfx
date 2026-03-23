package runfx

import (
	"bytes"
	"os"
	"testing"
)

func TestNewLoopWithConfigUsesConfiguredInput(t *testing.T) {
	input := bytes.NewBufferString("x")
	cfg := DefaultConfig()
	cfg.Input = input

	loop, ok := newLoopWithConfig(cfg).(*MainLoop)
	if !ok {
		t.Fatal("newLoopWithConfig did not return *MainLoop")
	}

	if loop.reader == nil {
		t.Fatal("loop reader is nil")
	}

	if loop.reader.input != input {
		t.Fatal("loop reader did not use configured input")
	}

	if loop.ticker != nil {
		t.Fatal("loop should not allocate ticker before Run")
	}
}

func TestTryStartRejectsInvalidMultipathArgs(t *testing.T) {
	if _, err := TryStart("bad"); err == nil {
		t.Fatal("expected invalid config error")
	}
}

func TestResolveTTYFilePrefersOutputThenInput(t *testing.T) {
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create output pipe: %v", err)
	}
	defer func() {
		_ = outReader.Close()
		_ = outWriter.Close()
	}()

	inReader, inWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create input pipe: %v", err)
	}
	defer func() {
		_ = inReader.Close()
		_ = inWriter.Close()
	}()

	if got := resolveTTYFile(outWriter, inReader); got != outWriter {
		t.Fatal("expected output file to take precedence")
	}
	if got := resolveTTYFile(bytes.NewBuffer(nil), inReader); got != inReader {
		t.Fatal("expected input file fallback when output is not a file")
	}
}
