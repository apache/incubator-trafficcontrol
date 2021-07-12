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
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
)

const ContentTypeParentDotConfig = ContentTypeTextASCII
const LineCommentParentDotConfig = LineCommentHash

const ParentConfigFileName = "parent.config"

const ParentConfigParamQStringHandling = "psel.qstring_handling"
const ParentConfigParamMSOAlgorithm = "mso.algorithm"
const ParentConfigParamMSOParentRetry = "mso.parent_retry"
const ParentConfigParamMSOUnavailableServerRetryResponses = "mso.unavailable_server_retry_responses"
const ParentConfigParamMSOMaxSimpleRetries = "mso.max_simple_retries"
const ParentConfigParamMSOMaxUnavailableServerRetries = "mso.max_unavailable_server_retries"
const ParentConfigParamAlgorithm = "algorithm"
const ParentConfigParamQString = "qstring"
const ParentConfigParamSecondaryMode = "try_all_primaries_before_secondary"

const ParentConfigParamParentRetry = "parent_retry"
const ParentConfigParamUnavailableServerRetryResponses = "unavailable_server_retry_responses"
const ParentConfigParamMaxSimpleRetries = "max_simple_retries"
const ParentConfigParamMaxUnavailableServerRetries = "max_unavailable_server_retries"

const ParentConfigDSParamDefaultMSOAlgorithm = "consistent_hash"
const ParentConfigDSParamDefaultMSOParentRetry = "both"
const ParentConfigDSParamDefaultMSOUnavailableServerRetryResponses = ""
const ParentConfigDSParamDefaultMaxSimpleRetries = "1"
const ParentConfigDSParamDefaultMaxUnavailableServerRetries = "1"

const ParentConfigCacheParamWeight = "weight"
const ParentConfigCacheParamPort = "port"
const ParentConfigCacheParamUseIP = "use_ip_address"
const ParentConfigCacheParamRank = "rank"
const ParentConfigCacheParamNotAParent = "not_a_parent"

type OriginHost string
type OriginFQDN string

