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
	"errors"
	"github.com/apache/trafficcontrol/lib/go-log"
	"sort"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
)

const CacheKeyParameterConfigFile = "cachekey.config"
const ContentTypeRemapDotConfig = ContentTypeTextASCII
const LineCommentRemapDotConfig = LineCommentHash

const RemapConfigRangeDirective = `__RANGE_DIRECTIVE__`

func MakeRemapDotConfig(
	server *Server,
	unfilteredDSes []DeliveryService,
	dss []DeliveryServiceServer,
	dsRegexArr []tc.DeliveryServiceRegexes,
	serverParams []tc.Parameter,
	cdn *tc.CDN,
	remapConfigParams []tc.Parameter, // includes cachekey.config
	topologies []tc.Topology,
	cacheGroupArr []tc.CacheGroupNullable,
	serverCapabilities map[int]map[ServerCapability]struct{},
	dsRequiredCapabilities map[int]map[ServerCapability]struct{},
	hdrComment string,
) (Cfg, error) {
	warnings := []string{}
	if server.HostName == nil {
		return Cfg{}, makeErr(warnings, "server HostName missing")
	} else if server.ID == nil {
		return Cfg{}, makeErr(warnings, "server ID missing")
	} else if server.Cachegroup == nil {
		return Cfg{}, makeErr(warnings, "server Cachegroup missing")
	} else if server.DomainName == nil {
		return Cfg{}, makeErr(warnings, "server DomainName missing")
	}

	cdnDomain := cdn.DomainName
	dsRegexes := makeDSRegexMap(dsRegexArr)
	// Returned DSes are guaranteed to have a non-nil XMLID, Type, DSCP, ID, and Active.
	dses, dsWarns := remapFilterDSes(server, dss, unfilteredDSes)
	warnings = append(warnings, dsWarns...)

	dsProfilesConfigParams, paramWarns, err := makeDSProfilesConfigParams(server, dses, remapConfigParams)
	warnings = append(warnings, paramWarns...)
	if err != nil {
		warnings = append(warnings, "making Delivery Service Cache Key Params, cache key will be missing! : "+err.Error())
	}

	atsMajorVersion, verWarns := getATSMajorVersion(serverParams)
	warnings = append(warnings, verWarns...)
	serverPackageParamData, paramWarns := makeServerPackageParamData(server, serverParams)
	warnings = append(warnings, paramWarns...)
	cacheGroups, err := makeCGMap(cacheGroupArr)
	if err != nil {
		return Cfg{}, makeErr(warnings, "making remap.config, config will be malformed! : "+err.Error())
	}

	nameTopologies := makeTopologyNameMap(topologies)

	hdr := makeHdrComment(hdrComment)
	txt := ""
	typeWarns := []string{}
	if tc.CacheTypeFromString(server.Type) == tc.CacheTypeMid {
		txt, typeWarns, err = getServerConfigRemapDotConfigForMid(atsMajorVersion, dsProfilesConfigParams, dses, dsRegexes, hdr, server, nameTopologies, cacheGroups, serverCapabilities, dsRequiredCapabilities)
	} else {
		txt, typeWarns, err = getServerConfigRemapDotConfigForEdge(dsProfilesConfigParams, serverPackageParamData, dses, dsRegexes, atsMajorVersion, hdr, server, nameTopologies, cacheGroups, serverCapabilities, dsRequiredCapabilities, cdnDomain)
	}
	warnings = append(warnings, typeWarns...)
	if err != nil {
		return Cfg{}, makeErr(warnings, err.Error()) // the GetFor funcs include error context
	}

	return Cfg{
		Text:        txt,
		ContentType: ContentTypeRemapDotConfig,
		LineComment: LineCommentRemapDotConfig,
		Warnings:    warnings,
	}, nil
}

// key is '<plugin>.pparam' or 'cachekey.config'
func classifyConfigParams(configParams []tc.Parameter) map[string][]tc.Parameter {
	var configParamMap = map[string][]tc.Parameter{}
	for _, param := range configParams {
		key := param.ConfigFile
		if "remap.config" == key {
			key = param.Name
		}
		configParamMap[key] = append(configParamMap[key], param)
	}
	return configParamMap
}

