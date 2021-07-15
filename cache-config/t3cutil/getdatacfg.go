package t3cutil

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
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/apache/trafficcontrol/cache-config/t3c-generate/toreq"
	"github.com/apache/trafficcontrol/lib/go-atscfg"
	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
)

const TrafficOpsProxyParameterName = `tm.rev_proxy.url`

type ConfigData struct {
	// Servers must be all the servers from Traffic Ops. May include servers not on the current cdn.
	Servers []atscfg.Server `json:"servers,omitempty"`

	// CacheGroups must be all cachegroups in Traffic Ops with Servers on the current server's cdn. May also include CacheGroups without servers on the current cdn.
	CacheGroups []tc.CacheGroupNullable `json:"cache_groups,omitempty"`

	// GlobalParams must be all Parameters in Traffic Ops on the tc.GlobalProfileName Profile. Must not include other parameters.
	GlobalParams []tc.Parameter `json:"global_parameters,omitempty"`

	// ServerParams must be all Parameters on the Profile of the current server. Must not include other Parameters.
	ServerParams []tc.Parameter `json:"server_parameters,omitempty"`

	// CacheKeyConfigParams must be all Parameters with the "cachekey.config" (compat)
	CacheKeyConfigParams []tc.Parameter `json:"cachekey_config_parameters,omitempty"`

	// RemapConfigParams must be all Parameters with the ConfigFile "remap.config"
	RemapConfigParams []tc.Parameter `json:"remap_config_parameters,omitempty"`

	// ParentConfigParams must be all Parameters with the ConfigFile "parent.config.
	ParentConfigParams []tc.Parameter `json:"parent_config_parameters,omitempty"`

	// DeliveryServices must include all Delivery Services on the current server's cdn, including those not assigned to the server. Must not include delivery services on other cdns.
	DeliveryServices []atscfg.DeliveryService `json:"delivery_services,omitempty"`

	// DeliveryServiceServers must include all delivery service servers in Traffic Ops for all delivery services on the current cdn, including those not assigned to the current server.
	DeliveryServiceServers []atscfg.DeliveryServiceServer `json:"delivery_service_servers,omitempty"`

	// Server must be the server we're fetching configs from
	Server *atscfg.Server `json:"server,omitempty"`

	// Jobs must be all Jobs on the server's CDN. May include jobs on other CDNs.
	Jobs []tc.InvalidationJob `json:"jobs,omitempty"`

	// CDN must be the CDN of the server.
	CDN *tc.CDN `json:"cdn,omitempty"`

	// DeliveryServiceRegexes must be all regexes on all delivery services on this server's cdn.
	DeliveryServiceRegexes []tc.DeliveryServiceRegexes `json:"delivery_service_regexes,omitempty"`

	// Profile must be the Profile of the server being requested.
	Profile tc.Profile `json:"profile,omitempty"`

	// URISigningKeys must be a map of every delivery service which is URI Signed, to its keys.
	URISigningKeys map[tc.DeliveryServiceName][]byte `json:"uri_signing_keys,omitempty"`

	// URLSigKeys must be a map of every delivery service which uses URL Sig, to its keys.
	URLSigKeys map[tc.DeliveryServiceName]tc.URLSigKeys `json:"url_sig_keys,omitempty"`

	// ServerCapabilities must be a map of all server IDs on this server's CDN, to a set of their capabilities. May also include servers from other cdns.
	ServerCapabilities map[int]map[atscfg.ServerCapability]struct{} `json:"server_capabilities,omitempty"`

	// DSRequiredCapabilities must be a map of all delivery service IDs on this server's CDN, to a set of their required capabilities. Delivery Services with no required capabilities may not have an entry in the map.
	DSRequiredCapabilities map[int]map[atscfg.ServerCapability]struct{} `json:"delivery_service_required_capabilities,omitempty"`

	// SSLKeys must be all the ssl keys for the server's cdn.
	SSLKeys []tc.CDNSSLKeys `json:"ssl_keys,omitempty"`

	// Topologies must be all the topologies for the server's cdn.
	// May incude topologies of other cdns.
	Topologies []tc.Topology `json:"topologies,omitempty"`

	// TrafficOpsAddresses is the list of IP addresses used to request data. Because of proxies and load balancers,
	// multiple addresses may be used for the multiple requests necessary to fetch all data.
	TrafficOpsAddresses []string `json:"traffic_ops_addresses,omitempty"`
	TrafficOpsURL       string   `json:"traffic_ops_url,omitempty"`

	MetaData ConfigDataMetaData `json:"metadata"`
}

