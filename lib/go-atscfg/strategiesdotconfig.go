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
	//	"bytes"
	"errors"
	"fmt"
	"net/url"
	//	"regexp"
	//	"sort"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	//	"github.com/apache/trafficcontrol/lib/go-util"
)

const ContentTypeStrategiesDotYAML = ContentTypeYAML
const LineCommentStrategiesDotYAML = LineCommentHash

// StrategiesYAMLOpts contains settings to configure strategies.config generation options.
type StrategiesYAMLOpts struct {
	// VerboseComments is whether to add informative comments to the generated file, about what was generated and why.
	// Note this does not include the header comment, which is configured separately with HdrComment.
	// These comments are human-readable and not guarnateed to be consistent between versions. Automating anything based on them is strongly discouraged.
	VerboseComments bool

	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeStrategiesDotYAML(
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
	opt StrategiesYAMLOpts,
) (Cfg, error) {
	warnings := []string{}

	hdrComment := makeHdrComment(opt.HdrComment)

	hosts := "hosts:"
	groups := "groups:"
	strategies := "strategies:"

	for _, ds := range dses {
		if ds.XMLID == nil {
			warnings = append(warnings, fmt.Sprintf("ds had nil XMLID, skipping!"))
			continue
		} else if ds.OrgServerFQDN == nil {
			warnings = append(warnings, fmt.Sprintf("ds '%v' had nil OrgServerFQDN, skipping!", *ds.XMLID))
			continue
		}

		dsName := tc.DeliveryServiceName(*ds.XMLID)

		nameTopologies := makeTopologyNameMap(topologies)

		cacheGroups, err := makeCGMap(cacheGroupArr)
		if err != nil {
			return Cfg{}, makeErr(warnings, "making CacheGroup map: "+err.Error())
		}

		isLastCacheTier, err := getIsLastCacheTier(server, &ds, cacheGroups, nameTopologies)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("ds '%v' error getting cache tier status, skipping ds!! error: %v", *ds.XMLID, err))
			continue
		}

		profileParentConfigParams, parentWarns := getProfileParentConfigParams(tcParentConfigParams)
		warnings = append(warnings, parentWarns...)

		serverParams := getServerParentConfigParams(server, profileParentConfigParams)
		dsParams, dsParamsWarnings := getParentDSParams(ds, profileParentConfigParams)
		warnings = append(warnings, dsParamsWarnings...)

		policy, policyWarns := getStrategyPolicy(serverParams, dsParams, isLastCacheTier)
		warnings = append(warnings, policyWarns...)

		hashKey, hashWarns := getStrategyHashKey(&ds, serverParams, dsParams, isLastCacheTier)
		warnings = append(warnings, hashWarns...)

		goDirect := isLastCacheTier
		parentIsProxy := !isLastCacheTier

		strategy := Strategy{
			DSName:             dsName,
			Policy:             policy,
			HashKey:            hashKey,
			GoDirect:           goDirect,
			ParentIsProxy:      parentIsProxy,
			MaxSimpleRetries:   2,
			Scheme:             StrategySchemeHTTP,
			RingMode:           StrategyRingModeExhaust,
			ErrorResponseCodes: []int{404, 502, 503},
		}
		strategies += strategy.String()

		originURI, err := url.Parse(*ds.OrgServerFQDN)
		if err != nil {
			log.Errorf("error parsing ds '%v' origin '%v' skipping ds!\n", dsName, *ds.OrgServerFQDN)
			continue
		}

		hosts += strategyHostStr(dsName, originURI)
		groups += strategyGroupStr(dsName, []*url.URL{originURI})
	}
	txt := hdrComment + "\n" + hosts + "\n\n" + groups + "\n\n" + strategies + "\n"

	return Cfg{
		Text:        txt,
		ContentType: ContentTypeStrategiesDotYAML,
		LineComment: LineCommentStrategiesDotYAML,
		Warnings:    warnings,
	}, nil
}

