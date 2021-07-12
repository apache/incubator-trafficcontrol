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
	"sort"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const ContentTypeYAML = "application/yaml; charset=us-ascii" // Note YAML has no IANA standard mime type. This is one of several common usages, and is likely to be the standardized value. If you're reading this, please check IANA to see if YAML has been added, and change this to the IANA definition if so. Also note we include 'charset=us-ascii' because YAML is commonly UTF-8, but ATS is likely to be unable to handle UTF.
const LineCommentYAML = LineCommentHash

const SSLServerNameYAMLFileName = "ssl_server_name.yaml"

const ContentTypeSSLServerNameYAML = ContentTypeYAML
const LineCommentSSLServerNameYAML = LineCommentYAML

// DefaultDefaultEnableH2 is whether Delivery Services will have HTTP/2 enabled by default if they don't have an explicit Parameter, and no Opt is passed to the Make func.
// We disable by default, to prevent potentially enabling broken clients.
const DefaultDefaultEnableH2 = false

type TLSVersion string

const TLSVersion1p0 = TLSVersion("1.0")
const TLSVersion1p1 = TLSVersion("1.1")
const TLSVersion1p2 = TLSVersion("1.2")
const TLSVersion1p3 = TLSVersion("1.3")
const TLSVersionInvalid = TLSVersion("")

// StringToTLSVersion returns the TLSVersion or TLSVersionInvalid if the string is not a TLS Version enum.
func StringToTLSVersion(st string) TLSVersion {
	switch TLSVersion(st) {
	case TLSVersion1p0:
		return TLSVersion1p0
	case TLSVersion1p1:
		return TLSVersion1p1
	case TLSVersion1p2:
		return TLSVersion1p2
	case TLSVersion1p3:
		return TLSVersion1p3
	}
	return TLSVersionInvalid
}

// tlsVersionsToATS maps TLS version strings to the string used by ATS in ssl_server_name.yaml.
var tlsVersionsToATS = map[TLSVersion]string{
	TLSVersion1p0: "TLSv1",
	TLSVersion1p1: "TLSv1_1",
	TLSVersion1p2: "TLSv1_2",
	TLSVersion1p3: "TLSv1_3",
}

const SSLServerNameYAMLParamEnableH2 = "enable_h2"
const SSLServerNameYAMLParamTLSVersions = "tls_versions"

// DefaultDefaultTLSVersions is the list of TLS versions to enable by default, if no Parameter exists and no Opt is passed to the Make func.
// By default, we enable all, even insecure versions.
// As a CDN, Traffic Control assumes it should not break clients, and it's the client's responsibility to use secure protocols.
// Note this enables certain downgrade attacks. Operators or tenants concerned about these attacks should disable older TLS versions.
var DefaultDefaultTLSVersions = []TLSVersion{
	TLSVersion1p0,
	TLSVersion1p1,
	TLSVersion1p2,
	TLSVersion1p3,
}

// SSLServerNameYAMLOpts contains settings to configure ssl_server_name.yaml generation options.
type SSLServerNameYAMLOpts struct {
	// VerboseComments is whether to add informative comments to the generated file, about what was generated and why.
	// Note this does not include the header comment, which is configured separately with HdrComment.
	// These comments are human-readable and not guaranteed to be consistent between versions. Automating anything based on them is strongly discouraged.
	VerboseComments bool

	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string

	// DefaultTLSVersions is the list of TLS versions to enable on delivery services with no Parameter.
	DefaultTLSVersions []TLSVersion

	// DefaultEnableH2 is whether to disable H2 on delivery services with no Parameter.
	DefaultEnableH2 bool
}

