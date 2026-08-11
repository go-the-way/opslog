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

package bootstrap

import (
	"fmt"

	"github.com/go-the-way/opslog/pkg/transport"
	"github.com/go-the-way/opslog/pkg/transport/grpctx"
	"github.com/go-the-way/opslog/pkg/transport/httpx"
	"github.com/go-the-way/opslog/pkg/transport/tcp"
	"github.com/go-the-way/opslog/pkg/transport/udp"
	"github.com/go-the-way/opslog/pkg/transport/ws"
	"github.com/go-the-way/opslog/server/contract"
	"github.com/go-the-way/opslog/server/internal/decoder"
	"github.com/go-the-way/opslog/server/internal/output/clickhouse"
	"github.com/go-the-way/opslog/server/internal/output/filesystem"
	"github.com/go-the-way/opslog/server/internal/output/mysql"
	"github.com/go-the-way/opslog/server/internal/processor"
	"github.com/go-the-way/opslog/server/internal/registry"
)

func NewRegistry() *registry.Mem {
	r := registry.New()
	r.RegisterDecoder("json", decoder.NewJSON)
	r.RegisterDecoder("plain", decoder.NewPlain)
	r.RegisterProcessor("normalize", processor.NewNormalize)
	r.RegisterOutput(contract.OutputFilesystem, filesystem.New)
	r.RegisterOutput(contract.OutputMySQL, mysql.New)
	r.RegisterOutput(contract.OutputClickHouse, clickhouse.New)
	r.RegisterOutput(contract.OutputKafka, reservedOutput("kafka"))
	r.RegisterOutput(contract.OutputElasticsearch, reservedOutput("elasticsearch"))
	r.RegisterTransport(transport.TypeUDP, udp.New)
	r.RegisterTransport(transport.TypeTCP, tcp.New)
	r.RegisterTransport(transport.TypeHTTP, httpx.New)
	r.RegisterTransport(transport.TypeWebSocket, ws.New)
	r.RegisterTransport(transport.TypeGRPC, grpctx.New)
	return r
}

func reservedOutput(typ string) contract.OutputFactory {
	return func(name string, cfg map[string]any) (contract.Output, error) {
		return nil, fmt.Errorf("output type %q is reserved and not implemented yet", typ)
	}
}
