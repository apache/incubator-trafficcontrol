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
	"os/exec"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const RecordsSeparator = " "
const RecordsFileName = "records.config"
const ContentTypeRecordsDotConfig = ContentTypeTextASCII
const LineCommentRecordsDotConfig = LineCommentHash

type RecordsConfigOpts struct {
	// ReleaseViaStr is whether or not we replace the via and server strings in ATS
	// responses to be the Release value from the rpm package. This can be a user
	// defined build hash (or whatever the user wants) type value to give more
	// specific info as well as obfuscating the real ATS version from prying eyes
	ReleaseViaStr bool

	// DNSLocalBindServiceAddr is whether to set the server's service addresses
	// as the records.config proxy.config.dns.local_ipv* settings.
	DNSLocalBindServiceAddr bool

	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeRecordsDotConfig(
	server *Server,
	serverParams []tc.Parameter,
	opt *RecordsConfigOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &RecordsConfigOpts{}
	}
	warnings := []string{}
	if server.Profile == nil {
		return Cfg{}, makeErr(warnings, "server profile missing")
	}

	params, paramWarns := paramsToMap(filterParams(serverParams, RecordsFileName, "", "", "location"))
	warnings = append(warnings, paramWarns...)

	hdr := makeHdrComment(opt.HdrComment)
	txt := genericProfileConfig(params, RecordsSeparator)
	if txt == "" {
		txt = "\n" // If no params exist, don't send "not found," but an empty file. We know the profile exists.
	}
	txt = replaceLineSuffixes(txt, "STRING __HOSTNAME__", "STRING __FULL_HOSTNAME__")
	txt = hdr + txt

	txt, overrideWarns := addRecordsDotConfigOverrides(txt, server, opt)
	warnings = append(warnings, overrideWarns...)

	return Cfg{
		Text:        txt,
		ContentType: ContentTypeRecordsDotConfig,
		LineComment: LineCommentRecordsDotConfig,
		Warnings:    warnings,
	}, nil
}

// addRecordsDotConfigOverrides modifies the records.config text and adds any overrides.
// Returns the modified text and any warnings.
func addRecordsDotConfigOverrides(txt string, server *Server, opt *RecordsConfigOpts) (string, []string) {
	warnings := []string{}
	txt, ipWarns := addRecordsDotConfigOutgoingIP(txt, server)
	warnings = append(warnings, ipWarns...)

	if opt.ReleaseViaStr {
		viaWarns := []string{}
		txt, viaWarns = addRecordsDotConfigViaStr(txt)
		warnings = append(warnings, viaWarns...)
	}

	if opt.DNSLocalBindServiceAddr {
		dnsWarns := []string{}
		txt, dnsWarns = addRecordsDotConfigDNSLocal(txt, server)
		warnings = append(warnings, dnsWarns...)
	}

	return txt, warnings
}

// addRecordsDotConfigOutgoingIP returns the outgoing IP added to the config text, and any warnings.
func addRecordsDotConfigOutgoingIP(txt string, server *Server) (string, []string) {
	warnings := []string{}

	outgoingIPConfig := `proxy.local.outgoing_ip_to_bind`
	if strings.Contains(txt, outgoingIPConfig) {
		warnings = append(warnings, "records.config had a proxy.local.outgoing_ip_to_bind Parameter! Using Parameter, not setting Outgoing IP from Server")
		return txt, warnings
	}

	v4, v6 := getServiceAddresses(server)
	if v4 == nil {
		warnings = append(warnings, "server had no IPv4 service address, cannot set "+outgoingIPConfig+"!")
		return txt, warnings
	}

	txt = txt + `LOCAL ` + outgoingIPConfig + ` STRING ` + v4.String()
	if v6 != nil {
		txt += ` [` + v6.String() + `]`
	}
	txt += "\n"
	return txt, warnings
}

// addRecordsDotConfigViaStr returns the request, response, and response server via strings with the current Release (a.k.a. build version and not ATS version), and any warnings.
func addRecordsDotConfigViaStr(txt string) (string, []string) {
	warnings := []string{}

	requestViaStr := `proxy.config.http.request_via_str`
	responseViaStr := `proxy.config.http.response_via_str`
	responseServerStr := `proxy.config.http.response_server_str`

	cmd := "yum info installed trafficserver | grep Release"
	yumOutput, err := exec.Command("sh", "-c", cmd).Output()

	if err != nil {
		warnings = append(warnings, "could not read trafficserver release information from yum! Not setting via strings")
		return txt, warnings
	}

	releaseVerSlice := strings.Split(string(yumOutput), " ")
	releaseVer := releaseVerSlice[len(releaseVerSlice)-1]

	if strings.Contains(txt, requestViaStr) {
		warnings = append(warnings, "records.config had a proxy.config.http.request_via_str Parameter! Using Parameter, not setting request via string")
	} else {
		txt = txt + `CONFIG ` + requestViaStr + ` STRING ` + releaseVer
		txt += "\n"
	}

	if strings.Contains(txt, responseViaStr) {
		warnings = append(warnings, "records.config had a proxy.config.http.response_via_str Parameter! Using Parameter, not setting response via string")
	} else {
		txt = txt + `CONFIG ` + responseViaStr + ` STRING ` + releaseVer
		txt += "\n"
	}

	if strings.Contains(txt, responseServerStr) {
		warnings = append(warnings, "records.config had a proxy.config.http.response_server_str Parameter! Using Parameter, not setting response server string")
	} else {
		txt = txt + `CONFIG ` + responseServerStr + ` STRING ` + releaseVer
		txt += "\n"
	}

	return txt, warnings
}

func addRecordsDotConfigDNSLocal(txt string, server *Server) (string, []string) {
	warnings := []string{}

	const dnsLocalV4 = `proxy.config.dns.local_ipv4`
	const dnsLocalV6 = `proxy.config.dns.local_ipv6`

	v4, v6 := getServiceAddresses(server)
	if v4 == nil {
		warnings = append(warnings, "server had no IPv4 Service Address, not setting records.config dns v4 local bind addr!")
	} else {
		txt += `CONFIG ` + dnsLocalV4 + ` STRING ` + v4.String() + "\n"
	}

	if v6 == nil {
		warnings = append(warnings, "server had no IPv6 Service Address, not setting records.config dns v6 local bind addr!")
	} else {
		txt += `CONFIG ` + dnsLocalV6 + ` STRING [` + v6.String() + `]` + "\n"
	}

	return txt, warnings
}

func replaceLineSuffixes(txt string, suffix string, newSuffix string) string {
	lines := strings.Split(txt, "\n")
	newLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasSuffix(line, suffix) {
			line = line[:len(line)-len(suffix)]
			line += newSuffix
		}
		newLines = append(newLines, line)
	}
	return strings.Join(newLines, "\n")
}