type ConfigDataMetaData struct {
	Servers                ReqMetaData                            `json:"servers"`
	CacheGroups            ReqMetaData                            `json:"cache_groups"`
	GlobalParams           ReqMetaData                            `json:"global_parameters"`
	ServerParams           ReqMetaData                            `json:"server_parameters"`
	CacheKeyConfigParams   ReqMetaData                            `json:"cachekey_config_parameters"`
	RemapConfigParams      ReqMetaData                            `json:"remap_config_parameters"`
	ParentConfigParams     ReqMetaData                            `json:"parent_config_parameters"`
	DeliveryServices       ReqMetaData                            `json:"delivery_services"`
	DeliveryServiceServers ReqMetaData                            `json:"delivery_service_servers"`
	Jobs                   ReqMetaData                            `json:"jobs"`
	CDN                    ReqMetaData                            `json:"cdn"`
	DeliveryServiceRegexes ReqMetaData                            `json:"delivery_service_regexes"`
	Profile                ReqMetaData                            `json:"profile"`
	URISigningKeys         map[tc.DeliveryServiceName]ReqMetaData `json:"uri_signing_keys"`
	URLSigKeys             map[tc.DeliveryServiceName]ReqMetaData `json:"url_sig_keys"`
	ServerCapabilities     ReqMetaData                            `json:"server_capabilities"`
	DSRequiredCapabilities ReqMetaData                            `json:"delivery_service_required_capabilities"`
	SSLKeys                ReqMetaData                            `json:"ssl_keys"`
	Topologies             ReqMetaData                            `json:"topologies"`
}

// ReqMetaData has response headers for Conditional Requests.
type ReqMetaData struct {
	LastModified string `json:"last_modified"`
	Date         string `json:"date"`
	ETag         string `json:"etag"`
}

func MakeReqHdr(md ReqMetaData) http.Header {
	if md.LastModified == "" && md.Date == "" && md.ETag == "" {
		return nil
	}
	hdr := http.Header{}
	if lm, ok := rfc.ParseHTTPDate(md.LastModified); ok {
		hdr.Set("If-Modified-Since", rfc.FormatHTTPDate(lm.Add(time.Second))) // add 1s, because TO rounds down, which will always be modified
	} else if date, ok := rfc.ParseHTTPDate(md.Date); ok {
		hdr.Set("If-Modified-Since", rfc.FormatHTTPDate(date.Add(time.Second))) // add 1s, because TO rounds down, which will always be modified
	}
	if md.ETag != "" {
		hdr.Set("If-None-Match", md.ETag)
	}
	return hdr
}

func MakeReqMetaData(respHdr http.Header) ReqMetaData {
	return ReqMetaData{
		LastModified: respHdr.Get(rfc.LastModified),
		Date:         respHdr.Get(rfc.Date),
		ETag:         respHdr.Get(rfc.ETagHeader),
	}
}

