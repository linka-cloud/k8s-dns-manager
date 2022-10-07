/*
Copyright 2020 The Linka Cloud Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package libdns

import (
	"context"
	"sync"

	"github.com/libdns/libdns"
)

var _ Client = (*syncClient)(nil)

type syncClient struct {
	mu sync.RWMutex
	c  Client
}

func (s *syncClient) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.c.GetRecords(ctx, zone)
}

func (s *syncClient) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.c.AppendRecords(ctx, zone, recs)
}

func (s *syncClient) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.c.DeleteRecords(ctx, zone, recs)
}
