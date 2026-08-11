// Copyright 2026 opslog Author. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//      http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"time"

	"github.com/go-the-way/opslog/server/internal/app"
)

// Run loads config and runs OpsLog server until ctx is cancelled.
func Run(ctx context.Context, configPath string) error {
	application, err := app.New(configPath)
	if err != nil {
		return err
	}
	if err := application.Start(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return application.Stop(stopCtx)
}
