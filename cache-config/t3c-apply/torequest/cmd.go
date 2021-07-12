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

// cmd.go has funcs to call t3c sub-command apps.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/cache-config/t3c-apply/config"
	"github.com/apache/trafficcontrol/cache-config/t3cutil"
	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

type ServerAndConfigs struct {
	ConfigData  json.RawMessage
	ConfigFiles json.RawMessage
}

// generate runs t3c-generate and returns the result.
func generate(cfg config.Cfg) ([]t3cutil.ATSConfigFile, error) {
	configData, err := request(cfg, "config")
	if err != nil {
		return nil, errors.New("requesting: " + err.Error())
	}
	args := []string{
		"--dir=" + cfg.TsConfigDir,
	}

	if cfg.LogLocationErr == log.LogLocationNull {
		args = append(args, "-s")
	}
	if cfg.LogLocationWarn != log.LogLocationNull {
		args = append(args, "-v")
	}
	if cfg.LogLocationInfo != log.LogLocationNull {
		args = append(args, "-v")
	}

	if cfg.DNSLocalBind {
		args = append(args, "--dns-local-bind")
	}
	if cfg.DefaultClientEnableH2 != nil {
		args = append(args, "--default-client-enable-h2="+strconv.FormatBool(*cfg.DefaultClientEnableH2))
	}
	if cfg.DefaultClientTLSVersions != nil {
		args = append(args, "--default-client-tls-versions="+*cfg.DefaultClientTLSVersions+"")
	}
	if cfg.Files == t3cutil.ApplyFilesFlagReval {
		args = append(args, "--revalidate-only")
	}
	args = append(args, "--via-string-release="+strconv.FormatBool(!cfg.OmitViaStringRelease))
	args = append(args, "--disable-parent-config-comments="+strconv.FormatBool(cfg.DisableParentConfigComments))

	generatedFiles, stdErr, code := t3cutil.DoInput(configData, config.GenerateCmd, args...)
	if code != 0 {
		return nil, fmt.Errorf("t3c-generate returned non-zero exit code %v stdout '%v' stderr '%v'", code, string(generatedFiles), string(stdErr))
	}
	if len(bytes.TrimSpace(stdErr)) > 0 {
		log.Warnln(`t3c-generate stderr start` + "\n" + string(stdErr))
		log.Warnln(`t3c-generate stderr end`)
	}

	preprocessedBytes, err := preprocess(cfg, configData, generatedFiles)
	if err != nil {
		return nil, errors.New("preprocessing config files: " + err.Error())
	}

	allFiles := []t3cutil.ATSConfigFile{}
	if err := json.Unmarshal(preprocessedBytes, &allFiles); err != nil {
		return nil, errors.New("unmarshalling generated files: " + err.Error())
	}

	return allFiles, nil
}

