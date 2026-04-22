package capture

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
)

type MacOSCapture struct{}

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (MacOSCapture) Fullscreen(ctx context.Context, path string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("fullscreen capture currently uses macOS screencapture; running on %s", runtime.GOOS)
	}
	cmd := exec.CommandContext(ctx, "screencapture", "-x", path)
	return cmd.Run()
}

func (MacOSCapture) Rect(ctx context.Context, path string, rect Rect) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("rect capture currently uses macOS screencapture; running on %s", runtime.GOOS)
	}
	if rect.Width <= 0 || rect.Height <= 0 {
		return fmt.Errorf("capture rect width and height must be greater than zero")
	}
	arg := fmt.Sprintf("%d,%d,%d,%d", rect.X, rect.Y, rect.Width, rect.Height)
	cmd := exec.CommandContext(ctx, "screencapture", "-x", "-R", arg, path)
	return cmd.Run()
}

func (MacOSCapture) Window(ctx context.Context, path string, windowID uint32) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("window capture currently uses macOS screencapture; running on %s", runtime.GOOS)
	}
	if windowID == 0 {
		return fmt.Errorf("window id must be greater than zero")
	}
	cmd := exec.CommandContext(ctx, "screencapture", "-x", "-l", strconv.FormatUint(uint64(windowID), 10), path)
	return cmd.Run()
}

func (MacOSCapture) ResizeMaxDimension(ctx context.Context, path string, maxDimension int) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("image resize currently uses macOS sips; running on %s", runtime.GOOS)
	}
	if maxDimension <= 0 {
		return nil
	}
	cmd := exec.CommandContext(ctx, "sips", "-Z", strconv.Itoa(maxDimension), path)
	return cmd.Run()
}
