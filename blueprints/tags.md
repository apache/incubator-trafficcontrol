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
# Tags

## Problem Description
Tags are a fairly long-requested feature; to be able to associate a key word or
phrase with a Delivery Service, or a user. Today, some organizations are abusing
the overloaded "description" properties of a Delivery Service to provide
comma-separated key words and phrases, essentially inventing their own tags by
forcing semantics on a field that does not have them.

## Proposed Change
A new, incredibly simple object will be added to the data model: Tags. A Tag is
nothing more than a unique name, which can be associated with the following
objects:

- Cache Groups
- Delivery Services
- Origins as they appear in Traffic Ops API requests and responses
- Physical Locations as they appear in Traffic Ops API requests and responses
- Profiles
- Servers as they appear in Traffic Ops API requests and responses
- Tenants as they appear in Traffic Ops API requests and responses
- Topologies as they appear in Traffic Ops API requests and responses
- Users as they appear in Traffic Ops API requests and responses

## Component Impact
The only impacted components will be Traffic Ops and Traffic Portal, as other
components are strictly not allowed to derive meaning from an object's Tags.

### Traffic Ops Impact
A new object will be added, the Tag, which will appear through the API like the
following TypeScript interface.

```typescript
interface Tag {
	name: string;
}
```

Further, the aforementioned object types - which may be referred to as "taggable
objects" - will all be augmented with a new property: Tags, which will be a set
of Tag Names.

```typescript
interface Taggable {
	tags: Set<string>;
}
```

The Tags will be "shared" globally across all object types; that is, there are
not separate "server tags" and "Delivery Service tags", only Tags.

#### Database Specifics
The suggested method for storing Tag data is with one table for storing what
Tags exist, and separate tables for each Taggable object to be paired with zero
or more Tags.

The main Tag table ought to look like this:
```text
               Table "public.tag"
 Column | Type | Collation | Nullable | Default
--------+------+-----------+----------+---------
 name   | text |           | not null |
Indexes:
    "tag_pkey" PRIMARY KEY, btree (name)
```

and as an example of an object-to-Tag relation table, the server-to-Tag relation
table should resemble this:
```text
            Table "public.server_tag"
 Column |  Type  | Collation | Nullable | Default
--------+--------+-----------+----------+---------
 tag    | text   |           | not null |
 server | bigint |           | not null |
Indexes:
    "server_tag_pkey" PRIMARY KEY, btree (tag, server)
Foreign-key constraints:
    "server_tag_server_fkey" FOREIGN KEY (server) REFERENCES server(id) ON UPDATE CASCADE ON DELETE CASCADE
    "server_tag_tag_fkey" FOREIGN KEY (tag) REFERENCES tag(name) ON UPDATE CASCADE ON DELETE CASCADE
```

Also, the release that includes Tags should also include a database migration
that adds new tags based on existing Types, to make replacing them easier.

Specifically, it should add Tags for:

