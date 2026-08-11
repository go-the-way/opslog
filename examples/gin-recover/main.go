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

// Demo: Gin panic recovery → OpsLog HTTP ingest.
//
// Prerequisites: start OpsLog server (configs/opslog.yml).
// If http.token is set there (default demo value "change-me"), pass the same token:
//
//	OPSLOG_HTTP_TOKEN=change-me go run ./examples/gin-recover
//	# or: go run ./examples/gin-recover -token change-me
//
// Then trigger a panic:
//
//	curl http://127.0.0.1:8088/panic
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/agent"
	"github.com/go-the-way/opslog/sdk/middleware/opslog4gin"
	"github.com/go-the-way/opslog/sdk/middleware/opslog4proc"
	"github.com/go-the-way/opslog/sdk/middleware/panicopt"
)

func main() {
	endpoint := flag.String("endpoint", envOr("OPSLOG_HTTP_URL", "http://127.0.0.1:8600/ingest"), "OpsLog HTTP ingest URL")
	token := flag.String("token", strings.TrimSpace(os.Getenv("OPSLOG_HTTP_TOKEN")), "Bearer token for ingest (OPSLOG_HTTP_TOKEN); must match server http.token")
	listen := flag.String("listen", ":8088", "demo HTTP listen address")
	flag.Parse()

	opts := []agent.Option{
		agent.WithService("gin-recover-demo"),
		agent.WithEndpoint(*endpoint),
	}
	if *token != "" {
		opts = append(opts, agent.WithToken(*token))
	}
	a, err := agent.NewAgent(opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()

	log.Printf("starting OpsLog agent endpoint=%s token=%v", *endpoint, *token != "")
	if err := a.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer a.Sync()
	log.Printf("OpsLog agent ready (stderr: http transport connecting/connected, agent started, send ok/failed)")

	logger := a.Logger()
	r := gin.New()
	r.Use(opslog4gin.Recovery(logger, panicopt.WithAttrs(sdk.String("component", "http"))))

	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/panic", func(c *gin.Context) {
		panic("demo panic")
	})
	r.GET("/worker", func(c *gin.Context) {
		go func() {
			// Closure form: Guard returns func() for defer (same as Recover).
			defer opslog4proc.Guard(logger, panicopt.WithAttrs(sdk.String("goroutine", "worker")))()
			panic("background panic")
		}()
		c.JSON(http.StatusAccepted, gin.H{"started": true})
	})

	hint := *listen
	if strings.HasPrefix(hint, ":") {
		hint = "127.0.0.1" + hint
	}
	log.Printf("listening on http://%s", hint)
	log.Printf("try: curl http://%s/panic", hint)
	if err := r.Run(*listen); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
