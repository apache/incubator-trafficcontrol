package cfgfile

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
	"github.com/apache/trafficcontrol/cache-config/t3c-generate/config"
	"github.com/apache/trafficcontrol/cache-config/t3cutil"
	"github.com/apache/trafficcontrol/lib/go-atscfg"
)

//
// This file has wrappers that turn lib/go-atscfg Make funcs into ConfigFileFunc types.
//
// We don't want to make lib/go-atscfg functions take a TOData, because then users wanting to generate a single file would have to fetch all kinds of data that file doesn't need, or else pass objects they know it doesn't currently need as nil and risk it crashing if that func is changed to use it in the future.
//
// But it's useful to map filenames to functions for dispatch. Hence these wrappers.
//

// MakeConfigFilesList returns the list of config files, any warnings, and any error.
func MakeConfigFilesList(toData *t3cutil.ConfigData, dir string) ([]atscfg.CfgMeta, []string, error) {
	configFiles, warnings, err := atscfg.MakeConfigFilesList(
		dir,
		toData.Server,
		toData.ServerParams,
		toData.DeliveryServices,
		toData.DeliveryServiceServers,
		toData.GlobalParams,
		toData.CacheGroups,
		toData.Topologies,
	)
	return configFiles, warnings, err
}

func Make12MFacts(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.Make12MFacts(toData.Server, hdrCommentTxt)
}

func MakeATSDotRules(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeATSDotRules(toData.Server, toData.ServerParams, hdrCommentTxt)
}

func MakeAstatsDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeAStatsDotConfig(toData.Server, toData.ServerParams, hdrCommentTxt)
}

func MakeBGFetchDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeBGFetchDotConfig(toData.Server, hdrCommentTxt)
}

func MakeCacheDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeCacheDotConfig(toData.Server, toData.Servers, toData.DeliveryServices, toData.DeliveryServiceServers, hdrCommentTxt)
}

func MakeChkconfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeChkconfig(toData.ServerParams)
}

func MakeDropQStringDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeDropQStringDotConfig(toData.Server, toData.ServerParams, hdrCommentTxt)
}

func MakeHostingDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeHostingDotConfig(toData.Server, toData.Servers, toData.ServerParams, toData.DeliveryServices, toData.DeliveryServiceServers, toData.Topologies, hdrCommentTxt)
}

func MakeIPAllowDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeIPAllowDotConfig(
		toData.ServerParams,
		toData.Server,
		toData.Servers,
		toData.CacheGroups,
		toData.Topologies,
		hdrCommentTxt,
	)
}

func MakeIPAllowDotYAML(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeIPAllowDotYAML(
		toData.ServerParams,
		toData.Server,
		toData.Servers,
		toData.CacheGroups,
		toData.Topologies,
		hdrCommentTxt,
	)
}

func MakeLoggingDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeLoggingDotConfig(toData.Server, toData.ServerParams, hdrCommentTxt)
}

func MakeLoggingDotYAML(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeLoggingDotYAML(toData.Server, toData.ServerParams, hdrCommentTxt)
}

func MakeSSLServerNameYAML(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeSSLServerNameYAML(
		toData.Server,
		toData.DeliveryServices,
		toData.DeliveryServiceServers,
		toData.DeliveryServiceRegexes,
		toData.ParentConfigParams,
		toData.CDN,
		toData.Topologies,
		toData.CacheGroups,
		toData.ServerCapabilities,
		toData.DSRequiredCapabilities,
		atscfg.SSLServerNameYAMLOpts{
			HdrComment:         hdrCommentTxt,
			VerboseComments:    true, // TODO add a CLI flag
			DefaultTLSVersions: cfg.DefaultTLSVersions,
			DefaultEnableH2:    cfg.DefaultEnableH2,
		},
	)
}

