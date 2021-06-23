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

# Running

## Prequisites

To run `traffic_ops_golang` proxy locally the following prerequisites are needed:

* Golang version greater or equal to the Go version found in the `GO_VERSION` file at the base of this repository. See: [https://golang.org/doc/install](https://golang.org/doc/install)
* Postgres 13.2 or greater


## Vendoring and Building

### vendoring
We treat `golang.org/x` as a part of the Go compiler so that means that we still vendor application dependencies for stability and reproducible builds.  The [govend](https://github.com/govend/govend) tool is helpful for managing dependencies.

### building
To download the remaining `golang.org/x` dependencies you need to:

`$ go mod vendor -v`

## Configuration

### cdn.conf changes

Copy `traffic_ops/app/conf/cdn.conf` to `$HOME/cdn.conf` so you can modify it for development purposes.

`$HOME/cdn.conf`

```
       "traffic_ops_golang" : {
          "port" : "443",
```

```
       "traffic_ops_golang" : {
          "port" : "8443",
```

## Logging

By default `/var/log/traffic_ops/error.log` is configured for output, to change this modify your `$HOME/cdn.conf` for the following:

`$HOME/cdn.conf`

```
    "traffic_ops_golang" : {
        "..."
        "log_location_error": "stdout",
        "log_location_warning": "stdout",
        "log_location_info": "stdout",
        "log_location_debug": "stdout",
        "log_location_event": "stdout",
        ...
     }
```

# Development

Go is a compiled language so any local changes will require you to CTRL-C the console and re-run the `traffic_ops_golang` Go binary locally:

`go build && ./traffic_ops_golang -cfg $HOME/cdn.conf -dbcfg ../app/conf/development/database.conf`

## Updating a Minor Version

Traffic Control implements [Semantic Versioning](https://semver.org). When adding new fields to the API, we must increase the minor version. If you're the first one adding a new field to a particular object in a particular release, you'll need to do this.

The structs with no version in the name are the latest version.

Most structs do not have versioning. If you are adding a field to a struct with no existing versioning. see `lib/go-tc/deliveryservices.go` for an example.

1. In `lib/go-tc`, rename the old struct to be the previous minor version.
    - For example, if you are adding a field to Delivery Service and existing minor version is 1.4 (so your new minor version is 1.5), in `lib/go-tc/deliveryservices.go` rename `type DeliveryServiceNullable struct` to `type DeliveryServiceNullableV14 struct`.

2. In `lib/go-tc`, create a new struct with an unversioned name, and anonymously embed the previous struct (that you just renamed), along with your new field.
    - For example:
```go
type DeliveryServiceNullable struct {
	DeliveryServiceNullableV14
	MyNewField *int `json:"myNewField" db:"my_new_field"`
}
```

3. In `lib/go-tc`, change the struct's type alias to the new minor version.
    - For example:
```go
type DeliveryServiceNullableV15 DeliveryServiceNullable
```

4. Update the `Sanitize` function on the unversioned struct, e.g. `func (ds *DeliveryServiceNullable) Sanitize()`, which sets your new field to a default value, if it is null.
```go
  func (ds *DeliveryServiceNullable) Sanitize() {
    if ds.MyNewField == nil { ... }
```

5. Update the `Validate` function on the unversioned struct to add validation for your new field.
    - For example, if your new field is a port, `Validate` should verify it is between 0 and 65535.
    - Almost all fields can be invalid! Don't skip this step. Proper validation is essential to Traffic Control functioning properly and rejecting invalid input.

6. Add new versioned Create and Update handlers for the new version in e.g. `deliveryservice/deliveryservices.go`. The added Create and Update handlers will decode requests into the latest version of the struct and should pass it to an underlying versioned `create` or `update` function:

  For example:
```go
func CreateV15(w http.ResponseWriter, r *http.Request) {
  ...
	ds := tc.DeliveryServiceNullableV15{}
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}

	res, status, userErr, sysErr := createV15(w, r, inf, ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, status, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.DeliveryServiceNullableV15{*res})
}

func createV15(w http.ResponseWriter, r *http.Request, inf *api.Info, reqDS tc.DeliveryServiceNullableV15) *tc.DeliveryServiceNullableV15 {
  ...
}
```

NOTE: the underlying `create` and `update` functions are chained together so that requests for previous minor versions are upgraded into requests of the next latest version until they are finally handled at the latest minor version.

Example call chains:
```
  CreateV12 -> createV12 -> createV13 -> createV14 -> createV15
  CreateV13         ->      createV13 -> createV14 -> createV15
  CreateV14                ->            createV14 -> createV15
  CreateV15                      ->                   createV15
  ```

In this example you would rename the existing `createV14` function to `createV15` and update its signature to accept and return a V15 struct. Then you would create a new `createV14` function, in which you would simply create a V15 struct, insert the V14 struct into it, and pass it to the `createV15` function. By doing that, the V14 request would essentially be upgraded into a V15 request for the underlying `createV15` handler to use.

For an `updateV14` function, you would follow the same pattern as the create function, but you also have to take into account any existing 1.5 fields that may already exist in the resource. So, you have to read existing 1.5 fields from the DB into your V15 struct before passing it to `updateV15`. That is how an "update" request can be upgraded from a 1.4 request to a 1.5 request.

7. Modify the `createV15` and `updateV15` functions (and associated INSERT and UPDATE SQL queries) to create and update the new field in e.g. `deliveryservice/deliveryservices.go`.

8. Modify the `Read` function (and associated SELECT SQL query) to read structs of the new version. For example in `deliveryservice/deliveryservices.go`, you would update the `switch` statement so that `version.Minor >= 5` returns structs of `DeliveryServiceNullable` (the latest version of the struct), and `version.Minor >= 4` returns structs of the embedded `DeliveryServiceNullableV14`. The SELECT SQL query should always be updated to read all of the latest fields, and the `Read` handler should always return the proper versioned struct for the requested API version.

NOTE: the `Delete` handler should not need any modification when adding a new minor version of an API endpoint.

9. Add the routes for your new `CreateV15` and `UpdateV15` handlers to `traffic_ops/traffic_ops_golang/routing/routes.go`.
    - The new latest route must go above the previous version. If the new version is below the old, the new version will never be routed to!

    For example, Change:
```go
		{1.4, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV14, auth.PrivLevelOperations, Authenticated, nil},
```

  To:

```go
		{1.5, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV15, auth.PrivLevelOperations, Authenticated, nil},
		{1.4, http.MethodPost, `deliveryservices/?(\.json)?$`, deliveryservice.CreateV14, auth.PrivLevelOperations, Authenticated, nil},
```

NOTE: the `Read` and `Delete` handlers should always point to the lowest minor version since they are meant to handle requests of any minor version, so the routes for these handlers should not change when adding a new minor version.

## Writing a new route

### Getting a "Handle" on Routes

Open [routes.go](./routing/routes.go). Routes are defined in the `Routes` function, of the form `{version, method, path, handler, ID}`. Notice the path can contain variables, of the form `/{var}/`. These variables will be made available to your handler.

NOTE: Route IDs are immutable and unique. DO NOT change the ID of an existing Route; otherwise, existing configurations may break. New Route IDs can be any integer between 0 and 2147483647 (inclusive), as long as it's unique.

### Creating a Handler

The first step is to create your handler. For an example, look at `monitoringHandler` in `monitoring.go`. Your handler arguments can be any data available to the router (the config and database, or what you can create from them). Passing the `db` or prepared `Stmt`s is common. The handler function must return a `RegexHandlerFunc`. In general, you want to return an inline function, `return func(w http.ResponseWriter, r *http.Request, p ParamMap) {...`.

The `ResponseWriter` and `Request` are standard Go `HandlerFunc` parameters. The `ParamMap` is a `map[string]string`, containing the variables from your route path.

Now, your handler just needs to load the data, format it, and write it to the `ResponseWriter`, like any other Go `HandlerFunc`.

 If you're just learning Go, look at some of the other endpoints like `monitoring.go`, and maybe google some Golang tutorials on SQL, JSON, and HTTP. The Go documentation is also helpful, particularly  https://golang.org/pkg/database/sql/ and https://golang.org/pkg/encoding/json/.

Your handler should be in its own file, where you can create any structs and helper functions you need.

### Registering the Handler

Back to `routes.go`, you need to add your handler to the `Routes` function. For example, `/api/2.0/cdns` would look like `{2.0, http.MethodGet, "cdns", wrapHeaders(wrapAuth(cdnsHandler(d.DB), d.Insecure, d.TOSecret, rd.PrivLevelStmt, CdnsPrivLevel))},`.

The only thing we haven't talked about are those `wrap` functions. They each take a `RegexHandlerFunc` and return a `RegexHandlerFunc`, which lets them 'wrap' your handler. You almost certainly need both of them; if you're not sure, ask on the mailing list or Slack. You'll notice the `wrapAuth` function also takes config parameters, as well as a `PrivLevel`. You should create a constant in your handler file of the form `EndpointPrivLevel` and pass that. If your endpoint modifies data, use `PrivLevelOperations`, otherwise `PrivLevelReadOnly`.

That's it! Test your endpoint, read [Contributing.md](https://github.com/apache/trafficcontrol/blob/master/CONTRIBUTING.md) if you haven't, and submit a pull request!

If you have any trouble, or suggestions for this guide, hit us up on the [mailing list](mailto:dev@trafficcontrol.apache.org) or [Slack](https://goo.gl/Suzakj).
