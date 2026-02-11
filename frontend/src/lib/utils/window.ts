/**
 * Monitors for display configuration changes (e.g. plugging in an external monitor)
 * and calls the provided callback when the screen dimensions change.
 *
 * Wails v2 on Linux sets GTK geometry hints at startup based on the initial monitor,
 * which locks the window's max size. This allows callers to re-evaluate constraints
 * when the display configuration changes.
 *
 * @param onScreenChange - Callback invoked when screen dimensions change
 * @returns Cleanup function to stop monitoring
 */
export function monitorScreenChanges(onScreenChange: () => void): () => void {
  let lastWidth = window.screen.width
  let lastHeight = window.screen.height

  const interval = setInterval(() => {
    const w = window.screen.width
    const h = window.screen.height
    if (w !== lastWidth || h !== lastHeight) {
      lastWidth = w
      lastHeight = h
      onScreenChange()
    }
  }, 2000)

  return () => clearInterval(interval)
}