// preprocess takes the to Data from 't3c-request --get-data=config' and the generated files from 't3c-generate', passes them to `t3c-preprocess`, and returns the result.
func preprocess(cfg config.Cfg, configData []byte, generatedFiles []byte) ([]byte, error) {
	args := []string{}

	if cfg.LogLocationErr == log.LogLocationNull {
		args = append(args, "-s")
	}
	if cfg.LogLocationWarn != log.LogLocationNull {
		args = append(args, "-v")
	}
	if cfg.LogLocationInfo != log.LogLocationNull {
		args = append(args, "-v")
	}

	cmd := exec.Command(`t3c-preprocess`, args...)
	outbuf := bytes.Buffer{}
	errbuf := bytes.Buffer{}
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, errors.New("getting command pipe: " + err.Error())
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.New("starting command: " + err.Error())
	}

	if _, err := stdinPipe.Write([]byte(`{"data":`)); err != nil {
		return nil, errors.New("writing opening json to input: " + err.Error())
	} else if _, err := stdinPipe.Write(configData); err != nil {
		return nil, errors.New("writing config data to input: " + err.Error())
	} else if _, err := stdinPipe.Write([]byte(`,"files":`)); err != nil {
		return nil, errors.New("writing files key to input: " + err.Error())
	} else if _, err := stdinPipe.Write(generatedFiles); err != nil {
		return nil, errors.New("writing generated files to input: " + err.Error())
	} else if _, err := stdinPipe.Write([]byte(`}`)); err != nil {
		return nil, errors.New("writing closing json input: " + err.Error())
	} else if err := stdinPipe.Close(); err != nil {
		return nil, errors.New("closing stdin writer: " + err.Error())
	}

	code := 0 // if cmd.Wait returns no error, that means the command returned 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok {
			return nil, errors.New("error running command: " + err.Error())
		} else {
			code = exitErr.ExitCode()
		}
	}

	stdOut := outbuf.Bytes()
	stdErr := errbuf.Bytes()
	if code != 0 {
		return nil, fmt.Errorf("t3c-preprocess returned non-zero exit code %v stdout '%v' stderr '%v'", code, string(stdOut), string(stdErr))
	}
	if len(bytes.TrimSpace(stdErr)) > 0 {
		log.Warnf("t3c-preprocess returned code 0 but stderr '%v'", string(stdErr)) // usually warnings
	}
	return stdOut, nil
}

func getStatuses(cfg config.Cfg) ([]string, error) {
	statuses := []tc.StatusNullable{}
	if err := requestJSON(cfg, "statuses", &statuses); err != nil {
		return nil, errors.New("requesting json: " + err.Error())
	}
	sl := []string{}
	for val := range statuses {
		if statuses[val].Name != nil {
			sl = append(sl, *statuses[val].Name)
		}
	}
	return sl, nil
}

func getChkconfig(cfg config.Cfg) ([]map[string]string, error) {
	result := []map[string]string{}
	if err := requestJSON(cfg, "chkconfig", &result); err != nil {
		return nil, errors.New("requesting json: " + err.Error())
	}
	return result, nil
}

func getUpdateStatus(cfg config.Cfg) (*tc.ServerUpdateStatus, error) {
	status := tc.ServerUpdateStatus{}
	if err := requestJSON(cfg, "update-status", &status); err != nil {
		return nil, errors.New("requesting json: " + err.Error())
	}
	return &status, nil
}

func getSystemInfo(cfg config.Cfg) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if err := requestJSON(cfg, "system-info", &result); err != nil {
		return nil, errors.New("requesting json: " + err.Error())
	}
	return result, nil
}

func getPackages(cfg config.Cfg) ([]Package, error) {
	pkgs := []Package{}
	if err := requestJSON(cfg, "packages", &pkgs); err != nil {
		return nil, errors.New("requesting json: " + err.Error())
	}
	return pkgs, nil
}

// sendUpdate updates the given cache's queue update and reval status in Traffic Ops.
// Note the statuses are the value to be set, not whether to set the value.
func sendUpdate(cfg config.Cfg, updateStatus bool, revalStatus bool) error {
	args := []string{
		"--traffic-ops-timeout-milliseconds=" + strconv.FormatInt(int64(cfg.TOTimeoutMS), 10),
		"--traffic-ops-user=" + cfg.TOUser,
		"--traffic-ops-password=" + cfg.TOPass,
		"--traffic-ops-url=" + cfg.TOURL,
		"--traffic-ops-insecure=" + strconv.FormatBool(cfg.TOInsecure),
		"--cache-host-name=" + cfg.CacheHostName,
		"--set-update-status=" + strconv.FormatBool(updateStatus),
		"--set-reval-status=" + strconv.FormatBool(revalStatus),
	}

	if cfg.LogLocationErr == log.LogLocationNull {
		args = append(args, "-s")
	}
	if cfg.LogLocationWarn != log.LogLocationNull {
		args = append(args, "-v")
	}
	if cfg.LogLocationInfo != log.LogLocationNull {
		args = append(args, "-v")
	}

	if _, used := os.LookupEnv("TO_USER"); !used {
		args = append(args, "--traffic-ops-user="+cfg.TOUser)
	}
	if _, used := os.LookupEnv("TO_PASS"); !used {
		args = append(args, "--traffic-ops-password="+cfg.TOPass)
	}
	if _, used := os.LookupEnv("TO_URL"); !used {
		args = append(args, "--traffic-ops-url="+cfg.TOURL)
	}
	stdOut, stdErr, code := t3cutil.Do(`t3c-update`, args...)
	if code != 0 {
		return fmt.Errorf("t3c-update returned non-zero exit code %v stdout '%v' stderr '%v'", code, string(stdOut), string(stdErr))
	}
	if len(bytes.TrimSpace(stdErr)) > 0 {
		log.Warnf("t3c-update returned code 0 but stderr '%v'", string(stdErr)) // usually warnings
	}
	log.Infoln("t3c-update succeeded")
	return nil
}