// ParentConfigOpts contains settings to configure parent.config generation options.
type ParentConfigOpts struct {
	// AddComments is whether to add informative comments to the generated file, about what was generated and why.
	// Note this does not include the header comment, which is configured separately with HdrComment.
	// These comments are human-readable and not guarnateed to be consistent between versions. Automating anything based on them is strongly discouraged.
	AddComments bool

	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeParentDotConfig(
	dses []DeliveryService,
	server *Server,
	servers []Server,
	topologies []tc.Topology,
	tcServerParams []tc.Parameter,
	tcParentConfigParams []tc.Parameter,
	serverCapabilities map[int]map[ServerCapability]struct{},
	dsRequiredCapabilities map[int]map[ServerCapability]struct{},
	cacheGroupArr []tc.CacheGroupNullable,
	dss []DeliveryServiceServer,
	cdn *tc.CDN,
	opt *ParentConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &ParentConfigOpts{}
	}
	warnings := []string{}

	if server.HostName == nil || *server.HostName == "" {
		return Cfg{}, makeErr(warnings, "server HostName missing")
	} else if server.CDNName == nil || *server.CDNName == "" {
		return Cfg{}, makeErr(warnings, "server CDNName missing")
	} else if server.Cachegroup == nil || *server.Cachegroup == "" {
		return Cfg{}, makeErr(warnings, "server Cachegroup missing")
	} else if server.Profile == nil || *server.Profile == "" {
		return Cfg{}, makeErr(warnings, "server Profile missing")
	} else if server.TCPPort == nil {
		return Cfg{}, makeErr(warnings, "server TCPPort missing")
	}

	atsMajorVer, verWarns := getATSMajorVersion(tcServerParams)
	warnings = append(warnings, verWarns...)

	cacheGroups, err := makeCGMap(cacheGroupArr)
	if err != nil {
		return Cfg{}, makeErr(warnings, "making CacheGroup map: "+err.Error())
	}
	serverParentCGData, err := getParentCacheGroupData(server, cacheGroups)
	if err != nil {
		return Cfg{}, makeErr(warnings, "getting server parent cachegroup data: "+err.Error())
	}
	cacheIsTopLevel := isTopLevelCache(serverParentCGData)
	serverCDNDomain := cdn.DomainName

	sort.Sort(dsesSortByName(dses))

	hdr := ""
	if opt.HdrComment != "" {
		hdr = makeHdrComment(opt.HdrComment)
	}

	textArr := []string{}
	processedOriginsToDSNames := map[string]tc.DeliveryServiceName{}

	parentConfigParamsWithProfiles, err := tcParamsToParamsWithProfiles(tcParentConfigParams)
	if err != nil {
		warnings = append(warnings, "error getting profiles from Traffic Ops Parameters, Parameters will not be considered for generation! : "+err.Error())
		parentConfigParamsWithProfiles = []parameterWithProfiles{}
	}
	parentConfigParams := parameterWithProfilesToMap(parentConfigParamsWithProfiles)

	// this is an optimization, to avoid looping over all params, for every DS. Instead, we loop over all params only once, and put them in a profile map.
	profileParentConfigParams := map[string]map[string]string{} // map[profileName][paramName]paramVal
	for _, param := range parentConfigParamsWithProfiles {
		for _, profile := range param.ProfileNames {
			if _, ok := profileParentConfigParams[profile]; !ok {
				profileParentConfigParams[profile] = map[string]string{}
			}
			profileParentConfigParams[profile][param.Name] = param.Value
		}
	}

	// We only need parent.config params, don't need all the params on the server
	serverParams := map[string]string{}
	if server.Profile == nil || *server.Profile != "" { // TODO warn/error if false? Servers requires profiles
		for name, val := range profileParentConfigParams[*server.Profile] {
			if name == ParentConfigParamQStringHandling ||
				name == ParentConfigParamAlgorithm ||
				name == ParentConfigParamQString {
				serverParams[name] = val
			}
		}
	}

	parentCacheGroups := map[string]struct{}{}
	if cacheIsTopLevel {
		for _, cg := range cacheGroups {
			if cg.Type == nil {
				return Cfg{}, makeErr(warnings, "cachegroup type is nil!")
			}
			if cg.Name == nil {
				return Cfg{}, makeErr(warnings, "cachegroup name is nil!")
			}

			if *cg.Type != tc.CacheGroupOriginTypeName {
				continue
			}
			parentCacheGroups[*cg.Name] = struct{}{}
		}
	} else {
		for _, cg := range cacheGroups {
			if cg.Type == nil {
				return Cfg{}, makeErr(warnings, "cachegroup type is nil!")
			}
			if cg.Name == nil {
				return Cfg{}, makeErr(warnings, "cachegroup name is nil!")
			}

			if *cg.Name == *server.Cachegroup {
				if cg.ParentName != nil && *cg.ParentName != "" {
					parentCacheGroups[*cg.ParentName] = struct{}{}
				}
				if cg.SecondaryParentName != nil && *cg.SecondaryParentName != "" {
					parentCacheGroups[*cg.SecondaryParentName] = struct{}{}
				}
				break
			}
		}
	}

	nameTopologies := makeTopologyNameMap(topologies)

	cgServers := map[int]Server{} // map[serverID]server
	for _, sv := range servers {
		if sv.ID == nil {
			warnings = append(warnings, "TO servers had server with missing ID, skipping!")
			continue
		} else if sv.CDNName == nil {
			warnings = append(warnings, "TO servers had server with missing CDNName, skipping!")
			continue
		} else if sv.Cachegroup == nil || *sv.Cachegroup == "" {
			warnings = append(warnings, "TO servers had server with missing Cachegroup, skipping!")
			continue
		} else if sv.Status == nil || *sv.Status == "" {
			warnings = append(warnings, "TO servers had server with missing Status, skipping!")
			continue
		} else if sv.Type == "" {
			warnings = append(warnings, "TO servers had server with missing Type, skipping!")
			continue
		}
		if *sv.CDNName != *server.CDNName {
			continue
		}
		if _, ok := parentCacheGroups[*sv.Cachegroup]; !ok {
			continue
		}
		if sv.Type != tc.OriginTypeName &&
			!strings.HasPrefix(sv.Type, tc.EdgeTypePrefix) &&
			!strings.HasPrefix(sv.Type, tc.MidTypePrefix) {
			continue
		}
		if *sv.Status != string(tc.CacheStatusReported) && *sv.Status != string(tc.CacheStatusOnline) {
			continue
		}
		cgServers[*sv.ID] = sv
	}

	cgServerIDs := map[int]struct{}{}
	for serverID, _ := range cgServers {
		cgServerIDs[serverID] = struct{}{}
	}
	cgServerIDs[*server.ID] = struct{}{}

	cgDSServers := filterDSS(dss, nil, cgServerIDs)
	parentServerDSes := map[int]map[int]struct{}{} // map[serverID][dsID]
	for _, dss := range cgDSServers {
		if parentServerDSes[dss.Server] == nil {
			parentServerDSes[dss.Server] = map[int]struct{}{}
		}
		parentServerDSes[dss.Server][dss.DeliveryService] = struct{}{}
	}

	originServers, profileCaches, orgProfWarns, err := getOriginServersAndProfileCaches(cgServers, parentServerDSes, profileParentConfigParams, dses, serverCapabilities)
	warnings = append(warnings, orgProfWarns...)
	if err != nil {
		return Cfg{}, makeErr(warnings, "getting origin servers and profile caches: "+err.Error())
	}

	parentInfos := makeParentInfo(serverParentCGData, serverCDNDomain, profileCaches, originServers)

	dsOrigins, dsOriginWarns := makeDSOrigins(dss, dses, servers)
	warnings = append(warnings, dsOriginWarns...)

	for _, ds := range dses {
		if ds.XMLID == nil || *ds.XMLID == "" {
			warnings = append(warnings, "got ds with missing XMLID, skipping!")
			continue
		} else if ds.ID == nil {
			warnings = append(warnings, "got ds with missing ID, skipping!")
			continue
		} else if ds.Type == nil {
			warnings = append(warnings, "got ds with missing Type, skipping!")
			continue
		}

		if !cacheIsTopLevel && ds.Topology == nil {
			if _, ok := parentServerDSes[*server.ID][*ds.ID]; !ok {
				continue // skip DSes not assigned to this server.
			}
		}

		if !ds.Type.IsHTTP() && !ds.Type.IsDNS() {
			continue // skip ANY_MAP, STEERING, etc
		}
		if ds.OrgServerFQDN == nil || *ds.OrgServerFQDN == "" {
			// this check needs to be after the HTTP|DNS check, because Steering DSes without origins are ok'
			warnings = append(warnings, "DS '"+*ds.XMLID+"' has no origin server! Skipping!")
			continue
		}

		// Note these Parameters are only used for MSO for legacy DeliveryServiceServers DeliveryServices (except QueryStringHandling which is used by all DeliveryServices).
		//      Topology DSes use them for all DSes, MSO and non-MSO.
		dsParams, dsParamsWarnings := getParentDSParams(ds, profileParentConfigParams)
		warnings = append(warnings, dsParamsWarnings...)

		if existingDS, ok := processedOriginsToDSNames[*ds.OrgServerFQDN]; ok {
			warnings = append(warnings, "duplicate origin! DS '"+*ds.XMLID+"' and '"+string(existingDS)+"' share origin '"+*ds.OrgServerFQDN+"': skipping '"+*ds.XMLID+"'!")
			continue
		}

		// TODO put these in separate functions. No if-statement should be this long.
		if ds.Topology != nil && *ds.Topology != "" {
			txt, topoWarnings, err := getTopologyParentConfigLine(
				server,
				servers,
				&ds,
				serverParams,
				parentConfigParams,
				nameTopologies,
				serverCapabilities,
				dsRequiredCapabilities,
				cacheGroups,
				dsParams,
				atsMajorVer,
				dsOrigins[DeliveryServiceID(*ds.ID)],
				opt.AddComments,
			)
			warnings = append(warnings, topoWarnings...)
			if err != nil {
				// we don't want to fail generation with an error if one ds is malformed
				warnings = append(warnings, err.Error()) // getTopologyParentConfigLine includes error context
				continue
			}

			if txt != "" { // will be empty with no error if this server isn't in the Topology, or if it doesn't have the Required Capabilities
				textArr = append(textArr, txt)
			}
		} else if isTopLevelCache(serverParentCGData) {
			parentQStr := "ignore"
			if dsParams.QueryStringHandling == "" && dsParams.Algorithm == tc.AlgorithmConsistentHash && ds.QStringIgnore != nil && tc.QStringIgnore(*ds.QStringIgnore) == tc.QStringIgnoreUseInCacheKeyAndPassUp {
				parentQStr = "consider"
			}

			orgURI, orgWarns, err := getOriginURI(*ds.OrgServerFQDN)
			warnings = append(warnings, orgWarns...)
			if err != nil {
				warnings = append(warnings, "DS '"+*ds.XMLID+"' has malformed origin URI: '"+*ds.OrgServerFQDN+"': skipping!"+err.Error())
				continue
			}

			textLine := ""

			if ds.OriginShield != nil && *ds.OriginShield != "" {
				algorithm := ""
				if parentSelectAlg := serverParams[ParentConfigParamAlgorithm]; strings.TrimSpace(parentSelectAlg) != "" {
					algorithm = "round_robin=" + parentSelectAlg
				}
				textLine += makeParentComment(opt.AddComments, *ds.XMLID, "")
				textLine += "dest_domain=" + orgURI.Hostname() + " port=" + orgURI.Port() + " parent=" + *ds.OriginShield + " " + algorithm + " go_direct=true\n"
			} else if ds.MultiSiteOrigin != nil && *ds.MultiSiteOrigin {
				textLine += makeParentComment(opt.AddComments, *ds.XMLID, "")
				textLine += "dest_domain=" + orgURI.Hostname() + " port=" + orgURI.Port() + " "
				if len(parentInfos) == 0 {
				}

				if len(parentInfos[OriginHost(orgURI.Hostname())]) == 0 {
					// TODO error? emulates Perl
					warnings = append(warnings, "DS "+*ds.XMLID+" has no parent servers")
				}

				parents, secondaryParents, parentWarns := getMSOParentStrs(&ds, parentInfos[OriginHost(orgURI.Hostname())], atsMajorVer, dsParams.Algorithm, dsParams.TryAllPrimariesBeforeSecondary)
				warnings = append(warnings, parentWarns...)

				textLine += parents + secondaryParents + ` round_robin=` + dsParams.Algorithm + ` qstring=` + parentQStr + ` go_direct=false parent_is_proxy=false`
				textLine += getParentRetryStr(true, atsMajorVer, dsParams.ParentRetry, dsParams.UnavailableServerRetryResponses, dsParams.MaxSimpleRetries, dsParams.MaxUnavailableServerRetries)
				textLine += "\n" // TODO remove, and join later on "\n" instead of ""?

				textArr = append(textArr, textLine)
			}
		} else {
			queryStringHandling := serverParams[ParentConfigParamQStringHandling] // "qsh" in Perl

			roundRobin := `round_robin=consistent_hash`
			goDirect := `go_direct=false`

			parents, secondaryParents, parentWarns := getParentStrs(&ds, dsRequiredCapabilities, parentInfos[deliveryServicesAllParentsKey], atsMajorVer, dsParams.TryAllPrimariesBeforeSecondary)
			warnings = append(warnings, parentWarns...)

			text := ""
			orgURI, orgWarns, err := getOriginURI(*ds.OrgServerFQDN)
			warnings = append(warnings, orgWarns...)
			if err != nil {
				warnings = append(warnings, "DS '"+*ds.XMLID+"' had malformed origin  URI: '"+*ds.OrgServerFQDN+"': skipping!"+err.Error())
				continue
			}

			text += makeParentComment(opt.AddComments, *ds.XMLID, "")
			// TODO encode this in a DSType func, IsGoDirect() ?
			if *ds.Type == tc.DSTypeHTTPNoCache || *ds.Type == tc.DSTypeHTTPLive || *ds.Type == tc.DSTypeDNSLive {
				text += `dest_domain=` + orgURI.Hostname() + ` port=` + orgURI.Port() + ` go_direct=true` + "\n"
			} else {

				// check for profile psel.qstring_handling.  If this parameter is assigned to the server profile,
				// then edges will use the qstring handling value specified in the parameter for all profiles.

				// If there is no defined parameter in the profile, then check the delivery service profile.
				// If psel.qstring_handling exists in the DS profile, then we use that value for the specified DS only.
				// This is used only if not overridden by a server profile qstring handling parameter.

				// TODO refactor this logic, hard to understand (transliterated from Perl)
				dsQSH := queryStringHandling
				if dsQSH == "" {
					dsQSH = dsParams.QueryStringHandling
				}
				parentQStr := dsQSH
				if parentQStr == "" {
					parentQStr = "ignore"
				}
				if ds.QStringIgnore != nil && tc.QStringIgnore(*ds.QStringIgnore) == tc.QStringIgnoreUseInCacheKeyAndPassUp && dsQSH == "" {
					parentQStr = "consider"
				}

				text += `dest_domain=` + orgURI.Hostname() + ` port=` + orgURI.Port() + ` ` + parents + ` ` + secondaryParents + ` ` + roundRobin + ` ` + goDirect + ` qstring=` + parentQStr + "\n"
			}

			textArr = append(textArr, text)
		}
		processedOriginsToDSNames[*ds.OrgServerFQDN] = tc.DeliveryServiceName(*ds.XMLID)
	}

	// TODO determine if this is necessary. It's super-dangerous, and moreover ignores Server Capabilitites.
	defaultDestText := ""
	if !isTopLevelCache(serverParentCGData) {
		invalidDS := &DeliveryService{}
		invalidDS.ID = util.IntPtr(-1)
		tryAllPrimariesBeforeSecondary := false
		parents, secondaryParents, parentWarns := getParentStrs(invalidDS, dsRequiredCapabilities, parentInfos[deliveryServicesAllParentsKey], atsMajorVer, tryAllPrimariesBeforeSecondary)
		warnings = append(warnings, parentWarns...)
		defaultDestText = `dest_domain=. ` + parents
		if serverParams[ParentConfigParamAlgorithm] == tc.AlgorithmConsistentHash {
			defaultDestText += secondaryParents
		}
		defaultDestText += ` round_robin=consistent_hash go_direct=false`

		if qStr := serverParams[ParentConfigParamQString]; qStr != "" {
			defaultDestText += ` qstring=` + qStr
		}
		defaultDestText += "\n"
	}

	sort.Sort(sort.StringSlice(textArr))
	text := hdr + strings.Join(textArr, "")

	text += makeParentComment(opt.AddComments, "", "") + defaultDestText

	return Cfg{
		Text:        text,
		ContentType: ContentTypeParentDotConfig,
		LineComment: LineCommentParentDotConfig,
		Warnings:    warnings,
	}, nil
}

