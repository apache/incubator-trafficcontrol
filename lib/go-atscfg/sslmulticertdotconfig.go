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

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

const ContentTypeSSLMultiCertDotConfig = ContentTypeTextASCII
const LineCommentSSLMultiCertDotConfig = LineCommentHash
const SSLMultiCertConfigFileName = `ssl_multicert.config`

// SSLMultiCertDotConfigOpts contains settings to configure generation options.
type SSLMultiCertDotConfigOpts struct {
	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeSSLMultiCertDotConfig(
	server *Server,
	deliveryServices []DeliveryService,
	opt *SSLMultiCertDotConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &SSLMultiCertDotConfigOpts{}
	}
	warnings := []string{}
	if server.CDNName == nil {
		return Cfg{}, makeErr(warnings, "server missing CDNName")
	}

	dses, dsWarns := DeliveryServicesToSSLMultiCertDSes(deliveryServices)
	warnings = append(warnings, dsWarns...)

	hdr := makeHdrComment(opt.HdrComment)

	dses = GetSSLMultiCertDotConfigDeliveryServices(dses)

	lines := []string{}
	for dsName, ds := range dses {
		cerName, keyName := GetSSLMultiCertDotConfigCertAndKeyName(dsName, ds)
		lines = append(lines, `ssl_cert_name=`+cerName+"\t"+` ssl_key_name=`+keyName+"\n")
	}
	sort.Strings(lines)

	txt := hdr + strings.Join(lines, "")

	return Cfg{
		Text:        txt,
		ContentType: ContentTypeSSLMultiCertDotConfig,
		LineComment: LineCommentSSLMultiCertDotConfig,
		Warnings:    warnings,
	}, nil
}

type sslMultiCertDS struct {
	XMLID       string
	Type        tc.DSType
	Protocol    int
	ExampleURLs []string
}

// deliveryServicesToSSLMultiCertDSes returns the "SSLMultiCertDS" map, and any warnings.
func DeliveryServicesToSSLMultiCertDSes(dses []DeliveryService) (map[tc.DeliveryServiceName]sslMultiCertDS, []string) {
	warnings := []string{}
	sDSes := map[tc.DeliveryServiceName]sslMultiCertDS{}
	for _, ds := range dses {
		if ds.Type == nil || ds.Protocol == nil || ds.XMLID == nil {
			if ds.XMLID == nil {
				warnings = append(warnings, "got unknown DS with nil values! Skipping!")
			} else {
				warnings = append(warnings, "got DS '"+*ds.XMLID+"' with nil values! Skipping!")
			}
			continue
		}
		sDSes[tc.DeliveryServiceName(*ds.XMLID)] = sslMultiCertDS{Type: *ds.Type, Protocol: *ds.Protocol, ExampleURLs: ds.ExampleURLs}
	}
	return sDSes, warnings
}

// GetSSLMultiCertDotConfigCertAndKeyName returns the cert file name and key file name for the given delivery service.
func GetSSLMultiCertDotConfigCertAndKeyName(dsName tc.DeliveryServiceName, ds sslMultiCertDS) (string, string) {
	hostName := ds.ExampleURLs[0] // first one is the one we want

	scheme := "https://"
	if !strings.HasPrefix(hostName, scheme) {
		scheme = "http://"
	}
	newHost := hostName
	if len(hostName) < len(scheme) {
		log.Errorln("MakeSSLMultiCertDotConfig got ds '" + string(dsName) + "' example url '" + hostName + "' with no scheme! ssl_multicert.config will likely be malformed!")
	} else {
		newHost = hostName[len(scheme):]
	}
	keyName := newHost + ".key"

	newHost = strings.Replace(newHost, ".", "_", -1)

	cerName := newHost + "_cert.cer"
	return cerName, keyName
}

// GetSSLMultiCertDotConfigDeliveryServices takes a list of delivery services, and returns the delivery services which will be inserted into the config by MakeSSLMultiCertDotConfig.
// This is public, so users can see which Delivery Services are used, without parsing the config file.
// For example, this is useful to determine which certificates are needed.
func GetSSLMultiCertDotConfigDeliveryServices(dses map[tc.DeliveryServiceName]sslMultiCertDS) map[tc.DeliveryServiceName]sslMultiCertDS {
	usedDSes := map[tc.DeliveryServiceName]sslMultiCertDS{}
	for dsName, ds := range dses {
		if ds.Type == tc.DSTypeAnyMap {
			continue
		}
		if ds.Type.IsSteering() {
			continue // Steering delivery service SSLs should not be on the edges.
		}
		if ds.Protocol == 0 {
			continue
		}
		if len(ds.ExampleURLs) == 0 {
			continue // TODO warn? error? Perl doesn't
		}
		usedDSes[dsName] = ds
	}
	return usedDSes
}
