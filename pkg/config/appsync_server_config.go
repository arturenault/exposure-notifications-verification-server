// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/google/exposure-notifications-verification-server/pkg/database"

	"github.com/google/exposure-notifications-server/pkg/observability"

	"github.com/sethvargo/go-envconfig"
)

// AppSyncConfig represents the environment based configuration for the app sync server.
type AppSyncConfig struct {
	Database      database.Config
	Observability observability.Config

	// DevMode produces additional debugging information. Do not enable in
	// production environments.
	DevMode bool `env:"DEV_MODE"`

	Port string `env:"PORT,default=8080"`

	RateLimit uint64 `env:"RATE_LIMIT,default=60"`

	// AppSync config
	AppSyncURL string `env:"APP_SYNC_URL, default=https://www.gstatic.com/exposurenotifications/apps.json"`
}

// NewAppSyncConfig returns the environment config for the appsync server.
// Only needs to be called once per instance, but may be called multiple times.
func NewAppSyncConfig(ctx context.Context) (*AppSyncConfig, error) {
	var config AppSyncConfig
	if err := ProcessWith(ctx, &config, envconfig.OsLookuper()); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *AppSyncConfig) Validate() error {
	if url, err := url.Parse(c.AppSyncURL); err != nil {
		return fmt.Errorf("unable to parse APP_SYNC_URL: %v", err)
	} else if url == nil {
		return errors.New("expecting a value for APP_SYNC_URL")
	}

	return nil
}

func (c *AppSyncConfig) ObservabilityExporterConfig() *observability.Config {
	return &c.Observability
}