// makeParentComment creates the parent line comment and returns it.
// If addComments is false, returns the empty string. This exists for composability.
// Either dsName or topology may be the empty string.
// The returned comment includes a trailing newline.
func makeParentComment(addComments bool, dsName string, topology string) string {
	if !addComments {
		return ""
	}
	return "# ds '" + dsName + "' topology '" + topology + "'" + "\n"
}

type parentConfigDS struct {
	Name                 tc.DeliveryServiceName
	QStringIgnore        tc.QStringIgnore
	OriginFQDN           string
	MultiSiteOrigin      bool
	OriginShield         string
	Type                 tc.DSType
	QStringHandling      string
	RequiredCapabilities map[ServerCapability]struct{}
	Topology             string
}

type parentConfigDSTopLevel struct {
	parentConfigDS
	MSOAlgorithm                       string
	MSOParentRetry                     string
	MSOUnavailableServerRetryResponses string
	MSOMaxSimpleRetries                string
	MSOMaxUnavailableServerRetries     string
}

type parentInfo struct {
	Host            string
	Port            int
	Domain          string
	Weight          string
	UseIP           bool
	Rank            int
	IP              string
	PrimaryParent   bool
	SecondaryParent bool
	Capabilities    map[ServerCapability]struct{}
}

func (p parentInfo) Format() string {
	host := ""
	if p.UseIP {
		host = p.IP
	} else {
		host = p.Host + "." + p.Domain
	}
	return host + ":" + strconv.Itoa(p.Port) + "|" + p.Weight + ";"
}

type parentInfos map[OriginHost]parentInfo

type parentInfoSortByRank []parentInfo

func (s parentInfoSortByRank) Len() int      { return len(s) }
func (s parentInfoSortByRank) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s parentInfoSortByRank) Less(i, j int) bool {
	if s[i].Rank != s[j].Rank {
		return s[i].Rank < s[j].Rank
	} else if s[i].Host != s[j].Host {
		return s[i].Host < s[j].Host
	} else if s[i].Domain != s[j].Domain {
		return s[i].Domain < s[j].Domain
	} else if s[i].Port != s[j].Port {
		return s[i].Port < s[j].Port
	}
	return s[i].IP < s[j].IP
}

type serverWithParams struct {
	Server
	Params profileCache
}

type serversWithParamsSortByRank []serverWithParams

func (ss serversWithParamsSortByRank) Len() int      { return len(ss) }
func (ss serversWithParamsSortByRank) Swap(i, j int) { ss[i], ss[j] = ss[j], ss[i] }
func (ss serversWithParamsSortByRank) Less(i, j int) bool {
	if ss[i].Params.Rank != ss[j].Params.Rank {
		return ss[i].Params.Rank < ss[j].Params.Rank
	}

	if ss[i].HostName == nil {
		if ss[j].HostName != nil {
			return true
		}
	} else if ss[j].HostName == nil {
		return false
	} else if ss[i].HostName != ss[j].HostName {
		return *ss[i].HostName < *ss[j].HostName
	}

	if ss[i].DomainName == nil {
		if ss[j].DomainName != nil {
			return true
		}
	} else if ss[j].DomainName == nil {
		return false
	} else if ss[i].DomainName != ss[j].DomainName {
		return *ss[i].DomainName < *ss[j].DomainName
	}

	if ss[i].Params.Port != ss[j].Params.Port {
		return ss[i].Params.Port < ss[j].Params.Port
	}

	iIP := getServerIPAddress(&ss[i].Server)
	jIP := getServerIPAddress(&ss[j].Server)

	if iIP == nil {
		if jIP != nil {
			return true
		}
	} else if jIP == nil {
		return false
	}
	return bytes.Compare(iIP, jIP) <= 0
}

type parentConfigDSTopLevelSortByName []parentConfigDSTopLevel

func (s parentConfigDSTopLevelSortByName) Len() int      { return len(s) }
func (s parentConfigDSTopLevelSortByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s parentConfigDSTopLevelSortByName) Less(i, j int) bool {
	return strings.Compare(string(s[i].Name), string(s[j].Name)) < 0
}

type dsesSortByName []DeliveryService

func (s dsesSortByName) Len() int      { return len(s) }
func (s dsesSortByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s dsesSortByName) Less(i, j int) bool {
	if s[i].XMLID == nil {
		return true
	}
	if s[j].XMLID == nil {
		return false
	}
	return *s[i].XMLID < *s[j].XMLID
}

type profileCache struct {
	Weight     string
	Port       int
	UseIP      bool
	Rank       int
	NotAParent bool
}

func defaultProfileCache() profileCache {
	return profileCache{
		Weight:     "0.999",
		Port:       0,
		UseIP:      false,
		Rank:       1,
		NotAParent: false,
	}
}

// cgServer is the server table data needed when selecting the servers assigned to a cachegroup.
type cgServer struct {
	ServerID       ServerID
	ServerHost     string
	ServerIP       string
	ServerPort     int
	CacheGroupID   int
	CacheGroupName string
	Status         int
	Type           int
	ProfileID      ProfileID
	ProfileName    string
	CDN            int
	TypeName       string
	Domain         string
	Capabilities   map[ServerCapability]struct{}
}

type originURI struct {
	Scheme string
	Host   string
	Port   string
}

// TODO change, this is terrible practice, using a hard-coded key. What if there were a delivery service named "all_parents" (transliterated Perl)
const deliveryServicesAllParentsKey = "all_parents"

