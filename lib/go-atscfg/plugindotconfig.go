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
	"github.com/apache/trafficcontrol/lib/go-tc"
)

const PluginSeparator = " "
const PluginFileName = "plugin.config"
const ContentTypePluginDotConfig = ContentTypeTextASCII
const LineCommentPluginDotConfig = LineCommentHash

// PluginDotConfigOpts contains settings to configure generation options.
type PluginDotConfigOpts struct {
	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakePluginDotConfig(
	server *Server,
	serverParams []tc.Parameter,
	opt *PluginDotConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &PluginDotConfigOpts{}
	}
	warnings := []string{}
	if server.Profile == nil {
		return Cfg{}, makeErr(warnings, "server profile missing")
	}

	paramData, paramWarns := paramsToMap(filterParams(serverParams, PluginFileName, "", "", "location"))
	warnings = append(warnings, paramWarns...)

	hdr := makeHdrComment(opt.HdrComment)
	txt := genericProfileConfig(paramData, PluginSeparator)
	if txt == "" {
		txt = "\n" // If no params exist, don't send "not found," but an empty file. We know the profile exists.
	}
	txt = hdr + txt

	return Cfg{
		Text:        txt,
		ContentType: ContentTypePluginDotConfig,
		LineComment: LineCommentPluginDotConfig,
		Warnings:    warnings,
	}, nil
}
