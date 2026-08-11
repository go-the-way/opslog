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

package contract

import "github.com/go-the-way/opslog/pkg/transport"

// Registry is the extension point for plugging inputs/decoders/processors/outputs/transports.
type Registry interface {
	RegisterInput(typ string, factory InputFactory)
	RegisterDecoder(typ string, factory DecoderFactory)
	RegisterProcessor(typ string, factory ProcessorFactory)
	RegisterOutput(typ OutputType, factory OutputFactory)
	RegisterTransport(typ transport.Type, factory transport.Factory)

	InputFactory(typ string) (InputFactory, bool)
	DecoderFactory(typ string) (DecoderFactory, bool)
	ProcessorFactory(typ string) (ProcessorFactory, bool)
	OutputFactory(typ OutputType) (OutputFactory, bool)
	TransportFactory(typ transport.Type) (transport.Factory, bool)

	BuildInput(typ, name string, cfg map[string]any) (Input, error)
	BuildDecoder(typ, name string, cfg map[string]any) (Decoder, error)
	BuildProcessor(typ, name string, cfg map[string]any) (Processor, error)
	BuildOutput(typ OutputType, name string, cfg map[string]any) (Output, error)
	BuildTransport(typ transport.Type, name string, cfg map[string]any) (transport.Transport, error)
}
