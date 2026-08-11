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

package registry

import (
	"fmt"
	"sync"

	"github.com/go-the-way/opslog/pkg/transport"
	"github.com/go-the-way/opslog/server/contract"
)

// Mem is an in-memory Registry implementation.
type Mem struct {
	mu         sync.RWMutex
	inputs     map[string]contract.InputFactory
	decoders   map[string]contract.DecoderFactory
	processors map[string]contract.ProcessorFactory
	outputs    map[contract.OutputType]contract.OutputFactory
	transports map[transport.Type]transport.Factory
}

func New() *Mem {
	return &Mem{
		inputs:     make(map[string]contract.InputFactory),
		decoders:   make(map[string]contract.DecoderFactory),
		processors: make(map[string]contract.ProcessorFactory),
		outputs:    make(map[contract.OutputType]contract.OutputFactory),
		transports: make(map[transport.Type]transport.Factory),
	}
}

func (r *Mem) RegisterInput(typ string, factory contract.InputFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inputs[typ] = factory
}

func (r *Mem) RegisterDecoder(typ string, factory contract.DecoderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decoders[typ] = factory
}

func (r *Mem) RegisterProcessor(typ string, factory contract.ProcessorFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.processors[typ] = factory
}

func (r *Mem) RegisterOutput(typ contract.OutputType, factory contract.OutputFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outputs[typ] = factory
}

func (r *Mem) RegisterTransport(typ transport.Type, factory transport.Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transports[typ] = factory
}

func (r *Mem) InputFactory(typ string) (contract.InputFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.inputs[typ]
	return f, ok
}

func (r *Mem) DecoderFactory(typ string) (contract.DecoderFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.decoders[typ]
	return f, ok
}

func (r *Mem) ProcessorFactory(typ string) (contract.ProcessorFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.processors[typ]
	return f, ok
}

func (r *Mem) OutputFactory(typ contract.OutputType) (contract.OutputFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.outputs[typ]
	return f, ok
}

func (r *Mem) TransportFactory(typ transport.Type) (transport.Factory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.transports[typ]
	return f, ok
}

func (r *Mem) BuildInput(typ, name string, cfg map[string]any) (contract.Input, error) {
	f, ok := r.InputFactory(typ)
	if !ok {
		return nil, fmt.Errorf("registry: input type %q not registered", typ)
	}
	return f(name, cfg)
}

func (r *Mem) BuildDecoder(typ, name string, cfg map[string]any) (contract.Decoder, error) {
	f, ok := r.DecoderFactory(typ)
	if !ok {
		return nil, fmt.Errorf("registry: decoder type %q not registered", typ)
	}
	return f(name, cfg)
}

func (r *Mem) BuildProcessor(typ, name string, cfg map[string]any) (contract.Processor, error) {
	f, ok := r.ProcessorFactory(typ)
	if !ok {
		return nil, fmt.Errorf("registry: processor type %q not registered", typ)
	}
	return f(name, cfg)
}

func (r *Mem) BuildOutput(typ contract.OutputType, name string, cfg map[string]any) (contract.Output, error) {
	f, ok := r.OutputFactory(typ)
	if !ok {
		return nil, fmt.Errorf("registry: output type %q is not registered (reserved or unimplemented)", typ)
	}
	return f(name, cfg)
}

func (r *Mem) BuildTransport(typ transport.Type, name string, cfg map[string]any) (transport.Transport, error) {
	f, ok := r.TransportFactory(typ)
	if !ok {
		return nil, fmt.Errorf("registry: transport type %q not registered", typ)
	}
	return f(name, cfg)
}

var _ contract.Registry = (*Mem)(nil)
