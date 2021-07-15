package main

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
	"os"
	"time"

	"github.com/apache/trafficcontrol/cache-config/t3c-apply/config"
	"github.com/apache/trafficcontrol/cache-config/t3c-apply/torequest"
	"github.com/apache/trafficcontrol/cache-config/t3c-apply/util"
	"github.com/apache/trafficcontrol/cache-config/t3cutil"
	"github.com/apache/trafficcontrol/lib/go-log"
)

// exit codes
const (
	Success           = 0
	AlreadyRunning    = 132
	ConfigFilesError  = 133
	ConfigError       = 134
	GeneralFailure    = 135
	PackagingError    = 136
	RevalidationError = 137
	ServicesError     = 138
	SyncDSError       = 139
	UserCheckError    = 140
)

func runSysctl(cfg config.Cfg) {
	if cfg.RunMode == t3cutil.ModeBadAss {
		_, rc, err := util.ExecCommand("/usr/sbin/sysctl", "-p")
		if err != nil {
			log.Errorln("sysctl -p failed")
		} else if rc == 0 {
			log.Debugf("sysctl -p ran succesfully.")
		}
	}
}

func main() {
	var syncdsUpdate torequest.UpdateStatus
	var lock util.FileLock
	cfg, err := config.GetCfg()
	if err != nil {
		fmt.Println(err)
		os.Exit(ConfigError)
	} else if cfg == (config.Cfg{}) { // user used the --help option
		os.Exit(Success)
	}

	if cfg.UseGit == config.UseGitYes {
		err := util.EnsureConfigDirIsGitRepo(config.TSConfigDir)
		if err != nil {
			log.Errorln("Ensuring config directory '" + config.TSConfigDir + "' is a git repo - config may not be a git repo! " + err.Error())
		} else {
			log.Infoln("Successfully ensured ATS config directory '" + config.TSConfigDir + "' is a git repo")
		}
	} else {
		log.Infoln("UseGit not 'yes', not creating git repo")
	}

	if cfg.UseGit == config.UseGitYes || cfg.UseGit == config.UseGitAuto {
		// commit anything someone else changed when we weren't looking,
		// with a keyword indicating it wasn't our change
		if err := util.MakeGitCommitAll(config.TSConfigDir, util.GitChangeNotSelf, cfg.RunMode, true); err != nil {
			log.Errorln("git committing existing changes, dir '" + config.TSConfigDir + "': " + err.Error())
		}
	}

	trops := torequest.NewTrafficOpsReq(cfg)

	// if doing os checks, insure there is a 'systemctl' or 'service' and 'chkconfig' commands.
	if !cfg.SkipOSCheck && cfg.SvcManagement == config.Unknown {
		log.Errorln("OS checks are enabled and unable to find any know service management tools.")
	}

	// create and clean the config.TmpBase (/tmp/ort)
	if !util.MkDir(config.TmpBase, cfg) {
		log.Errorln("mkdir TmpBase '" + config.TmpBase + "' failed, cannot continue")
		os.Exit(GeneralFailure)
	} else if !util.CleanTmpDir(cfg) {
		log.Errorln("CleanTmpDir failed, cannot continue")
		os.Exit(GeneralFailure)
	}
	if cfg.RunMode != t3cutil.ModeReport {
		if !lock.GetLock(config.TmpBase + "/to_ort.lock") {
			os.Exit(AlreadyRunning)
		}
	}

	fmt.Println(time.Now().Format(time.UnixDate))

	if !util.CheckUser(cfg) {
		lock.UnlockAndExit(UserCheckError)
	}

	toolName := trops.GetHeaderComment()
	log.Debugf("toolname: %s\n", toolName)

	// if running in Revalidate mode, check to see if it's
	// necessary to continue
	if cfg.RunMode == t3cutil.ModeRevalidate {
		syncdsUpdate, err = trops.CheckRevalidateState(false)
		if err != nil || syncdsUpdate == torequest.UpdateTropsNotNeeded {
			if err != nil {
				log.Errorln("Checking revalidate state: " + err.Error())
			} else {
				log.Infoln("Checking revalidate state: returned UpdateTropsNotNeeded")
			}
			GitCommitAndExit(RevalidationError, cfg)
		}
	} else {
		syncdsUpdate, err = trops.CheckSyncDSState()
		if err != nil {
			log.Errorln(err)
			GitCommitAndExit(SyncDSError, cfg)
		}
		if cfg.RunMode == t3cutil.ModeSyncDS && syncdsUpdate == torequest.UpdateTropsNotNeeded {
			// check for maxmind db updates even if we have no other updates
			if CheckMaxmindUpdate(cfg) {
				// We updated the db so we should touch and reload
				trops.RemapConfigReload = true
				path := config.TSConfigDir + "/remap.config"
				_, rc, err := util.ExecCommand("/usr/bin/touch", path)
				if err != nil {
					log.Errorf("failed to update the remap.config for reloading: %s\n", err.Error())
				} else if rc == 0 {
					log.Infoln("updated the remap.config for reloading.")
				}
				if err := trops.StartServices(&syncdsUpdate); err != nil {
					log.Errorln("failed to start services: " + err.Error())
					GitCommitAndExit(ServicesError, cfg)
				}
			}
			GitCommitAndExit(Success, cfg)
		}
	}

	if cfg.RunMode == t3cutil.ModeRevalidate {
		log.Infoln("======== Revalidating, no package processing needed ========")
	} else {
		log.Infoln("======== Start processing packages  ========")
		err = trops.ProcessPackages()
		if err != nil {
			log.Errorf("Error processing packages: %s\n", err)
			GitCommitAndExit(PackagingError, cfg)
		}

		// check and make sure packages are enabled for startup
		err = trops.CheckSystemServices()
		if err != nil {
			log.Errorf("Error verifying system services: %s\n", err.Error())
			GitCommitAndExit(ServicesError, cfg)
		}
	}

	log.Debugf("Preparing to fetch the config files for %s, cfg.RunMode: %s, syncdsUpdate: %s\n", cfg.CacheHostName, cfg.RunMode, syncdsUpdate)
	err = trops.GetConfigFileList()
	if err != nil {
		log.Errorf("Unable to continue: %s\n", err)
		GitCommitAndExit(ConfigFilesError, cfg)
	}
	syncdsUpdate, err = trops.ProcessConfigFiles()
	if err != nil {
		log.Errorf("Error while processing config files: %s\n", err.Error())
	}

	// check for maxmind db updates
	// If we've updated also reload remap to reload the plugin and pick up the new database
	if CheckMaxmindUpdate(cfg) {
		trops.RemapConfigReload = true
	}

	if trops.RemapConfigReload == true {
		cfg, ok := trops.GetConfigFile("remap.config")
		_, rc, err := util.ExecCommand("/usr/bin/touch", cfg.Path)
		if err != nil {
			log.Errorf("failed to update the remap.config for reloading: %s\n", err.Error())
		} else if rc == 0 && ok == true {
			log.Infoln("updated the remap.config for reloading.")
		}
	}

	if err := trops.StartServices(&syncdsUpdate); err != nil {
		log.Errorln("failed to start services: " + err.Error())
		GitCommitAndExit(ServicesError, cfg)
	}

	// start 'teakd' if installed.
	if trops.IsPackageInstalled("teakd") {
		svcStatus, pid, err := util.GetServiceStatus("teakd")
		if err != nil {
			log.Errorf("not starting 'teakd', error getting 'teakd' run status: %s\n", err)
		} else if svcStatus == util.SvcNotRunning {
			running, err := util.ServiceStart("teakd", "start")
			if err != nil {
				log.Errorf("'teakd' was not started: %s\n", err)
			} else if running {
				log.Infoln("service 'teakd' started.")
			} else if svcStatus == util.SvcRunning {
				log.Infof("service 'teakd' was already running, pid: %v\n", pid)
			}
		}
	}

	// reload sysctl
	if trops.SysCtlReload == true {
		runSysctl(cfg)
	}

	// update Traffic Ops
	result, err := trops.UpdateTrafficOps(&syncdsUpdate)
	if err != nil {
		log.Errorf("failed to update Traffic Ops: %s\n", err.Error())
	} else if result {
		log.Infoln("Traffic Ops has been updated.")
	}

	GitCommitAndExit(Success, cfg)
}

