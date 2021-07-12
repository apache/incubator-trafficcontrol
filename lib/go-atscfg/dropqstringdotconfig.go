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

const DropQStringDotConfigFileName = "drop_qstring.config"
const DropQStringDotConfigParamName = "content"
const ContentTypeDropQStringDotConfig = ContentTypeTextASCII
const LineCommentDropQStringDotConfig = LineCommentHash

// DropQStringDotConfigOpts contains settings to configure generation options.
type DropQStringDotConfigOpts struct {
	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeDropQStringDotConfig(
	server *Server,
	serverParams []tc.Parameter,
	opt *DropQStringDotConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &DropQStringDotConfigOpts{}
	}
	warnings := []string{}

	if server.Profile == nil {
		return Cfg{}, makeErr(warnings, "this server missing Profile")
	}

	dropQStringVal := (*string)(nil)
	for _, param := range serverParams {
		if param.ConfigFile != DropQStringDotConfigFileName {
			continue
		}
		if param.Name != DropQStringDotConfigParamName {
			continue
		}
		dropQStringVal = &param.Value
		break
	}

	text := makeHdrComment(opt.HdrComment)
	if dropQStringVal != nil {
		text += *dropQStringVal + "\n"
	} else {
		text += `/([^?]+) $s://$t/$1` + "\n"
	}

	return Cfg{
		Text:        text,
		ContentType: ContentTypeDropQStringDotConfig,
		LineComment: LineCommentDropQStringDotConfig,
		Warnings:    warnings,
	}, nil
}
