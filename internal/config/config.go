// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config // import "miniflux.app/v2/internal/config"

import (
	"slices"
	"sync"

	"miniflux.app/v2/internal/version"
)

// Opts holds parsed configuration options.
var Opts *configOptions

var defaultHTTPClientUserAgent = "Mozilla/5.0 (compatible; Miniflux/" + version.Version + "; +https://miniflux.app)"

var (
	reloadMu        sync.Mutex
	configFilePath  string
	changeCallbacks []func()
)

// SetConfigFilePath records the configuration file path used at startup so
// that Reload can re-parse it later.
func SetConfigFilePath(path string) {
	reloadMu.Lock()
	defer reloadMu.Unlock()
	configFilePath = path
}

// OnConfigChange registers a callback executed after a successful
// configuration reload. Callbacks are invoked in registration order.
func OnConfigChange(callback func()) {
	reloadMu.Lock()
	defer reloadMu.Unlock()
	changeCallbacks = append(changeCallbacks, callback)
}

// Reload re-parses the configuration file (if any) and the environment
// variables, validates the result, atomically swaps Opts and notifies the
// registered change callbacks. The previous configuration is kept if the
// new one is invalid.
func Reload() error {
	reloadMu.Lock()

	parser := NewConfigParser()

	var opts *configOptions
	var err error

	if configFilePath != "" {
		if opts, err = parser.ParseFile(configFilePath); err != nil {
			reloadMu.Unlock()
			return err
		}
	}

	if opts, err = parser.ParseEnvironmentVariables(); err != nil {
		reloadMu.Unlock()
		return err
	}

	if err := opts.Validate(); err != nil {
		reloadMu.Unlock()
		return err
	}

	Opts = opts
	callbacks := slices.Clone(changeCallbacks)
	reloadMu.Unlock()

	for _, callback := range callbacks {
		callback()
	}

	return nil
}
