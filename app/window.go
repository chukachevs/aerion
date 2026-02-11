package app

import (
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// RefreshWindowConstraints re-evaluates window max size constraints based on
// the current monitor. This works around a Wails v2 Linux limitation where
// geometry hints are set once at startup using the initial monitor's dimensions,
// causing the window to be stuck at that size when moving to a larger monitor.
func (a *App) RefreshWindowConstraints() {
	wailsRuntime.WindowSetMaxSize(a.ctx, 0, 0)
}

// RefreshWindowConstraints re-evaluates window max size constraints for the
// composer window. See App.RefreshWindowConstraints for details.
func (c *ComposerApp) RefreshWindowConstraints() {
	wailsRuntime.WindowSetMaxSize(c.ctx, 0, 0)
}