// getStrategyHashKey returns the strategy hash_key, primarily whether to use the query string in the parent selection consistent hash selection.
// Note this is not whether to use the query string in the cache key, or whether to pass it up. But rather, whether to use it when consistent-hashing to find the parent. That is, if it is used, parent selection will be different for different query strings but the same path.
// Returns the strategy hash key, and any warnings
func getStrategyHashKey(ds *DeliveryService, serverParams map[string]string, dsParams parentDSParams, isLastCacheTier bool) (StrategyHashKey, []string) {
	warnings := []string{}

	if isLastCacheTier {
		if ds.MultiSiteOrigin != nil && *ds.MultiSiteOrigin && dsParams.QueryStringHandling == "" && dsParams.Algorithm == tc.AlgorithmConsistentHash && ds.QStringIgnore != nil && tc.QStringIgnore(*ds.QStringIgnore) == tc.QStringIgnoreUseInCacheKeyAndPassUp {
			return StrategyHashKeyPathQuery, warnings
		}
		return StrategyHashKeyPath, warnings
	}

	hashKey := StrategyHashKeyPath
	if ds.QStringIgnore != nil && tc.QStringIgnore(*ds.QStringIgnore) == tc.QStringIgnoreUseInCacheKeyAndPassUp {
		return StrategyHashKeyPathQuery, warnings
	}

	if param := serverParams[ParentConfigParamQStringHandling]; param != "" {
		paramKey := parentConfigQStringHandlingToStrategyHashKey(param)
		if paramKey != StrategyHashKeyInvalid {
			hashKey = paramKey
		} else {
			warnings = append(warnings, "server parameters had unknown qstr '"+param+"': ignoring!")
		}
	}

	if dsParams.QueryStringHandling != "" {
		paramKey := parentConfigQStringHandlingToStrategyHashKey(dsParams.QueryStringHandling)
		if paramKey != StrategyHashKeyInvalid {
			hashKey = paramKey
		} else {
			warnings = append(warnings, "delivery service parameters had unknown qstr '"+dsParams.QueryStringHandling+"': ignoring!")
		}
	}

	return hashKey, warnings
}

func parentConfigQStringHandlingToStrategyHashKey(qStringHandling string) StrategyHashKey {
	qstr := strings.ToLower(strings.TrimSpace(qStringHandling))
	switch qstr {
	case "consider":
		return StrategyHashKeyPathQuery
	case "ignore":
		return StrategyHashKeyPath
	}
	return StrategyHashKeyInvalid
}

const DefaultStrategyPolicy = StrategyPolicyConsistentHash

// getStrategyPolicy returns the policy of the Strategy for the given Delivery Service on the given Server, and any warnings.
// The serverParams is as returned from getServerParentConfigParams.
func getStrategyPolicy(serverParams map[string]string, dsParams parentDSParams, isLastCacheTier bool) (StrategyPolicy, []string) {
	warnings := []string{}
	policy := DefaultStrategyPolicy

	if !isLastCacheTier {
		// the CDN always uses consistent-hash internally.
		return DefaultStrategyPolicy, warnings
	}

	if parentSelectAlg := serverParams[ParentConfigParamAlgorithm]; strings.TrimSpace(parentSelectAlg) != "" {
		if paramPolicy := parentConfigRoundRobinToStrategyPolicy(parentSelectAlg); paramPolicy != StrategyPolicyInvalid {
			policy = paramPolicy
		} else {
			warnings = append(warnings, "server parameters had unknown algorithm '"+parentSelectAlg+"': ignoring!")
		}
	}

	if dsParams.Algorithm != "" {
		if paramPolicy := parentConfigRoundRobinToStrategyPolicy(dsParams.Algorithm); paramPolicy != StrategyPolicyInvalid {
			policy = paramPolicy
		} else {
			warnings = append(warnings, "delivery service parameters had unknown algorithm '"+dsParams.Algorithm+"': ignoring!")
		}
	}

	return policy, warnings
}

// getProfileParentConfigParams returns a map[profileName][paramName]paramVal and any warnings
func getProfileParentConfigParams(tcParentConfigParams []tc.Parameter) (map[string]map[string]string, []string) {
	warnings := []string{}
	parentConfigParamsWithProfiles, err := tcParamsToParamsWithProfiles(tcParentConfigParams)
	if err != nil {
		warnings = append(warnings, "error getting profiles from Traffic Ops Parameters, Parameters will not be considered for generation! : "+err.Error())
		parentConfigParamsWithProfiles = []parameterWithProfiles{}
	}
	// parentConfigParams := parameterWithProfilesToMap(parentConfigParamsWithProfiles)

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
	return profileParentConfigParams, warnings
}