type parentDSParams struct {
	Algorithm                       string
	ParentRetry                     string
	UnavailableServerRetryResponses string
	MaxSimpleRetries                string
	MaxUnavailableServerRetries     string
	QueryStringHandling             string
	TryAllPrimariesBeforeSecondary  bool
}

// getDSParams returns the Delivery Service Profile Parameters used in parent.config, and any warnings.
// If Parameters don't exist, defaults are returned. Non-MSO Delivery Services default to no custom retry logic (we should reevaluate that).
// Note these Parameters are only used for MSO for legacy DeliveryServiceServers DeliveryServices.
//      Topology DSes use them for all DSes, MSO and non-MSO.
func getParentDSParams(ds DeliveryService, profileParentConfigParams map[string]map[string]string) (parentDSParams, []string) {
	warnings := []string{}
	params := parentDSParams{}
	isMSO := ds.MultiSiteOrigin != nil && *ds.MultiSiteOrigin
	if isMSO {
		params.Algorithm = ParentConfigDSParamDefaultMSOAlgorithm
		params.ParentRetry = ParentConfigDSParamDefaultMSOParentRetry
		params.UnavailableServerRetryResponses = ParentConfigDSParamDefaultMSOUnavailableServerRetryResponses
		params.MaxSimpleRetries = ParentConfigDSParamDefaultMaxSimpleRetries
		params.MaxUnavailableServerRetries = ParentConfigDSParamDefaultMaxUnavailableServerRetries
	}
	if ds.ProfileName == nil || *ds.ProfileName == "" {
		return params, warnings
	}
	dsParams, ok := profileParentConfigParams[*ds.ProfileName]
	if !ok {
		return params, warnings
	}

	params.QueryStringHandling = dsParams[ParentConfigParamQStringHandling] // may be blank, no default
	// TODO deprecate & remove "mso." Parameters - there was never a reason to restrict these settings to MSO.
	if isMSO {
		if v, ok := dsParams[ParentConfigParamMSOAlgorithm]; ok && strings.TrimSpace(v) != "" {
			params.Algorithm = v
		}
		if v, ok := dsParams[ParentConfigParamMSOParentRetry]; ok {
			params.ParentRetry = v
		}
		if v, ok := dsParams[ParentConfigParamMSOUnavailableServerRetryResponses]; ok {
			if v != "" && !unavailableServerRetryResponsesValid(v) {
				warnings = append(warnings, "DS '"+*ds.XMLID+"' had malformed "+ParentConfigParamMSOUnavailableServerRetryResponses+" parameter '"+v+"', not using!")
			} else if v != "" {
				params.UnavailableServerRetryResponses = v
			}
		}
		if v, ok := dsParams[ParentConfigParamMSOMaxSimpleRetries]; ok {
			params.MaxSimpleRetries = v
		}
		if v, ok := dsParams[ParentConfigParamMSOMaxUnavailableServerRetries]; ok {
			params.MaxUnavailableServerRetries = v
		}
	}

	// Even if the DS is MSO, non-"mso." Parameters override "mso." ones, because they're newer.
	if v, ok := dsParams[ParentConfigParamAlgorithm]; ok && strings.TrimSpace(v) != "" {
		params.Algorithm = v
	}
	if v, ok := dsParams[ParentConfigParamParentRetry]; ok {
		params.ParentRetry = v
	}
	if v, ok := dsParams[ParentConfigParamUnavailableServerRetryResponses]; ok {
		if v != "" && !unavailableServerRetryResponsesValid(v) {
			warnings = append(warnings, "DS '"+*ds.XMLID+"' had malformed "+ParentConfigParamUnavailableServerRetryResponses+" parameter '"+v+"', not using!")
		} else if v != "" {
			params.UnavailableServerRetryResponses = v
		}
	}
	if v, ok := dsParams[ParentConfigParamMaxSimpleRetries]; ok {
		params.MaxSimpleRetries = v
	}
	if v, ok := dsParams[ParentConfigParamMaxUnavailableServerRetries]; ok {
		params.MaxUnavailableServerRetries = v
	}
	if v, ok := dsParams[ParentConfigParamSecondaryMode]; ok {
		if v != "" {
			warnings = append(warnings, "DS '"+*ds.XMLID+"' had Parameter "+ParentConfigParamSecondaryMode+" which is used if it exists, the value is ignored! Non-empty value '"+v+"' will be ignored!")
		}
		params.TryAllPrimariesBeforeSecondary = true
	}

	return params, warnings
}

// getTopologyParentConfigLine returns the topology parent.config line, any warnings, and any error
func getTopologyParentConfigLine(
	server *Server,
	servers []Server,
	ds *DeliveryService,
	serverParams map[string]string,
	parentConfigParams []parameterWithProfilesMap, // all params with configFile parent.config
	nameTopologies map[TopologyName]tc.Topology,
	serverCapabilities map[int]map[ServerCapability]struct{},
	dsRequiredCapabilities map[int]map[ServerCapability]struct{},
	cacheGroups map[tc.CacheGroupName]tc.CacheGroupNullable,
	dsParams parentDSParams,
	atsMajorVer int,
	dsOrigins map[ServerID]struct{},
	addComments bool,
) (string, []string, error) {
	warnings := []string{}
	txt := ""

	if !hasRequiredCapabilities(serverCapabilities[*server.ID], dsRequiredCapabilities[*ds.ID]) {
		return "", warnings, nil
	}

	orgURI, orgWarns, err := getOriginURI(*ds.OrgServerFQDN)
	warnings = append(warnings, orgWarns...)
	if err != nil {
		return "", warnings, errors.New("DS '" + *ds.XMLID + "' has malformed origin URI: '" + *ds.OrgServerFQDN + "': skipping!" + err.Error())
	}

	topology := nameTopologies[TopologyName(*ds.Topology)]
	if topology.Name == "" {
		return "", warnings, errors.New("DS " + *ds.XMLID + " topology '" + *ds.Topology + "' not found in Topologies!")
	}

	txt += makeParentComment(addComments, *ds.XMLID, *ds.Topology)
	txt += "dest_domain=" + orgURI.Hostname() + " port=" + orgURI.Port()

	serverPlacement, err := getTopologyPlacement(tc.CacheGroupName(*server.Cachegroup), topology, cacheGroups, ds)
	if err != nil {
		return "", warnings, errors.New("getting topology placement: " + err.Error())
	}
	if !serverPlacement.InTopology {
		return "", warnings, nil // server isn't in topology, no error
	}
	// TODO add Topology/Capabilities to remap.config

	parents, secondaryParents, parentWarnings, err := getTopologyParents(server, ds, servers, parentConfigParams, topology, serverPlacement.IsLastTier, serverCapabilities, dsRequiredCapabilities, dsOrigins)
	warnings = append(warnings, parentWarnings...)
	if err != nil {
		return "", warnings, errors.New("getting topology parents for '" + *ds.XMLID + "': skipping! " + err.Error())
	}
	if len(parents) == 0 {
		return "", warnings, errors.New("getting topology parents for '" + *ds.XMLID + "': no parents found! skipping! (Does your Topology have a CacheGroup with no servers in it?)")
	}

	txt += ` parent="` + strings.Join(parents, `;`) + `"`
	if len(secondaryParents) > 0 {
		txt += ` secondary_parent="` + strings.Join(secondaryParents, `;`) + `"`

		secondaryModeStr, secondaryModeWarnings := getSecondaryModeStr(dsParams.TryAllPrimariesBeforeSecondary, atsMajorVer, tc.DeliveryServiceName(*ds.XMLID))
		warnings = append(warnings, secondaryModeWarnings...)
		txt += secondaryModeStr
	}
	txt += ` round_robin=` + getTopologyRoundRobin(ds, serverParams, serverPlacement.IsLastCacheTier, dsParams.Algorithm)
	txt += ` go_direct=` + getTopologyGoDirect(ds, serverPlacement.IsLastTier)
	txt += ` qstring=` + getTopologyQueryString(ds, serverParams, serverPlacement.IsLastCacheTier, dsParams.Algorithm, dsParams.QueryStringHandling)
	txt += getTopologyParentIsProxyStr(serverPlacement.IsLastCacheTier)
	txt += getParentRetryStr(serverPlacement.IsLastCacheTier, atsMajorVer, dsParams.ParentRetry, dsParams.UnavailableServerRetryResponses, dsParams.MaxSimpleRetries, dsParams.MaxUnavailableServerRetries)
	txt += "\n"

	return txt, warnings, nil
}

