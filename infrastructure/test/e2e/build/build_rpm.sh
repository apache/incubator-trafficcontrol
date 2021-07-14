#!/usr/bin/env bash

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#----------------------------------------
function importFunctions() {
	local script=$(readlink -f "$0")
	local scriptdir=$(dirname "$script")
	local e2edir=$(dirname "$scriptdir")
	local testdir=$(dirname "$e2edir")
	local infradir=$(dirname "$testdir")
	export TC_DIR=$(dirname "$infradir")
	functions_sh="$TC_DIR/build/functions.sh"
	if [[ ! -r $functions_sh ]]; then
		echo "error: can't find $functions_sh"
		exit 1
	fi
	. "$functions_sh"
}

# function importFunctions() {
# 	[ ! -z "$TC_DIR" ] || { echo "Cannot find repository root." >&2 ; exit 1; }
# 	export TC_DIR
# 	functions_sh="$TC_DIR/build/functions.sh"
# 	if [[ ! -r $functions_sh ]]; then
# 		echo "Error: Can't find $functions_sh"
# 		exit 1
# 	fi
# 	. "$functions_sh"
# }

#----------------------------------------
function checkAppEnvironment() {
	echo "Verifying the build configuration environment."

	local script=$(readlink -f "$0")
	local scriptdir=$(dirname "$script")

	export APP_DIR=$(dirname "$scriptdir")
	export APP_VERSION=`cat ${TC_DIR}/VERSION`
	export PACKAGE="tce2e"
	export BUILD_NUMBER=${BUILD_NUMBER:-$(getBuildNumber)}
	export RPMBUILD="${APP_DIR}/rpmbuild"
	export DIST="${TC_DIR}/dist"
	export RPM="${PACKAGE}-${APP_VERSION}-${BUILD_NUMBER}.x86_64.rpm"

	# # needs to be built with go 1.11 or greater
	# verify_and_set_go_version
	# if [[ $? -ne 1 ]]; then
	# 	exit 0
	# fi

	echo "=================================================="
	echo "GO_VERSION: $GO_VERSION"
	echo "TC_DIR: $TC_DIR"
	echo "PACKAGE: $PACKAGE"
	echo "APP_DIR: $APP_DIR"
	echo "APP_VERSION: $APP_VERSION"
	echo "BUILD_NUMBER: $BUILD_NUMBER"
	echo "DIST: $DIST"
	echo "RPM: $RPM"
	echo "RPMBUILD: $RPMBUILD"
	echo "--------------------------------------------------"
}

# ---------------------------------------
function initBuildArea() {
	cd "$APP_DIR"

	# prep build environment
	[ -e $RPMBUILD ] && rm -rf $RPMBUILD
	[ ! -e $RPMBUILD ] || { echo "Failed to clean up rpm build directory '$RPMBUILD': $?" >&2; exit 1; }
	mkdir -p $RPMBUILD/{BUILD,RPMS,SOURCES} || { echo "Failed to create build directory '$RPMBUILD': $?" >&2; exit 1; }
}

# ---------------------------------------
function buildRpm() {
	# build
	$GO get -v -d . || { echo "Failed to go get dependencies: $?" >&2; exit 1; }
	$GO build -v -ldflags "-X main.Version=$APP_VERSION" || { echo "Failed to build tce2e: $?" >&2; exit 1; }

	# tar
	tar -cvzf $RPMBUILD/SOURCES/tce2e-${APP_VERSION}.tgz tce2e cfg/cfg.json || { echo "Failed to create archive for rpmbuild: $?" >&2; exit 1; }

	# Work around bug in rpmbuild. Fixed in rpmbuild 4.13.
	# See: https://github.com/rpm-software-management/rpm/commit/916d528b0bfcb33747e81a57021e01586aa82139
	# Takes ownership of the spec file.
	spec=build/tce2e.spec
	spec_owner=$(stat -c%u $spec)
	spec_group=$(stat -c%g $spec)
	if ! id $spec_owner >/dev/null 2>&1; then
	  chown $(id -u):$(id -g) build/tce2e.spec

	  function give_spec_back {
		chown ${spec_owner}:${spec_group} build/tce2e.spec
	  }
	  trap give_spec_back EXIT
	fi

	# build RPM
	rpmbuild --define "_topdir $RPMBUILD" --define "version ${APP_VERSION}" --define "build_number ${BUILD_NUMBER}" -ba build/tce2e.spec || { echo "rpmbuild failed: $?" >&2; exit 1; }

	# copy build RPM to .
	[ -e $DIST ] || mkdir -p $DIST
	cp $RPMBUILD/RPMS/x86_64/${RPM} $DIST
}

importFunctions
checkAppEnvironment
initBuildArea
buildRpm