func MakeSNIDotYAML(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeSNIDotYAML(
		toData.Server,
		toData.DeliveryServices,
		toData.DeliveryServiceServers,
		toData.DeliveryServiceRegexes,
		toData.ParentConfigParams,
		toData.CDN,
		toData.Topologies,
		toData.CacheGroups,
		toData.ServerCapabilities,
		toData.DSRequiredCapabilities,
		atscfg.SNIDotYAMLOpts{
			HdrComment:         hdrCommentTxt,
			VerboseComments:    true, // TODO add a CLI flag
			DefaultTLSVersions: cfg.DefaultTLSVersions,
			DefaultEnableH2:    cfg.DefaultEnableH2,
		},
	)
}

func MakeLogsXMLDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeLogsXMLDotConfig(toData.Server, toData.ServerParams, hdrCommentTxt)
}

func MakePackages(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakePackages(toData.ServerParams)
}

func MakeParentDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeParentDotConfig(
		toData.DeliveryServices,
		toData.Server,
		toData.Servers,
		toData.Topologies,
		toData.ServerParams,
		toData.ParentConfigParams,
		toData.ServerCapabilities,
		toData.DSRequiredCapabilities,
		toData.CacheGroups,
		toData.DeliveryServiceServers,
		toData.CDN,
		atscfg.ParentConfigOpts{
			HdrComment:  hdrCommentTxt,
			AddComments: cfg.ParentComments, // TODO add a CLI flag?
		},
	)
}

func MakePluginDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakePluginDotConfig(toData.Server, toData.ServerParams, hdrCommentTxt)
}

func MakeRecordsDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeRecordsDotConfig(
		toData.Server,
		toData.ServerParams,
		hdrCommentTxt,
		atscfg.RecordsConfigOpts{
			ReleaseViaStr:           cfg.ViaRelease,
			DNSLocalBindServiceAddr: cfg.SetDNSLocalBind,
		},
	)
}

func MakeRegexRevalidateDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeRegexRevalidateDotConfig(toData.Server, toData.DeliveryServices, toData.GlobalParams, toData.Jobs, hdrCommentTxt)
}

func MakeRemapDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeRemapDotConfig(
		toData.Server,
		toData.DeliveryServices,
		toData.DeliveryServiceServers,
		toData.DeliveryServiceRegexes,
		toData.ServerParams,
		toData.CDN,
		append(toData.RemapConfigParams, toData.CacheKeyConfigParams...),
		toData.Topologies,
		toData.CacheGroups,
		toData.ServerCapabilities,
		toData.DSRequiredCapabilities,
		hdrCommentTxt,
	)
}

func MakeSSLMultiCertDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeSSLMultiCertDotConfig(toData.Server, toData.DeliveryServices, hdrCommentTxt)
}

func MakeStorageDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeStorageDotConfig(toData.Server, toData.ServerParams, hdrCommentTxt)
}

func MakeSysCtlDotConf(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeSysCtlDotConf(toData.Server, toData.ServerParams, hdrCommentTxt)
}

func MakeVolumeDotConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeVolumeDotConfig(toData.Server, toData.ServerParams, hdrCommentTxt)
}

func MakeHeaderRewrite(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeHeaderRewriteDotConfig(
		fileName,
		toData.DeliveryServices,
		toData.DeliveryServiceServers,
		toData.Server,
		toData.Servers,
		toData.CacheGroups,
		toData.ServerParams,
		toData.ServerCapabilities,
		toData.DSRequiredCapabilities,
		toData.Topologies,
		hdrCommentTxt,
	)
}

func MakeRegexRemap(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeRegexRemapDotConfig(fileName, toData.Server, toData.DeliveryServices, hdrCommentTxt)
}

func MakeSetDSCP(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeSetDSCPDotConfig(fileName, toData.Server, hdrCommentTxt)
}

func MakeURLSigConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeURLSigConfig(fileName, toData.Server, toData.ServerParams, toData.URLSigKeys, hdrCommentTxt)
}

func MakeURISigningConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeURISigningConfig(fileName, toData.URISigningKeys)
}

func MakeUnknownConfig(toData *t3cutil.ConfigData, fileName string, hdrCommentTxt string, cfg config.Cfg) (atscfg.Cfg, error) {
	return atscfg.MakeServerUnknown(fileName, toData.Server, toData.ServerParams, hdrCommentTxt)
}
