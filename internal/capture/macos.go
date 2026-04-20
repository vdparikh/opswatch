package capture

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

type MacOSCapture struct{}

func (MacOSCapture) Fullscreen(ctx context.Context, path string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("fullscreen capture currently uses macOS screencapture; running on %s", runtime.GOOS)
	}
	cmd := exec.CommandContext(ctx, "screencapture", "-x", path)
	return cmd.Run()
}
