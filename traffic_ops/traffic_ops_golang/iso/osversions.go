package iso

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
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/jmoiron/sqlx"
)

// GetOSVersions returns a map of available Operating System (OS) versions for ISO generation,
// as well as the name of the directory where the "kickstarter" files are found.
//
// The returned data comes from a configuration file. There's a default location of the config file
// which can be overridden via a Parameter database entry.
//
// Note: The api.CRUDer interface could not be used for this endpoint because the original Perl
// endpoint returned a single JSON object, whereas the CRUDer interface returns an array of JSON
// objects per the Read method.
func GetOSVersions(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	api.RespWriter(w, r, inf.Tx.Tx)(getOSVersions(inf.Tx))
}

// getOSVersions is used in conjunction with GetOSVersions. It returns
// a mapping of OS names to their respective directories.
func getOSVersions(tx *sqlx.Tx) (tc.OSVersionsResponse, error) {
	cfgPath, err := osversionCfgPath(tx)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		// Warn if the old Perl config file exists, but there's an
		// error reading the file from the new location.
		oldCfgPath := filepath.Join(filepath.Dir(cfgPath), cfgFilenamePerl)
		if _, statErr := os.Stat(oldCfgPath); statErr == nil {
			log.Warnf("The configuration file for the /osversions endpoint has changed from %q to %q: %v", oldCfgPath, cfgPath, err)
		}
		return nil, err
	}

	var data tc.OSVersionsResponse
	if err = json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	return data, nil
}

// osversionsCfgPath returns a path to the configuration file
// containing the OS versions data. The name of the configuration
// file is constant, but a specific Parameter database entry can update the
// directory where the file is expected.
func osversionCfgPath(tx *sqlx.Tx) (string, error) {
	var cfgDir string
	err := tx.QueryRow(
		`SELECT value FROM parameter WHERE name = $1 AND config_file = $2 LIMIT 1`,
		ksFilesParamName,
		ksFilesParamConfigFile,
	).Scan(&cfgDir)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	if cfgDir == "" {
		cfgDir = cfgDefaultDir
	}

	return filepath.Join(cfgDir, cfgFilename), nil
}
