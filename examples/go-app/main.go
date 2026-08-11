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

package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/agent"
	"github.com/go-the-way/opslog/sdk/collectors"
	"github.com/go-the-way/opslog/sdk/transport"
)

func main() {
	// If server configs/opslog.yml sets http.token, export OPSLOG_HTTP_TOKEN to match.
	httpTransport, err := transport.NewHTTPTransport("http", "http://127.0.0.1:8600/ingest", os.Getenv("OPSLOG_HTTP_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	a, err := agent.NewAgent(
		agent.WithService("demo-app"),
		agent.WithLevel("debug"),
		agent.WithTransport(httpTransport),
		agent.WithCollector(collectors.NewRuntime()),
		agent.WithCollector(collectors.NewHost()),
		agent.WithVersion("0.1.0"),
		agent.WithCollector(collectors.NewConfig("0.1.0")), // default env allow-list
		agent.WithCollector(collectors.NewProbe(collectors.Target{
			Name: "opslog-http", Target: "http://127.0.0.1:8600/api/health",
		})),
		agent.WithCollectInterval(10*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()

	if err := a.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	logger := a.Logger()
	logger.Info("demo app started", sdk.String("component", "main"))
	logger.Error("sample error", sdk.Int("code", 500))

	time.Sleep(2 * time.Second)
	_ = a.Sync()
}
