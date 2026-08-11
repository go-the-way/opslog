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

package input

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/server/contract"
	disp "github.com/go-the-way/opslog/server/internal/dispatcher"
)

type UDP struct {
	name     string
	listen   string
	decoder  contract.Decoder
	conn     *net.UDPConn
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

func NewUDP(name string, cfg map[string]any, decoder contract.Decoder) (contract.Input, error) {
	listen := cfgutil.String(cfg, "listen", ":8140")
	if name == "" {
		name = "udp"
	}
	if decoder == nil {
		return nil, fmt.Errorf("udp input: decoder required")
	}
	return &UDP{name: name, listen: listen, decoder: decoder}, nil
}

func (u *UDP) Name() string { return u.name }
func (u *UDP) Type() string { return "udp" }

func (u *UDP) Start(ctx context.Context, dispatcher contract.Dispatcher) error {
	addr, err := net.ResolveUDPAddr("udp", u.listen)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	u.conn = conn
	cctx, cancel := context.WithCancel(ctx)
	u.cancel = cancel
	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		buf := make([]byte, 65535)
		for {
			select {
			case <-cctx.Done():
				return
			default:
			}
			_ = conn.SetReadDeadline(deadlineSoon())
			n, remote, err := conn.ReadFromUDP(buf)
			if err != nil {
				if cctx.Err() != nil {
					return
				}
				continue
			}
			payload := append([]byte(nil), buf[:n]...)
			remoteAddr := remote.String()
			sig, err := u.decoder.Decode(remoteAddr, payload)
			if err != nil {
				continue
			}
			_ = dispatcher.Dispatch(disp.WithMeta(cctx, u.name, remoteAddr), sig)
		}
	}()
	return nil
}

func (u *UDP) Stop(context.Context) error {
	if u.cancel != nil {
		u.cancel()
	}
	if u.conn != nil {
		_ = u.conn.Close()
	}
	u.wg.Wait()
	return nil
}
