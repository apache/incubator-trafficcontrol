// Package crconfig provides logic and handlers for the CDN Snapshot-related
// API endpoints.
package crconfig

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
 */

import (
	"database/sql"
	"errors"
	"net/url"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

// Make creates and returns the CRConfig from the database.
func Make(tx *sql.Tx, cdn, user, toHost, toVersion string, useClientReqHost bool, emulateOldPath bool) (*tc.CRConfig, error) {
	crc := tc.CRConfig{}
	err := error(nil)

	cdnDomain, dnssecEnabled, err := getCDNInfo(cdn, tx)
	if err != nil {
		return nil, errors.New("Error getting CDN info: " + err.Error())
	}

	if crc.Config, err = makeCRConfigConfig(cdn, tx, dnssecEnabled, cdnDomain); err != nil {
		return nil, errors.New("Error getting Config: " + err.Error())
	}

	if crc.ContentServers, crc.ContentRouters, crc.Monitors, err = makeCRConfigServers(cdn, tx, cdnDomain); err != nil {
		return nil, errors.New("Error getting Servers: " + err.Error())
	}
	if crc.EdgeLocations, crc.RouterLocations, err = makeLocations(cdn, tx); err != nil {
		return nil, errors.New("Error getting Edge Locations: " + err.Error())
	}
	if crc.DeliveryServices, err = makeDSes(cdn, cdnDomain, tx); err != nil {
		return nil, errors.New("Error getting Delivery Services: " + err.Error())
	}
	if crc.Topologies, err = makeTopologies(tx); err != nil {
		return nil, errors.New("Error getting Topologies: " + err.Error())
	}

	if !useClientReqHost {
		paramTMURL, ok, err := getGlobalParam(tx, "tm.url")
		if err != nil {
			return nil, errors.New("getting global 'tm.url' parameter: " + err.Error())
		}
		if !ok {
			log.Warnln("Making CRConfig: no global tm.url parameter found! Using request host header instead!")
		}
		toHost = getTMURLHost(paramTMURL)
	}

	crc.Stats = makeStats(cdn, user, toHost, toVersion)
	if emulateOldPath {
		crc.Stats.TMPath = new(string)
		*crc.Stats.TMPath = "/tools/write_crconfig/" + cdn
	}
	return &crc, nil
}

// getTMURLHost returns the FQDN from a tm.url global parameter, which should be either an FQDN or a Hostname.
// If tmURL is a valid URL, the FQDN is returned.
// if tmURL is not a valid URL, it is returned with any leading 'http://' or 'http://' removed, and everything after the next '/' removed.
func getTMURLHost(tmURL string) string {
	if !strings.HasPrefix(tmURL, "http://") && !strings.HasPrefix(tmURL, "https://") {
		tmURL = "http://" + tmURL // if it doesn't begin with "http://", add it so it's a valid URL to parse
	}
	uri, err := url.Parse(tmURL)
	if err == nil {
		return uri.Host
	}

	// if it isn't a valid URL, do the best we can: strip the protocol and path
	tmURL = strings.TrimPrefix(tmURL, "https://")
	tmURL = strings.TrimPrefix(tmURL, "http://")
	pathStart := strings.Index(tmURL, "/")
	if pathStart == -1 {
		return tmURL
	}
	tmURL = tmURL[:pathStart]
	return tmURL
}