// getParentRetryStr builds the parent retry directive(s).
// If atsMajorVer < 6, "" is returned (ATS 5 and below don't support retry directives).
// If isLastCacheTier is false, "" is returned. This argument exists to simplify usage.
// If parentRetry is "", "" is returned (because the other directives are unused if parent_retry doesn't exist). This is allowed to simplify usage.
// If unavailableServerRetryResponses is not "", it must be valid. Use unavailableServerRetryResponsesValid to check.
// If maxSimpleRetries is "", ParentConfigDSParamDefaultMaxSimpleRetries will be used.
// If maxUnavailableServerRetries is "", ParentConfigDSParamDefaultMaxUnavailableServerRetries will be used.
func getParentRetryStr(isLastCacheTier bool, atsMajorVer int, parentRetry string, unavailableServerRetryResponses string, maxSimpleRetries string, maxUnavailableServerRetries string) string {
	if !isLastCacheTier || // allow !isLastCacheTier, to simplify usage.
		parentRetry == "" || // allow parentRetry to be empty, to simplify usage.
		atsMajorVer < 6 { // ATS 5 and below don't support parent_retry directives
		return ""
	}

	if maxSimpleRetries == "" {
		maxSimpleRetries = ParentConfigDSParamDefaultMaxSimpleRetries
	}
	if maxUnavailableServerRetries == "" {
		maxUnavailableServerRetries = ParentConfigDSParamDefaultMaxUnavailableServerRetries
	}

	txt := ` parent_retry=` + parentRetry
	if unavailableServerRetryResponses != "" {
		txt += ` unavailable_server_retry_responses=` + unavailableServerRetryResponses
	}
	txt += ` max_simple_retries=` + maxSimpleRetries + ` max_unavailable_server_retries=` + maxUnavailableServerRetries
	return txt
}

// getSecondaryModeStr returns the secondary_mode string, and any warnings.
func getSecondaryModeStr(tryAllPrimariesBeforeSecondary bool, atsMajorVer int, ds tc.DeliveryServiceName) (string, []string) {
	warnings := []string{}
	if !tryAllPrimariesBeforeSecondary {
		return "", warnings
	}
	if atsMajorVer < 8 {
		warnings = append(warnings, "DS '"+string(ds)+"' had Parameter "+ParentConfigParamSecondaryMode+" but this cache is "+strconv.Itoa(atsMajorVer)+" and secondary_mode isn't supported in ATS until 8. Not using!")
		return "", warnings
	}
	return ` secondary_mode=2`, warnings // See https://docs.trafficserver.apache.org/en/8.0.x/admin-guide/files/parent.config.en.html
}

func getTopologyParentIsProxyStr(serverIsLastCacheTier bool) string {
	if serverIsLastCacheTier {
		return ` parent_is_proxy=false`
	}
	return ""
}

func getTopologyRoundRobin(
	ds *DeliveryService,
	serverParams map[string]string,
	serverIsLastTier bool,
	algorithm string,
) string {
	roundRobinConsistentHash := "consistent_hash"
	if !serverIsLastTier {
		return roundRobinConsistentHash
	}
	if parentSelectAlg := serverParams[ParentConfigParamAlgorithm]; ds.OriginShield != nil && *ds.OriginShield != "" && strings.TrimSpace(parentSelectAlg) != "" {
		return parentSelectAlg
	}
	if algorithm != "" {
		return algorithm
	}
	return roundRobinConsistentHash
}

func getTopologyGoDirect(ds *DeliveryService, serverIsLastTier bool) string {
	if !serverIsLastTier {
		return "false"
	}
	if ds.OriginShield != nil && *ds.OriginShield != "" {
		return "true"
	}
	if ds.MultiSiteOrigin != nil && *ds.MultiSiteOrigin {
		return "false"
	}
	return "true"
}

func getTopologyQueryString(
	ds *DeliveryService,
	serverParams map[string]string,
	serverIsLastTier bool,
	algorithm string,
	qStringHandling string,
) string {
	if serverIsLastTier {
		if ds.MultiSiteOrigin != nil && *ds.MultiSiteOrigin && qStringHandling == "" && algorithm == tc.AlgorithmConsistentHash && ds.QStringIgnore != nil && tc.QStringIgnore(*ds.QStringIgnore) == tc.QStringIgnoreUseInCacheKeyAndPassUp {
			return "consider"
		}
		return "ignore"
	}

	if param := serverParams[ParentConfigParamQStringHandling]; param != "" {
		return param
	}
	if qStringHandling != "" {
		return qStringHandling
	}
	if ds.QStringIgnore != nil && tc.QStringIgnore(*ds.QStringIgnore) == tc.QStringIgnoreUseInCacheKeyAndPassUp {
		return "consider"
	}
	return "ignore"
}

// serverParentageParams gets the Parameters used for parent= line, or defaults if they don't exist
// Returns the Parameters used for parent= lines for the given server, and any warnings.
func serverParentageParams(sv *Server, params []parameterWithProfilesMap) (profileCache, []string) {
	warnings := []string{}
	// TODO deduplicate with atstccfg/parentdotconfig.go
	profileCache := defaultProfileCache()
	if sv.TCPPort != nil {
		profileCache.Port = *sv.TCPPort
	}
	for _, param := range params {
		if _, ok := param.ProfileNames[*sv.Profile]; !ok {
			continue
		}
		switch param.Name {
		case ParentConfigCacheParamWeight:
			profileCache.Weight = param.Value
		case ParentConfigCacheParamPort:
			if i, err := strconv.Atoi(param.Value); err != nil {
				warnings = append(warnings, "port param is not an integer, skipping! : "+err.Error())
			} else {
				profileCache.Port = i
			}
		case ParentConfigCacheParamUseIP:
			profileCache.UseIP = param.Value == "1"
		case ParentConfigCacheParamRank:
			if i, err := strconv.Atoi(param.Value); err != nil {
				warnings = append(warnings, "rank param is not an integer, skipping! : "+err.Error())
			} else {
				profileCache.Rank = i
			}
		case ParentConfigCacheParamNotAParent:
			profileCache.NotAParent = param.Value != "false"
		}
	}
	return profileCache, warnings
}

func serverParentStr(sv *Server, svParams profileCache) (string, error) {
	if svParams.NotAParent {
		return "", nil
	}
	host := ""
	if svParams.UseIP {
		// TODO get service interface here
		ip := getServerIPAddress(sv)
		if ip == nil {
			return "", errors.New("server params Use IP, but has no valid IPv4 Service Address")
		}
		host = ip.String()
	} else {
		host = *sv.HostName + "." + *sv.DomainName
	}
	return host + ":" + strconv.Itoa(svParams.Port) + "|" + svParams.Weight, nil
}

