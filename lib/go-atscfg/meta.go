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
	"path/filepath"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

type CfgMeta struct {
	Name string
	Path string
}

// ConfigFilesListOpts contains settings to configure generation options.
type ConfigFilesListOpts struct {
}

// MakeMetaObj returns the list of config files, any warnings, and any errors.
func MakeConfigFilesList(
	configDir string,
	server *Server,
	serverParams []tc.Parameter,
	deliveryServices []DeliveryService,
	deliveryServiceServers []DeliveryServiceServer,
	globalParams []tc.Parameter,
	cacheGroupArr []tc.CacheGroupNullable,
	topologies []tc.Topology,
	opt *ConfigFilesListOpts,
) ([]CfgMeta, []string, error) {
	if opt == nil {
		opt = &ConfigFilesListOpts{}
	}
	warnings := []string{}

	if server.Cachegroup == nil {
		return nil, warnings, errors.New("this server missing Cachegroup")
	} else if server.CachegroupID == nil {
		return nil, warnings, errors.New("this server missing CachegroupID")
	} else if server.ProfileID == nil {
		return nil, warnings, errors.New("server missing ProfileID")
	} else if server.TCPPort == nil {
		return nil, warnings, errors.New("server missing TCPPort")
	} else if server.HostName == nil {
		return nil, warnings, errors.New("server missing HostName")
	} else if server.CDNID == nil {
		return nil, warnings, errors.New("server missing CDNID")
	} else if server.CDNName == nil {
		return nil, warnings, errors.New("server missing CDNName")
	} else if server.ID == nil {
		return nil, warnings, errors.New("server missing ID")
	} else if server.Profile == nil {
		return nil, warnings, errors.New("server missing Profile")
	}

	atsMajorVer, verWarns := getATSMajorVersion(serverParams)
	warnings = append(warnings, verWarns...)

	tmURL, tmReverseProxyURL := getTOURLAndReverseProxy(globalParams)
	if tmURL == "" {
		warnings = append(warnings, "global tm.url parameter missing or empty! Setting empty in meta config!")
	}

	dses, dsWarns := filterConfigFileDSes(server, deliveryServices, deliveryServiceServers)
	warnings = append(warnings, dsWarns...)

	locationParams := getLocationParams(serverParams)

	uriSignedDSes, signDSWarns := getURISignedDSes(dses)
	warnings = append(warnings, signDSWarns...)

	configFiles := []CfgMeta{}

	if locationParams["remap.config"].Path != "" {
		configLocation := locationParams["remap.config"].Path
		for _, ds := range uriSignedDSes {
			cfgName := "uri_signing_" + string(ds) + ".config"
			// If there's already a parameter for it, don't clobber it. The user may wish to override the location.
			if _, ok := locationParams[cfgName]; !ok {
				p := locationParams[cfgName]
				p.Name = cfgName
				p.Path = configLocation
				locationParams[cfgName] = p
			}
		}
	}

locationParamsFor:
	for cfgFile, cfgParams := range locationParams {
		if strings.HasSuffix(cfgFile, ".config") {
			dsConfigFilePrefixes := []string{
				"hdr_rw_mid_", // must come before hdr_rw_, to avoid thinking we have a "hdr_rw_" with a ds of "mid_x"
				"hdr_rw_",
				"regex_remap_",
				"url_sig_",
				"uri_signing_",
			}
		prefixFor:
			for _, prefix := range dsConfigFilePrefixes {
				if strings.HasPrefix(cfgFile, prefix) {
					dsName := strings.TrimSuffix(strings.TrimPrefix(cfgFile, prefix), ".config")
					if _, ok := dses[tc.DeliveryServiceName(dsName)]; !ok {
						warnings = append(warnings, "server profile had 'location' Parameter '"+cfgFile+"', but delivery Service '"+dsName+"' is not assigned to this Server! Not including in meta config!")
						continue locationParamsFor
					}
					break prefixFor // if it has a prefix, don't check the next prefix. This is important for hdr_rw_mid_, which will match hdr_rw_ and result in a "ds name" of "mid_x" if we don't continue here.
				}
			}
		}

		atsCfg := CfgMeta{
			Name: cfgParams.Name,
			Path: cfgParams.Path,
		}

		configFiles = append(configFiles, atsCfg)
	}

	configFiles, configDirWarns, err := addMetaObjConfigDir(configFiles, configDir, server, tmURL, tmReverseProxyURL, locationParams, uriSignedDSes, dses, cacheGroupArr, topologies, atsMajorVer)
	warnings = append(warnings, configDirWarns...)
	return configFiles, warnings, err
}