// GetTOData gets all the data from Traffic Ops needed to generate config.
// Returns the data, the addresses of all Traffic Ops' requested, and any error.
//
// The toClient is the Traffic Ops client, which should already be initialized and connected.
//
// The disableProxy arg is whether to disable using any Traffic Ops proxy configured via the global TrafficOpsProxyParameterName. If the Parameter exists, this will connect to Traffic Ops to get the global parameters, get the Parameter, and then change the toClient to use it.
//
// The cacheHostName is the hostname of the cache to get config generation data for.
//
// The revalOnly arg is whether to only get data necessary to revalidate, versus all data necessary to generate cache config.
func GetConfigData(toClient *toreq.TOClient, disableProxy bool, cacheHostName string, revalOnly bool, oldCfg *ConfigData) (*ConfigData, error) {
	start := time.Now()
	defer func() { log.Infof("GetTOData took %v\n", time.Since(start)) }()

	toIPs := &sync.Map{} // each Traffic Ops request could get a different IP, so track them all
	toData := &ConfigData{}

	{
		reqHdr := (http.Header)(nil)
		if oldCfg != nil {
			reqHdr = MakeReqHdr(oldCfg.MetaData.GlobalParams)
		}
		globalParams, reqInf, err := toClient.GetGlobalParameters(reqHdr)
		if err != nil {
			return nil, errors.New("getting global parameters: " + err.Error())
		}
		if reqInf.StatusCode == http.StatusNotModified {
			log.Infof("Getting config: %v not modified, using old config", "Global Params")
			toData.GlobalParams = oldCfg.GlobalParams
		} else {
			log.Infof("Getting config: %v is modified, using new response", "Global Params")
			toData.GlobalParams = globalParams
		}
		toData.MetaData.GlobalParams = MakeReqMetaData(reqInf.RespHeaders)
		toIPs.Store(reqInf.RemoteAddr, nil)
	}

	if !disableProxy {
		toProxyURLStr := ""
		for _, param := range toData.GlobalParams {
			if param.Name == TrafficOpsProxyParameterName {
				toProxyURLStr = param.Value
				break
			}
		}
		if toProxyURLStr != "" {
			realTOURL := toClient.C.URL
			toClient.C.URL = toProxyURLStr
			log.Infoln("using Traffic Ops proxy '" + toProxyURLStr + "'")
			if _, _, err := toClient.C.GetCDNs(); err != nil {
				log.Warnln("Traffic Ops proxy '" + toProxyURLStr + "' failed to get CDNs, falling back to real Traffic Ops")
				toClient.C.URL = realTOURL
			}
		} else {
			log.Infoln("Traffic Ops proxy enabled, but GLOBAL Parameter '" + TrafficOpsProxyParameterName + "' missing or empty, not using proxy")
		}
	} else {
		log.Infoln("Traffic Ops proxy is disabled, not checking or using GLOBAL Parameter '" + TrafficOpsProxyParameterName)
	}

	serversF := func() error {
		defer func(start time.Time) { log.Infof("serversF took %v\n", time.Since(start)) }(time.Now())
		// TODO TOAPI add /servers?cdn=1 query param

		{
			reqHdr := (http.Header)(nil)
			if oldCfg != nil {
				reqHdr = MakeReqHdr(oldCfg.MetaData.Servers)
			}
			servers, reqInf, err := toClient.GetServers(reqHdr)
			if err != nil {
				return errors.New("getting servers: " + err.Error())
			}
			if reqInf.StatusCode == http.StatusNotModified {
				log.Infof("Getting config: %v not modified, using old config", "Servers")
				toData.Servers = oldCfg.Servers
			} else {
				log.Infof("Getting config: %v is modified, using new response", "Servers")
				toData.Servers = servers
			}
			toData.MetaData.Servers = MakeReqMetaData(reqInf.RespHeaders)
			toIPs.Store(reqInf.RemoteAddr, nil)
		}

		server := &atscfg.Server{}
		for _, toServer := range toData.Servers {
			if toServer.HostName != nil && *toServer.HostName == cacheHostName {
				server = &toServer
				break
			}
		}
		if server.ID == nil {
			return errors.New("server '" + cacheHostName + " not found in servers")
		} else if server.CDNName == nil {
			return errors.New("server '" + cacheHostName + " missing CDNName")
		} else if server.CDNID == nil {
			return errors.New("server '" + cacheHostName + " missing CDNID")
		} else if server.Profile == nil {
			return errors.New("server '" + cacheHostName + " missing Profile")
		}

		toData.Server = server

		sslF := func() error {
			defer func(start time.Time) { log.Infof("sslF took %v\n", time.Since(start)) }(time.Now())

			{
				reqHdr := (http.Header)(nil)
				if oldCfg != nil {
					reqHdr = MakeReqHdr(oldCfg.MetaData.SSLKeys)
				}
				keys, reqInf, err := toClient.GetCDNSSLKeys(tc.CDNName(*server.CDNName), reqHdr)
				if err != nil {
					return errors.New("getting cdn '" + *server.CDNName + "': " + err.Error())
				}

				if reqInf.StatusCode == http.StatusNotModified {
					log.Infof("Getting config: %v not modified, using old config", "SSLKeys")
					toData.SSLKeys = oldCfg.SSLKeys
				} else {
					log.Infof("Getting config: %v is modified, using new response", "SSLKeys")
					toData.SSLKeys = keys
				}
				toData.MetaData.SSLKeys = MakeReqMetaData(reqInf.RespHeaders)
				toIPs.Store(reqInf.RemoteAddr, nil)
			}
			return nil
		}
		dsF := func() error {
			defer func(start time.Time) { log.Infof("dsF took %v\n", time.Since(start)) }(time.Now())

			{
				reqHdr := (http.Header)(nil)
				if oldCfg != nil {
					reqHdr = MakeReqHdr(oldCfg.MetaData.DeliveryServices)
				}
				dses, reqInf, err := toClient.GetCDNDeliveryServices(*server.CDNID, reqHdr)
				if err != nil {
					return errors.New("getting delivery services: " + err.Error())
				}

				if reqInf.StatusCode == http.StatusNotModified {
					log.Infof("Getting config: %v not modified, using old config", "DeliveryServices")
					toData.DeliveryServices = oldCfg.DeliveryServices
				} else {
					log.Infof("Getting config: %v is modified, using new response", "DeliveryServices")
					toData.DeliveryServices = dses
				}
				toData.MetaData.DeliveryServices = MakeReqMetaData(reqInf.RespHeaders)
				toIPs.Store(reqInf.RemoteAddr, nil)
			}

			// TODO uncomment when MSO Origins are changed to not use DSS, to avoid the DSS call if it isn't necessary
			// allDSesHaveTopologies := true
			// for _, ds := range toData.DeliveryServices {
			// 	if ds.CDNID == nil || *ds.CDNID != *server.CDNID {
			// 		continue
			// 	}
			// 	if ds.Topology == nil {
			// 		allDSesHaveTopologies = false
			// 		break
			// 	}
			// }

			dssF := func() error {
				defer func(start time.Time) { log.Infof("dssF took %v\n", time.Since(start)) }(time.Now())

				{
					reqHdr := (http.Header)(nil)
					if oldCfg != nil {
						reqHdr = MakeReqHdr(oldCfg.MetaData.DeliveryServiceServers)
					}
					dss, reqInf, err := toClient.GetDeliveryServiceServers(nil, nil, reqHdr)
					if err != nil {
						return errors.New("getting delivery service servers: " + err.Error())
					}

					if reqInf.StatusCode == http.StatusNotModified {
						log.Infof("Getting config: %v not modified, using old config", "DeliveryServiceServers")
						toData.DeliveryServiceServers = oldCfg.DeliveryServiceServers
					} else {
						log.Infof("Getting config: %v is modified, using new response", "DeliveryServiceServers")
						toData.DeliveryServiceServers = filterUnusedDSS(dss, *toData.Server.CDNID, toData.Servers, toData.DeliveryServices)
					}
					toData.MetaData.DeliveryServiceServers = MakeReqMetaData(reqInf.RespHeaders)
					toIPs.Store(reqInf.RemoteAddr, nil)
				}
				return nil
			}

			uriSignKeysF := func() error {
				defer func(start time.Time) { log.Infof("uriF took %v\n", time.Since(start)) }(time.Now())
				uriSigningKeys := map[tc.DeliveryServiceName][]byte{}
				toData.MetaData.URISigningKeys = map[tc.DeliveryServiceName]ReqMetaData{}
				for _, ds := range toData.DeliveryServices {
					if ds.XMLID == nil {
						continue // TODO warn?
					}
					// TODO read meta config gen, and only include servers which are included in the meta (assigned to edge or all for mids? read the meta gen to find out)
					if ds.SigningAlgorithm == nil || *ds.SigningAlgorithm != tc.SigningAlgorithmURISigning {
						continue
					}

					reqHdr := (http.Header)(nil)
					if oldCfg != nil && oldCfg.MetaData.URISigningKeys != nil {
						reqHdr = MakeReqHdr(oldCfg.MetaData.URISigningKeys[tc.DeliveryServiceName(*ds.XMLID)])
					}
					keys, reqInf, err := toClient.GetURISigningKeys(*ds.XMLID, reqHdr)
					if err != nil {
						if strings.Contains(strings.ToLower(err.Error()), "not found") {
							log.Errorln("Delivery service '" + *ds.XMLID + "' is uri_signing, but keys not found! Skipping!")
							continue
						} else {
							return errors.New("getting uri signing keys for ds '" + *ds.XMLID + "': " + err.Error())
						}
					}

					if reqInf.StatusCode == http.StatusNotModified {
						log.Infof("Getting config: %v not modified, using old config", "URISigningKeys["+*ds.XMLID+"]")
						uriSigningKeys[tc.DeliveryServiceName(*ds.XMLID)] = oldCfg.URISigningKeys[tc.DeliveryServiceName(*ds.XMLID)]
					} else {
						log.Infof("Getting config: %v is modified, using new response", "URISigningKeys["+*ds.XMLID+"]")
						uriSigningKeys[tc.DeliveryServiceName(*ds.XMLID)] = keys
					}
					toData.MetaData.URISigningKeys[tc.DeliveryServiceName(*ds.XMLID)] = MakeReqMetaData(reqInf.RespHeaders)
					toIPs.Store(reqInf.RemoteAddr, nil)
				}
				toData.URISigningKeys = uriSigningKeys
				return nil
			}

			urlSigKeysF := func() error {
				defer func(start time.Time) { log.Infof("urlF took %v\n", time.Since(start)) }(time.Now())
				urlSigKeys := map[tc.DeliveryServiceName]tc.URLSigKeys{}
				toData.MetaData.URLSigKeys = map[tc.DeliveryServiceName]ReqMetaData{}
				for _, ds := range toData.DeliveryServices {
					if ds.XMLID == nil {
						continue // TODO warn?
					}
					// TODO read meta config gen, and only include servers which are included in the meta (assigned to edge or all for mids? read the meta gen to find out)
					if ds.SigningAlgorithm == nil || *ds.SigningAlgorithm != tc.SigningAlgorithmURLSig {
						continue
					}

					reqHdr := (http.Header)(nil)
					if oldCfg != nil {
						reqHdr = MakeReqHdr(oldCfg.MetaData.URLSigKeys[tc.DeliveryServiceName(*ds.XMLID)])
					}
					keys, reqInf, err := toClient.GetURLSigKeys(*ds.XMLID, reqHdr)
					if err != nil {
						if strings.Contains(strings.ToLower(err.Error()), "not found") {
							log.Errorln("Delivery service '" + *ds.XMLID + "' is url_sig, but keys not found! Skipping!: " + err.Error())
							continue
						} else {
							return errors.New("getting url sig keys for ds '" + *ds.XMLID + "': " + err.Error())
						}
					}

					if reqInf.StatusCode == http.StatusNotModified {
						log.Infof("Getting config: %v not modified, using old config", "URLSigKeys["+*ds.XMLID+"]")
						urlSigKeys[tc.DeliveryServiceName(*ds.XMLID)] = oldCfg.URLSigKeys[tc.DeliveryServiceName(*ds.XMLID)]
					} else {
						log.Infof("Getting config: %v is modified, using new response", "URLSigKeys["+*ds.XMLID+"]")
						urlSigKeys[tc.DeliveryServiceName(*ds.XMLID)] = keys
					}
					toData.MetaData.URLSigKeys[tc.DeliveryServiceName(*ds.XMLID)] = MakeReqMetaData(reqInf.RespHeaders)
					toIPs.Store(reqInf.RemoteAddr, nil)
				}
				toData.URLSigKeys = urlSigKeys
				return nil
			}

			fs := []func() error{}
			if !revalOnly {
				fs = append([]func() error{uriSignKeysF, urlSigKeysF}, fs...) // skip keys for reval-only, which doesn't need them
			}
			if !revalOnly { // TODO when MSO Origins are changed to not use DSS, we can add `&& !allDSesHaveTopologies` here, to avoid the DSS call if it isn't necessary
				// skip DSS if reval-only (which doesn't need DSS)
				fs = append([]func() error{dssF}, fs...)
			}

			return util.JoinErrs(runParallel(fs))
		}
		serverParamsF := func() error {
			defer func(start time.Time) { log.Infof("serverParamsF took %v\n", time.Since(start)) }(time.Now())
			{
				reqHdr := (http.Header)(nil)
				if oldCfg != nil {
					reqHdr = MakeReqHdr(oldCfg.MetaData.ServerParams)
				}
				params, reqInf, err := toClient.GetServerProfileParameters(*server.Profile, reqHdr)
				if err != nil {
					return errors.New("getting server profile '" + *server.Profile + "' parameters: " + err.Error())
				} else if len(params) == 0 {
					return errors.New("getting server profile '" + *server.Profile + "' parameters: no parameters (profile not found?)")
				}

				if reqInf.StatusCode == http.StatusNotModified {
					log.Infof("Getting config: %v not modified, using old config", "ServerParams")
					toData.ServerParams = oldCfg.ServerParams
				} else {
					log.Infof("Getting config: %v is modified, using new response", "ServerParams")
					toData.ServerParams = params
				}
				toData.MetaData.ServerParams = MakeReqMetaData(reqInf.RespHeaders)
				toIPs.Store(reqInf.RemoteAddr, nil)
			}
			return nil
		}
		cdnF := func() error {
			defer func(start time.Time) { log.Infof("cdnF took %v\n", time.Since(start)) }(time.Now())
			{
				reqHdr := (http.Header)(nil)
				if oldCfg != nil {
					reqHdr = MakeReqHdr(oldCfg.MetaData.CDN)
				}
				cdn, reqInf, err := toClient.GetCDN(tc.CDNName(*server.CDNName), reqHdr)
				if err != nil {
					return errors.New("getting cdn '" + *server.CDNName + "': " + err.Error())
				}
				if reqInf.StatusCode == http.StatusNotModified {
					log.Infof("Getting config: %v not modified, using old config", "CDN")
					toData.CDN = oldCfg.CDN
				} else {
					log.Infof("Getting config: %v is modified, using new response", "CDN")
					toData.CDN = &cdn
				}
				toData.MetaData.CDN = MakeReqMetaData(reqInf.RespHeaders)
				toIPs.Store(reqInf.RemoteAddr, nil)
			}
			return nil
		}
		profileF := func() error {
			defer func(start time.Time) { log.Infof("profileF took %v\n", time.Since(start)) }(time.Now())
			{
				reqHdr := (http.Header)(nil)
				if oldCfg != nil {
					reqHdr = MakeReqHdr(oldCfg.MetaData.GlobalParams)
				}
				profile, reqInf, err := toClient.GetProfileByName(*server.Profile, reqHdr)
				if err != nil {
					return errors.New("getting profile '" + *server.Profile + "': " + err.Error())
				}
				if reqInf.StatusCode == http.StatusNotModified {
					log.Infof("Getting config: %v not modified, using old config", "Profile")
					toData.Profile = oldCfg.Profile
				} else {
					log.Infof("Getting config: %v is modified, using new response", "Profile")
					toData.Profile = profile
				}
				toData.MetaData.Profile = MakeReqMetaData(reqInf.RespHeaders)
				toIPs.Store(reqInf.RemoteAddr, nil)
			}
			return nil
		}
		fs := []func() error{dsF, serverParamsF, cdnF, profileF}
		if !revalOnly {
			fs = append([]func() error{sslF}, fs...) // skip ssl keys for reval only, which doesn't need them
		}
		return util.JoinErrs(runParallel(fs))
	}

	cgF := func() error {
		defer func(start time.Time) { log.Infof("cfF took %v\n", time.Since(start)) }(time.Now())
		{
			reqHdr := (http.Header)(nil)
			if oldCfg != nil {
				reqHdr = MakeReqHdr(oldCfg.MetaData.CacheGroups)
			}
			cacheGroups, reqInf, err := toClient.GetCacheGroups(reqHdr)
			if err != nil {
				return errors.New("getting cachegroups: " + err.Error())
			}
			if reqInf.StatusCode == http.StatusNotModified {
				log.Infof("Getting config: %v not modified, using old config", "CacheGroups")
				toData.CacheGroups = oldCfg.CacheGroups
			} else {
				log.Infof("Getting config: %v is modified, using new response", "CacheGroups")
				toData.CacheGroups = cacheGroups
			}
			toData.MetaData.CacheGroups = MakeReqMetaData(reqInf.RespHeaders)
			toIPs.Store(reqInf.RemoteAddr, nil)
		}
		return nil
	}
	jobsF := func() error {
		defer func(start time.Time) { log.Infof("jobsF took %v\n", time.Since(start)) }(time.Now())
		{
			reqHdr := (http.Header)(nil)
			if oldCfg != nil {
				reqHdr = MakeReqHdr(oldCfg.MetaData.Jobs)
			}
			jobs, reqInf, err := toClient.GetJobs(reqHdr) // TODO add cdn query param to jobs endpoint
			if err != nil {
				return errors.New("getting jobs: " + err.Error())
			}
			if reqInf.StatusCode == http.StatusNotModified {
				log.Infof("Getting config: %v not modified, using old config", "Jobs")
				toData.Jobs = oldCfg.Jobs
			} else {
				log.Infof("Getting config: %v is modified, using new response", "Jobs")
				toData.Jobs = jobs
			}
			toData.MetaData.Jobs = MakeReqMetaData(reqInf.RespHeaders)
			toIPs.Store(reqInf.RemoteAddr, nil)
		}
		return nil
	}
	capsF := func() error {
		defer func(start time.Time) { log.Infof("capsF took %v\n", time.Since(start)) }(time.Now())
		{
			reqHdr := (http.Header)(nil)
			if oldCfg != nil {
				reqHdr = MakeReqHdr(oldCfg.MetaData.ServerCapabilities)
			}
			log.Infof("Getting config: ServerCapabilities reqHdr %+v", reqHdr)
			caps, reqInf, err := toClient.GetServerCapabilitiesByID(nil, reqHdr) // TODO change to not take a param; it doesn't use it to request TO anyway.
			if err != nil {
				log.Errorln("Server Capabilities error, skipping!")
				// return errors.New("getting server caps from Traffic Ops: " + err.Error())
			} else {
				if reqInf.StatusCode == http.StatusNotModified {
					log.Infof("Getting config: %v not modified, using old config", "ServerCapabilities")
					toData.ServerCapabilities = oldCfg.ServerCapabilities
				} else {
					log.Infof("Getting config: %v is modified, using new response", "ServerCapabilities")
					toData.ServerCapabilities = caps
				}
				toData.MetaData.ServerCapabilities = MakeReqMetaData(reqInf.RespHeaders)
				toIPs.Store(reqInf.RemoteAddr, nil)
			}
		}
		return nil
	}
	dsCapsF := func() error {
		defer func(start time.Time) { log.Infof("dscapsF took %v\n", time.Since(start)) }(time.Now())
		{
			reqHdr := (http.Header)(nil)
			if oldCfg != nil {
				reqHdr = MakeReqHdr(oldCfg.MetaData.DSRequiredCapabilities)
			}
			caps, reqInf, err := toClient.GetDeliveryServiceRequiredCapabilitiesByID(nil, reqHdr)
			if err != nil {
				log.Errorln("DS Required Capabilities error, skipping!")
				// return errors.New("getting DS required capabilities: " + err.Error())
			} else {
				if reqInf.StatusCode == http.StatusNotModified {
					log.Infof("Getting config: %v not modified, using old config", "DSRequiredCapabilities")
					toData.DSRequiredCapabilities = oldCfg.DSRequiredCapabilities
				} else {
					log.Infof("Getting config: %v is modified, using new response", "DSRequiredCapabilities")
					toData.DSRequiredCapabilities = caps
				}
				toData.MetaData.DSRequiredCapabilities = MakeReqMetaData(reqInf.RespHeaders)
				toIPs.Store(reqInf.RemoteAddr, nil)
			}
		}
		return nil
	}
	dsrF := func() error {
		defer func(start time.Time) { log.Infof("dsrF took %v\n", time.Since(start)) }(time.Now())
		{
			reqHdr := (http.Header)(nil)
			if oldCfg != nil {
				reqHdr = MakeReqHdr(oldCfg.MetaData.DeliveryServiceRegexes)
			}
			dsr, reqInf, err := toClient.GetDeliveryServiceRegexes(reqHdr)
			if err != nil {
				return errors.New("getting delivery service regexes: " + err.Error())
			}
			if reqInf.StatusCode == http.StatusNotModified {
				log.Infof("Getting config: %v not modified, using old config", "DeliveryServiceRegexes")
				toData.DeliveryServiceRegexes = oldCfg.DeliveryServiceRegexes
			} else {
				log.Infof("Getting config: %v is modified, using new response", "DeliveryServiceRegexes")
				toData.DeliveryServiceRegexes = dsr
			}
			toData.MetaData.DeliveryServiceRegexes = MakeReqMetaData(reqInf.RespHeaders)
			toIPs.Store(reqInf.RemoteAddr, nil)
		}
		return nil
	}

	cacheKeyConfigParamsF := func() error {
		defer func(start time.Time) { log.Infof("cacheKeyConfigParamsF took %v\n", time.Since(start)) }(time.Now())
		{
			reqHdr := (http.Header)(nil)
			if oldCfg != nil {
				reqHdr = MakeReqHdr(oldCfg.MetaData.CacheKeyConfigParams)
			}
			params, reqInf, err := toClient.GetConfigFileParameters("cachekey.config", reqHdr)
			if err != nil {
				return errors.New("getting cache key parameters: " + err.Error())
			}
			if reqInf.StatusCode == http.StatusNotModified {
				log.Infof("Getting config: %v not modified, using old config", "CacheKeyParams")
				toData.CacheKeyConfigParams = oldCfg.CacheKeyConfigParams
			} else {
				log.Infof("Getting config: %v is modified, using new response", "CacheKeyParams")
				toData.CacheKeyConfigParams = params
			}
			toData.MetaData.CacheKeyConfigParams = MakeReqMetaData(reqInf.RespHeaders)
			toIPs.Store(reqInf.RemoteAddr, nil)
		}
		return nil
	}

	remapConfigParamsF := func() error {
		defer func(start time.Time) { log.Infof("remapConfigParamsF took %v\n", time.Since(start)) }(time.Now())
		{
			reqHdr := (http.Header)(nil)
			if oldCfg != nil {
				reqHdr = MakeReqHdr(oldCfg.MetaData.RemapConfigParams)
			}
			params, reqInf, err := toClient.GetConfigFileParameters(atscfg.CacheKeyParameterConfigFile, reqHdr)
			if err != nil {
				return errors.New("getting cache key parameters: " + err.Error())
			}
			if reqInf.StatusCode == http.StatusNotModified {
				log.Infof("Getting config: %v not modified, using old config", "RemapConfigParams")
				toData.RemapConfigParams = oldCfg.RemapConfigParams
			} else {
				log.Infof("Getting config: %v is modified, using new response", "RemapConfigParams")
				toData.RemapConfigParams = params
			}
			toData.MetaData.RemapConfigParams = MakeReqMetaData(reqInf.RespHeaders)
			toIPs.Store(reqInf.RemoteAddr, nil)
		}
		return nil
	}

	parentConfigParamsF := func() error {
		defer func(start time.Time) { log.Infof("parentConfigParamsF took %v\n", time.Since(start)) }(time.Now())
		{
			reqHdr := (http.Header)(nil)
			if oldCfg != nil {
				reqHdr = MakeReqHdr(oldCfg.MetaData.ParentConfigParams)
			}
			parentConfigParams, reqInf, err := toClient.GetConfigFileParameters("parent.config", reqHdr) // TODO make const in lib/go-atscfg
			if err != nil {
				return errors.New("getting parent.config parameters: " + err.Error())
			}
			if reqInf.StatusCode == http.StatusNotModified {
				log.Infof("Getting config: %v not modified, using old config", "ParentConfigParams")
				toData.ParentConfigParams = oldCfg.ParentConfigParams
			} else {
				log.Infof("Getting config: %v is modified, using new response", "ParentConfigParams")
				toData.ParentConfigParams = parentConfigParams
			}
			toData.MetaData.ParentConfigParams = MakeReqMetaData(reqInf.RespHeaders)
			toIPs.Store(reqInf.RemoteAddr, nil)
		}
		return nil
	}

	topologiesF := func() error {
		defer func(start time.Time) { log.Infof("topologiesF took %v\n", time.Since(start)) }(time.Now())
		{
			reqHdr := (http.Header)(nil)
			if oldCfg != nil {
				reqHdr = MakeReqHdr(oldCfg.MetaData.Topologies)
			}
			topologies, reqInf, err := toClient.GetTopologies(reqHdr)
			if err != nil {
				return errors.New("getting topologies: " + err.Error())
			}
			if reqInf.StatusCode == http.StatusNotModified {
				log.Infof("Getting config: %v not modified, using old config", "Topologies")
				toData.Topologies = oldCfg.Topologies
			} else {
				log.Infof("Getting config: %v is modified, using new response", "Topologies")
				toData.Topologies = topologies
			}
			toData.MetaData.Topologies = MakeReqMetaData(reqInf.RespHeaders)
			toIPs.Store(reqInf.RemoteAddr, nil)
		}
		return nil
	}

	fs := []func() error{serversF, cgF, jobsF}
	if !revalOnly {
		// skip data not needed for reval, if we're reval-only
		fs = append([]func() error{dsrF, cacheKeyConfigParamsF, remapConfigParamsF, parentConfigParamsF, capsF, dsCapsF, topologiesF}, fs...)
	}
	errs := runParallel(fs)

	toAddrSet := map[string]struct{}{} // use a set to remove duplicates
	toIPs.Range(func(key, val interface{}) bool {
		toAddrSet[key.(net.Addr).String()] = struct{}{}
		return true
	})
	for addr, _ := range toAddrSet {
		toData.TrafficOpsAddresses = append(toData.TrafficOpsAddresses, addr)
	}
	toData.TrafficOpsURL = toClient.C.URL

	return toData, util.JoinErrs(errs)
}

