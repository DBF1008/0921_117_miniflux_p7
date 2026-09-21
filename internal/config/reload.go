// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

import (
	"log/slog"
	"sync"
)

var reloadState = struct {
	sync.RWMutex
	callbacks []func()
}{}

// OnReload registers a callback that is invoked each time the configuration
// is reloaded at runtime (see Reload). Callbacks are executed in the order
// they were registered, after Opts has been swapped to the new configuration.
func OnReload(callback func()) {
	reloadState.Lock()
	defer reloadState.Unlock()
	reloadState.callbacks = append(reloadState.callbacks, callback)
}

// Reload re-parses the configuration from the environment variables, swaps
// the global Opts atomically, and notifies all registered reload callbacks.
//
// It allows runtime configuration changes (e.g. DATABASE_MAX_CONNS) to take
// effect without restarting the process. If parsing or validation fails, the
// current configuration is left untouched and an error is returned.
func Reload() error {
	parser := NewConfigParser()

	newOpts, err := parser.ParseEnvironmentVariables()
	if err != nil {
		return err
	}

	if err := newOpts.Validate(); err != nil {
		return err
	}

	reloadState.Lock()
	Opts = newOpts
	callbacks := make([]func(), len(reloadState.callbacks))
	copy(callbacks, reloadState.callbacks)
	reloadState.Unlock()

	slog.Info("Configuration reloaded, notifying subscribers", slog.Int("subscribers", len(callbacks)))

	for _, callback := range callbacks {
		callback()
	}

	return nil
}
