#!/bin/bash
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

set -o errexit -o nounset
trap 'echo "Error on line ${LINENO} of ${0}"; exit 1' ERR

readonly red_fg="$(printf '%s%s' $'\x1B' '[31m')"
readonly green_fg="$(printf '%s%s' $'\x1B' '[32m')"
readonly normal_fg="$(printf '%s%s' $'\x1B' '[39m')"
colored_text() {
	color="$1"
	sed "s/^/${color}/" | sed "s/$/${normal_fg}/"
}

vendor_dependencies() {
	go mod tidy
	go mod vendor
}

check_vendored_deps() {
	status_output="$(git status --porcelain  -- vendor)"
	if [[ "$(<<<"$status_output" sed '/^$/d' | wc -l)" -eq 0 ]]; then
		echo 'No deleted, modified, or untracked vendor files were found.' | colored_text "$green_fg"
		return
	fi

	declare -A porcelain_symbols
	porcelain_symbols[' D']=deleted
	porcelain_symbols[' M']=modified
	porcelain_symbols[??]=untracked

	for symbol in "${!porcelain_symbols[@]}"; do
		output_of_type="$(<<<"$status_output" grep "^${symbol} " || true)"
		file_count="$(<<<"$output_of_type" sed '/^$/d' | wc -l)"
		file_type="${porcelain_symbols[$symbol]}"
		if [[ "$file_count" -eq 0 ]]; then
			continue
		fi
		echo "${file_count} ${file_type} files were found:" | colored_text "$red_fg"
		<<<"$output_of_type" sed "s/^${symbol} //"
		echo
	done
	exit_code=1
}

check_go_file() {
	go_file="$1"
	if git diff --exit-code --no-patch -- "$go_file"; then
		echo "${go_file} is up-to-date." | colored_text "$green_fg"
		return
	fi
	printf "Changes were found in %s! Please commit them and try again.\n\n" "$go_file" | colored_text "$red_fg"
	exit_code=1
}

GOROOT=/usr/local/go
export PATH="${PATH}:${GOROOT}/bin"
export GOPATH="${HOME}/go"

exit_code=0
declare -A command_exists
command_exists[vendor_dependencies]=1
command_exists[check_vendored_deps]=1
command_exists[check_go_file]=1
requested_command="$1"
shift
if : "${command_exists[$requested_command]}"; then
	"$requested_command" "$@"
else
	printf '`%s` is not a valid command.\n' "${requested_command}"
	exit_code=1
fi

exit $exit_code
