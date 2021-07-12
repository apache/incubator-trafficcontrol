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
	"encoding/json"
	"sort"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const PackagesFileName = `packages`
const PackagesParamConfigFile = `package`

const ContentTypePackages = ContentTypeTextASCII
const LineCommentPackages = ""

// PackagesOpts contains settings to configure generation options.
type PackagesOpts struct {
}

// MakePackages returns the 'packages' ATS config file endpoint.
// This is a JSON object, and should be served with an 'application/json' Content-Type.
func MakePackages(
	serverParams []tc.Parameter,
	opts *PackagesOpts,
) (Cfg, error) {
	if opts == nil {
		opts = &PackagesOpts{}
	}
	warnings := []string{}

	params := paramsToMultiMap(filterParams(serverParams, PackagesParamConfigFile, "", "", ""))

	pkgs := []pkg{}
	for name, versions := range params {
		for _, version := range versions {
			pkgs = append(pkgs, pkg{Name: name, Version: version})
		}
	}
	sort.Sort(packages(pkgs))
	bts, err := json.Marshal(&pkgs)
	if err != nil {
		// should never happen
		return Cfg{}, makeErr(warnings, "marshalling chkconfig NameVersions: "+err.Error())
	}

	return Cfg{
		Text:        string(bts),
		ContentType: ContentTypePackages,
		LineComment: LineCommentPackages,
		Warnings:    warnings,
	}, nil
}

type pkg struct {
	Name    string
	Version string
}

type packages []pkg

func (ps packages) Len() int { return len(ps) }
func (ps packages) Less(i, j int) bool {
	if ps[i].Name != ps[j].Name {
		return ps[i].Name < ps[j].Name
	}
	return ps[i].Version < ps[j].Version
}
func (ps packages) Swap(i, j int) { ps[i], ps[j] = ps[j], ps[i] }
