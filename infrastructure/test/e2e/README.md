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

# Traffic Control End-to-End Test Framework

The TC e2e test framework tests the CDN end-to-end, making requests like a client would, and verifying expected behavior.

To run:

```bash
go test -v -cfg mycfg.json
```

# Configuring

The `cfg` parameter must be a file containing a JSON object that looks like:

```json
{
	"log_location_info": "stdout",
	"traffic_ops_uri": "https://localhost",
	"traffic_ops_user": "myuser",
	"traffic_ops_pass": "mypass",
	"traffic_ops_insecure": false,
	"ds_assets": {
		"ds-name-0": "/a/b/c.m3u8",
		"ds-name-1": "/d/e/f.png"
	}
}
```

Config options:

- `traffic_ops_uri` The URI of the Traffic Ops instance to use, to get data for making test requests from. Full URI, including protocol.
- `traffic_ops_user` The Traffic Ops user to log in with.
- `traffic_ops_pass` The password of the given Traffic Ops user to log in with.
- `traffic_ops_insecure` Whether to ignore certificate failures when connecting to Traffic Ops
- `log_location_info` The location to log info. Valid values are `stdout`, `stderr`, `null`, and valid file paths.
- `log_location_error` The location to log errors. Valid values are `stdout`, `stderr`, `null`, and valid file paths.
- `log_location_warning` The location to log warnings. Valid values are `stdout`, `stderr`, `null`, and valid file paths.
- `log_location_event` The location to log events. Valid values are `stdout`, `stderr`, `null`, and valid file paths.
- `ds_assets` A JSON object mapping delivery service names (xml_id) to assets. Assets are only used to make valid requests to the delivery service, and need only be a valid object on the origin server which will return a 200 OK status code and a nonzero body. Assets should be the full URI path, beginning with a `/`.

# Writing Tests

The e2e tests are ordinary Go tests. See `https://golang.org/pkg/testing/`.

To write a e2e test, simply create a new file and create an ordinary Go test func, `func TestWhatever(t *testing.T) {`.

The test main provides global variables which may be used by your test:

- `Cfg` The main configuration object. See `config.go`.
- `TOClient` A Traffic Ops client. This will be created and initialized in `TestMain` before your test is run.
- `TO` A struct of common Traffic Ops data, such as `Servers` and `DeliveryServices`, so every test doesn't have to request the same data repeatedly.

**NOTE** Tests _must not_ modify the global variables. They are used by every test, and not re-initialized for each test.

# Running the Tests

To run, you will need a complete CDN, including all services in the request path, as well as delivery services with valid origins.

With a complete CDN, you should be able to set the Traffic Ops config fields, and run `go test -v -cfg mycfg.json` from the `trafficcontrol/infrastructure/test/e2e` directory.

# Docker

The e2e framework comes with a simple Dockerfile.

Example build:
```bash
docker build --tag tc-e2e:`cat ../../../VERSION`.`git rev-parse --short head` .
```

Example run (with a config file at `./cfg/cfg.json`):
```bash
docker run -it --name tc-e2e --hostname tc-e2e --volume cfg:/lang/go/src/e2e/cfg --network="host" --rm -- tc-e2e:`cat ../../../VERSION`.`git rev-parse --short head`
```