func MakeSSLServerNameYAML(
	server *Server,
	dses []DeliveryService,
	dss []DeliveryServiceServer,
	dsRegexArr []tc.DeliveryServiceRegexes,
	tcParentConfigParams []tc.Parameter,
	cdn *tc.CDN,
	topologies []tc.Topology,
	cacheGroupArr []tc.CacheGroupNullable,
	serverCapabilities map[int]map[ServerCapability]struct{},
	dsRequiredCapabilities map[int]map[ServerCapability]struct{},
	opt *SSLServerNameYAMLOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &SSLServerNameYAMLOpts{}
	}
	if len(opt.DefaultTLSVersions) == 0 {
		opt.DefaultTLSVersions = DefaultDefaultTLSVersions
	}

	sslDatas, warnings, err := GetServerSSLData(
		server,
		dses,
		dss,
		dsRegexArr,
		tcParentConfigParams,
		cdn,
		topologies,
		cacheGroupArr,
		serverCapabilities,
		dsRequiredCapabilities,
		opt.DefaultTLSVersions,
		opt.DefaultEnableH2,
	)

	if err != nil {
		return Cfg{}, makeErr(warnings, "getting ssl data: "+err.Error())
	}

	txt := ""
	if opt.HdrComment != "" {
		txt += makeHdrComment(opt.HdrComment)
	}

	seenFQDNs := map[string]struct{}{}

	for _, sslData := range sslDatas {
		tlsVersionsATS := []string{}
		for _, tlsVersion := range sslData.TLSVersions {
			tlsVersionsATS = append(tlsVersionsATS, `'`+tlsVersionsToATS[tlsVersion]+`'`)
		}

		for _, requestFQDN := range sslData.RequestFQDNs {
			// TODO let active DSes take precedence?
			if _, ok := seenFQDNs[requestFQDN]; ok {
				warnings = append(warnings, "ds '"+sslData.DSName+"' had the same FQDN '"+requestFQDN+"' as some other delivery service, skipping!")
				continue
			}
			seenFQDNs[requestFQDN] = struct{}{}

			dsTxt := "\n"
			if opt.VerboseComments {
				dsTxt += LineCommentYAML + ` ds '` + sslData.DSName + `'` + "\n"
			}
			dsTxt += `- fqdn: '` + requestFQDN + `'`
			dsTxt += "\n" + `  disable_h2: ` + strconv.FormatBool(!sslData.EnableH2)
			dsTxt += "\n" + `  valid_tls_versions_in: [` + strings.Join(tlsVersionsATS, `,`) + `]`

			txt += dsTxt + "\n"
		}

	}

	return Cfg{
		Text:        txt,
		ContentType: ContentTypeSSLServerNameYAML,
		LineComment: LineCommentSSLServerNameYAML,
		Warnings:    warnings,
	}, nil
}

// SSLData has the DS data needed for both sni.yaml (ATS 9+)  and ssl_server_name.yaml (ATS 8).
type SSLData struct {
	DSName       string
	RequestFQDNs []string
	EnableH2     bool
	TLSVersions  []TLSVersion
}

// GetServerSSLData gets the SSLData for all Delivery Services assigned to the given Server, any warnings, and any error.
func GetServerSSLData(
	server *Server,
	dses []DeliveryService,
	dss []DeliveryServiceServer,
	dsRegexArr []tc.DeliveryServiceRegexes,
	tcParentConfigParams []tc.Parameter,
	cdn *tc.CDN,
	topologies []tc.Topology,
	cacheGroupArr []tc.CacheGroupNullable,
	serverCapabilities map[int]map[ServerCapability]struct{},
	dsRequiredCapabilities map[int]map[ServerCapability]struct{},
	defaultTLSVersions []TLSVersion,
	defaultEnableH2 bool,
) ([]SSLData, []string, error) {
	warnings := []string{}

	if server.Profile == nil {
		return nil, warnings, errors.New("this server missing Profile")
	}

	dsRegexes := makeDSRegexMap(dsRegexArr)

	parentConfigParamsWithProfiles, err := tcParamsToParamsWithProfiles(tcParentConfigParams)
	if err != nil {
		warnings = append(warnings, "error getting profiles from Traffic Ops Parameters, Parameters will not be considered for generation! : "+err.Error())
		parentConfigParamsWithProfiles = []parameterWithProfiles{}
	}

	profileParentConfigParams := map[string]map[string]string{} // map[profileName][paramName]paramVal
	for _, param := range parentConfigParamsWithProfiles {
		for _, profile := range param.ProfileNames {
			if _, ok := profileParentConfigParams[profile]; !ok {
				profileParentConfigParams[profile] = map[string]string{}
			}
			profileParentConfigParams[profile][param.Name] = param.Value
		}
	}

	cacheGroups, err := makeCGMap(cacheGroupArr)
	if err != nil {
		return nil, warnings, errors.New("making cachegroup map: " + err.Error())
	}

	nameTopologies := makeTopologyNameMap(topologies)

	sort.Sort(dsesSortByName(dses))

	sslDatas := []SSLData{}

	for _, ds := range dses {
		hasDS, err := dsUsesServer(&ds, server, dss, nameTopologies, cacheGroups, serverCapabilities, dsRequiredCapabilities)
		if err != nil {
			warnings = append(warnings, "error checking if ds uses this server, considering false! Error: "+err.Error())
			continue
		}
		if !hasDS {
			continue
		}

		dsParentConfigParams := map[string]string{}
		if ds.ProfileName != nil {
			dsParentConfigParams = profileParentConfigParams[*ds.ProfileName]
		}

		requestFQDNs, err := getDSRequestFQDNs(&ds, dsRegexes[tc.DeliveryServiceName(*ds.XMLID)], server, cdn.DomainName)
		if err != nil {
			warnings = append(warnings, "error getting ds '"+*ds.XMLID+"' request fqdns, skipping! Error: "+err.Error())
			continue
		}

		enableH2 := defaultEnableH2
		tlsVersions := defaultTLSVersions

		paramValEnableH2 := dsParentConfigParams[SSLServerNameYAMLParamEnableH2]
		paramValEnableH2 = strings.TrimSpace(paramValEnableH2)
		paramValEnableH2 = strings.ToLower(paramValEnableH2)

		if paramValEnableH2 != "" {
			enableH2 = strings.HasPrefix(paramValEnableH2, "t") || strings.HasPrefix(paramValEnableH2, "y")
		}

		paramValTLSVersions := dsParentConfigParams[SSLServerNameYAMLParamTLSVersions]
		paramValTLSVersions = strings.Replace(paramValTLSVersions, " ", "", -1)
		paramValTLSVersions = strings.TrimSpace(paramValTLSVersions)

		paramTLSVersions := []TLSVersion{}
		if paramValTLSVersions != "" {
			// Allow delimiting with commas, semicolons, spaces, or newlines.
			delim := ","
			if !strings.Contains(paramValTLSVersions, delim) {
				delim = ";"
			}
			if !strings.Contains(paramValTLSVersions, delim) {
				delim = " "
			}
			if !strings.Contains(paramValTLSVersions, delim) {
				delim = "\n"
			}

			tlsVersionsParamArr := strings.Split(paramValTLSVersions, delim)
			for _, tlsVersion := range tlsVersionsParamArr {
				if _, ok := tlsVersionsToATS[TLSVersion(tlsVersion)]; !ok {
					warnings = append(warnings, "ds '"+*ds.XMLID+"' had unknown "+SSLServerNameYAMLParamTLSVersions+" parameter '"+tlsVersion+"' - ignoring!")
					continue
				}
				paramTLSVersions = append(paramTLSVersions, TLSVersion(tlsVersion))
			}
		}

		if len(paramTLSVersions) != 0 {
			tlsVersions = paramTLSVersions
		}

		sslDatas = append(sslDatas, SSLData{
			DSName:       *ds.XMLID,
			RequestFQDNs: requestFQDNs,
			EnableH2:     enableH2,
			TLSVersions:  tlsVersions,
		})
	}

	return sslDatas, warnings, nil
}