// getServerParentConfigParams returns a map[name]value.
// Intended to be called with the result of getProfileParentConfigParams.
func getServerParentConfigParams(server *Server, profileParentConfigParams map[string]map[string]string) map[string]string {
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
	return serverParams
}

func parentConfigRoundRobinToStrategyPolicy(rr string) StrategyPolicy {
	rr = strings.ToLower(strings.TrimSpace(rr))
	switch rr {
	case "true":
		return StrategyPolicyRoundRobinIP
	case "strict":
		return StrategyPolicyRoundRobinStrict
	case "false":
		return StrategyPolicyFirstLive
	case "latched":
		return StrategyPolicyLatched
	case "consistent_hash":
		return StrategyPolicyConsistentHash
	}
	return StrategyPolicyInvalid
}

type Strategy struct {
	DSName             tc.DeliveryServiceName
	Policy             StrategyPolicy
	HashKey            StrategyHashKey
	GoDirect           bool
	ParentIsProxy      bool
	MaxSimpleRetries   int
	Scheme             StrategyScheme
	RingMode           StrategyRingMode
	ErrorResponseCodes []int
}

func (st Strategy) String() string {
	responseCodesStr := func() string {
		str := ``
		for _, code := range st.ErrorResponseCodes {
			str += `
        - ` + strconv.Itoa(code)
		}
		return str
	}
	return `
  - strategy: '` + string(st.DSName) + `'
    policy: ` + string(st.Policy) + `
    hash_key: ` + string(st.HashKey) + `
    go_direct: ` + strconv.FormatBool(st.GoDirect) + `
    ignore_self_detect: true
    parent_is_proxy: ` + strconv.FormatBool(st.ParentIsProxy) + `
    groups:
      - *` + strategyNameOfGroup(st.DSName) + `
    scheme: ` + string(st.Scheme) + `
    failover:
      max_simple_retries: ` + strconv.Itoa(st.MaxSimpleRetries) + `
      ring_mode: ` + string(st.RingMode) + `
      response_codes: ` + responseCodesStr() + `
      health_check:
        - passive
`
}

func strategyGroupStr(ds tc.DeliveryServiceName, parentURIs []*url.URL) string {
	str := `
  - &` + strategyNameOfGroup(ds)
	for _, parentURI := range parentURIs {
		str += strategyHostGroupStr(ds, parentURI)
	}
	return str
}

func strategyHostGroupStr(ds tc.DeliveryServiceName, parentURI *url.URL) string {
	return `
    - <<: *` + strategyNameOfHost(ds, parentURI) + `
      weight: 1.0`
}

// uriPort returns uri.Port() if nonempty, else 80 if scheme is http, else 443 if scheme is https, else "".
func uriPortStr(uri *url.URL) string {
	if port := uri.Port(); port != "" {
		return port
	}
	scheme := strings.ToLower(uri.Scheme)
	if scheme == "http" {
		return "80"
	}
	if scheme == "https" {
		return "443"
	}
	return ""
}

func strategyNameOfHost(ds tc.DeliveryServiceName, uri *url.URL) string {
	return "host_" + string(ds) + "_" + uri.Scheme + "_" + uri.Hostname() + "_" + uri.Port()
}

func strategyNameOfGroup(ds tc.DeliveryServiceName) string {
	return "group_" + string(ds)
}

func strategyHostStr(ds tc.DeliveryServiceName, uri *url.URL) string {
	return `
  - &` + strategyNameOfHost(ds, uri) + `
    host: ` + uri.Hostname() + `
    ignore_self_detect: true
    protocol:
      - scheme: ` + strings.ToLower(uri.Scheme) + `
        port:  ` + uriPortStr(uri) + `
        ignore_self_detect: true
`
}

type StrategyPolicy string

const StrategyPolicyRoundRobinIP = StrategyPolicy(`rr_ip`)
const StrategyPolicyRoundRobinStrict = StrategyPolicy(`rr_strict`)
const StrategyPolicyFirstLive = StrategyPolicy(`first_live`)
const StrategyPolicyLatched = StrategyPolicy(`latched`)
const StrategyPolicyConsistentHash = StrategyPolicy(`consistent_hash`)
const StrategyPolicyInvalid = StrategyPolicy(``)

type StrategyHashKey string

