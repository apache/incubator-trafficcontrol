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
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const ContentTypeURISigningDotConfig = `application/json; charset=us-ascii`
const LineCommentURISigningDotConfig = ""

// URISigningConfigOpts contains settings to configure generation options.
type URISigningConfigOpts struct {
}

func MakeURISigningConfig(
	fileName string,
	uriSigningKeys map[tc.DeliveryServiceName][]byte,
	opt *URISigningConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &URISigningConfigOpts{}
	}
	warnings := []string{}

	dsName := getDSFromURISigningConfigFileName(fileName)
	if dsName == "" {
		return Cfg{}, makeErr(warnings, "getting ds name: malformed config file '"+fileName+"'")
	}

	uriSigningKeyBts, ok := uriSigningKeys[dsName]
	if !ok {
		warnings = append(warnings, "no keys fetched for ds '"+string(dsName)+"!")
		uriSigningKeyBts = []byte{}
	}

	return Cfg{
		Text:        string(uriSigningKeyBts),
		ContentType: ContentTypeURISigningDotConfig,
		LineComment: LineCommentURISigningDotConfig,
		Warnings:    warnings,
	}, nil
}

// getDSFromURISigningConfigFileName returns the DS of a URI Signing config file name.
// For example, "uri_signing_foobar.config" returns "foobar".
// If the given string is shorter than len("uri_signing_a.config"), the empty string is returned.
func getDSFromURISigningConfigFileName(fileName string) tc.DeliveryServiceName {
	if !strings.HasPrefix(fileName, "uri_signing_") || !strings.HasSuffix(fileName, ".config") || len(fileName) <= len("uri_signing_")+len(".config") {
		return ""
	}
	fileName = fileName[len("uri_signing_"):]
	fileName = fileName[:len(fileName)-len(".config")]
	return tc.DeliveryServiceName(fileName)
}
