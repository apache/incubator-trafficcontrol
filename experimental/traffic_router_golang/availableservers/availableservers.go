package availableservers

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
	"errors"
	"fmt"
	"github.com/apache/trafficcontrol/lib/go-tc/tce"
	"sync"
)

type AvailableServersMap map[tce.DeliveryServiceName]map[tce.CacheGroupName][]tce.CacheName

// AvailableServers provides access to the currently available servers, by Delivery Service and Cache Group. This is safe for access by multiple goroutines.
type AvailableServers struct {
	p **AvailableServersMap
	m *sync.RWMutex
}

func New() AvailableServers {
	mp := &AvailableServersMap{}
	return AvailableServers{p: &mp, m: &sync.RWMutex{}}
}

func (a *AvailableServers) Get(ds tce.DeliveryServiceName, cg tce.CacheGroupName) ([]tce.CacheName, error) {
	a.m.RLock()
	s := *a.p
	a.m.RUnlock()

	cgs, ok := (*s)[ds]
	if !ok {
		return nil, errors.New("deliveryservice not found")
	}
	cs, ok := cgs[cg]
	if !ok {
		return nil, errors.New("cachegroup not found")
	}
	return cs, nil
}

func (a *AvailableServers) Set(m AvailableServersMap) {
	a.m.Lock()
	defer a.m.Unlock()
	*a.p = &m
}

// TODO put in _test.go file
func Test() {
	a := New()

	as := map[tce.DeliveryServiceName]map[tce.CacheGroupName][]tce.CacheName{}
	as[tce.DeliveryServiceName("dsOne")] = map[tce.CacheGroupName][]tce.CacheName{}

	cs := as[tce.DeliveryServiceName("dsOne")]
	cs[tce.CacheGroupName("cgOne")] = []tce.CacheName{"cacheOne", "cacheTwo"}

	fmt.Printf("testAvailableServers as %+v\n", as)

	a.Set(as)

	newCs, err := a.Get(tce.DeliveryServiceName("dsOne"), tce.CacheGroupName("cgOne"))

	if err != nil {
		fmt.Println("testAvailableServers err ", err.Error())
	}
	fmt.Println("testAvailableServers caches ", newCs)
}