// GetTopologyParents returns the parents, secondary parents, any warnings, and any error.
func getTopologyParents(
	server *Server,
	ds *DeliveryService,
	servers []Server,
	parentConfigParams []parameterWithProfilesMap, // all params with configFile parent.confign
	topology tc.Topology,
	serverIsLastTier bool,
	serverCapabilities map[int]map[ServerCapability]struct{},
	dsRequiredCapabilities map[int]map[ServerCapability]struct{},
	dsOrigins map[ServerID]struct{}, // for Topology DSes, MSO still needs DeliveryServiceServer assignments.
) ([]string, []string, []string, error) {
	warnings := []string{}
	// If it's the last tier, then the parent is the origin.
	// Note this doesn't include MSO, whose final tier cachegroup points to the origin cachegroup.
	if serverIsLastTier {
		orgURI, orgWarns, err := getOriginURI(*ds.OrgServerFQDN) // TODO pass, instead of calling again
		warnings = append(warnings, orgWarns...)
		if err != nil {
			return nil, nil, warnings, err
		}
		return []string{orgURI.Host}, nil, warnings, nil
	}

	svNode := tc.TopologyNode{}
	for _, node := range topology.Nodes {
		if node.Cachegroup == *server.Cachegroup {
			svNode = node
			break
		}
	}
	if svNode.Cachegroup == "" {
		return nil, nil, warnings, errors.New("This server '" + *server.HostName + "' not in DS " + *ds.XMLID + " topology, skipping")
	}

	if len(svNode.Parents) == 0 {
		return nil, nil, warnings, errors.New("DS " + *ds.XMLID + " topology '" + *ds.Topology + "' is last tier, but NonLastTier called! Should never happen")
	}
	if numParents := len(svNode.Parents); numParents > 2 {
		warnings = append(warnings, "DS "+*ds.XMLID+" topology '"+*ds.Topology+"' has "+strconv.Itoa(numParents)+" parent nodes, but Apache Traffic Server only supports Primary and Secondary (2) lists of parents. CacheGroup nodes after the first 2 will be ignored!")
	}
	if len(topology.Nodes) <= svNode.Parents[0] {
		return nil, nil, warnings, errors.New("DS " + *ds.XMLID + " topology '" + *ds.Topology + "' node parent " + strconv.Itoa(svNode.Parents[0]) + " greater than number of topology nodes " + strconv.Itoa(len(topology.Nodes)) + ". Cannot create parents!")
	}
	if len(svNode.Parents) > 1 && len(topology.Nodes) <= svNode.Parents[1] {
		warnings = append(warnings, "DS "+*ds.XMLID+" topology '"+*ds.Topology+"' node secondary parent "+strconv.Itoa(svNode.Parents[1])+" greater than number of topology nodes "+strconv.Itoa(len(topology.Nodes))+". Secondary parent will be ignored!")
	}

	parentCG := topology.Nodes[svNode.Parents[0]].Cachegroup
	secondaryParentCG := ""
	if len(svNode.Parents) > 1 && len(topology.Nodes) > svNode.Parents[1] {
		secondaryParentCG = topology.Nodes[svNode.Parents[1]].Cachegroup
	}

	if parentCG == "" {
		return nil, nil, warnings, errors.New("Server '" + *server.HostName + "' DS " + *ds.XMLID + " topology '" + *ds.Topology + "' cachegroup '" + *server.Cachegroup + "' topology node parent " + strconv.Itoa(svNode.Parents[0]) + " is not in the topology!")
	}

	parentStrs := []string{}
	secondaryParentStrs := []string{}

	serversWithParams := []serverWithParams{}
	for _, sv := range servers {
		serverParentParams, parentWarns := serverParentageParams(&sv, parentConfigParams)
		warnings = append(warnings, parentWarns...)
		serversWithParams = append(serversWithParams, serverWithParams{
			Server: sv,
			Params: serverParentParams,
		})
	}
	sort.Sort(serversWithParamsSortByRank(serversWithParams))

	for _, sv := range serversWithParams {
		if sv.ID == nil {
			warnings = append(warnings, "TO Servers server had nil ID, skipping")
			continue
		} else if sv.Cachegroup == nil {
			warnings = append(warnings, "TO Servers server had nil Cachegroup, skipping")
			continue
		} else if sv.CDNName == nil {
			warnings = append(warnings, "TO servers had server with missing CDNName, skipping!")
			continue
		} else if sv.Status == nil || *sv.Status == "" {
			warnings = append(warnings, "TO servers had server with missing Status, skipping!")
			continue
		}

		if !strings.HasPrefix(sv.Type, tc.EdgeTypePrefix) && !strings.HasPrefix(sv.Type, tc.MidTypePrefix) && sv.Type != tc.OriginTypeName {
			continue // only consider edges, mids, and origins in the CacheGroup.
		}
		if _, dsHasOrigin := dsOrigins[ServerID(*sv.ID)]; sv.Type == tc.OriginTypeName && !dsHasOrigin {
			continue
		}
		if *sv.CDNName != *server.CDNName {
			continue
		}
		if *sv.Status != string(tc.CacheStatusReported) && *sv.Status != string(tc.CacheStatusOnline) {
			continue
		}

		if sv.Type != tc.OriginTypeName && !hasRequiredCapabilities(serverCapabilities[*sv.ID], dsRequiredCapabilities[*ds.ID]) {
			continue
		}
		if *sv.Cachegroup == parentCG {
			parentStr, err := serverParentStr(&sv.Server, sv.Params)
			if err != nil {
				return nil, nil, warnings, errors.New("getting server parent string: " + err.Error())
			}
			if parentStr != "" { // will be empty if server is not_a_parent (possibly other reasons)
				parentStrs = append(parentStrs, parentStr)
			}
		}
		if *sv.Cachegroup == secondaryParentCG {
			parentStr, err := serverParentStr(&sv.Server, sv.Params)
			if err != nil {
				return nil, nil, warnings, errors.New("getting server parent string: " + err.Error())
			}
			secondaryParentStrs = append(secondaryParentStrs, parentStr)
		}
	}

	return parentStrs, secondaryParentStrs, warnings, nil
}

// getOriginURI returns the URL, any warnings, and any error.
func getOriginURI(fqdn string) (*url.URL, []string, error) {
	warnings := []string{}

	orgURI, err := url.Parse(fqdn) // TODO verify origin is always a host:port
	if err != nil {
		return nil, warnings, errors.New("parsing: " + err.Error())
	}
	if orgURI.Port() == "" {
		if orgURI.Scheme == "http" {
			orgURI.Host += ":80"
		} else if orgURI.Scheme == "https" {
			orgURI.Host += ":443"
		} else {
			warnings = append(warnings, "non-top-level: origin '"+fqdn+"' is unknown scheme '"+orgURI.Scheme+"', but has no port! Using as-is! ")
		}
	}
	return orgURI, warnings, nil
}

// getParentStrs returns the parents= and secondary_parents= strings for ATS parent.config lines, and any warnings.
func getParentStrs(
	ds *DeliveryService,
	dsRequiredCapabilities map[int]map[ServerCapability]struct{},
	parentInfos []parentInfo,
	atsMajorVer int,
	tryAllPrimariesBeforeSecondary bool,
) (string, string, []string) {
	warnings := []string{}
	parentInfo := []string{}
	secondaryParentInfo := []string{}

	sort.Sort(parentInfoSortByRank(parentInfos))

	for _, parent := range parentInfos { // TODO fix magic key
		if !hasRequiredCapabilities(parent.Capabilities, dsRequiredCapabilities[*ds.ID]) {
			continue
		}

		pTxt := parent.Format()
		if parent.PrimaryParent {
			parentInfo = append(parentInfo, pTxt)
		} else if parent.SecondaryParent {
			secondaryParentInfo = append(secondaryParentInfo, pTxt)
		}
	}

	if len(parentInfo) == 0 {
		parentInfo = secondaryParentInfo
		secondaryParentInfo = []string{}
	}

	// TODO remove duplicate code with top level if block
	seen := map[string]struct{}{} // TODO change to host+port? host isn't unique
	parentInfo, seen = util.RemoveStrDuplicates(parentInfo, seen)
	secondaryParentInfo, seen = util.RemoveStrDuplicates(secondaryParentInfo, seen)

	dsName := tc.DeliveryServiceName("")
	if ds != nil && ds.XMLID != nil {
		dsName = tc.DeliveryServiceName(*ds.XMLID)
	}

	parents := ""
	secondaryParents := "" // "secparents" in Perl

	if atsMajorVer >= 6 && len(secondaryParentInfo) > 0 {
		parents = `parent="` + strings.Join(parentInfo, "") + `"`
		secondaryParents = ` secondary_parent="` + strings.Join(secondaryParentInfo, "") + `"`
		secondaryModeStr, secondaryModeWarnings := getSecondaryModeStr(tryAllPrimariesBeforeSecondary, atsMajorVer, dsName)
		warnings = append(warnings, secondaryModeWarnings...)
		secondaryParents += secondaryModeStr
	} else {
		parents = `parent="` + strings.Join(parentInfo, "") + strings.Join(secondaryParentInfo, "") + `"`
	}

	return parents, secondaryParents, warnings
}

