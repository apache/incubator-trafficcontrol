package config

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
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-atscfg"
	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"

	"github.com/pborman/getopt/v2"
)

const AppName = "t3c-generate"
const Version = "0.3"
const AppVersion = AppName + "/" + Version

const ExitCodeSuccess = 0
const ExitCodeErrGeneric = 1
const ExitCodeNotFound = 104
const ExitCodeBadRequest = 100

var ErrNotFound = errors.New("not found")
var ErrBadRequest = errors.New("bad request")

type Cfg struct {
	ListPlugins        bool
	LogLocationErr     string
	LogLocationInfo    string
	LogLocationDebug   string
	LogLocationWarn    string
	RevalOnly          bool
	Dir                string
	ViaRelease         bool
	SetDNSLocalBind    bool
	ParentComments     bool
	DefaultEnableH2    bool
	DefaultTLSVersions []atscfg.TLSVersion
}

func (cfg Cfg) ErrorLog() log.LogLocation   { return log.LogLocation(cfg.LogLocationErr) }
func (cfg Cfg) WarningLog() log.LogLocation { return log.LogLocation(cfg.LogLocationWarn) }
func (cfg Cfg) InfoLog() log.LogLocation    { return log.LogLocation(cfg.LogLocationInfo) }
func (cfg Cfg) DebugLog() log.LogLocation   { return log.LogLocation(cfg.LogLocationDebug) }
func (cfg Cfg) EventLog() log.LogLocation   { return log.LogLocation(log.LogLocationNull) } // app doesn't use the event logger.

// GetCfg gets the application configuration, from arguments and environment variables.
func GetCfg() (Cfg, error) {
	version := getopt.BoolLong("version", 'V', "Print version information and exit.")
	listPlugins := getopt.BoolLong("list-plugins", 'l', "Print the list of plugins.")
	help := getopt.BoolLong("help", 'h', "Print usage information and exit")
	revalOnly := getopt.BoolLong("revalidate-only", 'y', "Whether to exclude files not named 'regex_revalidate.config'")
	dir := getopt.StringLong("dir", 'D', "", "ATS config directory, used for config files without location parameters or with relative paths. May be blank. If blank and any required config file location parameter is missing or relative, will error.")
	viaRelease := getopt.BoolLong("via-string-release", 'r', "Whether to use the Release value from the RPM package as a replacement for the ATS version specified in the build that is returned in the Via and Server headers from ATS.")
	dnsLocalBind := getopt.BoolLong("dns-local-bind", 'b', "Whether to use the server's Service Addresses to set the ATS DNS local bind address.")
	disableParentConfigComments := getopt.BoolLong("disable-parent-config-comments", 'c', "Disable adding a comments to parent.config individual lines")
	defaultEnableH2 := getopt.BoolLong("default-client-enable-h2", '2', "Whether to enable HTTP/2 on Delivery Services by default, if they have no explicit Parameter. This is irrelevant if ATS records.config is not serving H2. If omitted, H2 is disabled.")
	defaultTLSVersionsStr := getopt.StringLong("default-client-tls-versions", 'T', "", "Comma-delimited list of default TLS versions for Delivery Services with no Parameter, e.g. '--default-tls-versions=1.1,1.2,1.3'. If omitted, all versions are enabled.")
	verbosePtr := getopt.CounterLong("verbose", 'v', `Log verbosity. Logging is output to stderr. By default, errors are logged. To log warnings, pass '-v'. To log info, pass '-vv'. To omit error logging, see '-s'`)
	silentPtr := getopt.BoolLong("silent", 's', `Silent. Errors are not logged, and the 'verbose' flag is ignored. If a fatal error occurs, the return code will be non-zero but no text will be output to stderr`)

	getopt.Parse()

	if *version {
		fmt.Println(AppName + " v" + Version)
		os.Exit(0)
	} else if *help {
		getopt.PrintUsage(os.Stdout)
		os.Exit(0)
	} else if *listPlugins {
		return Cfg{ListPlugins: true}, nil
	}

	logLocationError := log.LogLocationStderr
	logLocationWarn := log.LogLocationNull
	logLocationInfo := log.LogLocationNull
	logLocationDebug := log.LogLocationNull
	if *silentPtr {
		logLocationError = log.LogLocationNull
	} else {
		if *verbosePtr >= 1 {
			logLocationWarn = log.LogLocationStderr
		}
		if *verbosePtr >= 2 {
			logLocationInfo = log.LogLocationStderr
			logLocationDebug = log.LogLocationStderr // t3c only has 3 verbosity options: none (-s), error (default or --verbose=0), warning (-v), and info (-vv). Any code calling log.Debug is treated as Info.
		}
	}

	if *verbosePtr > 2 {
		return Cfg{}, errors.New("Too many verbose options. The maximum log verbosity level is 2 (-vv or --verbose=2) for errors (0), warnings (1), and info (2)")
	}

	defaultTLSVersions := atscfg.DefaultDefaultTLSVersions

	*defaultTLSVersionsStr = strings.TrimSpace(*defaultTLSVersionsStr)
	if len(*defaultTLSVersionsStr) > 0 {
		defaultTLSVersionsStrs := strings.Split(*defaultTLSVersionsStr, ",")

		defaultTLSVersions = []atscfg.TLSVersion{}
		for _, tlsVersionStr := range defaultTLSVersionsStrs {
			tlsVersion := atscfg.StringToTLSVersion(tlsVersionStr)
			if tlsVersion == atscfg.TLSVersionInvalid {
				return Cfg{}, errors.New("unknown TLS Version '" + tlsVersionStr + "' in '" + *defaultTLSVersionsStr + "'")
			}
			defaultTLSVersions = append(defaultTLSVersions, tlsVersion)
		}
	}

	cfg := Cfg{
		LogLocationErr:     logLocationError,
		LogLocationWarn:    logLocationWarn,
		LogLocationInfo:    logLocationInfo,
		LogLocationDebug:   logLocationDebug,
		ListPlugins:        *listPlugins,
		RevalOnly:          *revalOnly,
		Dir:                *dir,
		ViaRelease:         *viaRelease,
		SetDNSLocalBind:    *dnsLocalBind,
		ParentComments:     !(*disableParentConfigComments),
		DefaultEnableH2:    *defaultEnableH2,
		DefaultTLSVersions: defaultTLSVersions,
	}
	if err := log.InitCfg(cfg); err != nil {
		return Cfg{}, errors.New("Initializing loggers: " + err.Error() + "\n")
	}
	return cfg, nil
}

