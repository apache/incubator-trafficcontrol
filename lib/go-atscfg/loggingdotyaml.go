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
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const LoggingYAMLFileName = "logging.yaml"
const ContentTypeLoggingDotYAML = ContentTypeYAML
const LineCommentLoggingDotYAML = LineCommentHash

func MakeLoggingDotYAML(
	server *Server,
	serverParams []tc.Parameter,
	hdrComment string,
) (Cfg, error) {
	warnings := []string{}
	requiredIndent := 0

	if server.Profile == nil {
		return Cfg{}, makeErr(warnings, "this server missing Profile")
	}

	paramData, paramWarns := paramsToMap(filterParams(serverParams, LoggingYAMLFileName, "", "", "location"))
	warnings = append(warnings, paramWarns...)

	hdr := makeHdrComment(hdrComment)

	version, vWarn := getATSMajorVersion(serverParams)
	warnings = append(warnings, vWarn...)
	// note we use the same const as logs.xml - this isn't necessarily a requirement, and we may want to make separate variables in the future.
	maxLogObjects := MaxLogObjects

	text := hdr
	if version >= 9 {
		text += "\nlogging:"
		requiredIndent += 2
	}

	indentSpaces := strings.Repeat(" ", requiredIndent)
	text += "\n" + indentSpaces + "formats: \n"
	for i := 0; i < maxLogObjects; i++ {
		logFormatField := "LogFormat"
		if i > 0 {
			logFormatField += strconv.Itoa(i)
		}
		logFormatName := paramData[logFormatField+".Name"]
		if logFormatName != "" {
			format := paramData[logFormatField+".Format"]
			if format == "" {
				// TODO determine if the line should be excluded. Perl includes it anyway, without checking.
				warnings = append(warnings, fmt.Sprintf("profile '%v' has logging.yaml format '%v' Name Parameter but no Format Parameter. Setting blank Format!\n", *server.Profile, logFormatField))
			}
			text += indentSpaces + " - name: " + logFormatName + " \n"
			text += indentSpaces + "   format: '" + format + "'\n"
		}
	}

	text += indentSpaces + "filters:\n"
	for i := 0; i < maxLogObjects; i++ {
		logFilterField := "LogFilter"
		if i > 0 {
			logFilterField += strconv.Itoa(i)
		}
		if logFilterName := paramData[logFilterField+".Name"]; logFilterName != "" {
			filter := paramData[logFilterField+".Filter"]
			if filter == "" {
				// TODO determine if the line should be excluded. Perl includes it anyway, without checking.
				warnings = append(warnings, fmt.Sprintf("profile '%v' has logging.yaml filter '%v' Name Parameter but no Filter Parameter. Setting blank Filter!\n", *server.Profile, logFilterField))
			}
			logFilterType := paramData[logFilterField+".Type"]
			if logFilterType == "" {
				logFilterType = "accept"
			}
			text += indentSpaces + " - name: " + logFilterName + "\n"
			text += indentSpaces + "   action: " + logFilterType + "\n"
			text += indentSpaces + "   condition: " + filter + "\n"
		}
	}

	var firstObject = true
	for i := 0; i < maxLogObjects; i++ {
		logObjectField := "LogObject"
		if i > 0 {
			logObjectField += strconv.Itoa(i)
		}

		if logObjectFilename := paramData[logObjectField+".Filename"]; logObjectFilename != "" {
			logObjectType := paramData[logObjectField+".Type"]
			if logObjectType == "" {
				logObjectType = "ascii"
			}
			logObjectFormat := paramData[logObjectField+".Format"]
			logObjectRollingEnabled := paramData[logObjectField+".RollingEnabled"]
			logObjectRollingIntervalSec := paramData[logObjectField+".RollingIntervalSec"]
			logObjectRollingOffsetHr := paramData[logObjectField+".RollingOffsetHr"]
			logObjectRollingSizeMb := paramData[logObjectField+".RollingSizeMb"]
			logObjectFilters := paramData[logObjectField+".Filters"]

			if firstObject {
				text += "\n" + indentSpaces + "logs:\n"
				firstObject = false
			}
			text += indentSpaces + " - mode: " + logObjectType + "\n"
			text += indentSpaces + "   filename: " + logObjectFilename + "\n"
			text += indentSpaces + "   format: " + logObjectFormat + "\n"

			if logObjectType != "pipe" {
				if logObjectRollingEnabled != "" {
					text += indentSpaces + "   rolling_enabled: " + logObjectRollingEnabled + "\n"
				}
				if logObjectRollingIntervalSec != "" {
					text += indentSpaces + "   rolling_interval_sec: " + logObjectRollingIntervalSec + "\n"
				}
				if logObjectRollingOffsetHr != "" {
					text += indentSpaces + "   rolling_offset_hr: " + logObjectRollingOffsetHr + "\n"
				}
				if logObjectRollingSizeMb != "" {
					text += indentSpaces + "   rolling_size_mb: " + logObjectRollingSizeMb + "\n"
				}
			}
			if logObjectFilters != "" {
				logObjectFilters = strings.Replace(logObjectFilters, "\v", "", -1)
				text += indentSpaces + "   filters: [" + logObjectFilters + "]\n"
			}
		}
	}

	return Cfg{
		Text:        text,
		ContentType: ContentTypeLoggingDotYAML,
		LineComment: LineCommentLoggingDotYAML,
		Warnings:    warnings,
	}, nil
}