// addMetaObjConfigDir takes the Meta Object generated from TO data, and the ATS config directory
// and prepends the config directory to all relative paths,
// and creates MetaObj entries for all required config files which have no location parameter.
// If configDir is empty and any location Parameters have relative paths, returns an error.
// Returns the amended config files list, any warnings, and any error.
func addMetaObjConfigDir(
	configFiles []CfgMeta,
	configDir string,
	server *Server,
	tmURL string, // global tm.url Parameter
	tmReverseProxyURL string, // global tm.rev_proxy.url Parameter
	locationParams map[string]configProfileParams, // map[configFile]params; 'location' and 'URL' Parameters on serverHostName's Profile
	uriSignedDSes []tc.DeliveryServiceName,
	dses map[tc.DeliveryServiceName]DeliveryService,
	cacheGroupArr []tc.CacheGroupNullable,
	topologies []tc.Topology,
	atsMajorVer int,
) ([]CfgMeta, []string, error) {
	warnings := []string{}

	if server.Cachegroup == nil {
		return nil, warnings, errors.New("server missing Cachegroup")
	}

	cacheGroups, err := makeCGMap(cacheGroupArr)
	if err != nil {
		return nil, warnings, errors.New("making CG map: " + err.Error())
	}

	// Note there may be multiple files with the same name in different directories.
	configFilesM := map[string][]CfgMeta{} // map[fileShortName]CfgMeta
	for _, fi := range configFiles {
		configFilesM[fi.Name] = append(configFilesM[fi.Path], fi)
	}

	// add all strictly required files, all of which should be in the base config directory.
	// If they don't exist, create them.
	// If they exist with a relative path, prepend configDir.
	// If any exist with a relative path, or don't exist, and configDir is empty, return an error.
	for _, fileName := range requiredFiles(atsMajorVer) {
		if _, ok := configFilesM[fileName]; ok {
			continue
		}
		if configDir == "" {
			return nil, warnings, errors.New("required file '" + fileName + "' has no location Parameter, and ATS config directory not found.")
		}
		configFilesM[fileName] = []CfgMeta{{
			Name: fileName,
			Path: configDir,
		}}
	}

	for fileName, fis := range configFilesM {
		newFis := []CfgMeta{}
		for _, fi := range fis {
			if !filepath.IsAbs(fi.Path) {
				if configDir == "" {
					return nil, warnings, errors.New("file '" + fileName + "' has location Parameter with relative path '" + fi.Path + "', but ATS config directory was not found.")
				}
				absPath := filepath.Join(configDir, fi.Path)
				fi.Path = absPath
			}
			newFis = append(newFis, fi)
		}
		configFilesM[fileName] = newFis
	}

	nameTopologies := makeTopologyNameMap(topologies)

	for _, ds := range dses {
		if ds.XMLID == nil {
			warnings = append(warnings, "got Delivery Service with nil XMLID - not considering!")
			continue
		}

		err := error(nil)
		// Note we log errors, but don't return them.
		// If an individual DS has an error, we don't want to break the rest of the CDN.
		if ds.Topology != nil && *ds.Topology != "" {
			topology := nameTopologies[TopologyName(*ds.Topology)]

			placement, err := getTopologyPlacement(tc.CacheGroupName(*server.Cachegroup), topology, cacheGroups, &ds)
			if err != nil {
				return nil, warnings, errors.New("getting topology placement: " + err.Error())
			}
			if placement.IsFirstCacheTier {
				if (ds.FirstHeaderRewrite != nil && *ds.FirstHeaderRewrite != "") || ds.MaxOriginConnections != nil || ds.ServiceCategory != nil {
					fileName := FirstHeaderRewriteConfigFileName(*ds.XMLID)
					if configFilesM, err = ensureConfigFile(configFilesM, fileName, configDir); err != nil {
						warnings = append(warnings, "ensuring config file '"+fileName+"': "+err.Error())
					}
				}
			}
			if placement.IsInnerCacheTier {
				if (ds.InnerHeaderRewrite != nil && *ds.InnerHeaderRewrite != "") || ds.MaxOriginConnections != nil || ds.ServiceCategory != nil {
					fileName := InnerHeaderRewriteConfigFileName(*ds.XMLID)
					if configFilesM, err = ensureConfigFile(configFilesM, fileName, configDir); err != nil {
						warnings = append(warnings, "ensuring config file '"+fileName+"': "+err.Error())
					}
				}
			}
			if placement.IsLastCacheTier {
				if (ds.LastHeaderRewrite != nil && *ds.LastHeaderRewrite != "") || ds.MaxOriginConnections != nil || ds.ServiceCategory != nil {
					fileName := LastHeaderRewriteConfigFileName(*ds.XMLID)
					if configFilesM, err = ensureConfigFile(configFilesM, fileName, configDir); err != nil {
						warnings = append(warnings, "ensuring config file '"+fileName+"': "+err.Error())
					}
				}
			}
		} else if strings.HasPrefix(server.Type, tc.EdgeTypePrefix) {
			if (ds.EdgeHeaderRewrite != nil || ds.MaxOriginConnections != nil || ds.ServiceCategory != nil) &&
				strings.HasPrefix(server.Type, tc.EdgeTypePrefix) {
				fileName := "hdr_rw_" + *ds.XMLID + ".config"
				if configFilesM, err = ensureConfigFile(configFilesM, fileName, configDir); err != nil {
					warnings = append(warnings, "ensuring config file '"+fileName+"': "+err.Error())
				}
			}
		} else if strings.HasPrefix(server.Type, tc.MidTypePrefix) {
			if (ds.MidHeaderRewrite != nil || ds.MaxOriginConnections != nil || ds.ServiceCategory != nil) &&
				ds.Type != nil && ds.Type.UsesMidCache() &&
				strings.HasPrefix(server.Type, tc.MidTypePrefix) {
				fileName := "hdr_rw_mid_" + *ds.XMLID + ".config"
				if configFilesM, err = ensureConfigFile(configFilesM, fileName, configDir); err != nil {
					warnings = append(warnings, "ensuring config file '"+fileName+"': "+err.Error())
				}
			}
		}
		if ds.RegexRemap != nil {
			configFile := "regex_remap_" + *ds.XMLID + ".config"
			if configFilesM, err = ensureConfigFile(configFilesM, configFile, configDir); err != nil {
				warnings = append(warnings, "ensuring config file '"+configFile+"': "+err.Error())
			}
		}
		if ds.SigningAlgorithm != nil && *ds.SigningAlgorithm == tc.SigningAlgorithmURLSig {
			configFile := "url_sig_" + *ds.XMLID + ".config"
			if configFilesM, err = ensureConfigFile(configFilesM, configFile, configDir); err != nil {
				warnings = append(warnings, "ensuring config file '"+configFile+"': "+err.Error())
			}
		}
		if ds.SigningAlgorithm != nil && *ds.SigningAlgorithm == tc.SigningAlgorithmURISigning {
			configFile := "uri_signing_" + *ds.XMLID + ".config"
			if configFilesM, err = ensureConfigFile(configFilesM, configFile, configDir); err != nil {
				warnings = append(warnings, "ensuring config file '"+configFile+"': "+err.Error())
			}
		}
	}

	newFiles := []CfgMeta{}
	for _, fis := range configFilesM {
		for _, fi := range fis {
			newFiles = append(newFiles, fi)
		}
	}
	return newFiles, warnings, nil
}

