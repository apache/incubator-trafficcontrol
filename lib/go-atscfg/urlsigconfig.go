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
	"sort"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const ContentTypeURLSig = ContentTypeTextASCII
const LineCommentURLSig = LineCommentHash

// URLSigConfigOpts contains settings to configure generation options.
type URLSigConfigOpts struct {
	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeURLSigConfig(
	fileName string,
	server *Server,
	serverParams []tc.Parameter,
	allURLSigKeys map[tc.DeliveryServiceName]tc.URLSigKeys,
	opt *URLSigConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &URLSigConfigOpts{}
	}
	warnings := []string{}

	if server.Profile == nil {
		return Cfg{}, makeErr(warnings, "server missing Profile")
	}

	paramData, paramWarns := paramsToMap(filterParams(serverParams, fileName, "", "", "location"))
	warnings = append(warnings, paramWarns...)

	dsName := getDSFromURLSigConfigFileName(fileName)
	if dsName == "" {
		return Cfg{}, makeErr(warnings, "getting ds name: malformed config file '"+fileName+"'")
	}

	urlSigKeys, ok := allURLSigKeys[tc.DeliveryServiceName(dsName)]
	if !ok {
		warnings = append(warnings, "no keys fetched for ds '"+dsName+"!")
		urlSigKeys = tc.URLSigKeys{}
	}

	hdr := makeHdrComment(opt.HdrComment)

	sep := " = "

	text := hdr

	paramLines := []string{}
	for paramName, paramVal := range paramData {
		if len(urlSigKeys) == 0 || !strings.HasPrefix(paramName, "key") {
			paramLines = append(paramLines, paramName+sep+paramVal+"\n")
		}
	}
	sort.Strings(paramLines)
	text += strings.Join(paramLines, "")

	keyLines := []string{}
	for key, val := range urlSigKeys {
		keyLines = append(keyLines, key+sep+val+"\n")
	}
	sort.Strings(keyLines)
	text += strings.Join(keyLines, "")

	return Cfg{
		Text:        text,
		ContentType: ContentTypeURLSig,
		LineComment: LineCommentURLSig,
		Warnings:    warnings,
	}, nil
}

// getDSFromURLSigConfigFileName returns the DS of a URLSig config file name.
// For example, "url_sig_foobar.config" returns "foobar".
// If the given string is shorter than len("url_sig_a.config"), the empty string is returned.
func getDSFromURLSigConfigFileName(fileName string) string {
	if !strings.HasPrefix(fileName, "url_sig_") || !strings.HasSuffix(fileName, ".config") || len(fileName) <= len("url_sig_")+len(".config") {
		return ""
	}
	fileName = fileName[len("url_sig_"):]
	fileName = fileName[:len(fileName)-len(".config")]
	return fileName
}
