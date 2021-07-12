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

const ContentTypeServerUnknownConfig = ContentTypeTextASCII

// ServerUnknownOpts contains settings to configure generation options.
type ServerUnknownOpts struct {
	// HdrComment is the header comment to include at the beginning of the file.
	// This should be the text desired, without comment syntax (like # or //). The file's comment syntax will be added.
	// To omit the header comment, pass the empty string.
	HdrComment string
}

func MakeServerUnknown(
	fileName string,
	server *Server,
	serverParams []tc.Parameter,
	opt *ServerUnknownOpts,
) (Cfg, error) {
	if opt == nil {
		opt = &ServerUnknownOpts{}
	}
	warnings := []string{}

	if server.HostName == nil {
		return Cfg{}, makeErr(warnings, "server missing HostName")
	} else if server.DomainName == nil {
		return Cfg{}, makeErr(warnings, "server missing DomainName")
	}

	params := paramsToMultiMap(filterParams(serverParams, fileName, "", "", ""))

	hdr := makeHdrComment(opt.HdrComment)

	txt := ""

	sortedParams := sortParams(params)
	for _, pa := range sortedParams {
		if pa.Name == "location" {
			continue
		}
		if pa.Name == "header" {
			if pa.Val == "none" {
				hdr = ""
			} else {
				hdr = pa.Val + "\n"
			}
			continue
		}
		txt += pa.Val + "\n"
	}

	txt = strings.Replace(txt, `__HOSTNAME__`, *server.HostName+`.`+*server.DomainName, -1)
	txt = strings.Replace(txt, `__RETURN__`, "\n", -1)

	lineComment := getServerUnknownConfigCommentType(params)

	txt = hdr + txt

	return Cfg{
		Text:        txt,
		ContentType: ContentTypeServerUnknownConfig,
		LineComment: lineComment,
		Warnings:    warnings,
	}, nil
}

type param struct {
	Name string
	Val  string
}

type paramsSort []param

func (a paramsSort) Len() int           { return len(a) }
func (a paramsSort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a paramsSort) Less(i, j int) bool { return a[i].Name < a[j].Name }

func sortParams(params map[string][]string) []param {
	sortedParams := []param{}
	for name, vals := range params {
		for _, val := range vals {
			sortedParams = append(sortedParams, param{Name: name, Val: val})
		}
	}
	sort.Sort(paramsSort(sortedParams))
	return sortedParams
}

// getServerUnknownConfigCommentType takes the same data as MakeUnknownConfig and returns the comment type for that config.
// In particular, it returns # unless there is a 'header' parameter, in which case it returns an empty string.
// Wwe don't actually know that the first characters of a custom header are a comment, or how many characters it might be.
func getServerUnknownConfigCommentType(
	params map[string][]string,
) string {
	for name, _ := range params {
		if name == "header" {
			return ""
		}
	}
	return LineCommentHash
}