// For general <plugin>.pparam parameters
func paramsStringFor(parameters []tc.Parameter, warnings *[]string) (paramsString string) {
	uniquemap := map[string]int{}

	for _, param := range parameters {
		paramsString += " @pparam=" + param.Value

		// Try to extract argument
		index := strings.IndexAny(param.Value, "= ")
		var arg string
		if 0 < index {
			arg = param.Value[:index]
		} else {
			arg = param.Value
		}

		// Warn on detection, but don't correct
		if _, exists := uniquemap[arg]; !exists {
			uniquemap[arg] = 1
		} else {
			*warnings = append(*warnings, "Multiple repeated arguments '"+arg+"'")
		}
	}
	return paramsString
}

// for cachekey.config parameters
func paramsStringOldFor(parameters []tc.Parameter, warnings *[]string) (paramsString string) {
	// check for duplicate parameters
	uniquemap := map[string]int{}
	paramKeyVals := []keyVal{}
	for _, param := range parameters {
		key := param.Name
		val := param.Value

		if _, exists := uniquemap[key]; !exists {
			uniquemap[key] = 1
			paramKeyVals = append(paramKeyVals, keyVal{Key: key, Val: val})
		} else {
			uniquemap[key]++
			*warnings = append(*warnings, "got multiple parameters for name '"+key+"' - ignoring '"+val+"'")
		}
	}

	sort.Sort(keyVals(paramKeyVals))
	for _, keyVal := range paramKeyVals {
		paramsString += " @pparam=--" + keyVal.Key + "=" + keyVal.Val
	}

	return paramsString
}

// getServerConfigRemapDotConfigForMid returns the remap lines, any warnings, and any error.
func getServerConfigRemapDotConfigForMid(
	atsMajorVersion int,
	profilesConfigParams map[int][]tc.Parameter,
	dses []DeliveryService,
	dsRegexes map[tc.DeliveryServiceName][]tc.DeliveryServiceRegex,
	header string,
	server *Server,
	nameTopologies map[TopologyName]tc.Topology,
	cacheGroups map[tc.CacheGroupName]tc.CacheGroupNullable,
	serverCapabilities map[int]map[ServerCapability]struct{},
	dsRequiredCapabilities map[int]map[ServerCapability]struct{},
) (string, []string, error) {
	warnings := []string{}
	midRemaps := map[string]string{}
	for _, ds := range dses {
		if !hasRequiredCapabilities(serverCapabilities[*server.ID], dsRequiredCapabilities[*ds.ID]) {
			continue
		}

		topology, hasTopology := nameTopologies[TopologyName(*ds.Topology)]
		if *ds.Topology != "" && hasTopology {
			topoIncludesServer, err := topologyIncludesServerNullable(topology, server)
			if err != nil {
				return "", warnings, errors.New("getting Topology Server inclusion: " + err.Error())
			}
			if !topoIncludesServer {
				continue
			}
		}

		if !ds.Type.UsesMidCache() && (!hasTopology || *ds.Topology == "") {
			continue // Live local delivery services skip mids (except Topologies ignore DS types)
		}

		if ds.OrgServerFQDN == nil || *ds.OrgServerFQDN == "" {
			warnings = append(warnings, "ds '"+*ds.XMLID+"' has no origin fqdn, skipping!") // TODO confirm - Perl uses without checking!
			continue
		}

		if midRemaps[*ds.OrgServerFQDN] != "" {
			continue // skip remap rules from extra HOST_REGEXP entries
		}

		// multiple uses of cachekey plugins don't work right in ATS, but Perl has always done it.
		// So for now, keep track of it, so we can log an error when it happens.
		hasCacheKey := false

		midRemap := ""

		if *ds.Topology != "" {
			topoTxt, err := makeDSTopologyHeaderRewriteTxt(ds, tc.CacheGroupName(*server.Cachegroup), topology, cacheGroups)
			if err != nil {
				return "", warnings, err
			}
			midRemap += topoTxt
		} else if (ds.MidHeaderRewrite != nil && *ds.MidHeaderRewrite != "") || (ds.MaxOriginConnections != nil && *ds.MaxOriginConnections > 0) || (ds.ServiceCategory != nil && *ds.ServiceCategory != "") {
			midRemap += ` @plugin=header_rewrite.so @pparam=` + midHeaderRewriteConfigFileName(*ds.XMLID)
		}

		if ds.QStringIgnore != nil && *ds.QStringIgnore == tc.QueryStringIgnoreIgnoreInCacheKeyAndPassUp {
			qstr, addedCacheKey := getQStringIgnoreRemap(atsMajorVersion)
			if addedCacheKey {
				hasCacheKey = true
			}
			midRemap += qstr
		}

		var dsConfigParamsMap map[string][]tc.Parameter

		if nil != ds.ProfileID {
			dsConfigParamsMap = classifyConfigParams(profilesConfigParams[*ds.ProfileID])
			var cachekeyParams string

			// Prefer to use cachekey.pparam
			if params, ok := dsConfigParamsMap["cachekey.pparam"]; ok {
				cachekeyParams = paramsStringFor(params, &warnings)

				if _, ok := dsConfigParamsMap["cachekey.config"]; ok {
					warnings = append(warnings, "Delivery Service '"+*ds.XMLID+"': has both old cachekey.config and new cachekey.pparam parameters assigned, ignoring old cachekey.config ones")
				}
			} else if params, ok := dsConfigParamsMap["cachekey.config"]; ok {
				cachekeyParams = paramsStringOldFor(params, &warnings)
			}

			if cachekeyParams != "" {
				if hasCacheKey {
					warnings = append(warnings, "Delivery Service '"+*ds.XMLID+"': qstring_ignore and cachekey params both add cachekey, but ATS cachekey doesn't work correctly with multiple entries! Adding anyway!")
				}
				midRemap += " @plugin=cachekey.so" + cachekeyParams
			}
		}

		if ds.RangeRequestHandling != nil && (*ds.RangeRequestHandling == tc.RangeRequestHandlingCacheRangeRequest || *ds.RangeRequestHandling == tc.RangeRequestHandlingSlice) {
			midRemap += " @plugin=cache_range_requests.so" +
				paramsStringFor(dsConfigParamsMap["cache_range_requests.pparam"], &warnings)
		}

		if midRemap != "" {
			midRemaps[*ds.OrgServerFQDN] = midRemap
		}
	}

	textLines := []string{}
	for originFQDN, midRemap := range midRemaps {
		textLines = append(textLines, "map "+originFQDN+" "+originFQDN+midRemap+"\n")
	}
	sort.Strings(textLines)

	text := header
	text += strings.Join(textLines, "")
	return text, warnings, nil
}

