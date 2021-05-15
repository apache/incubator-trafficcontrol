#!/usr/bin/env bash

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

maybe_debug() {
	set -o errexit -o nounset
	debug_port="$1"
	shift
	actual_binary="$1"
	shift
	if [[ "$T3C_DEBUG_COMPONENT" == "${actual_binary%.actual}" ]]; then
		command=(dlv --listen=":${debug_port}" --headless=true --api-version=2 exec "/usr/bin/${actual_binary}" --)
	else
		command=("$actual_binary")
	fi
	exec "${command[@]}" "$@"
}

for t3c_tool in $(compgen -c t3c | sort | uniq); do
	(path="$(type -p "$t3c_tool")"
	cd "$(dirname "$path")"
	dlv_script="${t3c_tool}.debug"
	actual_binary="${t3c_tool}.actual"
	touch "$dlv_script"
	chmod +x "$dlv_script"
	<<-DLV_SCRIPT cat > "$dlv_script"
	#!/usr/bin/env bash
	$(type maybe_debug | tail -n+2)
	maybe_debug "${DEBUG_PORT}" "${actual_binary}" "\$@"
	DLV_SCRIPT
	mv "$t3c_tool" "$actual_binary"
	ln -s "$dlv_script" "$t3c_tool"
	)
done
