package torequest

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
	"testing"

	"github.com/apache/trafficcontrol/cache-config/t3c-apply/config"
	"github.com/apache/trafficcontrol/cache-config/t3cutil"
)

var testCfg config.Cfg = config.Cfg{
	Dispersion:          300,
	LogLocationDebug:    "stdout",
	LogLocationErr:      "stdout",
	LogLocationInfo:     "stdout",
	LogLocationWarn:     "stdout",
	LoginDispersion:     60,
	CacheHostName:       "cache-01.cdn.com",
	SvcManagement:       config.SystemD,
	Retries:             3,
	RevalWaitTime:       2,
	ReverseProxyDisable: false,
	Files:               t3cutil.ApplyFilesFlagReval,
	SkipOSCheck:         false,
	TOTimeoutMS:         1000,
	TOUser:              "mickey",
	TOPass:              "mouse",
	TOURL:               "http://mouse.com",
	WaitForParents:      false,
	YumOptions:          "none",
}

func TestCommentsFilter(t *testing.T) {
	input := []string{"#\n", "#\n", "#this is a comment\n", "proxy.http.retries: 10\n"}

	output := commentsFilter(input)
	length := len(output)
	if length != 1 {
		t.Fatal("commentsFilter() failed, expected a length of '1'")
	}
	if output[0] != "proxy.http.retries: 10\n" {
		t.Errorf("commentsFilter() failed, expected 'proxy.http.retries: 10' actual '" +
			output[0] + "'")
	}
}

func TestNewLineFilter(t *testing.T) {
	input := "the quick brown fox\r\njumped over\r\nthe lazy dogs\r\nback\n"
	expected := "the quick brown fox\njumped over\nthe lazy dogs\nback"

	output := newLineFilter(input)
	if output != expected {
		t.Errorf("newLineFilter() failed, expected '" + expected + "', actual '" + output)
	}
}

func TestUnencodeFilter(t *testing.T) {
	input := []string{" the  quick&lt;p&gt;brown&amp;fox"}
	expected := "the quick<p>brown&fox"

	output := unencodeFilter(input)

	length := len(output)
	if length != 1 {
		t.Errorf("unencodeFilter() failed, expected a length of '1'")
	}
	if output[0] != expected {
		t.Errorf("unencodeFilter() failed, expected '" + expected + "' actual '" + output[0] + "'")
	}
}

func TestIsPackageInstalled(t *testing.T) {
	trops := NewTrafficOpsReq(testCfg)
	trops.pkgs["trafficserver"] = true

	if trops.IsPackageInstalled("mouse") {
		t.Errorf("isPackageInstalled() failed, expected 'false' got 'true'.")
	}

	if !trops.IsPackageInstalled("trafficserver") {
		t.Errorf("isPackageInstalled() failed, expected 'true' got 'false'.")
	}

	trops.pkgs["trafficserver"] = false
	if trops.IsPackageInstalled("trafficserver") {
		t.Errorf("isPackageInstalled() failed, expected 'false' got 'true'.")
	}
}

func TestGetConfigFile(t *testing.T) {
	trops := NewTrafficOpsReq(testCfg)

	cfgFile := ConfigFile{
		Name:              "remap.config",
		Dir:               "/tmp",
		Path:              "/tmp/trafficserver/remap.config",
		Service:           "trafficserver",
		CfgBackup:         "/tmp/trafficserver/backup",
		TropsBackup:       "/tmp/trafficops/backu",
		AuditComplete:     true,
		AuditFailed:       false,
		ChangeApplied:     true,
		ChangeNeeded:      false,
		PreReqFailed:      false,
		RemapPluginConfig: false,
		Body:              nil,
		Uid:               100,
		Gid:               100,
	}

	trops.configFiles["remap.config"] = &cfgFile

	_, ok := trops.GetConfigFile("parent.config")
	if ok {
		t.Errorf("GetConfigFile('parent.config') failed, expected 'false' got 'true'.")
	}

	cfg, ok := trops.GetConfigFile("remap.config")
	if !ok {
		t.Errorf("GetConfigFile('remap.config') failed, expected 'true' got 'false'.")
	}
	if cfg.Name != "remap.config" {
		t.Errorf("GetConfigFile('remap.config') failed, expected 'remap.config' got '" + cfg.Name + "'.")
	}
}