// getServerConfigRemapDotConfigForEdge returns the remap lines, any warnings, and any error.
func getServerConfigRemapDotConfigForEdge(
	profilesRemapConfigParams map[int][]tc.Parameter,
	serverPackageParamData map[string]string, // map[paramName]paramVal for this server, config file 'package'
	dses []DeliveryService,
	dsRegexes map[tc.DeliveryServiceName][]tc.DeliveryServiceRegex,
	atsMajorVersion int,
	header string,
	server *Server,
	nameTopologies map[TopologyName]tc.Topology,
	cacheGroups map[tc.CacheGroupName]tc.CacheGroupNullable,
	serverCapabilities map[int]map[ServerCapability]struct{},
	dsRequiredCapabilities map[int]map[ServerCapability]struct{},
	cdnDomain string,
) (string, []string, error) {
	warnings := []string{}
	textLines := []string{}

	for _, ds := range dses {
		if !hasRequiredCapabilities(serverCapabilities[*server.ID], dsRequiredCapabilities[*ds.ID]) {
			continue
		}

		topology, hasTopology := nameTopologies[TopologyName(*ds.Topology)]
		if *ds.Topology != "" && hasTopology {
			topoIncludesServer, err := topologyIncludesServerNullable(topology, server)
			if err != nil {
				return "", warnings, errors.New("getting topology server inclusion: " + err.Error())
			}
			if !topoIncludesServer {
				continue
			}
		}
		remapText := ""
		if *ds.Type == tc.DSTypeAnyMap {
			if ds.RemapText == nil {
				warnings = append(warnings, "ds '"+*ds.XMLID+"' is ANY_MAP, but has no remap text - skipping")
				continue
			}
			remapText = *ds.RemapText + "\n"
			textLines = append(textLines, remapText)
			continue
		}

		requestFQDNs, err := getDSRequestFQDNs(&ds, dsRegexes[tc.DeliveryServiceName(*ds.XMLID)], server, cdnDomain)
		if err != nil {
			warnings = append(warnings, "error getting ds '"+*ds.XMLID+"' request fqdns, skipping! Error: "+err.Error())
			continue
		}

		for _, requestFQDN := range requestFQDNs {
			remapLines, err := makeEdgeDSDataRemapLines(ds, requestFQDN, server, cdnDomain)
			if err != nil {
				warnings = append(warnings, "DS '"+*ds.XMLID+"' - skipping! : "+err.Error())
				continue
			}

			for _, line := range remapLines {
				profileremapConfigParams := []tc.Parameter{}
				if ds.ProfileID != nil {
					profileremapConfigParams = profilesRemapConfigParams[*ds.ProfileID]
				}
				remapWarns := []string{}
				remapText, remapWarns, err = buildEdgeRemapLine(atsMajorVersion, server, serverPackageParamData, remapText, ds, line.From, line.To, profileremapConfigParams, cacheGroups, nameTopologies)
				warnings = append(warnings, remapWarns...)

				if err != nil {
					return "", warnings, err
				}
				if hasTopology {
					remapText += " # topology '" + topology.Name + "'"
				}
				remapText += "\n"
			}
		}

		textLines = append(textLines, remapText)
	}

	text := header
	sort.Strings(textLines)
	text += strings.Join(textLines, "")
	return text, warnings, nil
}