// getMSOParentStrs returns the parents= and secondary_parents= strings for ATS parent.config lines for MSO, and any warnings.
func getMSOParentStrs(
	ds *DeliveryService,
	parentInfos []parentInfo,
	atsMajorVer int,
	msoAlgorithm string,
	tryAllPrimariesBeforeSecondary bool,
) (string, string, []string) {
	warnings := []string{}
	// TODO determine why MSO is different, and if possible, combine with getParentAndSecondaryParentStrs.

	rankedParents := parentInfoSortByRank(parentInfos)
	sort.Sort(rankedParents)

	parentInfoTxt := []string{}
	secondaryParentInfo := []string{}
	nullParentInfo := []string{}
	for _, parent := range ([]parentInfo)(rankedParents) {
		if parent.PrimaryParent {
			parentInfoTxt = append(parentInfoTxt, parent.Format())
		} else if parent.SecondaryParent {
			secondaryParentInfo = append(secondaryParentInfo, parent.Format())
		} else {
			nullParentInfo = append(nullParentInfo, parent.Format())
		}
	}

	if len(parentInfoTxt) == 0 {
		// If no parents are found in the secondary parent either, then set the null parent list (parents in neither secondary or primary)
		// as the secondary parent list and clear the null parent list.
		if len(secondaryParentInfo) == 0 {
			secondaryParentInfo = nullParentInfo
			nullParentInfo = []string{}
		}
		parentInfoTxt = secondaryParentInfo
		secondaryParentInfo = []string{} // TODO should thi be '= secondary'? Currently emulates Perl
	}

	// TODO benchmark, verify this isn't slow. if it is, it could easily be made faster
	seen := map[string]struct{}{} // TODO change to host+port? host isn't unique
	parentInfoTxt, seen = util.RemoveStrDuplicates(parentInfoTxt, seen)
	secondaryParentInfo, seen = util.RemoveStrDuplicates(secondaryParentInfo, seen)
	nullParentInfo, seen = util.RemoveStrDuplicates(nullParentInfo, seen)

	secondaryParentStr := strings.Join(secondaryParentInfo, "") + strings.Join(nullParentInfo, "")

	dsName := tc.DeliveryServiceName("")
	if ds != nil && ds.XMLID != nil {
		dsName = tc.DeliveryServiceName(*ds.XMLID)
	}

	// If the ats version supports it and the algorithm is consistent hash, put secondary and non-primary parents into secondary parent group.
	// This will ensure that secondary and tertiary parents will be unused unless all hosts in the primary group are unavailable.

	parents := ""
	secondaryParents := ""

	if atsMajorVer >= 6 && msoAlgorithm == "consistent_hash" && len(secondaryParentStr) > 0 {
		parents = `parent="` + strings.Join(parentInfoTxt, "") + `"`
		secondaryParents = ` secondary_parent="` + secondaryParentStr + `"`
		secondaryModeStr, secondaryModeWarnings := getSecondaryModeStr(tryAllPrimariesBeforeSecondary, atsMajorVer, dsName)
		warnings = append(warnings, secondaryModeWarnings...)
		secondaryParents += secondaryModeStr
	} else {
		parents = `parent="` + strings.Join(parentInfoTxt, "") + secondaryParentStr + `"`
	}
	return parents, secondaryParents, warnings
}

func makeParentInfo(
	serverParentCGData serverParentCacheGroupData,
	serverDomain string, // getCDNDomainByProfileID(tx, server.ProfileID)
	profileCaches map[ProfileID]profileCache, // getServerParentCacheGroupProfiles(tx, server)
	originServers map[OriginHost][]cgServer, // getServerParentCacheGroupProfiles(tx, server)
) map[OriginHost][]parentInfo {
	parentInfos := map[OriginHost][]parentInfo{}

	// note servers also contains an "all" key
	for originHost, servers := range originServers {
		for _, row := range servers {
			profile := profileCaches[row.ProfileID]
			if profile.NotAParent {
				continue
			}
			// Perl has this check, but we only select servers ("deliveryServices" in Perl) with the right CDN in the first place
			// if profile.Domain != serverDomain {
			// 	continue
			// }

			parentInf := parentInfo{
				Host:            row.ServerHost,
				Port:            profile.Port,
				Domain:          row.Domain,
				Weight:          profile.Weight,
				UseIP:           profile.UseIP,
				Rank:            profile.Rank,
				IP:              row.ServerIP,
				PrimaryParent:   serverParentCGData.ParentID == row.CacheGroupID,
				SecondaryParent: serverParentCGData.SecondaryParentID == row.CacheGroupID,
				Capabilities:    row.Capabilities,
			}
			if parentInf.Port < 1 {
				parentInf.Port = row.ServerPort
			}
			parentInfos[originHost] = append(parentInfos[originHost], parentInf)
		}
	}
	return parentInfos
}

// unavailableServerRetryResponsesValid returns whether a unavailable_server_retry_responses parameter is valid for an ATS parent rule.
func unavailableServerRetryResponsesValid(s string) bool {
	// optimization if param is empty
	if s == "" {
		return false
	}
	re := regexp.MustCompile(`^"(:?\d{3},)+\d{3}"\s*$`) // TODO benchmark, cache if performance matters
	return re.MatchString(s)
}

// getOriginServersAndProfileCaches returns the origin servers, ProfileCaches, any warnings, and any error.
func getOriginServersAndProfileCaches(
	cgServers map[int]Server,
	parentServerDSes map[int]map[int]struct{},
	profileParentConfigParams map[string]map[string]string, // map[profileName][paramName]paramVal
	dses []DeliveryService,
	serverCapabilities map[int]map[ServerCapability]struct{},
) (map[OriginHost][]cgServer, map[ProfileID]profileCache, []string, error) {
	warnings := []string{}
	originServers := map[OriginHost][]cgServer{}  // "deliveryServices" in Perl
	profileCaches := map[ProfileID]profileCache{} // map[profileID]ProfileCache

	dsIDMap := map[int]DeliveryService{}
	for _, ds := range dses {
		if ds.ID == nil {
			return nil, nil, warnings, errors.New("delivery services got nil ID!")
		}
		if !ds.Type.IsHTTP() && !ds.Type.IsDNS() {
			continue // skip ANY_MAP, STEERING, etc
		}
		dsIDMap[*ds.ID] = ds
	}

	allDSMap := map[int]DeliveryService{} // all DSes for this server, NOT all dses in TO
	for _, dsIDs := range parentServerDSes {
		for dsID, _ := range dsIDs {
			if _, ok := dsIDMap[dsID]; !ok {
				// this is normal if the TO was too old to understand our /deliveryserviceserver?servers= query param
				// In which case, the DSS will include DSes from other CDNs, which aren't in the dsIDMap
				// If the server was new enough to respect the params, this should never happen.
				// warnings = append(warnings, ("getting delivery services: parent server DS %v not in dsIDMap\n", dsID)
				continue
			}
			if _, ok := allDSMap[dsID]; !ok {
				allDSMap[dsID] = dsIDMap[dsID]
			}
		}
	}

	dsOrigins, dsOriginWarns, err := getDSOrigins(allDSMap)
	warnings = append(warnings, dsOriginWarns...)
	if err != nil {
		return nil, nil, warnings, errors.New("getting DS origins: " + err.Error())
	}

	profileParams, profParamWarns := getParentConfigProfileParams(cgServers, profileParentConfigParams)
	warnings = append(warnings, profParamWarns...)

	for _, cgSv := range cgServers {
		if cgSv.ID == nil {
			warnings = append(warnings, "getting origin servers: got server with nil ID, skipping!")
			continue
		} else if cgSv.HostName == nil {
			warnings = append(warnings, "getting origin servers: got server with nil HostName, skipping!")
			continue
		} else if cgSv.TCPPort == nil {
			warnings = append(warnings, "getting origin servers: got server with nil TCPPort, skipping!")
			continue
		} else if cgSv.CachegroupID == nil {
			warnings = append(warnings, "getting origin servers: got server with nil CachegroupID, skipping!")
			continue
		} else if cgSv.StatusID == nil {
			warnings = append(warnings, "getting origin servers: got server with nil StatusID, skipping!")
			continue
		} else if cgSv.TypeID == nil {
			warnings = append(warnings, "getting origin servers: got server with nil TypeID, skipping!")
			continue
		} else if cgSv.ProfileID == nil {
			warnings = append(warnings, "getting origin servers: got server with nil ProfileID, skipping!")
			continue
		} else if cgSv.CDNID == nil {
			warnings = append(warnings, "getting origin servers: got server with nil CDNID, skipping!")
			continue
		} else if cgSv.DomainName == nil {
			warnings = append(warnings, "getting origin servers: got server with nil DomainName, skipping!")
			continue
		}

		ipAddr := getServerIPAddress(&cgSv)
		if ipAddr == nil {
			warnings = append(warnings, "getting origin servers: got server with no valid IP Address, skipping!")
			continue
		}

		realCGServer := cgServer{
			ServerID:     ServerID(*cgSv.ID),
			ServerHost:   *cgSv.HostName,
			ServerIP:     ipAddr.String(),
			ServerPort:   *cgSv.TCPPort,
			CacheGroupID: *cgSv.CachegroupID,
			Status:       *cgSv.StatusID,
			Type:         *cgSv.TypeID,
			ProfileID:    ProfileID(*cgSv.ProfileID),
			CDN:          *cgSv.CDNID,
			TypeName:     cgSv.Type,
			Domain:       *cgSv.DomainName,
			Capabilities: serverCapabilities[*cgSv.ID],
		}

		if cgSv.Type == tc.OriginTypeName {
			for dsID, _ := range parentServerDSes[*cgSv.ID] { // map[serverID][]dsID
				orgURI := dsOrigins[dsID]
				if orgURI == nil {
					// warnings = append(warnings, fmt.Sprintf(("ds %v has no origins! Skipping!\n", dsID) // TODO determine if this is normal
					continue
				}
				orgHost := OriginHost(orgURI.Host)
				originServers[orgHost] = append(originServers[orgHost], realCGServer)
			}
		} else {
			originServers[deliveryServicesAllParentsKey] = append(originServers[deliveryServicesAllParentsKey], realCGServer)
		}

		if _, profileCachesHasProfile := profileCaches[realCGServer.ProfileID]; !profileCachesHasProfile {
			if profileCache, profileParamsHasProfile := profileParams[*cgSv.Profile]; !profileParamsHasProfile {
				warnings = append(warnings, fmt.Sprintf("cachegroup has server with profile %+v but that profile has no parameters\n", *cgSv.ProfileID))
				profileCaches[realCGServer.ProfileID] = defaultProfileCache()
			} else {
				profileCaches[realCGServer.ProfileID] = profileCache
			}
		}
	}

	return originServers, profileCaches, warnings, nil
}