- Cache Group Types other than
	- `ORG_LOC`
	- `EDGE_LOC`
	- `MID_LOC`
	- `INFRA_LOC` (which will need to be created if it doesn't already exist)
- Server Types other than
	- `EDGE`
	- `MID`
	- `CCR`
	- `TRAFFIC_ROUTER` (which will need to be created if it doesn't already exist)
	- `ORG`
	- `INFLUXDB` (which will need to be created if it doesn't already exist)
	- `RASCAL`
	- `TRAFFIC_MONITOR` (which will need to be created if it doesn't already exist)
	- `RIAK`
	- `TRAFFIC_VAULT` (which will need to be created if it doesn't already exist)
	- `ORG`
	- `TRAFFIC_OPS` (which will need to be created if it doesn't already exist)
	- `TRAFFIC_OPS_DB` (which will need to be created if it doesn't already exist)
	- `TRAFFIC_STATS` (which will need to be created if it doesn't already exist)
	- `INFRA` (which will need to be created if it doesn't already exist)
- Delivery Service Types other than
	- `HTTP`
	- `HTTP_NO_CACHE`
	- `HTTP_LIVE`
	- `HTTP_LIVE_NATNL`
	- `DNS`
	- `DNS_LIVE`
	- `DNS_LIVE_NATNL`
	- `ANY_MAP`
	- `CLIENT_STEERING`
	- `STEERING`

These created Tags should also be associated with the objects that have these
Types at the time they are created.

#### API Impact
The actual amount of work to add Tags to all Taggable objects is rather large,
considering the number of endpoints that need to be changed, but each change is
quite non-invasive, and can all be done atomically without breaking any
functionality between them.

The new Tags endpoint (`/tags`) shall support the HTTP request methods GET and
POST. A further endpoint, `/tags/{{Tag Name}}` will also be added that
supports GET, PUT and DELETE. These endpoints are herein specified.

##### `/tags`
This endpoint deals with creating Tags and retrieving representations thereof.

###### GET
Retrieves Tag representations.

- *Required Roles* None
- *Response Type* Array

**Request Structure**
This method of this endpoint implements the standard pagination query string
parameters. It provides no additional query string parameters.

*Request Example*
```http
GET /api/4.0/tags?limit=1 HTTP/1.1
Host: trafficops.infra.ciab.test
Accept: application/json, */*;q=0.9
Cookie: mojolicious=...

```

**Response Structure**
The response is an array of representations of Tags, each representation
extended with the `lastUpdated` property containing the Date/Time at which
the Tag was last modified.

This method of this endpoint also implements the `count` property of the
top-level `summary` object.

*Response Example*
```http
HTTP/1.1 200 OK
Content-Type: application/json
Server: Traffic Ops/5.0
Date: Tue, 23 Jun 2020 20:46:57 GMT
Transfer-Encoding: chunked

{ "response": [
	{
		"name": "Foo",
		"lastUpdated": "2020-06-23T20:45:00.000Z"
	}
],
"summary": {
	"count": 347
}}
```

###### POST
Creates a new Tag or Tags.

- *Required Roles* Operations or Admin
- *Response Type* Array

**Request Structure**
This method of this endpoint provides no query string parameters.
The request body of a POST request to `/tags` is either a representation of
a Tag, or an array thereof.

*Request Example*
```http
POST /api/4.0/tags HTTP/1.1
Host: trafficops.infra.ciab.test
Accept: application/json, */*;q=0.9
Cookie: mojolicious=...
Content-Type: application-json
Content-Length: 52

[
	{
		"name": "Test"
	},
	{
		"name": "Quest"
	}
]
```

The request must be rejected with an appropriate HTTP Response code on the range
[400, 500) if the request is not properly encoded as either a single Tag or an
array thereof, any submitted Tag already exists, or any submitted Tag has a Name
containing characters that are not alphanumeric, <kbd>=</kbd>, or <kbd>\_</kbd>.

**Response Structure**
The response is an array - always, even if the request body contained only a
single object - of representations of the Tag objects created.

*Response Example*
```http
HTTP/1.1 201 Created
Content-Type: application/json
Server: Traffic Ops/5.0
Date: Tue, 23 Jun 2020 20:46:57 GMT
Location: /api/4.0/tags?newerThan=2020-06-23T20:46:56.999Z&olderThan=2020-06-23T20:46:57.001Z
Transfer-Encoding: chunked

{ "alerts": [{
	"level": "success",
	"text": "Created 2 tags"
}],
"response": [
	{
		"name": "Test"
	},
	{
		"name": "Quest"
	}
]}
```

##### `/tags/{{Tag Name}}`
This endpoint deals with manipulations and representations of singular Tags.

###### GET
Retrieves a Tag's representation.

- *Required Roles* None
- *Response Type* Object

**Request Structure**
The single route parameter `Tag Name` must be the name of an existing
Tag.

This method of this endpoint provides no query string parameters.

*Request Example*
```http
GET /api/4.0/tags/Test HTTP/1.1
Host: trafficops.infra.ciab.test
Accept: application/json, */*;q=0.9
Cookie: mojolicious=...

```

**Response Structure**
The response is a representation of the requested Tag, augmented with the
`lastUpdated` property containing the Date/Time at which the Tag was last
modified.

*Response Example*
```http
HTTP/1.1 200 OK
Content-Type: application/json
Server: Traffic Ops/5.0
Date: Tue, 23 Jun 2020 20:46:57 GMT
Transfer-Encoding: chunked

{ "response":
	{
		"name": "Test",
		"lastUpdated": "2020-06-23T20:45:00.000Z"
	}
}
```

###### PUT
Edits a Tag.

- *Required Roles* Operations or Admin
- *Response Type* Array

**Request Structure**
The single route parameter `Tag Name` must be the name of an existing
Tag.

This method of this endpoint provides no query string parameters.

The request body of a PUT request to `/tags/{{Tag Name}}` is a representation of
a Tag.

*Request Example*
```http
POST /api/4.0/tags HTTP/1.1
Host: trafficops.infra.ciab.test
Accept: application/json, */*;q=0.9
Cookie: mojolicious=...
Content-Type: application-json
Content-Length: 20

{
	"name": "Bar"
}
```

**Response Structure**
The response is a representation of the edited Tag, augmented with the
`lastUpdated` property containing the Date/Time at which the Tag was last
modified.

The request must be rejected with an appropriate HTTP response code on the
interval [400, 500) if the Tag named in the request path does not exist, the new
name is the name of a pre-existing Tag (excluding the Tag itself, which should
allow for successful completion as a no-op), the new name contains characters
that are not alphanumeric, <kbd>=</kbd>, or <kbd>\_</kbd>, or the request body
cannot be understood as a JSON representation of a Tag.

*Response Example*
```http
HTTP/1.1 200 OK
Content-Type: application/json
Server: Traffic Ops/5.0
Date: Tue, 23 Jun 2020 20:46:57 GMT
Transfer-Encoding: chunked

{ "alerts": [{
	"level": "success",
	"text": "Edited Tag 'Test', name changed to 'Bar'"
}]
"response":
	{
		"name": "Bar",
		"lastUpdated": "2020-06-23T20:46:57.000Z"
	}
}
```

###### DELETE
Deletes a Tag.

- *Required Roles* Operations or Admin
- *Response Type* Object

**Request Structure**
The single route parameter `Tag Name` must be the name of an existing
Tag.

This method of this endpoint provides no query string parameters.

*Request Example*
```http
DELETE /api/4.0/tags/Bar HTTP/1.1
Host: trafficops.infra.ciab.test
Accept: application/json, */*;q=0.9
Cookie: mojolicious=...

```

**Response Structure**
The response is a representation of the deleted Tag.

*Response Example*
```http
HTTP/1.1 200 OK
Content-Type: application/json
Server: Traffic Ops/5.0
Date: Tue, 23 Jun 2020 20:46:57 GMT
Transfer-Encoding: chunked

{ "alerts": [{
	"level": "success",
	"text": "Deleted Tag 'Bar'"
}],
"response":
	{
		"name": "Bar"
	}
}
```


#### Client Impact
The return data structures will change, but no code change to any Traffic Ops
client should be necessary for the addition of Tags to existing objects.
However, new methods/functions/procedures must be added to manipulate Tags
themselves.

### Traffic Portal Impact
A new UI section, complete with tables and forms for editing and creation, will
need to be added to accommodate the addition, editing and deletion of Tags.
Also, existing tables and forms for now-taggable objects will need to be updated
to accommodate the use of Tags. Each separate change can be done atomically, and
since the new collections need not be required, they can be done in non-invasive
fashion of arbitrary order (provided Tag manipulation itself is done first)
without disrupting any existing processes or data.

## Documentation Impact
Tags will need to be documented in the data model, both themselves and on the
modeled objects which now contain them. Also, API endpoints will need to be updated
to reflect the new return structures, where applicable.

## Testing Impact
All new Traffic Ops client methods/functions/procedures will need to be
accompanied by corresponding tests, and ideally some part of the route handlers
will also be testable by unit tests.

## Performance Impact
For most objects this would merely add a column to the database query structure,
so the performance impact is expected to be negligible.

## Security Impact
Tags have no implied functionality, and expose no sensitive data, so there is
expected to be no security impact.

## Upgrade Impact
This will require probably multiple database migrations (or possibly one if the
implementer is feeling brave), but should require no change to upgrade processes
nor cause any problems on roll-back, as the only lost data would be the new Tags
that would be inaccessible in an older version anyway.

## Operations Impact
Tags have no operational meaning, but operators may need to be made aware of
Tags' existences and different organizations may place their own, proprietary
importance on a Tag or Tags that they may then need to communicate to operators.

## Developer Impact
Developers should be fairly free from impact, but may just need to be aware that
certain objects now have additional properties.

## Alternatives
There are no alternatives of which I'm aware, other than the obvious "just not
doing tags".