// getURISignedDSes returns the URI-signed Delivery Services, and any warnings.
func getURISignedDSes(dses map[tc.DeliveryServiceName]DeliveryService) ([]tc.DeliveryServiceName, []string) {
	warnings := []string{}

	uriSignedDSes := []tc.DeliveryServiceName{}
	for _, ds := range dses {
		if ds.ID == nil {
			warnings = append(warnings, "got delivery service with no id, skipping!")
			continue
		}
		if ds.XMLID == nil {
			warnings = append(warnings, "got delivery service with no xmlId (name), skipping!")
			continue
		}
		if _, ok := dses[tc.DeliveryServiceName(*ds.XMLID)]; !ok {
			continue // skip: this ds isn't assigned to this server, this is normal
		}
		if ds.SigningAlgorithm == nil || *ds.SigningAlgorithm != tc.SigningAlgorithmURISigning {
			continue // not signed, so not in our list of signed dses to make config files for.
		}
		uriSignedDSes = append(uriSignedDSes, tc.DeliveryServiceName(*ds.XMLID))
	}

	return uriSignedDSes, warnings
}

// filterConfigFileDSes returns the DSes that should have config files for the given server.
// Returns the delivery services and any warnings.
func filterConfigFileDSes(server *Server, deliveryServices []DeliveryService, deliveryServiceServers []DeliveryServiceServer) (map[tc.DeliveryServiceName]DeliveryService, []string) {
	warnings := []string{}

	dses := map[tc.DeliveryServiceName]DeliveryService{}

	if tc.CacheTypeFromString(server.Type) != tc.CacheTypeMid {
		dsIDs := map[int]struct{}{}
		for _, ds := range deliveryServices {
			if ds.ID == nil {
				warnings = append(warnings, "got delivery service with no ID, skipping!")
				continue
			}
			dsIDs[*ds.ID] = struct{}{}
		}

		// TODO verify?
		//		serverIDs := []int{server.ID}

		dssMap := map[int]struct{}{}
		for _, dss := range deliveryServiceServers {
			if dss.Server != *server.ID {
				continue
			}
			if _, ok := dsIDs[dss.DeliveryService]; !ok {
				continue
			}
			dssMap[dss.DeliveryService] = struct{}{}
		}

		for _, ds := range deliveryServices {
			if ds.ID == nil {
				warnings = append(warnings, "got deliveryservice with nil id, skipping!")
				continue
			}
			if ds.XMLID == nil {
				warnings = append(warnings, "got deliveryservice with nil xmlId (name), skipping!")
				continue
			}
			if _, ok := dssMap[*ds.ID]; !ok && ds.Topology == nil {
				continue
			}
			dses[tc.DeliveryServiceName(*ds.XMLID)] = ds
		}
	} else {
		for _, ds := range deliveryServices {
			if ds.ID == nil {
				warnings = append(warnings, "got deliveryservice with nil id, skipping!")
				continue
			}
			if ds.XMLID == nil {
				warnings = append(warnings, "got deliveryservice with nil xmlId (name), skipping!")
				continue
			}
			if ds.CDNID == nil || *ds.CDNID != *server.CDNID {
				continue
			}
			dses[tc.DeliveryServiceName(*ds.XMLID)] = ds
		}
	}
	return dses, warnings
}