// runParallel runs all funcs in fs in parallel goroutines, and returns after all funcs have returned.
// Returns a slice of the errors returned by each func. The order of the errors will not be the same as the order of fs.
// All funcs in fs must be safe to run in parallel.
func runParallel(fs []func() error) []error {
	errs := []error{}
	doneChan := make(chan error, len(fs))
	for _, fPtr := range fs {
		f := fPtr // because functions are pointers, f will change in the loop. Need create a new variable here to close around.
		go func() { doneChan <- f() }()
	}
	for i := 0; i < len(fs); i++ {
		errs = append(errs, <-doneChan)
	}
	return errs
}

func FilterDSS(dsses []tc.DeliveryServiceServer, dsIDs map[int]struct{}, serverIDs map[int]struct{}) []tc.DeliveryServiceServer {
	// TODO filter only DSes on this server's CDN? Does anything ever needs DSS cross-CDN? Surely not.
	//      Then, we can remove a bunch of config files that filter only DSes on the current cdn.
	filtered := []tc.DeliveryServiceServer{}
	for _, dss := range dsses {
		if dss.Server == nil || dss.DeliveryService == nil {
			continue // TODO warn?
		}
		if len(dsIDs) > 0 {
			if _, ok := dsIDs[*dss.DeliveryService]; !ok {
				continue
			}
		}
		if len(serverIDs) > 0 {
			if _, ok := serverIDs[*dss.Server]; !ok {
				continue
			}
		}
		filtered = append(filtered, dss)
	}
	return filtered
}