// buildEdgeRemapLine builds the remap line for the given server and delivery service.
// The cacheKeyConfigParams map may be nil, if this ds profile had no cache key config params.
// Returns the remap line, any warnings, and any error.
func buildEdgeRemapLine(
	atsMajorVersion int,
	server *Server,
	pData map[string]string,
	text string,
	ds DeliveryService,
	mapFrom string,
	mapTo string,
	remapConfigParams []tc.Parameter,
	cacheGroups map[tc.CacheGroupName]tc.CacheGroupNullable,
	nameTopologies map[TopologyName]tc.Topology,
) (string, []string, error) {
	warnings := []string{}
	// ds = 'remap' in perl
	mapFrom = strings.Replace(mapFrom, `__http__`, *server.HostName, -1)

	if _, hasDSCPRemap := pData["dscp_remap"]; hasDSCPRemap {
		text += "map	" + mapFrom + "     " + mapTo + ` @plugin=dscp_remap.so @pparam=` + strconv.Itoa(*ds.DSCP)
	} else {
		text += "map	" + mapFrom + "     " + mapTo + ` @plugin=header_rewrite.so @pparam=dscp/set_dscp_` + strconv.Itoa(*ds.DSCP) + ".config"
	}

	if *ds.Topology != "" {
		topoTxt, err := makeDSTopologyHeaderRewriteTxt(ds, tc.CacheGroupName(*server.Cachegroup), nameTopologies[TopologyName(*ds.Topology)], cacheGroups)
		if err != nil {
			return "", warnings, err
		}
		text += topoTxt
	} else if (ds.EdgeHeaderRewrite != nil && *ds.EdgeHeaderRewrite != "") || (ds.ServiceCategory != nil && *ds.ServiceCategory != "") || (ds.MaxOriginConnections != nil && *ds.MaxOriginConnections != 0) {
		text += ` @plugin=header_rewrite.so @pparam=` + edgeHeaderRewriteConfigFileName(*ds.XMLID)
	}

	dsConfigParamsMap := classifyConfigParams(remapConfigParams)

	if ds.SigningAlgorithm != nil && *ds.SigningAlgorithm != "" {
		if *ds.SigningAlgorithm == tc.SigningAlgorithmURLSig {
			text += ` @plugin=url_sig.so @pparam=url_sig_` + *ds.XMLID + ".config" +
				paramsStringFor(dsConfigParamsMap["url_sig.pparam"], &warnings)
		} else if *ds.SigningAlgorithm == tc.SigningAlgorithmURISigning {
			text += ` @plugin=uri_signing.so @pparam=uri_signing_` + *ds.XMLID + ".config" +
				paramsStringFor(dsConfigParamsMap["uri_signing.pparam"], &warnings)
		}
	}

	// multiple uses of cachekey plugins don't work right in ATS, but Perl has always done it.
	// So for now, keep track of it, so we can log an error when it happens.
	hasCacheKey := false

	if ds.QStringIgnore != nil {
		if *ds.QStringIgnore == tc.QueryStringIgnoreDropAtEdge {
			dqsFile := "drop_qstring.config"
			text += ` @plugin=regex_remap.so @pparam=` + dqsFile
		} else if *ds.QStringIgnore == tc.QueryStringIgnoreIgnoreInCacheKeyAndPassUp {
			qstr, addedCacheKey := getQStringIgnoreRemap(atsMajorVersion)
			if addedCacheKey {
				hasCacheKey = true
			}
			text += qstr
		}
	}

	var cachekeyParams string

	// Prefer to use cachekey.pparam
	if params, ok := dsConfigParamsMap["cachekey.pparam"]; ok {
		cachekeyParams = paramsStringFor(params, &warnings)

		if _, ok := dsConfigParamsMap["cachekey.config"]; ok {
			warnings = append(warnings, "Delivery Service '"+*ds.XMLID+"': has both old cachekey.config and new cachekey.pparam parameters assigned, ignoring old cachekey.config ones")
		}
	} else if params, ok := dsConfigParamsMap["cachekey.config"]; ok {
		cachekeyParams = paramsStringOldFor(params, &warnings)
	}

	if cachekeyParams != "" {
		if hasCacheKey {
			warnings = append(warnings, "Delivery Service '"+*ds.XMLID+"': qstring_ignore and cachekey params both add cachekey, but ATS cachekey doesn't work correctly with multiple entries! Adding anyway!")
		}
		text += " @plugin=cachekey.so" + cachekeyParams
	}

	// Note: should use full path here?
	if ds.RegexRemap != nil && *ds.RegexRemap != "" {
		text += ` @plugin=regex_remap.so @pparam=regex_remap_` + *ds.XMLID + ".config"
	}

	rangeReqTxt := ""
	if ds.RangeRequestHandling != nil {
		crr := false

		if *ds.RangeRequestHandling == tc.RangeRequestHandlingBackgroundFetch {
			rangeReqTxt = " @plugin=background_fetch.so @pparam=--config=bg_fetch.config" +
				paramsStringFor(dsConfigParamsMap["background_fetch.pparam"], &warnings)
		} else if *ds.RangeRequestHandling == tc.RangeRequestHandlingSlice && ds.RangeSliceBlockSize != nil {

			rangeReqTxt = " @plugin=slice.so @pparam=--blockbytes=" + strconv.Itoa(*ds.RangeSliceBlockSize) +
				paramsStringFor(dsConfigParamsMap["slice.pparam"], &warnings)
			crr = true
		} else if *ds.RangeRequestHandling == tc.RangeRequestHandlingCacheRangeRequest {
			crr = true
		}

		if crr {
			rangeReqTxt += " @plugin=cache_range_requests.so " +
				paramsStringFor(dsConfigParamsMap["cache_range_requests.pparam"], &warnings)
		}
	}

	remapText := ""
	if ds.RemapText != nil {
		remapText = *ds.RemapText
	}

	// Temporary hack for moving the range directive into the raw remap text
	if strings.Contains(remapText, RemapConfigRangeDirective) {
		remapText = strings.Replace(remapText, RemapConfigRangeDirective, rangeReqTxt, 1)
	} else {
		text += rangeReqTxt
	}

	if remapText != "" {
		text += " " + remapText
	}

	if ds.FQPacingRate != nil && *ds.FQPacingRate > 0 {
		text += ` @plugin=fq_pacing.so @pparam=--rate=` + strconv.Itoa(*ds.FQPacingRate)
	}

	return text, warnings, nil
}