// diff calls t3c-diff to diff the given new file and the file on disk. Returns whether they're different.
// Logs the difference.
// If the file on disk doesn't exist, returns true and logs the entire file as a diff.
func diff(cfg config.Cfg, newFile []byte, fileLocation string) (bool, error) {
	stdOut, stdErr, code := t3cutil.DoInput(newFile, `t3c-diff`, `stdin`, fileLocation)
	if code > 1 {
		return false, fmt.Errorf("t3c-diff returned error code %v stdout '%v' stderr '%v'", code, string(stdOut), string(stdErr))
	}
	if len(bytes.TrimSpace(stdErr)) > 0 {
		log.Warnf("t3c-diff returned non-error code %v but stderr '%v'", code, string(stdErr))
	}

	if code == 0 {
		log.Infof("All lines match TrOps for config file: %s\n", fileLocation)
		return false, nil // 0 is only returned if there's no diff
	}
	// code 1 means a diff, difference text will be on stdout

	lines := strings.Split(string(stdOut), "\n")
	log.Infoln("file '" + fileLocation + "' changes begin")
	for _, line := range lines {
		log.Infoln("diff: " + line)
	}
	log.Infoln("file '" + fileLocation + "' changes end")

	return true, nil
}

// checkRefs calls t3c-check-refs to verify the given cfgFile.
// The cfgFile should be the full text of either a plugin.config or remap.config.
// Returns nil if t3c-check-refs returned no errors found, or the error found if any.
func checkRefs(cfg config.Cfg, cfgFile []byte, filesAdding []string) error {
	args := []string{`check`, `refs`,
		"--files-adding=" + strings.Join(filesAdding, ","),
	}
	if cfg.LogLocationErr == log.LogLocationNull {
		args = append(args, "-s")
	}
	if cfg.LogLocationWarn != log.LogLocationNull {
		args = append(args, "-v")
	}
	if cfg.LogLocationInfo != log.LogLocationNull {
		args = append(args, "-v")
	}

	stdOut, stdErr, code := t3cutil.DoInput(cfgFile, `t3c`, args...)

	if code != 0 {
		log.Errorf(`check-refs errors start
` + string(stdOut))
		log.Errorf(`check-refs errors end`)
		if strings.TrimSpace(string(stdErr)) != "" {
			log.Errorf(`check-refs output start
` + string(stdErr))
			log.Errorf(`check-refs output end`)
		}
		return fmt.Errorf("%d plugins failed to verify. See log for details.", code)
	}
	if len(bytes.TrimSpace(stdErr)) > 0 {
		log.Warnf("t3c-check-refs returned non-error code %v but stderr '%v'", code, string(stdErr))
	}
	if len(bytes.TrimSpace(stdOut)) > 0 {
		log.Warnf("t3c-check-refs returned non-error code %v but output '%v'", code, string(stdOut))
	}
	return nil
}