func ValidateURL(u *url.URL) error {
	if u == nil {
		return errors.New("nil url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("scheme expected 'http' or 'https', actual '" + u.Scheme + "'")
	}
	if strings.TrimSpace(u.Host) == "" {
		return errors.New("no host")
	}
	return nil
}

// TOData is the Traffic Ops data needed to generate configs.
// See each field for details on the data required.
// - If a field says 'must', the creation of TOData is guaranteed to do so, and users of the struct may rely on that.
// - If it says 'may', the creation may or may not do so, and therefore users of the struct must filter if they
//   require the potential fields to be omitted to generate correctly.
type TOData struct {
	// Servers must be all the servers from Traffic Ops. May include servers not on the current cdn.
	Servers []atscfg.Server

	// CacheGroups must be all cachegroups in Traffic Ops with Servers on the current server's cdn. May also include CacheGroups without servers on the current cdn.
	CacheGroups []tc.CacheGroupNullable

	// GlobalParams must be all Parameters in Traffic Ops on the tc.GlobalProfileName Profile. Must not include other parameters.
	GlobalParams []tc.Parameter

	// ServerParams must be all Parameters on the Profile of the current server. Must not include other Parameters.
	ServerParams []tc.Parameter

	// RemapConfigParams must be all Parameters with the ConfigFile "remap.config". Also includes cachekey.config parameters
	RemapConfigParams []tc.Parameter

	// ParentConfigParams must be all Parameters with the ConfigFile "parent.config.
	ParentConfigParams []tc.Parameter

	// DeliveryServices must include all Delivery Services on the current server's cdn, including those not assigned to the server. Must not include delivery services on other cdns.
	DeliveryServices []atscfg.DeliveryService

	// DeliveryServiceServers must include all delivery service servers in Traffic Ops for all delivery services on the current cdn, including those not assigned to the current server.
	DeliveryServiceServers []tc.DeliveryServiceServer

	// Server must be the server we're fetching configs from
	Server *atscfg.Server

	// Jobs must be all Jobs on the server's CDN. May include jobs on other CDNs.
	Jobs []tc.Job

	// CDN must be the CDN of the server.
	CDN *tc.CDN

	// DeliveryServiceRegexes must be all regexes on all delivery services on this server's cdn.
	DeliveryServiceRegexes []tc.DeliveryServiceRegexes

	// Profile must be the Profile of the server being requested.
	Profile tc.Profile

	// URISigningKeys must be a map of every delivery service which is URI Signed, to its keys.
	URISigningKeys map[tc.DeliveryServiceName][]byte

	// URLSigKeys must be a map of every delivery service which uses URL Sig, to its keys.
	URLSigKeys map[tc.DeliveryServiceName]tc.URLSigKeys

	// ServerCapabilities must be a map of all server IDs on this server's CDN, to a set of their capabilities. May also include servers from other cdns.
	ServerCapabilities map[int]map[atscfg.ServerCapability]struct{}

	// DSRequiredCapabilities must be a map of all delivery service IDs on this server's CDN, to a set of their required capabilities. Delivery Services with no required capabilities may not have an entry in the map.
	DSRequiredCapabilities map[int]map[atscfg.ServerCapability]struct{}

	// SSLKeys must be all the ssl keys for the server's cdn.
	SSLKeys []tc.CDNSSLKeys

	// Topologies must be all the topologies for the server's cdn.
	// May incude topologies of other cdns.
	Topologies []tc.Topology
}