// makeDSTopologyHeaderRewriteTxt returns the appropriate header rewrite remap line text for the given DS on the given server, and any error.
// May be empty, if the DS has no header rewrite for the server's position in the topology.
func makeDSTopologyHeaderRewriteTxt(ds DeliveryService, cg tc.CacheGroupName, topology tc.Topology, cacheGroups map[tc.CacheGroupName]tc.CacheGroupNullable) (string, error) {
	placement, err := getTopologyPlacement(cg, topology, cacheGroups, &ds)
	if err != nil {
		return "", errors.New("getting topology placement: " + err.Error())
	}
	txt := ""
	const pluginTxt = ` @plugin=header_rewrite.so @pparam=`
	if placement.IsFirstCacheTier && ((ds.FirstHeaderRewrite != nil && *ds.FirstHeaderRewrite != "") || (ds.ServiceCategory != nil && *ds.ServiceCategory != "")) {
		txt += pluginTxt + FirstHeaderRewriteConfigFileName(*ds.XMLID) + ` `
	}
	if placement.IsInnerCacheTier && ((ds.InnerHeaderRewrite != nil && *ds.InnerHeaderRewrite != "") || (ds.ServiceCategory != nil && *ds.ServiceCategory != "")) {
		txt += pluginTxt + InnerHeaderRewriteConfigFileName(*ds.XMLID) + ` `
	}
	if placement.IsLastCacheTier && ((ds.LastHeaderRewrite != nil && *ds.LastHeaderRewrite != "") || (ds.ServiceCategory != nil && *ds.ServiceCategory != "") || (ds.MaxOriginConnections != nil && *ds.MaxOriginConnections != 0)) {
		txt += pluginTxt + LastHeaderRewriteConfigFileName(*ds.XMLID) + ` `
	}
	return txt, nil
}