// getTOURLAndReverseProxy returns the toURL and toReverseProxyURL if they exist, or empty strings if they don't.
func getTOURLAndReverseProxy(globalParams []tc.Parameter) (string, string) {
	toReverseProxyURL := ""
	toURL := ""
	for _, param := range globalParams {
		if param.Name == "tm.rev_proxy.url" {
			toReverseProxyURL = param.Value
		} else if param.Name == "tm.url" {
			toURL = param.Value
		}
		if toReverseProxyURL != "" && toURL != "" {
			break
		}
	}
	return toURL, toReverseProxyURL
}

func getLocationParams(serverParams []tc.Parameter) map[string]configProfileParams {
	locationParams := map[string]configProfileParams{}
	for _, param := range serverParams {
		if param.Name == "location" {
			p := locationParams[param.ConfigFile]
			p.Name = param.ConfigFile
			p.Path = param.Value
			locationParams[param.ConfigFile] = p
		}
	}
	return locationParams
}

type configProfileParams struct {
	Name string
	Path string
}

func requiredFiles(atsMajorVer int) []string {
	if atsMajorVer >= 9 {
		return requiredFiles9()
	}
	return requiredFiles8()
}

// requiredFiles8 the list of config files required by ATS 8.
// Note these are not exhaustive. This is only used to error if these are missing.
// The presence of these is no guarantee the location Parameters are complete and correct.
func requiredFiles8() []string {
	return []string{
		"cache.config",
		"hosting.config",
		"ip_allow.config",
		"parent.config",
		"plugin.config",
		"records.config",
		"remap.config",
		"ssl_server_name.yaml",
		"storage.config",
		"volume.config",
	}
}

// requiredFiles9 is the list of config files required by ATS 9.
// Note these are not exhaustive. This is only used to error if these are missing.
// The presence of these is no guarantee the location Parameters are complete and correct.
func requiredFiles9() []string {
	return []string{
		"cache.config",
		"hosting.config",
		"ip_allow.yaml",
		"parent.config",
		"plugin.config",
		"records.config",
		"remap.config",
		"sni.yaml",
		"storage.config",
		"volume.config",
	}
}

// ensureConfigFile ensures files contains the given fileName. If so, returns files unmodified.
// If not, if configDir is empty, returns an error.
// If not, and configDir is nonempty, creates the given file, configDir location, and returns files.
func ensureConfigFile(files map[string][]CfgMeta, fileName string, configDir string) (map[string][]CfgMeta, error) {
	if _, ok := files[fileName]; ok {
		return files, nil
	}
	if configDir == "" {
		return files, errors.New("required file '" + fileName + "' has no location Parameter, and ATS config directory not found.")
	}
	files[fileName] = []CfgMeta{{
		Name: fileName,
		Path: configDir,
	}}
	return files, nil
}