// TODO change code to always create git commits, if the dir is a repo
// We only want --use-git to init the repo. If someone init'd the repo, t3c-apply should _always_ commit.
// We don't want someone doing manual badass's and not having that log

// GitCommitAndExit attempts to git commit all changes, logs any error, and calls os.Exit with the given code.
func GitCommitAndExit(exitCode int, cfg config.Cfg) {
	success := exitCode == Success
	if cfg.UseGit == config.UseGitYes || cfg.UseGit == config.UseGitAuto {
		if err := util.MakeGitCommitAll(config.TSConfigDir, util.GitChangeIsSelf, cfg.RunMode, success); err != nil {
			log.Errorln("git committing existing changes, dir '" + config.TSConfigDir + "': " + err.Error())
		}
	}
	os.Exit(exitCode)
}

// CheckMaxmindUpdate will (if a url is set) check for a db on disk.
// If it exists, issue an IMS to determine if it needs to update the db.
// If no file or if an update is needed to be done it is downloaded and unpacked.
func CheckMaxmindUpdate(cfg config.Cfg) bool {
	// Check if we have a URL for a maxmind db
	// If we do, test if the file exists, do IMS based on disk time
	// and download and unpack as needed
	result := false
	if cfg.MaxMindLocation != "" {
		// Check if the maxmind db needs to be updated before reload
		result = util.UpdateMaxmind(cfg)
		if result {
			log.Infoln("maxmind database was updated from " + cfg.MaxMindLocation)
		} else {
			log.Infoln("maxmind database not updated. Either not needed or curl/gunzip failure")
		}
	} else {
		log.Infoln(("maxmindlocation is empty, not checking for DB update"))
	}

	return result
}
