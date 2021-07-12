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

const ContentTypeRegexRemapDotConfig = ContentTypeTextASCII
const LineCommentRegexRemapDotConfig = LineCommentHash

// RegexRemapDotConfigOpts contains settings to configure generation options.
type RegexRemapDotConfigOpts struct {
	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeRegexRemapDotConfig(
	fileName string,
	server *Server,
	deliveryServices []DeliveryService,
	opt *RegexRemapDotConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &RegexRemapDotConfigOpts{}
	}
	warnings := []string{}
	if server.CDNName == nil {
		return Cfg{}, makeErr(warnings, "server CDNName missing")
	}

	configSuffix := `.config`
	if !strings.HasPrefix(fileName, RegexRemapPrefix) || !strings.HasSuffix(fileName, configSuffix) {
		return Cfg{}, makeErr(warnings, "file '"+fileName+"' not of the form 'regex_remap_*.config! Please file a bug with Traffic Control, this should never happen")
	}

	// TODO verify prefix and suffix exist, and warn if they don't? Perl doesn't
	dsName := strings.TrimSuffix(strings.TrimPrefix(fileName, RegexRemapPrefix), configSuffix)
	if dsName == "" {
		return Cfg{}, makeErr(warnings, "file '"+fileName+"' has no delivery service name!")
	}

	// only send the requested DS to atscfg. The atscfg.Make will work correctly even if we send it other DSes, but this will prevent deliveryServicesToCDNDSes from logging errors about AnyMap and Steering DSes without origins.
	ds := DeliveryService{}
	for _, dsesDS := range deliveryServices {
		if dsesDS.XMLID == nil {
			continue // TODO log?
		}
		if *dsesDS.XMLID != dsName {
			continue
		}
		ds = dsesDS
	}
	if ds.ID == nil {
		return Cfg{}, makeErr(warnings, "delivery service '"+dsName+"' not found! Do you have a regex_remap_*.config location Parameter for a delivery service that doesn't exist?")
	}

	dses, dsWarns := deliveryServicesToCDNDSes([]DeliveryService{ds})
	warnings = append(warnings, dsWarns...)

	text := makeHdrComment(opt.HdrComment)

	cdnDS, ok := dses[tc.DeliveryServiceName(dsName)]
	if !ok {
		warnings = append(warnings, "ds '"+dsName+"' not in dses, skipping!")
	} else {
		text += cdnDS.RegexRemap + "\n"
		text = strings.Replace(text, `__RETURN__`, "\n", -1)
	}

	return Cfg{
		Text:        text,
		ContentType: ContentTypeRegexRemapDotConfig,
		LineComment: LineCommentRegexRemapDotConfig,
		Warnings:    warnings,
	}, nil
}

type cdnDS struct {
	OrgServerFQDN string
	QStringIgnore int
	RegexRemap    string
}

// deliveryServicesToCDNDSes returns the CDNDSes and any warnings.
func deliveryServicesToCDNDSes(dses []DeliveryService) (map[tc.DeliveryServiceName]cdnDS, []string) {
	warnings := []string{}
	sDSes := map[tc.DeliveryServiceName]cdnDS{}
	for _, ds := range dses {
		if ds.OrgServerFQDN == nil || ds.QStringIgnore == nil || ds.XMLID == nil {
			if ds.XMLID == nil {
				warnings = append(warnings, "got unknown DS with nil values! Skipping!")
			} else {
				warnings = append(warnings, "got DS '"+*ds.XMLID+"' with nil values! Skipping!")
			}
			continue
		}
		sds := cdnDS{OrgServerFQDN: *ds.OrgServerFQDN, QStringIgnore: *ds.QStringIgnore}
		if ds.RegexRemap != nil {
			sds.RegexRemap = *ds.RegexRemap
		}
		sDSes[tc.DeliveryServiceName(*ds.XMLID)] = sds
	}
	return sDSes, warnings
}