// FilterParams filters params and returns only the parameters which match configFile, name, and value.
// If configFile, name, or value is the empty string, it is not filtered.
// Returns a slice of parameters.
func FilterParams(params []tc.Parameter, configFile string, name string, value string, omitName string) []tc.Parameter {
	filtered := []tc.Parameter{}
	for _, param := range params {
		if configFile != "" && param.ConfigFile != configFile {
			continue
		}
		if name != "" && param.Name != name {
			continue
		}
		if value != "" && param.Value != value {
			continue
		}
		if omitName != "" && param.Name == omitName {
			continue
		}
		filtered = append(filtered, param)
	}
	return filtered
}

// ParamsToMap converts a []tc.Parameter to a map[paramName]paramValue.
// If multiple params have the same value, the first one in params will be used an an error will be logged.
// See ParamArrToMultiMap.
func ParamsToMap(params []tc.Parameter) map[string]string {
	mp := map[string]string{}
	for _, param := range params {
		if val, ok := mp[param.Name]; ok {
			if val < param.Value {
				log.Errorln("config generation got multiple parameters for name '" + param.Name + "' - ignoring '" + param.Value + "'")
				continue
			} else {
				log.Errorln("config generation got multiple parameters for name '" + param.Name + "' - ignoring '" + val + "'")
			}
		}
		mp[param.Name] = param.Value
	}
	return mp
}

