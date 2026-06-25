package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Copy copies text to the system clipboard.
func Copy(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel, then wl-copy (Wayland).
		if path, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command(path, "-selection", "clipboard")
		} else if path, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command(path, "--clipboard", "--input")
		} else if path, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command(path)
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// Clear clears the system clipboard by writing an empty string.
func Clear() error {
	return Copy("")
}

// CopyWithTimeout copies text to the clipboard and clears it after the given duration.
// It runs the clearing in a goroutine and returns immediately.
// The done channel is closed when the clipboard has been cleared.
func CopyWithTimeout(text string, timeout time.Duration) (done chan struct{}, err error) {
	if err := Copy(text); err != nil {
		return nil, fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	done = make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(timeout)
		Clear()
	}()

	return done, nil
}
