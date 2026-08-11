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

package opslog4gin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/middleware/internal/panicx"
	"github.com/go-the-way/opslog/sdk/middleware/panicopt"
)

// Response configures the HTTP error body written after a panic is reported.
// Zero value means: status 500, JSON {"error":"internal server error"}.
type Response struct {
	Status    int    // 0 => 500
	PlainText bool   // false => JSON
	Message   string // empty => "internal server error"
}

func (r Response) normalized() Response {
	if r.Status == 0 {
		r.Status = http.StatusInternalServerError
	}
	if r.Message == "" {
		r.Message = "internal server error"
	}
	return r
}

// Recovery returns Gin middleware that recovers from panics, reports them through
// an OpsLog Logger (shared panicx path), and responds with HTTP 500 JSON.
//
//	r.Use(opslog4gin.Recovery(logger))
//	r.Use(opslog4gin.Recovery(logger, panicopt.WithAttrs(sdk.String("component", "http"))))
//
// Customize the HTTP response with RecoveryResponse.
func Recovery(logger sdk.Logger, opts ...panicopt.Option) gin.HandlerFunc {
	return RecoveryResponse(logger, Response{}, opts...)
}

// RecoveryResponse is like Recovery but customizes the HTTP error response.
//
//	r.Use(opslog4gin.RecoveryResponse(logger, opslog4gin.Response{PlainText: true}))
//	r.Use(opslog4gin.RecoveryResponse(logger, opslog4gin.Response{Status: 503, Message: "unavailable"},
//	    panicopt.WithAttrs(sdk.String("component", "http"))))
func RecoveryResponse(logger sdk.Logger, resp Response, opts ...panicopt.Option) gin.HandlerFunc {
	cfg := resp.normalized()
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				path := c.FullPath()
				if path == "" && c.Request.URL != nil {
					path = c.Request.URL.Path
				}
				url := ""
				if c.Request.URL != nil {
					url = c.Request.URL.String()
				}
				method := ""
				if c.Request != nil {
					method = c.Request.Method
				}
				reportOpts := append([]panicopt.Option{}, opts...)
				reportOpts = append(reportOpts,
					panicopt.WithMessage("gin panic recovered"),
					panicopt.WithAttrs(
						sdk.String("http.method", method),
						sdk.String("http.path", path),
						sdk.String("http.url", url),
					),
				)
				panicx.ReportPanic(logger, r, reportOpts...)

				if c.Writer.Written() {
					c.Abort()
					return
				}
				if !cfg.PlainText {
					c.AbortWithStatusJSON(cfg.Status, gin.H{"error": cfg.Message})
					return
				}
				c.Header("Content-Type", "text/plain; charset=utf-8")
				c.AbortWithStatus(cfg.Status)
				_, _ = c.Writer.Write([]byte(cfg.Message))
			}
		}()
		c.Next()
	}
}
