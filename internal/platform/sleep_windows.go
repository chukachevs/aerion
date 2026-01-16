//go:build windows

package platform

import (
	"context"

	"github.com/hkdb/aerion/internal/logging"
)

// WindowsSleepWakeMonitor monitors sleep/wake events on Windows
// TODO: Implement using RegisterPowerSettingNotification or WM_POWERBROADCAST
type WindowsSleepWakeMonitor struct {
	events   chan SleepWakeEvent
	stopChan chan struct{}
	running  bool
}

// NewSleepWakeMonitor creates a new sleep/wake monitor for Windows
func NewSleepWakeMonitor() SleepWakeMonitor {
	return &WindowsSleepWakeMonitor{
		events:   make(chan SleepWakeEvent, 10),
		stopChan: make(chan struct{}),
	}
}

// Start begins monitoring for sleep/wake events
// TODO: Implement using Win32 Power API
func (m *WindowsSleepWakeMonitor) Start(ctx context.Context) error {
	log := logging.WithComponent("sleep-wake")

	if m.running {
		return nil
	}

	m.running = true

	// TODO: Implement Windows sleep/wake detection using:
	// Option 1: RegisterPowerSettingNotification with GUID_MONITOR_POWER_ON
	// Option 2: Create hidden window to receive WM_POWERBROADCAST messages
	// For now, this is a stub that doesn't detect sleep/wake events

	log.Info().Msg("Sleep/wake monitor started (Windows stub - not implemented)")
	return nil
}

// Events returns the channel for receiving sleep/wake events
func (m *WindowsSleepWakeMonitor) Events() <-chan SleepWakeEvent {
	return m.events
}

// Stop stops the monitor and cleans up resources
func (m *WindowsSleepWakeMonitor) Stop() error {
	log := logging.WithComponent("sleep-wake")

	if !m.running {
		return nil
	}

	m.running = false
	close(m.stopChan)

	log.Info().Msg("Sleep/wake monitor stopped (Windows)")
	return nil
}