const StrategyHashKeyHostName = StrategyHashKey(`hostname`)
const StrategyHashKeyPath = StrategyHashKey(`path`)
const StrategyHashKeyPathQuery = StrategyHashKey(`path+query`)
const StrategyHashKeyPathFragment = StrategyHashKey(`path+fragment`)
const StrategyHashKeyCacheKey = StrategyHashKey(`cache_key`)
const StrategyHashKeyURL = StrategyHashKey(`url`)
const StrategyHashKeyInvalid = StrategyHashKey(``)

type StrategyScheme string

const StrategySchemeHTTP = StrategyScheme(`http`)
const StrategySchemeHTTPS = StrategyScheme(`https`)
const StrategySchemeInvalid = StrategyScheme(``)

type StrategyRingMode string

const StrategyRingModeExhaust = StrategyRingMode(`exhaust_ring`)
const StrategyRingModeAlternate = StrategyRingMode(`alternate_ring`)
const StrategyRingModeInvalid = StrategyRingMode(``)

// func getTopologyParents(

const debugfile = `
#include unit-tests/hosts.yaml
#
strategies:
  - strategy: 'strategy-1'
    policy: consistent_hash
    hash_key: cache_key
    go_direct: false
    groups:
      - *g1
      - *g2
    scheme http
    failover:
      ring_mode: exhaust_ring
      response_codes:
        - 404
        - 503
      health_check:
        - passive
  - strategy: 'strategy-2'
    policy: rr_strict
    hash_key: cache_key
    go_direct: true
    groups:
      - *g1
      - *g2
    scheme http
    failover:
      ring_mode: exhaust_ring
      response_codes:
        - 404
        - 503
      health_check:
        - passive
`

const ParentConfigQueryStringConsider = "consider"
const ParentConfigQueryStringIgnore = "ignore"

// getIsLastCacheTier returns whether the given server is the last cache tier for the given delivery service.
// Works with both Topologies and pre-Topology DeliveryServices.
func getIsLastCacheTier(server *Server, ds *DeliveryService, cacheGroups map[tc.CacheGroupName]tc.CacheGroupNullable, nameTopologies map[TopologyName]tc.Topology) (bool, error) {
	if ds.Topology != nil && *ds.Topology != "" {
		topology := nameTopologies[TopologyName(*ds.Topology)]
		serverPlacement, err := getTopologyPlacement(tc.CacheGroupName(*server.Cachegroup), topology, cacheGroups, ds)
		if err != nil {
			return false, errors.New("getting topology placement: " + err.Error())
		}
		return serverPlacement.IsLastCacheTier, nil
	}

	if ds.Type == nil {
		return false, errors.New("Delivery service type was nil")
	}

	if ds.OriginShield != nil && *ds.OriginShield != "" {
		return false, nil // TODO verify
	}

	return (!ds.Type.UsesMidCache() && strings.HasPrefix(server.Type, tc.EdgeTypePrefix)) ||
		(ds.Type.UsesMidCache() && strings.HasPrefix(server.Type, tc.MidTypePrefix)), nil
}

// func isQueryStringInParentHash(
// 	server *Server,
// 	ds *DeliveryService,
// 	serverParams map[string]string,
// 	serverIsLastTier bool,
// 	algorithm string,
// 	qStringHandling string,
// ) bool {
// 	if serverIsLastTier {
// 		if ds.MultiSiteOrigin != nil && *ds.MultiSiteOrigin && qStringHandling == "" && algorithm == tc.AlgorithmConsistentHash && ds.QStringIgnore != nil && tc.QStringIgnore(*ds.QStringIgnore) == tc.QStringIgnoreUseInCacheKeyAndPassUp {
// 			return true
// 		}
// 		return false
// 	}

// 	if param := serverParams[ParentConfigParamQStringHandling]; param != "" {
// 		return strings.ToLower(strings.TrimSpace(param)) == ParentConfigQueryStringConsider
// 	}
// 	if qStringHandling != "" {
// 		return strings.ToLower(strings.TrimSpace(qStringHandling)) == ParentConfigQueryStringConsider
// 	}
// 	if ds.QStringIgnore != nil && tc.QStringIgnore(*ds.QStringIgnore) == tc.QStringIgnoreUseInCacheKeyAndPassUp {
// 		return true
// 	}
// 	return false
// }
