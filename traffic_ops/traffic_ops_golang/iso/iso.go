// Package iso provides support for generating ISO images.
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
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-rfc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/jmoiron/sqlx"
)

// Various directories and filenames related to ISO generation.
const (
	cfgDefaultDir   = "/var/www/files"  // Default directory containing config file
	cfgFilename     = "osversions.json" // The JSON config file containing mapping of OS names to directories
	cfgFilenamePerl = "osversions.cfg"  // Config file name in the Perl version

	// This is the directory name inside each OS directory where
	// configuration files for kickstart scripts are placed.
	ksCfgDir = "ks_scripts"

	// Configuration files that are generated inside the ks_scripts directory.
	ksCfgNetwork     = "network.cfg"
	ksCfgMgmtNetwork = "mgmt_network.cfg"
	ksCfgPassword    = "password.cfg"
	ksCfgDisk        = "disk.cfg"
	ksStateOut       = "state.out"

	ksAltCommand = "generate" // Optional executable that is invoked instead of mkisofs
)

// Various database columns and values.
const (
	ksFilesParamName       = "kickstart.files.location"
	ksFilesParamConfigFile = "mkisofs"
)

// ISOs handler is responsible for generating and returning an ISO image,
// as a streaming download.
//
// Response types:
//
// Error:
//   HTTP 400
//   {
//     "alerts": [
//       {"level":"error","text":"hostName is required"},
//       {"level":"error","text":"disk is required"},
//       ...,
//     ]
//   }
//
// Success:
//   HTTP 200
//   Content-Disposition: attachment; filename="db.infra.ciab.test-centos72.iso"
//   Content-Type: application/download
//
func ISOs(w http.ResponseWriter, req *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(w, req, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, req, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	ir := isoRequest{}

	if err := api.Parse(req.Body, inf.Tx.Tx, &ir); err != nil {
		api.HandleErr(w, req, inf.Tx.Tx, http.StatusBadRequest, err, nil)
		return
	}

	isos(w, req, inf.Tx, inf.User, ir)
}

// cmdOverwriteCtxKey is used in an http.Request's context
// to set a cmd override value.
var cmdOverwriteCtxKey struct{}

// isos performs the majority of work for the /isos endpoint handler. It is separated out from
// the exported handler for testability.
func isos(w http.ResponseWriter, req *http.Request, tx *sqlx.Tx, user *auth.CurrentUser, ir isoRequest) {

	// Ensure that the given OSVersionDir is defined in the osversions.json config
	// file as a valid directory. This directory is later referenced for ISO creation
	// and therefore must an allowed value.
	if ok, err := ir.validateOSDir(tx); err != nil {
		api.HandleErr(w, req, tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("unable to read osversions configuration: %v", err))
		return
	} else if !ok {
		api.HandleErr(w, req, tx.Tx, http.StatusBadRequest, fmt.Errorf("invalid OS version directory: %q", ir.OSVersionDir), nil)
		return
	}

	// Determine the kickstart root directory, which is either a default
	// value or may be overridden by a database/Parameter entry.
	ksDir, err := kickstarterDir(tx, ir.OSVersionDir)
	if err != nil {
		api.HandleErr(w, req, tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("unable to determine kickstarter directory: %v", err))
		return
	}

	// cfgDir holds the kickstart config files within the root
	// kickstart directory.
	cfgDir := filepath.Join(ksDir, ksCfgDir)
	log.Infof("cfg_dir: %s", cfgDir)

	genISOCmd, err := newStreamISOCmd(ksDir)
	if err != nil {
		api.HandleErr(w, req, tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("unable to initialize genISO command: %v", err))
		return
	}
	defer genISOCmd.cleanup()

	// Allow for the request context to carry a modifier function that can change the
	// genISOCmd's command. This is purely used for testing.
	if cmdMod, ok := req.Context().Value(cmdOverwriteCtxKey).(func(in *exec.Cmd) *exec.Cmd); ok {
		genISOCmd.cmd = cmdMod(genISOCmd.cmd)
	}

	log.Infof("Using %s ISO generation command: %s", genISOCmd.cmdType, genISOCmd.String())

	if err = writeKSCfgs(cfgDir, ir, genISOCmd.String()); err != nil {
		api.HandleErr(w, req, tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("unable to create kickstarter files: %v", err))
		return
	}

	isoFilename := fmt.Sprintf("%s-%s.iso", ir.fqdn(), ir.OSVersionDir)
	// strings.ReplaceAll was added in Go 1.12
	isoFilename = strings.Replace(isoFilename, "/", "_", -1)

	w.Header().Set(rfc.ContentDisposition, fmt.Sprintf("attachment; filename=%q", isoFilename))
	w.Header().Set(rfc.ContentType, rfc.ApplicationOctetStream)

	if err = genISOCmd.stream(w); err != nil {
		api.HandleErr(w, req, tx.Tx, http.StatusInternalServerError, nil, fmt.Errorf("unable to generate ISO: %v", err))
		return
	}

	// Create changelog entry
	err = api.CreateChangeLogBuildMsg(
		api.ApiChange,
		api.Created,
		user,
		tx.Tx,
		"ISO",
		ir.fqdn(),
		map[string]interface{}{"OS": ir.OSVersionDir},
	)
	if err != nil {
		// At this point, it's not possible to modify the HTTP response.
		log.Errorf("error creating changelog entry for ISO creation: %v", err)
	}
}

// isoRequest represents the JSON object clients use to
// request an ISO be generated.
type isoRequest struct {
	OSVersionDir  string          `json:"osversionDir"`
	HostName      string          `json:"hostName"`
	DomainName    string          `json:"domainName"`
	RootPass      string          `json:"rootPass"`
	DHCP          boolStr         `json:"dhcp"`
	IPAddr        net.IP          `json:"ipAddress"`
	IPNetmask     net.IP          `json:"ipNetmask"`
	IPGateway     net.IP          `json:"ipGateway"`
	IP6Address    string          `json:"ip6Address"`
	IP6Gateway    net.IP          `json:"ip6Gateway"`
	InterfaceName string          `json:"interfaceName"`
	InterfaceMTU  util.JSONIntStr `json:"interfaceMtu"`
	Disk          string          `json:"disk"`
	MgmtIPAddress net.IP          `json:"mgmtIpAddress"`
	MgmtIPNetmask net.IP          `json:"mgmtIpNetmask"`
	MgmtIPGateway net.IP          `json:"mgmtIpGateway"`
	MgmtInterface string          `json:"mgmtInterface"`
}

func (i *isoRequest) fqdn() string {
	fqdn := i.HostName
	if i.DomainName != "" {
		fqdn += "." + i.DomainName
	}
	return fqdn
}

// validate returns an empty slice if the isoRequest is valid. Otherwise,
// it returns a slice of error messages.
func (i *isoRequest) Validate(tx *sql.Tx) error {
	errs := []error{}
	addErr := func(msg string) { errs = append(errs, errors.New(msg)) }

	if i.OSVersionDir == "" {
		addErr("osversionDir is required")
	}
	if i.HostName == "" {
		addErr("hostName is required")
	}
	if i.DomainName == "" {
		addErr("domainName is required")
	}
	if i.RootPass == "" {
		addErr("rootPass is required")
	}
	if !i.DHCP.isSet {
		addErr("dhcp is required")
	}
	if i.InterfaceMTU == 0 {
		addErr("interfaceMtu is required")
	}
	if i.Disk == "" {
		addErr("disk is required")
	}
	if len(i.MgmtIPAddress) > 0 {
		if i.MgmtInterface == "" {
			addErr("mgmtInterface is required when mgmtIpAddress is provided")
		}
		if len(i.MgmtIPGateway) == 0 {
			addErr("mgmtIpGateway is required when mgmtIpAddress is provided")
		}
	}
	if v, ok := i.DHCP.val(); ok && !v {
		if len(i.IPAddr) == 0 {
			addErr("ipAddress is required if DHCP is no")
		}
		if len(i.IPNetmask) == 0 {
			addErr("ipNetmask is required if DHCP is no")
		}
		if len(i.IPGateway) == 0 {
			addErr("ipGateway is required if DHCP is no")
		}
	}

	if i.IP6Address != "" {
		if ipv6, _, err := net.ParseCIDR(i.IP6Address); err != nil {
			if ipv6 = net.ParseIP(i.IP6Address); len(ipv6) != 16 || ipv6.To4() != nil {
				addErr("ip6Address must be a valid IPv6 address (with optional CIDR prefix)")
			}
		} else if len(ipv6) != 16 || ipv6.To4() != nil {
			addErr("ip6Address must be a valid IPv6 address (with optional CIDR prefix)")
		}
	}

	return util.JoinErrs(errs)
}

// validateOSDir ensures that the OSDir value corresponds to a
// valid directory, as determined by the osversions.json config
// file. Since this directory is acted upon during ISO generation,
// it's important that it be valid.
func (i *isoRequest) validateOSDir(tx *sqlx.Tx) (bool, error) {
	validOSVersions, err := getOSVersions(tx)
	if err != nil {
		return false, fmt.Errorf("unable to read osversions configuration: %v", err)
	}

	var isValid bool
	for _, validOSDir := range validOSVersions {
		if i.OSVersionDir == validOSDir {
			isValid = true
			break
		}
	}

	return isValid, nil
}

// boolStr is used to decode boolean strings (e.g. "yes") as
// part of a JSON response. Part of the /isos JSON request
// generated by TrafficPortal uses this format.
// If an unrecognize or empty string is given, then
// the 'val' and 'isSet' fields will be false. Otherwise,
// 'isSet' will be true.
type boolStr struct {
	isSet bool // false if UnmarshalText is given an unrecognized value
	v     bool
}

// UnmarshalText decodes strings representing boolean values.
// It nevers returns an error to allow for all validation errors
// to be grouped together.
func (b *boolStr) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "yes", "true", "1":
		b.v = true
		b.isSet = true
	case "no", "false", "0":
		b.v = false
		b.isSet = true
	}
	return nil
}

// val returns the boolean value and whether
// the value was set or not.
func (b *boolStr) val() (value, ok bool) {
	return b.v, b.isSet
}
