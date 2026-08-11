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

// Package opslog is the OpsLog root module.
//
// OpsLog is a pluggable log/ops telemetry platform:
// collect logs, host config, resource metrics and connectivity probes,
// ship them via selectable transports, and write to filesystem/MySQL/ClickHouse.
//
// Module layout:
//
//	pkg/signal      shared Signal model (log/metric/config/probe)
//	pkg/transport   client Transport interface (udp/tcp/http/websocket/grpc)
//	pkg/query       query/archive shared types
//	sdk             embeddable Go Agent/Logger/Collector interfaces
//	server/contract server Input/Decoder/Dispatcher/Processor/Output/Registry
//
// Design docs: see README.md and the docs/ directory (English default; Chinese as *_zh.md).
//
// Implementations will live under server/internal and sdk/internal later.
package opslog