// ParamArrToMultiMap converts a []tc.Parameter to a map[paramName][]paramValue.
func ParamsToMultiMap(params []tc.Parameter) map[string][]string {
	mp := map[string][]string{}
	for _, param := range params {
		mp[param.Name] = append(mp[param.Name], param.Value)
	}
	return mp
}

// filterUnusedDSS removes entries not on the given server's cdn from dsses, and returns a atscfg.DeliveryServiceServer.
func filterUnusedDSS(dsses []tc.DeliveryServiceServer, cdnID int, servers []atscfg.Server, dses []atscfg.DeliveryService) []atscfg.DeliveryServiceServer {
	serverIDs := map[int]struct{}{}
	for _, sv := range servers {
		if sv.ID == nil {
			log.Errorln("filterUnusedDSS got server with nil id, skipping!")
			continue
		} else if sv.CDNID == nil {
			log.Errorln("filterUnusedDSS got server with nil cdnId, skipping!")
			continue
		} else if *sv.CDNID != cdnID {
			continue
		}
		serverIDs[*sv.ID] = struct{}{}
	}
	dsIDs := map[int]struct{}{}
	for _, ds := range dses {
		if ds.ID == nil {
			log.Errorln("filterUnusedDSS got delivery service with nil id, skipping!")
			continue
		} else if ds.CDNID == nil {
			log.Errorln("filterUnusedDSS got delivery service with nil cdnId, skipping!")
			continue
		} else if *ds.CDNID != cdnID {
			continue
		}
		dsIDs[*ds.ID] = struct{}{}
	}

	cfgDSS := []atscfg.DeliveryServiceServer{}
	for _, dss := range dsses {
		if dss.Server == nil {
			log.Errorln("filterUnusedDSS got deliveryserviceserver with nil server, skipping!")
			continue
		} else if dss.DeliveryService == nil {
			log.Errorln("filterUnusedDSS got deliveryserviceserver with nil deliveryservice, skipping!")
			continue
		}
		if _, ok := serverIDs[*dss.Server]; !ok {
			continue // no log, this is normal if servers are fetched per-cdn
		}
		if _, ok := dsIDs[*dss.DeliveryService]; !ok {
			continue // no log, this is normal if dses are fetched per-cdn
		}
		cfgDSS = append(cfgDSS, atscfg.DeliveryServiceServer{Server: *dss.Server, DeliveryService: *dss.DeliveryService})
	}
	return cfgDSS
}