// checkReload is a helper for the sub-command t3c-check-reload.
func checkReload(pluginPackagesInstalled []string, changedConfigFiles []string) (t3cutil.ServiceNeeds, error) {
	log.Infof("t3c-check-reload calling with pluginPackagesInstalled '%v' changedConfigFiles '%v'\n", pluginPackagesInstalled, changedConfigFiles)

	stdOut, stdErr, code := t3cutil.Do(`t3c`, `check`, `reload`,
		"--plugin-packages-installed="+strings.Join(pluginPackagesInstalled, ","),
		"--changed-config-paths="+strings.Join(changedConfigFiles, ","),
	)

	if code != 0 {
		log.Errorf(`t3c-check-reload errors start
` + string(stdErr))
		log.Errorf(`t3c-check-reload errors end`)
		if strings.TrimSpace(string(stdErr)) != "" {
			log.Errorf(`t3c-check-reload output start
` + string(stdOut))
			log.Errorf(`t3c-check-reload output end`)
		}
		return t3cutil.ServiceNeedsInvalid, fmt.Errorf("t3c-check-reload returned error code %d - see log for details.", code)
	} else if strings.TrimSpace(string(stdErr)) != "" {
		log.Errorf(`t3c-check-reload returned success code but nonempty stderr. determine-restart errors start
` + string(stdErr))
		log.Errorf(`t3c-check-reload errors end`)

	}
	needs := t3cutil.StrToServiceNeeds(strings.TrimSpace(string(stdOut)))
	if needs == t3cutil.ServiceNeedsInvalid {
		return t3cutil.ServiceNeedsInvalid, errors.New("t3c-check-reload returned unknown string '" + string(stdOut) + "'")
	}
	return needs, nil
}

// requestJSON calls t3c-request with the given command, and deserializes the result as JSON into obj.
func requestJSON(cfg config.Cfg, command string, obj interface{}) error {
	stdOut, err := request(cfg, command)
	if err != nil {
		return errors.New("requesting: " + err.Error())
	}
	if err := json.Unmarshal(stdOut, obj); err != nil {
		return errors.New("unmarshalling '" + string(stdOut) + "': " + err.Error())
	}
	return nil
}

// request calls t3c-request with the given command, and returns the stdout bytes.
func request(cfg config.Cfg, command string) ([]byte, error) {
	args := []string{
		"--traffic-ops-insecure=" + strconv.FormatBool(cfg.TOInsecure),
		"--traffic-ops-timeout-milliseconds=" + strconv.FormatInt(int64(cfg.TOTimeoutMS), 10),
		"--cache-host-name=" + cfg.CacheHostName,
		`--get-data=` + command,
	}

	if cfg.LogLocationErr == log.LogLocationNull {
		args = append(args, "-s")
	}
	if cfg.LogLocationWarn != log.LogLocationNull {
		args = append(args, "-v")
	}
	if cfg.LogLocationInfo != log.LogLocationNull {
		args = append(args, "-v")
	}

	if _, used := os.LookupEnv("TO_USER"); !used {
		args = append(args, "--traffic-ops-user="+cfg.TOUser)
	}
	if _, used := os.LookupEnv("TO_PASS"); !used {
		args = append(args, "--traffic-ops-password="+cfg.TOPass)
	}
	if _, used := os.LookupEnv("TO_URL"); !used {
		args = append(args, "--traffic-ops-url="+cfg.TOURL)
	}
	stdOut, stdErr, code := t3cutil.Do(`t3c-request`, args...)
	if code != 0 {
		return nil, fmt.Errorf("t3c-request returned non-zero exit code %v stdout '%v' stderr '%v'", code, string(stdOut), string(stdErr))
	}
	if len(bytes.TrimSpace(stdErr)) > 0 {
		log.Warnf("t3c-request returned code 0 but stderr '%v'", string(stdErr)) // usually warnings
	}
	return stdOut, nil
}

// outToErr returns stderr if logLocation is stdout, otherwise returns logLocation unchanged.
// This is a helper to avoid logging to stdout for commands whose output is on stdout.
func outToErr(logLocation string) string {
	if logLocation == "stdout" {
		return "stderr"
	}
	return logLocation
}
