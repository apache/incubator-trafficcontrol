package util

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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/cache-config/t3c-apply/config"
)

func EnsureConfigDirIsGitRepo(cfg config.Cfg) error {
	cmd := exec.Command("git", "status")
	cmd.Dir = cfg.TsConfigDir

	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return errors.New("getting stderr pipe for command: " + err.Error())
	}

	if err := cmd.Start(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// this means Go failed to run the command, not that the command returned an error.
			return errors.New("git status returned: " + err.Error())
		}
	}

	errOutput, err := ioutil.ReadAll(errPipe)
	if err != nil {
		return errors.New("reading stderr: " + err.Error())
	}

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// this means Go failed to run the command, not that the command returned an error.
			return errors.New("waiting for git command: " + err.Error())
		}
	}

	const GitNotARepoMsgPrefix = `fatal: not a git repository`

	errOutput = bytes.ToLower(errOutput)
	if !bytes.Contains(errOutput, []byte(GitNotARepoMsgPrefix)) {
		return nil // it's already a git repo
	}

	if err := makeConfigDirGitRepo(cfg); err != nil {
		return errors.New("making config dir '" + cfg.TsConfigDir + "' a git repo: " + err.Error())
	}

	return nil
}

func makeConfigDirGitRepo(cfg config.Cfg) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = cfg.TsConfigDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init in config dir '%v' returned err %v msg '%v'", cfg.TsConfigDir, err, string(output))
	}

	if err := makeConfigDirGitUser(cfg); err != nil {
		return fmt.Errorf("git creating user in config dir '%v': %v", cfg.TsConfigDir, err)
	}

	if err := makeInitialGitCommit(cfg); err != nil {
		return errors.New("creating initial git commit: " + err.Error())
	}

	if err := MakeGitCommitAll(cfg, GitChangeIsSelf, true); err != nil {
		return errors.New("creating first files git commit: " + err.Error())
	}

	return nil
}

const gitEmail = "traffic-control-cache-config@apache-traffic-control.invalid"
const gitUser = "t3c"

func makeConfigDirGitUser(cfg config.Cfg) error {
	{
		cmd := exec.Command("git", "config", "user.email", gitEmail)
		cmd.Dir = cfg.TsConfigDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git config user.email in config dir '%v' returned err %v msg '%v'", cfg.TsConfigDir, err, string(output))
		}
	}
	{
		cmd := exec.Command("git", "config", "user.name", gitUser)
		cmd.Dir = cfg.TsConfigDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git config user.email in config dir '%v' returned err %v msg '%v'", cfg.TsConfigDir, err, string(output))
		}
	}
	return nil
}

// makeInitialGitCommit makes the initial commit for a new git repo.
// An initial empty commit is desirable, because the first commit is difficult to manipulate.
func makeInitialGitCommit(cfg config.Cfg) error {
	// TODO git config author?

	cmd := exec.Command("git", "commit", "--allow-empty", "-m", "Initial commit")
	cmd.Dir = cfg.TsConfigDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git initial commit error: in config dir '%v' returned err %v msg '%v'", cfg.TsConfigDir, err, string(output))
	}
	return nil
}

const GitChangeIsSelf = true
const GitChangeNotSelf = false

// makeGitCommitAll makes a git commit of all changes in cfg.TsConfigDir, including untracked files.
func MakeGitCommitAll(cfg config.Cfg, self bool, success bool) error {
	{
		// if there are no changes, don't do anything
		cmd := exec.Command("git", "status", "--porcelain")
		cmd.Dir = cfg.TsConfigDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git status error: in config dir '%v' returned err %v msg '%v'", cfg.TsConfigDir, err, string(output))
		}
		if len(output) == 0 {
			// no error and no output means there were zero changes, so just return
			return nil
		}
	}

	{
		cmd := exec.Command("git", "add", "-A")
		cmd.Dir = cfg.TsConfigDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git add error: in config dir '%v' returned err %v msg '%v'", cfg.TsConfigDir, err, string(output))
		}
	}

	now := time.Now() // TODO get a single consistent time when ORT starts?
	msg := makeGitCommitMsg(cfg, now, self, success)

	{
		cmd := exec.Command("git", "commit", "--message", msg)
		cmd.Dir = cfg.TsConfigDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git commit error: in config dir '%v' returned err %v msg '%v'", cfg.TsConfigDir, err, string(output))
		}
	}

	return nil
}

func makeGitCommitMsg(cfg config.Cfg, now time.Time, self bool, success bool) string {
	const appStr = "t3c"
	selfStr := "other"
	if self {
		selfStr = "self"
	}
	timeStr := now.UTC().Format(time.RFC3339)
	// TODO use full args string literal instead?
	modeStr := "report-only=" + strconv.FormatBool(cfg.ReportOnly) +
		" files=" + cfg.Files.String() +
		" install-packages=" + strconv.FormatBool(cfg.InstallPackages) +
		" service-action=" + cfg.ServiceAction.String() +
		" ignore-update-flag=" + strconv.FormatBool(cfg.IgnoreUpdateFlag) +
		" update-ipallow=" + strconv.FormatBool(cfg.UpdateIPAllow) +
		" wait-for-parents=" + strconv.FormatBool(cfg.WaitForParents) +
		" no-unset-update-flag=" + strconv.FormatBool(cfg.NoUnsetUpdateFlag) +
		" report-only=" + strconv.FormatBool(cfg.ReportOnly)

	successStr := "fail"
	if success {
		successStr = "success"
	}
	const sep = " "
	return strings.Join([]string{appStr, selfStr, modeStr, successStr, timeStr}, sep)
}
