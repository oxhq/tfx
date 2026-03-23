package main

import (
	"io"
	"os"
	"time"

	// progrefx provides progress bars and spinners integrated with runfx.
	"github.com/oxhq/tfx/progrefx"
	"github.com/oxhq/tfx/runfx"
)

func runFormFXDemo() {
	runFormFXDemoWithOptions(os.Stdout, time.Sleep)
}

func runFormFXDemoWithOptions(output io.Writer, sleep func(time.Duration)) {
	if output == nil {
		output = os.Stdout
	}
	if sleep == nil {
		sleep = time.Sleep
	}

	// Create a progress bar using the express path.
	progressBar := progrefx.Start(progrefx.ProgressConfig{
		Total:     100,
		Label:     "Downloading files...",
		Writer:    output,
		DetectTTY: func() runfx.TTYInfo { return runfx.DetectTTYForOutput(output) },
	})

	_, _ = io.WriteString(output, "Starting main task...\n")
	for i := 0; i <= 100; i++ {
		progressBar.Set(i)
		_, _ = io.WriteString(output, progressBar.Render())
		sleep(30 * time.Millisecond)
	}

	progressBar.Finish()
	_, _ = io.WriteString(output, "\nTask completed.\n")

	// Demonstrate spinner usage.
	spinner := progrefx.StartSpinner(progrefx.SpinnerConfig{
		Label:     "Processing",
		Frames:    progrefx.DefaultSpinnerConfig().Frames,
		Writer:    output,
		DetectTTY: func() runfx.TTYInfo { return runfx.DetectTTYForOutput(output) },
	})
	for i := 0; i < 20; i++ {
		spinner.Tick()
		_, _ = io.WriteString(output, spinner.Render())
		sleep(100 * time.Millisecond)
	}
	_, _ = io.WriteString(output, "\nDone.\n")
}
