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

const HostingConfigFileName = `hosting.config`
const HostingConfigParamConfigFile = `storage.config`
const ContentTypeHostingDotConfig = ContentTypeTextASCII
const LineCommentHostingDotConfig = LineCommentHash

const ParamDrivePrefix = "Drive_Prefix"
const ParamRAMDrivePrefix = "RAM_Drive_Prefix"

const ServerHostingDotConfigMidIncludeInactive = false
const ServerHostingDotConfigEdgeIncludeInactive = true

// HostingDotConfigOpts contains settings to configure generation options.
type HostingDotConfigOpts struct {
	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeHostingDotConfig(
	server *Server,
	servers []Server,
	serverParams []tc.Parameter,
	deliveryServices []DeliveryService,
	deliveryServiceServers []DeliveryServiceServer,
	topologies []tc.Topology,
	opt *HostingDotConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &HostingDotConfigOpts{}
	}
	warnings := []string{}

	if server.CDNID == nil {
		return Cfg{}, makeErr(warnings, "this server missing CDNID")
	}
	if server.HostName == nil || *server.HostName == "" {
		return Cfg{}, makeErr(warnings, "server had no host name!")
	}
	if server.ID == nil {
		return Cfg{}, makeErr(warnings, "this server missing ID")
	}

	params, paramWarns := paramsToMap(filterParams(serverParams, HostingConfigParamConfigFile, "", "", ""))
	warnings = append(warnings, paramWarns...)

	cdnServers := map[tc.CacheName]Server{}
	for _, sv := range servers {
		if sv.HostName == nil {
			warnings = append(warnings, "TO Servers had server missing HostName, skipping!")
			continue
		} else if sv.CDNID == nil {
			warnings = append(warnings, "TO Servers had server missing CDNID, skipping!")
			continue
		}
		if *sv.CDNID != *server.CDNID {
			continue
		}
		cdnServers[tc.CacheName(*sv.HostName)] = sv
	}

	serverIDs := map[int]struct{}{}
	for _, sv := range cdnServers {
		if sv.CDNID == nil {
			warnings = append(warnings, "TO Servers had server missing CDNID, skipping!")
			continue
		}
		serverIDs[*sv.ID] = struct{}{}
	}

	dsIDs := map[int]struct{}{}
	for _, ds := range deliveryServices {
		if ds.ID != nil {
			dsIDs[*ds.ID] = struct{}{}
		}
	}

	dsServers := filterDSS(deliveryServiceServers, dsIDs, serverIDs)

	dsServerMap := map[int]map[int]struct{}{} // set[dsID][serverID]
	for _, dss := range dsServers {
		if _, ok := dsServerMap[dss.DeliveryService]; !ok {
			dsServerMap[dss.DeliveryService] = map[int]struct{}{}
		}
		dsServerMap[dss.DeliveryService][dss.Server] = struct{}{}
	}

	isMid := strings.HasPrefix(server.Type, tc.MidTypePrefix)

	filteredDSes := []DeliveryService{}
	for _, ds := range deliveryServices {
		if ds.Active == nil || ds.Type == nil || ds.XMLID == nil || ds.CDNID == nil || ds.ID == nil || ds.OrgServerFQDN == nil {
			// some DSes have nil origins. I think MSO? TODO: verify
			continue
		}
		if *ds.CDNID != *server.CDNID {
			continue
		}
		if ds.Topology == nil {
			if !*ds.Active && ((!isMid && !ServerHostingDotConfigEdgeIncludeInactive) || (isMid && !ServerHostingDotConfigMidIncludeInactive)) {
				continue
			}

			if isMid {
				if !strings.HasSuffix(string(*ds.Type), tc.DSTypeLiveNationalSuffix) {
					continue
				}

				// mids: include all DSes with at least one server assigned
				if len(dsServerMap[*ds.ID]) == 0 {
					continue
				}
			} else {
				if !strings.HasSuffix(string(*ds.Type), tc.DSTypeLiveNationalSuffix) && !strings.HasSuffix(string(*ds.Type), tc.DSTypeLiveSuffix) {
					continue
				}

				// edges: only include DSes assigned to this edge
				if dsServerMap[*ds.ID] == nil {
					continue
				}

				if _, ok := dsServerMap[*ds.ID][*server.ID]; !ok {
					continue
				}
			}
		}

		filteredDSes = append(filteredDSes, ds)
	}

	text := makeHdrComment(opt.HdrComment)

	nameTopologies := makeTopologyNameMap(topologies)

	lines := []string{}
	if _, ok := params[ParamRAMDrivePrefix]; ok {
		nextVolume := 1
		if _, ok := params[ParamDrivePrefix]; ok {
			diskVolume := nextVolume
			text += `# TRAFFIC OPS NOTE: volume ` + strconv.Itoa(diskVolume) + ` is the Disk volume` + "\n"
			nextVolume++
		}
		ramVolume := nextVolume
		text += `# TRAFFIC OPS NOTE: volume ` + strconv.Itoa(ramVolume) + ` is the RAM volume` + "\n"

		seenOrigins := map[string]struct{}{}
		for _, ds := range filteredDSes {
			if ds.OrgServerFQDN == nil || ds.XMLID == nil || ds.Active == nil {
				warnings = append(warnings, "got DS with nil values, skipping!")
				continue
			}

			origin := *ds.OrgServerFQDN
			if _, ok := seenOrigins[origin]; ok {
				continue
			}

			if ds.Topology != nil && *ds.Topology != "" {
				topology, hasTopology := nameTopologies[TopologyName(*ds.Topology)]
				if hasTopology {
					topoHasServer, err := topologyIncludesServerNullable(topology, server)
					if err != nil {
						warnings = append(warnings, "checking if topology has server, skipping! : "+err.Error())
						topoHasServer = false
					}
					if !topoHasServer {
						continue
					}
				}
			}

			seenOrigins[origin] = struct{}{}
			origin = strings.TrimPrefix(origin, `http://`)
			origin = strings.TrimPrefix(origin, `https://`)
			lines = append(lines, `hostname=`+origin+` volume=`+strconv.Itoa(ramVolume)+"\n")
		}
	}
	diskVolume := 1 // note this will actually be the RAM (RAM_Drive_Prefix) volume if there is no Drive_Prefix parameter.

	lines = append(lines, `hostname=*   volume=`+strconv.Itoa(diskVolume)+"\n")

	sort.Strings(lines)
	text += strings.Join(lines, "")

	return Cfg{
		Text:        text,
		ContentType: ContentTypeHostingDotConfig,
		LineComment: LineCommentHostingDotConfig,
		Warnings:    warnings,
	}, nil
}