// GetParentConfigProfileParams returns the parent config profile params, and any warnings.
func getParentConfigProfileParams(
	cgServers map[int]Server,
	profileParentConfigParams map[string]map[string]string, // map[profileName][paramName]paramVal
) (map[string]profileCache, []string) {
	warnings := []string{}
	parentConfigServerCacheProfileParams := map[string]profileCache{} // map[profileName]ProfileCache
	for _, cgServer := range cgServers {
		if cgServer.Profile == nil {
			warnings = append(warnings, "getting parent config profile params: server has nil profile, skipping!")
			continue
		}
		profileCache, ok := parentConfigServerCacheProfileParams[*cgServer.Profile]
		if !ok {
			profileCache = defaultProfileCache()
		}
		params, ok := profileParentConfigParams[*cgServer.Profile]
		if !ok {
			parentConfigServerCacheProfileParams[*cgServer.Profile] = profileCache
			continue
		}
		for name, val := range params {
			switch name {
			case ParentConfigCacheParamWeight:
				// f, err := strconv.ParseFloat(param.Val, 64)
				// if err != nil {
				// 	warnings = append(warnings, "parent.config generation: weight param is not a float, skipping! : " + err.Error())
				// } else {
				// 	profileCache.Weight = f
				// }
				// TODO validate float?
				profileCache.Weight = val
			case ParentConfigCacheParamPort:
				if i, err := strconv.Atoi(val); err != nil {
					warnings = append(warnings, "port param is not an integer, skipping! : "+err.Error())
				} else {
					profileCache.Port = i
				}
			case ParentConfigCacheParamUseIP:
				profileCache.UseIP = val == "1"
			case ParentConfigCacheParamRank:
				if i, err := strconv.Atoi(val); err != nil {
					warnings = append(warnings, "rank param is not an integer, skipping! : "+err.Error())
				} else {
					profileCache.Rank = i
				}
			case ParentConfigCacheParamNotAParent:
				profileCache.NotAParent = val != "false"
			}
		}
		parentConfigServerCacheProfileParams[*cgServer.Profile] = profileCache
	}
	return parentConfigServerCacheProfileParams, warnings
}

// getDSOrigins takes a map[deliveryServiceID]DeliveryService, and returns a map[DeliveryServiceID]OriginURI, any warnings, and any error.
func getDSOrigins(dses map[int]DeliveryService) (map[int]*originURI, []string, error) {
	warnings := []string{}
	dsOrigins := map[int]*originURI{}
	for _, ds := range dses {
		if ds.ID == nil {
			return nil, warnings, errors.New("ds has nil ID")
		}
		if ds.XMLID == nil {
			return nil, warnings, errors.New("ds has nil XMLID")
		}
		if ds.OrgServerFQDN == nil {
			warnings = append(warnings, fmt.Sprintf("GetDSOrigins ds %v got nil OrgServerFQDN, skipping!\n", *ds.XMLID))
			continue
		}
		orgURL, err := url.Parse(*ds.OrgServerFQDN)
		if err != nil {
			return nil, warnings, errors.New("parsing ds '" + *ds.XMLID + "' OrgServerFQDN '" + *ds.OrgServerFQDN + "': " + err.Error())
		}
		if orgURL.Scheme == "" {
			return nil, warnings, errors.New("parsing ds '" + *ds.XMLID + "' OrgServerFQDN '" + *ds.OrgServerFQDN + "': " + "missing scheme")
		}
		if orgURL.Host == "" {
			return nil, warnings, errors.New("parsing ds '" + *ds.XMLID + "' OrgServerFQDN '" + *ds.OrgServerFQDN + "': " + "missing scheme")
		}

		scheme := orgURL.Scheme
		host := orgURL.Hostname()
		port := orgURL.Port()
		if port == "" {
			if scheme == "http" {
				port = "80"
			} else if scheme == "https" {
				port = "443"
			} else {
				warnings = append(warnings, "parsing ds '"+*ds.XMLID+"' OrgServerFQDN '"+*ds.OrgServerFQDN+"': "+"unknown scheme '"+scheme+"' and no port, leaving port empty!")
			}
		}
		dsOrigins[*ds.ID] = &originURI{Scheme: scheme, Host: host, Port: port}
	}
	return dsOrigins, warnings, nil
}

// makeDSOrigins returns the DS Origins and any warnings.
func makeDSOrigins(dsses []DeliveryServiceServer, dses []DeliveryService, servers []Server) (map[DeliveryServiceID]map[ServerID]struct{}, []string) {
	warnings := []string{}
	dssMap := map[DeliveryServiceID]map[ServerID]struct{}{}
	for _, dss := range dsses {
		dsID := DeliveryServiceID(dss.DeliveryService)
		serverID := ServerID(dss.Server)
		if dssMap[dsID] == nil {
			dssMap[dsID] = map[ServerID]struct{}{}
		}
		dssMap[dsID][serverID] = struct{}{}
	}

	svMap := map[ServerID]Server{}
	for _, sv := range servers {
		if sv.ID == nil {
			warnings = append(warnings, "got server with missing ID, skipping!")
		}
		svMap[ServerID(*sv.ID)] = sv
	}

	dsOrigins := map[DeliveryServiceID]map[ServerID]struct{}{}
	for _, ds := range dses {
		if ds.ID == nil {
			warnings = append(warnings, "got ds with missing ID, skipping!")
			continue
		}
		dsID := DeliveryServiceID(*ds.ID)
		assignedServers := dssMap[dsID]
		for svID, _ := range assignedServers {
			sv := svMap[svID]
			if sv.Type != tc.OriginTypeName {
				continue
			}
			if dsOrigins[dsID] == nil {
				dsOrigins[dsID] = map[ServerID]struct{}{}
			}
			dsOrigins[dsID][svID] = struct{}{}
		}
	}
	return dsOrigins, warnings
}
