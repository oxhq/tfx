package main

import (
	"io"
	"os"
)

func main() {
	_, _ = io.WriteString(os.Stdout, "TFX demo: running formfx showcase\n")
	runFormFXDemo()
}
