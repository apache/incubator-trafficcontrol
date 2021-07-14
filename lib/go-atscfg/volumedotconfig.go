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
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const VolumeFileName = StorageFileName
const ContentTypeVolumeDotConfig = ContentTypeTextASCII
const LineCommentVolumeDotConfig = LineCommentHash

// MakeVolumeDotConfig creates volume.config for a given ATS Profile.
// The paramData is the map of parameter names to values, for all parameters assigned to the given profile, with the config_file "storage.config".
func MakeVolumeDotConfig(
	server *Server,
	serverParams []tc.Parameter,
	hdrComment string,
) (Cfg, error) {
	warnings := []string{}

	if server.Profile == nil {
		return Cfg{}, makeErr(warnings, "server missing Profile")
	}

	paramData, paramWarns := paramsToMap(filterParams(serverParams, VolumeFileName, "", "", ""))
	warnings = append(warnings, paramWarns...)

	hdr := makeHdrComment(hdrComment)

	numVolumes := getNumVolumes(paramData)

	hdr += "# TRAFFIC OPS NOTE: This is running with forced volumes - the size is irrelevant\n"
	text := ""
	nextVolume := 1
	if drivePrefix := paramData["Drive_Prefix"]; drivePrefix != "" {
		text += volumeText(strconv.Itoa(nextVolume), numVolumes)
		nextVolume++
	}
	if ramDrivePrefix := paramData["RAM_Drive_Prefix"]; ramDrivePrefix != "" {
		text += volumeText(strconv.Itoa(nextVolume), numVolumes)
		nextVolume++
	}
	if ssdDrivePrefix := paramData["SSD_Drive_Prefix"]; ssdDrivePrefix != "" {
		text += volumeText(strconv.Itoa(nextVolume), numVolumes)
		nextVolume++
	}

	if text == "" {
		text = "\n" // If no params exist, don't send "not found," but an empty file. We know the profile exists.
	}

	return Cfg{
		Text:        hdr + text,
		ContentType: ContentTypeVolumeDotConfig,
		LineComment: LineCommentVolumeDotConfig,
		Warnings:    warnings,
	}, nil
}

func volumeText(volume string, numVolumes int) string {
	return "volume=" + volume + " scheme=http size=" + strconv.Itoa(100/numVolumes) + "%\n"
}

func getNumVolumes(paramData map[string]string) int {
	num := 0
	drivePrefixes := []string{"Drive_Prefix", "SSD_Drive_Prefix", "RAM_Drive_Prefix"}
	for _, pre := range drivePrefixes {
		if _, ok := paramData[pre]; ok {
			num++
		}
	}
	return num
}