type remapLine struct {
	From string
	To   string
}

// makeEdgeDSDataRemapLines returns the remap lines for the given server and delivery service.
// Returns nil, if the given server and ds have no remap lines, i.e. the DS match is not a host regex, or has no origin FQDN.
func makeEdgeDSDataRemapLines(
	ds DeliveryService,
	requestFQDN string,
	//	dsRegex tc.DeliveryServiceRegex,
	server *Server,
	cdnDomain string,
) ([]remapLine, error) {
	if ds.Protocol == nil {
		return nil, errors.New("ds had nil protocol")
	}

	remapLines := []remapLine{}
	mapTo := *ds.OrgServerFQDN + "/"

	portStr := ""
	if !ds.Type.IsDNS() && server.TCPPort != nil && *server.TCPPort > 0 && *server.TCPPort != 80 {
		portStr = ":" + strconv.Itoa(*server.TCPPort)
	}

	httpsPortStr := ""
	if !ds.Type.IsDNS() && server.HTTPSPort != nil && *server.HTTPSPort > 0 && *server.HTTPSPort != 443 {
		httpsPortStr = ":" + strconv.Itoa(*server.HTTPSPort)
	}

	mapFromHTTP := "http://" + requestFQDN + portStr + "/"
	mapFromHTTPS := "https://" + requestFQDN + httpsPortStr + "/"

	if *ds.Protocol == tc.DSProtocolHTTP || *ds.Protocol == tc.DSProtocolHTTPAndHTTPS {
		remapLines = append(remapLines, remapLine{From: mapFromHTTP, To: mapTo})
	}
	if *ds.Protocol == tc.DSProtocolHTTPS || *ds.Protocol == tc.DSProtocolHTTPToHTTPS || *ds.Protocol == tc.DSProtocolHTTPAndHTTPS {
		remapLines = append(remapLines, remapLine{From: mapFromHTTPS, To: mapTo})
	}

	return remapLines, nil
}

func edgeHeaderRewriteConfigFileName(dsName string) string {
	return "hdr_rw_" + dsName + ".config"
}

func midHeaderRewriteConfigFileName(dsName string) string {
	return "hdr_rw_mid_" + dsName + ".config"
}

// getQStringIgnoreRemap returns the remap, whether cachekey was added.
func getQStringIgnoreRemap(atsMajorVersion int) (string, bool) {
	if atsMajorVersion < 7 {
		log.Errorf("Unsupport version of ats found %v", atsMajorVersion)
		return "", false
	}
	return ` @plugin=cachekey.so @pparam=--separator= @pparam=--remove-all-params=true @pparam=--remove-path=true @pparam=--capture-prefix-uri=/^([^?]*)/$1/`, true
}

// makeServerPackageParamData returns a map[paramName]paramVal for this server, config file 'package'.
// Returns the param data, and any warnings
func makeServerPackageParamData(server *Server, serverParams []tc.Parameter) (map[string]string, []string) {
	warnings := []string{}

	serverPackageParamData := map[string]string{}
	for _, param := range serverParams {
		if param.ConfigFile != "package" { // TODO put in const
			continue
		}
		if param.Name == "location" { // TODO put in const
			continue
		}

		paramName := param.Name
		// some files have multiple lines with the same key... handle that with param id.
		if _, ok := serverPackageParamData[param.Name]; ok {
			paramName += "__" + strconv.Itoa(param.ID)
		}
		paramValue := param.Value
		if paramValue == "STRING __HOSTNAME__" {
			paramValue = *server.HostName + "." + *server.DomainName // TODO strings.Replace to replace all anywhere, instead of just an exact match?
		}

		if val, ok := serverPackageParamData[paramName]; ok {
			if val < paramValue {
				warnings = append(warnings, "got multiple parameters for server package name '"+paramName+"' - ignoring '"+paramValue+"'")
				continue
			} else {
				warnings = append(warnings, "got multiple parameters for server package name '"+paramName+"' - ignoring '"+val+"'")
			}
		}
		serverPackageParamData[paramName] = paramValue
	}
	return serverPackageParamData, warnings
}

