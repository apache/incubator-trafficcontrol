package atscfg

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
	"sort"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const ServerCacheDotConfigIncludeInactiveDSes = false

func makeCacheDotConfigMid(
	server *Server,
	deliveryServices []DeliveryService,
	opt *CacheDotConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &CacheDotConfigOpts{}
	}
	warnings := []string{}

	if server.HostName == nil {
		return Cfg{}, makeErr(warnings, "server missing HostName")
	}
	if !strings.HasPrefix(string(server.Type), tc.MidTypePrefix) {
		return Cfg{}, makeErr(warnings, "server cache.config generation called for non-Mid server, this is a code error and should never happen! Please file a bug.")
	}

	dses := map[tc.DeliveryServiceName]serverCacheConfigDS{}
	for _, ds := range deliveryServices {
		if ds.XMLID == nil || ds.Active == nil || ds.OrgServerFQDN == nil || ds.Type == nil {
			// TODO orgserverfqdn is nil for some DSes - MSO? Verify.
			continue
			//			return "", fmt.Errorf("getting delivery services: got DS with nil values! '%v' %v %+v\n", *ds.XMLID, *ds.ID, ds)
		}
		if !ServerCacheDotConfigIncludeInactiveDSes && !*ds.Active {
			continue
		}
		dses[tc.DeliveryServiceName(*ds.XMLID)] = serverCacheConfigDS{OrgServerFQDN: *ds.OrgServerFQDN, Type: *ds.Type}
	}

	text := makeHdrComment(opt.HdrComment)

	lines := []string{}

	seenOrigins := map[string]struct{}{}
	for _, ds := range dses {
		if ds.Type != tc.DSTypeHTTPNoCache {
			continue
		}
		if _, ok := seenOrigins[ds.OrgServerFQDN]; ok {
			continue
		}
		seenOrigins[ds.OrgServerFQDN] = struct{}{}

		originFQDN, originPort := getOriginFQDNAndPort(ds.OrgServerFQDN)
		if originPort != nil {
			lines = append(lines, `dest_domain=`+originFQDN+` port=`+strconv.Itoa(*originPort)+` scheme=http action=never-cache`+"\n")
		} else {
			lines = append(lines, `dest_domain=`+originFQDN+` scheme=http action=never-cache`+"\n")
		}
	}
	sort.Strings(lines)
	text += strings.Join(lines, "")

	return Cfg{
		Text:        text,
		ContentType: ContentTypeCacheDotConfig,
		LineComment: LineCommentCacheDotConfig,
		Warnings:    warnings,
	}, nil
}

// TODO unit test
func getOriginFQDNAndPort(origin string) (string, *int) {
	origin = strings.TrimSpace(origin)
	origin = strings.Replace(origin, `https://`, ``, -1)
	origin = strings.Replace(origin, `http://`, ``, -1)

	// if the origin includes a path, strip it

	slashI := strings.Index(origin, `/`)
	if slashI != -1 {
		origin = origin[:slashI]
	}

	hostName := origin

	colonI := strings.Index(origin, ":")
	if colonI == -1 {
		return hostName, nil // no :, the origin must not include a port
	}
	portStr := origin[colonI+1:]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		// either the port isn't an integer, or the : we found was something else.
		// Return the origin, as if it didn't contain a port.
		return hostName, nil
	}

	hostName = origin[:colonI]
	return hostName, &port
}

type serverCacheConfigDS struct {
	OrgServerFQDN string
	Type          tc.DSType
}
