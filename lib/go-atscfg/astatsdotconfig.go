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

const AstatsSeparator = "="
const AstatsFileName = "astats.config"

const ContentTypeAstatsDotConfig = ContentTypeTextASCII
const LineCommentAstatsDotConfig = LineCommentHash

// AStatsDotConfigOpts contains settings to configure generation options.
type AStatsDotConfigOpts struct {
	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeAStatsDotConfig(
	server *Server,
	serverParams []tc.Parameter,
	opt *AStatsDotConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &AStatsDotConfigOpts{}
	}
	warnings := []string{}

	if server.Profile == nil {
		return Cfg{}, makeErr(warnings, "server missing Profile")
	}

	serverParams = filterParams(serverParams, AstatsFileName, "", "", "location")
	paramData, paramWarns := paramsToMap(serverParams)
	warnings = append(warnings, paramWarns...)
	hdr := makeHdrComment(opt.HdrComment)
	txt := genericProfileConfig(paramData, AstatsSeparator)
	if txt == "" {
		txt = "\n" // If no params exist, don't send "not found," but an empty file. We know the profile exists.
	}
	txt = hdr + txt

	return Cfg{
		Text:        txt,
		ContentType: ContentTypeAstatsDotConfig,
		LineComment: LineCommentAstatsDotConfig,
		Warnings:    warnings,
	}, nil
}