// remapFilterDSes filters Delivery Services to be used to generate remap.config for the given server.
// Returned DSes are guaranteed to have a non-nil XMLID, Type, DSCP, ID, Active, and Topology.
// If a DS has a nil Topology, OrgServerFQDN, FirstHeaderRewrite, InnerHeaderRewrite, or LastHeaderRewrite, "" is assigned.
// Returns the filtered delivery services, and any warnings
func remapFilterDSes(server *Server, dss []DeliveryServiceServer, dses []DeliveryService) ([]DeliveryService, []string) {
	warnings := []string{}
	isMid := strings.HasPrefix(server.Type, string(tc.CacheTypeMid))

	serverIDs := map[int]struct{}{}
	if !isMid {
		// mids use all servers, so pass empty=all. Edges only use this current server
		serverIDs[*server.ID] = struct{}{}
	}

	dsIDs := map[int]struct{}{}
	for _, ds := range dses {
		if ds.ID == nil {
			// TODO log error?
			continue
		}
		dsIDs[*ds.ID] = struct{}{}
	}

	dsServers := filterDSS(dss, dsIDs, serverIDs)

	dssMap := map[int]map[int]struct{}{} // set of map[dsID][serverID]
	for _, dss := range dsServers {
		if dssMap[dss.DeliveryService] == nil {
			dssMap[dss.DeliveryService] = map[int]struct{}{}
		}
		dssMap[dss.DeliveryService][dss.Server] = struct{}{}
	}

	useInactive := false
	if !isMid {
		// mids get inactive DSes, edges don't. This is how it's always behaved, not necessarily how it should.
		useInactive = true
	}

	filteredDSes := []DeliveryService{}
	for _, ds := range dses {
		if ds.Topology == nil {
			ds.Topology = util.StrPtr("")
		}
		if ds.OrgServerFQDN == nil {
			ds.OrgServerFQDN = util.StrPtr("")
		}
		if ds.FirstHeaderRewrite == nil {
			ds.FirstHeaderRewrite = util.StrPtr("")
		}
		if ds.InnerHeaderRewrite == nil {
			ds.InnerHeaderRewrite = util.StrPtr("")
		}
		if ds.LastHeaderRewrite == nil {
			ds.LastHeaderRewrite = util.StrPtr("")
		}
		if ds.XMLID == nil {
			warnings = append(warnings, "got Delivery Service with nil XMLID, skipping!")
			continue
		} else if ds.Type == nil {
			warnings = append(warnings, "got Delivery Service '"+*ds.XMLID+"'  with nil Type, skipping!")
			continue
		} else if ds.DSCP == nil {
			warnings = append(warnings, "got Delivery Service '"+*ds.XMLID+"'  with nil DSCP, skipping!")
			continue
		} else if ds.ID == nil {
			warnings = append(warnings, "got Delivery Service '"+*ds.XMLID+"'  with nil ID, skipping!")
			continue
		} else if ds.Active == nil {
			warnings = append(warnings, "got Delivery Service '"+*ds.XMLID+"'  with nil Active, skipping!")
			continue
		} else if _, ok := dssMap[*ds.ID]; !ok && *ds.Topology == "" {
			continue // normal, not an error, this DS just isn't assigned to our Cache
		} else if !useInactive && !*ds.Active {
			continue // normal, not an error, DS just isn't active and we aren't including inactive DSes
		}
		filteredDSes = append(filteredDSes, ds)
	}
	return filteredDSes, warnings
}

