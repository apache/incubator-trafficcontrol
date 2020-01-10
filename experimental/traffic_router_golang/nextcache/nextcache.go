package nextcache

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 */

import (
	"github.com/apache/trafficcontrol/lib/go-tc/enum"
	"sync/atomic"
)

// New creates and returns a new NextCacher. The returned NextCacher is safe for use by multiple goroutines.
func New(dses []enum.DeliveryServiceName) NextCacher {
	m := make(map[enum.DeliveryServiceName]*uint64, len(dses))
	for _, ds := range dses {
		i := uint64(0)
		m[ds] = &i
	}
	return nextCacher(m)
}

// NextCacher is the interface that wraps the NextCache method.
//
// NextCache returns the next cache to use for the given delivery service. This is neither pure nor idempotent, and successive calls will return different numbers. The underlying mechanism may not be aware of the number of caches, and the returned number MAY exceed the number of caches. Typically, callers should mod the returned number by the size of their cache list, to determine the cache to use. Returns false if the given delivery service is not found.
type NextCacher interface {
	NextCache(enum.DeliveryServiceName) (uint64, bool)
}

type nextCacher map[enum.DeliveryServiceName]*uint64

func (c nextCacher) NextCache(ds enum.DeliveryServiceName) (uint64, bool) {
	m := (map[enum.DeliveryServiceName]*uint64)(c)
	i, ok := m[ds]
	if !ok {
		return 0, false
	}
	return atomic.AddUint64(i, 1), true
}
