package main

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
	"fmt"
	"os"
	"strings"

	"github.com/apache/trafficcontrol/cache-config/t3cutil"

	"github.com/pborman/getopt/v2"
)

func main() {
	// presumably calculated by by t3c-check-refs
	// TODO remove? The blueprint says t3c/ORT will no longer install packages
	pluginPackagesInstalledStr := getopt.StringLong("plugin-packages-installed", 'p', "", "comma-delimited list of ATS plugin packages which were installed by t3c")
	// presumably calculated by t3c-diff
	changedConfigFilesStr := getopt.StringLong("changed-config-paths", 'c', "", "comma-delimited list of the full paths of all files changed by t3c")
	help := getopt.BoolLong("help", 'h', "Print usage information and exit")
	getopt.Parse()

	if *help {
		getopt.PrintUsage(os.Stdout)
		os.Exit(0)
	}

	changedConfigFiles := strings.Split(*changedConfigFilesStr, ",")
	changedConfigFiles = StrMap(changedConfigFiles, strings.TrimSpace)
	changedConfigFiles = StrRemoveIf(changedConfigFiles, StrIsEmpty)

	// TODO determine if determining which installed packages were plugins should be part of this app's job?
	// Probably not, because whatever told the installer to install them already knew that,
	// we shouldn't re-calculate it.

	pluginPackagesInstalled := strings.Split(*pluginPackagesInstalledStr, ",")
	pluginPackagesInstalled = StrMap(pluginPackagesInstalled, strings.TrimSpace)
	pluginPackagesInstalled = StrRemoveIf(pluginPackagesInstalled, StrIsEmpty)

	// ATS restart is needed if:
	// [x] 1. mode was badass
	// [x] 2. any ATS Plugin was installed
	// [x] 3. plugin.config or 50-ats.rules was changed
	// [ ] 4. package 'trafficserver' was installed

	// ATS reload is needed if:
	// [ ] 1. new SSL keys were installed AND ssl_multicert.config was changed
	// [ ] 2. any of the following were changed: url_sig*, uri_signing*, hdr_rw*, (plugin.config), (50-ats.rules),
	//        ssl/*.cer, ssl/*.key, anything else in /trafficserver,
	//

	if len(pluginPackagesInstalled) > 0 {
		ExitRestart()
	}

	for _, fileRequiringRestart := range configFilesRequiringRestart {
		for _, changedPath := range changedConfigFiles {
			if strings.HasSuffix(changedPath, fileRequiringRestart) {
				ExitRestart()
			}
		}
	}

	for _, path := range changedConfigFiles {
		// TODO add && ssl keys install
		if strings.Contains(path, "ssl_multicert.config") /* && sslKeysInstalled */ {
			ExitReload()
		}
		if strings.Contains(path, "/trafficserver/") {
			ExitReload()
		}
		if strings.Contains(path, "hdr_rw_") ||
			strings.Contains(path, "url_sig_") ||
			strings.Contains(path, "uri_signing_") ||
			strings.Contains(path, "plugin.config") ||
			strings.Contains(path, "50-ats.rules") {
			ExitReload()
		}
	}

	ExitNothing()
}

// ExitRestart returns the "needs restart" message and exits.
func ExitRestart() {
	fmt.Fprintf(os.Stdout, t3cutil.ServiceNeedsRestart.String()+"\n")
	os.Exit(0)
}

// ExitReload returns the "needs reload" message and exits.
func ExitReload() {
	fmt.Fprintf(os.Stdout, t3cutil.ServiceNeedsReload.String()+"\n")
	os.Exit(0)
}

// ExitNothing returns the "needs nothing" message and exits.
func ExitNothing() {
	os.Exit(0)
}

var configFilesRequiringRestart = []string{"plugin.config", "50-ats.rules"}

// StrMap applies the given function fn to all strings in strs.
func StrMap(strs []string, fn func(str string) string) []string {
	news := make([]string, 0, len(strs))
	for _, str := range strs {
		news = append(news, fn(str))
	}
	return news
}

// StrRemoveIf removes all strings in strs for which fn returns true.
func StrRemoveIf(strs []string, fn func(str string) bool) []string {
	news := []string{}
	for _, str := range strs {
		if fn(str) {
			continue
		}
		news = append(news, str)
	}
	return news
}

// StrIsEmpty returns whether str == "". Helper function for composing with other functions.
func StrIsEmpty(str string) bool { return str == "" }