// makeDSProfilesConfigParams returns a map[ProfileID][ParamName]ParamValue for the cache key params for each profile.
// Returns the params, any warnings, and any error.
func makeDSProfilesConfigParams(server *Server, dses []DeliveryService, remapConfigParams []tc.Parameter) (map[int][]tc.Parameter, []string, error) {
	warnings := []string{}
	dsConfigParamsWithProfiles, err := tcParamsToParamsWithProfiles(remapConfigParams)
	if err != nil {
		return nil, warnings, errors.New("decoding cache key parameter profiles: " + err.Error())
	}

	configParamsWithProfilesMap := parameterWithProfilesToMap(dsConfigParamsWithProfiles)

	dsProfileNamesToIDs := map[string]int{}
	for _, ds := range dses {
		if ds.ProfileID == nil || ds.ProfileName == nil {
			continue // TODO log
		}
		dsProfileNamesToIDs[*ds.ProfileName] = *ds.ProfileID
	}

	dsProfilesConfigParams := map[int][]tc.Parameter{}
	for _, param := range configParamsWithProfilesMap {
		for dsProfileName, dsProfileID := range dsProfileNamesToIDs {
			if _, ok := param.ProfileNames[dsProfileName]; ok {
				if _, ok := dsProfilesConfigParams[dsProfileID]; !ok {
					dsProfilesConfigParams[dsProfileID] = []tc.Parameter{}
				}
				dsProfilesConfigParams[dsProfileID] = append(dsProfilesConfigParams[dsProfileID], param.Parameter)
			}
		}
	}
	return dsProfilesConfigParams, warnings, nil
}

type deliveryServiceRegexesSortByTypeThenSetNum []tc.DeliveryServiceRegex

func (r deliveryServiceRegexesSortByTypeThenSetNum) Len() int { return len(r) }
func (r deliveryServiceRegexesSortByTypeThenSetNum) Less(i, j int) bool {
	if rc := strings.Compare(r[i].Type, r[j].Type); rc != 0 {
		return rc < 0
	}
	return r[i].SetNumber < r[j].SetNumber
}
func (r deliveryServiceRegexesSortByTypeThenSetNum) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

func makeDSRegexMap(regexes []tc.DeliveryServiceRegexes) map[tc.DeliveryServiceName][]tc.DeliveryServiceRegex {
	dsRegexMap := map[tc.DeliveryServiceName][]tc.DeliveryServiceRegex{}
	for _, dsRegex := range regexes {
		sort.Sort(deliveryServiceRegexesSortByTypeThenSetNum(dsRegex.Regexes))
		dsRegexMap[tc.DeliveryServiceName(dsRegex.DSName)] = dsRegex.Regexes
	}
	return dsRegexMap
}

type keyVal struct {
	Key string
	Val string
}

type keyVals []keyVal

func (ks keyVals) Len() int      { return len(ks) }
func (ks keyVals) Swap(i, j int) { ks[i], ks[j] = ks[j], ks[i] }
func (ks keyVals) Less(i, j int) bool {
	if ks[i].Key != ks[j].Key {
		return ks[i].Key < ks[j].Key
	}
	return ks[i].Val < ks[j].Val
}

// getDSRequestFQDNs returns the FQDNs that clients will request from the edge.
func getDSRequestFQDNs(ds *DeliveryService, regexes []tc.DeliveryServiceRegex, server *Server, cdnDomain string) ([]string, error) {
	if server.HostName == nil {
		return nil, errors.New("server missing hostname")
	}

	fqdns := []string{}
	for _, dsRegex := range regexes {
		if tc.DSMatchType(dsRegex.Type) != tc.DSMatchTypeHostRegex || ds.OrgServerFQDN == nil || *ds.OrgServerFQDN == "" {
			continue
		}
		if dsRegex.Pattern == "" {
			return nil, errors.New("ds missing regex pattern")
		}
		if ds.Protocol == nil {
			return nil, errors.New("ds missing protocol")
		}
		if cdnDomain == "" {
			return nil, errors.New("ds missing domain")
		}

		hostRegex := dsRegex.Pattern
		fqdn := hostRegex

		if strings.HasSuffix(hostRegex, `.*`) {
			re := hostRegex
			re = strings.Replace(re, `\`, ``, -1)
			re = strings.Replace(re, `.*`, ``, -1)

			hName := *server.HostName
			if ds.Type.IsDNS() {
				if ds.RoutingName == nil {
					return nil, errors.New("ds is dns, but missing routing name")
				}
				hName = *ds.RoutingName
			}

			fqdn = hName + re + cdnDomain
		}
		fqdns = append(fqdns, fqdn)
	}
	return fqdns, nil
}
