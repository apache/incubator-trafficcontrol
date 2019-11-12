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

# Mock
This directory contains code for building a mock Traffic Ops server - that is, a server with no real database (and thus no persistent data) meant only to test clients of the Traffic Ops API. The _structure_ of the responses is guaranteed to be valid as a client might expect from a real Traffic Ops server, but the data should not be regarded as canonical or special in any way - it will be consistent with itself, but not with operations performed by the client. For example, if a client creates a new user, the mock server will respond that the operation was succesful (supposing the request was syntactically valid, actual content nonwithstanding), but a further query to a `/api/{{version}}/users` endpoint will **NOT** include the new user.

!!! Note !!!
	While most Traffic Ops API endpoints support `.json` suffixes, this mock server does **NOT** (and in fact developers are discouraged from depending on this suffix in general).

## Building and Running
Building the mock server should be very straightforward, assuming the directory was properly cloned into the user's `$GOPATH` (e.g. via `go get github.com/apache/trafficcontrol`). Simply run the `go build` command as normal, then run the utility.

```bash
pushd $GOPATH/src/github.com/apache/trafficcontrol/traffic_ops/client_tests/
go build -o mock . # `-o mock` is suggested to create a useful binary name

# Because it binds to port 443, super-user permissions are generally required
# Note that this requires a valid (though probably self-signed) private key and
# certificate to exist in the same directory, by the names `localhost.key` and
# `localhost.crt`, respectively.
sudo ./mock
# Also note that on some systems it may be necessary to modify your
# `LD_LIBRARY_PATH` to include `libgo.so`, e.g.
# sudo LD_LIBRARY_PATH=/usr/local/lib64 ./mock
```

For additional configuration options, see the output of the `-h`/`--help` flag and/or (the official documentation)[https://traffic-control-cdn.readthedocs.io/en/latest/tools/mock_trafficops.html].
