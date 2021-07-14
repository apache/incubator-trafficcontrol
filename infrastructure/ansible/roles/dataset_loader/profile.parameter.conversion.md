<!--
    Licensed to the Apache Software Foundation (ASF) under one
    or more contributor license agreements.  See the NOTICE file
    distributed with this work for additional information
    regarding copyright ownership.  The ASF licenses this file
    to you under the Apache License, Version 2.0 (the
    "License"); you may not use this file except in compliance
    with the License.  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing,
    software distributed under the License is distributed on an
    "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
    KIND, either express or implied.  See the License for the
    specific language governing permissions and limitations
    under the License.
-->
# Conversion of existing profile paramters to yaml dictionary

1. Install `jq`
2. Install `ruby`
3. Curl to get the response of https://{{ TO_BASE_URL }}/api/{{ dl_to_api_version }}/profiles/name/{{ TARGET_PROFILE }}/parameters and save that to a file named `profile.parameters.json`
4. `jq -r '[.[] | sort_by(.configFile,.name,.value)[] | {name: .name,configFile: .configFile,value: .value,secure: (if .secure then 1 else 0 end)}]' profile.parameters.json > profile.parameters.filtered.json`
5. `ruby -ryaml -rjson -e 'puts YAML.dump(JSON.parse(STDIN.read))' < profile.parameters.filtered.json > profile.parameters.yml`
6. Transplant into appropriate ansible var file
