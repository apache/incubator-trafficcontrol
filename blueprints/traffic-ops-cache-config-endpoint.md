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
# Cache Config Snapshot

## Problem Description

Currently, ORT needs to request many large endpoints from Traffic Ops. These requests have a performance cost, and can cause issues for CDNs with many caches, running Traffic Ops or its PostgreSQL database on small hardware.

The cache config generation doesn't need a lot of data, but there don't exist TO endpoints with only the data it does need, or a way to filter it.

## Proposed Change

Traffic Ops will add a “Cache Config Snapshot” feature.

When the cache config is Snapshotted, all data will be serialized and stored, just like the CRConfig.

Snapshot history will be stored. New snapshots will insert with a timestamp, retaining previous versions so they can easily be restored if necessary. Snapshots should be automatically pruned with a configurable number (which maybe be infinity or zero), and/or a configurable amount of time.

The snapshot data will be cached in Traffic Ops' memory, with a configurable cache time (suggested default: 5s).

If the cache is under that age, it will be served by Traffic Ops from memory with no database request.

Initially, the Cache Config Snapshot will be created on Queue, so no interface change is necessary. In the future, Queue and Cache Config Snapshot could be easily separated, if desired.

### Implementation

Note endpoint names are provisional and may be changed to fit Traffic Ops API Requirements and OSS preferences.

The data currently needed by the cache config generation which will be snapshotted can be seen here:

https://godoc.org/github.com/apache/trafficcontrol/traffic_ops/ort/atstccfg/config#TOData

### Traffic Portal Impact

n/a

### Traffic Ops Impact

As above, TO should cache the snapshot in memory, and only fetch from the database if the cache has expired, and pre-check if the timestamp is unchanged (common) before initiating the full expensive snapshot retrieval.

- The snapshot should use DeliveryService-Server IDs, not names, and omit the DSS last_modified, for both the database query and the response.
    - DSS is 99% of the snapshot, using IDs and omitting the timestamp is a large size  reduction, saving network both DB->TO  and TO->client.
    - The snapshot should use one-letter JSON names for DSS, i.e. `“d”: 42, “s”: 24`. This will further reduce the network to the client by about 50%, and improve TO scalability.
    - The snapshot cache should store the serialized JSON bytes, so it doesn’t have to re-serialize for every request. JSON serializing in Go is a huge performance cost, and this will drastically improve scalability.

Queueing Updates will automatically create the Snapshot
    - This may be changed to a button on the Server in the future. But for now, automatically snapshotting on Queue avoids changing the User Interface.

GET `/servers/{{name}}/cache-config-snapshot`
    - Returns the cache config snapshot
    - Per-server, because a lot of data is only needed by one server. For example, that server’s Parameters.
    - TO should load the Snapshot, and then remove the unnecessary data before serializing it in the GET for one server.

#### REST API Impact

See Traffic Ops Impact

#### Client Impact

See Traffic Ops Impact - client functions for each endpoint.

#### Data Model / Database Impact

New table for Cache Config Snapshots.

### ORT Impact

ORT will be changed to request this endpoint if it exists, instead of the API endpoints.

This should be a small change, and should only require changing the single request function. See https://github.com/apache/trafficcontrol/blob/b3e8605e9bdcaf097372c9c5dff337b72ff0bc66/traffic_ops/ort/atstccfg/cfgfile/cfgfile.go#L43

ORT will need to continue to request the API endpoints for the prior Traffic Ops version, if the endpoint doesn't exist, for at least one major Traffic Control version.

### Traffic Monitor Impact

n/a

### Traffic Router Impact

n/a

### Traffic Stats Impact

n/a

### Traffic Vault Impact

n/a

### Documentation Impact

The new API endpoint will be documented. See Traffic Ops Impact.

### Testing Impact

The new API endpoint, and ATS config changes, will have TO API tests and unit tests. See Traffic Ops Impact.

### Performance Impact

No significant performance impact expected. Load impact on Traffic Ops should be reduced.

### Security Impact

n/a

### Upgrade Impact

None. ORT will continue to request the API endpoints if the new endpoint does not exist, so it can be upgraded prior to TO. Likewise, Traffic Ops will continue to serve its API endpoints needed by the existing ORT, if TO is upgraded first.

### Operations Impact

None necessary. Operators will Queue Updates as they currently do, which will automatically create a new Cache Config Snapshot.


### Developer Impact

New TO endpoint, and config generation change. Neither are anticipated to be overly complex to develop or maintain.

## Alternatives

A large percentage of the ORT request data is not necessary, importantly the entire `deliveryserviceserver` endpoint, if it were possible to filter and get only the necessary data from Traffic Ops.

Cache config generation needs to know how many servers are assigned at a tier, to calculate MaxOriginConnections.

There are several other ways to do this. New query parameters, possibly new endpoints, could be added to Traffic Ops, to only get the data necessary.

Also, the Kill-The-Chicken Timestamp Plan includes Server Snapshots, which are stored as timestamps and built on request. Combined with query params to only get necessary data, that adds the advantages of Snapshotting cache configs, without the disadvantages of storing a blob (such as easily breaking backward-compatibility).

Toward the goal of performance, and specifically reducing load on the database, Traffic Ops could also add a Read-While-Writer feature to only process duplicate concurrent requests once, as well as a small (~1 second) cache, to improve scalability and reduce database load.

We're of the opinion that the above alternatives are not independently sufficient, and that the advantages of Cache Config Snapshots outweigh the disadvantages.

## Dependencies

n/a

## References

n/a