func dsUsesServer(
	ds *DeliveryService,
	server *Server,
	dss []DeliveryServiceServer,
	nameTopologies map[TopologyName]tc.Topology,
	cacheGroups map[tc.CacheGroupName]tc.CacheGroupNullable,
	serverCapabilities map[int]map[ServerCapability]struct{},
	dsRequiredCapabilities map[int]map[ServerCapability]struct{},
) (bool, error) {
	if ds.XMLID == nil || *ds.XMLID == "" {
		return false, errors.New("ds missing xmlId")
	} else if ds.ID == nil {
		return false, errors.New("ds missing id")
	} else if server.ID == nil {
		return false, errors.New("server missing id")
	} else if ds.Type == nil {
		return false, errors.New("ds missing type")
	}

	if !hasRequiredCapabilities(serverCapabilities[*server.ID], dsRequiredCapabilities[*ds.ID]) {
		return false, nil
	}

	serverParentCGData, err := getParentCacheGroupData(server, cacheGroups)
	if err != nil {
		return false, errors.New("getting server parent cachegroup data: " + err.Error())
	}
	cacheIsTopLevel := isTopLevelCache(serverParentCGData)

	if !cacheIsTopLevel && (ds.Topology == nil || *ds.Topology == "") {
		if !dsAssignedServer(*ds.ID, *server.ID, dss) {
			return false, nil
		}
	}

	if ds.Topology != nil && *ds.Topology != "" {
		topology, ok := nameTopologies[TopologyName(*ds.Topology)]
		if !ok {
			return false, errors.New("ds topology '" + *ds.Topology + "' not found in topologies")
		}

		serverPlacement, err := getTopologyPlacement(tc.CacheGroupName(*server.Cachegroup), topology, cacheGroups, ds)
		if err != nil {
			return false, errors.New("getting topology placement: " + err.Error())
		}
		if !serverPlacement.InTopology {
			return false, nil
		}
	}

	return true, nil
}

// dsAssignedServer returns whether the Delivery Service Servers has an assignment between the server and the DS.
// Does not check Topologies, or parentage. Only useful for Edges and pre-topology DSS.
func dsAssignedServer(dsID int, serverID int, dsses []DeliveryServiceServer) bool {
	for _, dss := range dsses {
		if dss.Server == serverID && dss.DeliveryService == dsID {
			return true
		}
	}
	return false
}
