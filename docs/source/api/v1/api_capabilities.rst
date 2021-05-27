..
..
.. Licensed under the Apache License, Version 2.0 (the "License");
.. you may not use this file except in compliance with the License.
.. You may obtain a copy of the License at
..
..     http://www.apache.org/licenses/LICENSE-2.0
..
.. Unless required by applicable law or agreed to in writing, software
.. distributed under the License is distributed on an "AS IS" BASIS,
.. WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
.. See the License for the specific language governing permissions and
.. limitations under the License.
..

.. _to-api-v1-api_capabilities:

********************
``api_capabilities``
********************
Deals with the capabilities that may be associated with API endpoints and methods. These capabilities are assigned to :term:`Roles`, of which a user may have one or more. Capabilities support "wildcarding" or "globbing" using asterisks to group multiple routes into a single capability

``GET``
=======
Get all API-capability mappings.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Query Parameters

	+----------------+----------+--------+------------------------------------+
	|    Name        | Required | Type   |         Description                |
	+================+==========+========+====================================+
	|   capability   |   no     | string | Capability name                    |
	+----------------+----------+--------+------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/1.1/api_capabilities?capability=types-write HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
:capability:  Capability name
:httpMethod:  An HTTP request method, practically one of:

	- ``GET``
	- ``POST``
	- ``PUT``
	- ``PATCH``
	- ``DELETE``

:httpRoute:   The request route for which this capability applies - relative to the Traffic Ops server's URL
:id:          An integer which uniquely identifies this capability
:lastUpdated: The time at which this capability was last updated, in an ISO-like format

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Cache-Control: no-cache, no-store, max-age=0, must-revalidate
	Content-Type: application/json
	Date: Thu, 01 Nov 2018 14:45:24 GMT
	Server: Mojolicious (Perl)
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: wptErtIop/AfTTQ+1MZdA2YpPXEOuLFfrPQvvaHqO/uX5fRruOVYW+7p8JTrtH1xg1WN+x6FnjQnSHuWwcpyJg==
	Content-Length: 393

	{ "response": [
		{
			"httpMethod": "POST",
			"lastUpdated": "2018-11-01 14:10:22.794114+00",
			"httpRoute": "types",
			"id": 261,
			"capability": "types-write"
		},
		{
			"httpMethod": "PUT",
			"lastUpdated": "2018-11-01 14:10:22.795917+00",
			"httpRoute": "types/*",
			"id": 262,
			"capability": "types-write"
		},
		{
			"httpMethod": "DELETE",
			"lastUpdated": "2018-11-01 14:10:22.799748+00",
			"httpRoute": "types/*",
			"id": 263,
			"capability": "types-write"
		}
	]}

``POST``
========
.. deprecated:: 1.1
	This endpoint does not have an alternative. API Capabilities must be seeded when new endpoints are created.

Create an API-capability mapping.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
:capability: Capability name
:httpMethod: An HTTP request method, one of: 'GET', 'POST', 'PUT', 'PATCH', 'DELETE'
:httpRoute:  The API endpoint for which to create capabilities

.. code-block:: http
	:caption: Request Example

	POST /api/1.1/api_capabilities HTTP/1.1
	Host: ipcdn-cache-51.cdnlab.comcast.net:6443
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 94
	Content-Type: application/x-www-form-urlencoded

	{
		"capability": "types-write",
		"httpRoute": "/api/1.1/api_capabilities",
		"httpMethod": "PATCH"
	}

Response Structure
------------------
:capability:  Capability name
:httpMethod:  An HTTP request method, practically one of:

	- ``GET``
	- ``POST``
	- ``PUT``
	- ``PATCH``
	- ``DELETE``

:httpRoute:   The request route for which this capability applies - relative to the Traffic Ops server's URL
:id:          An integer which uniquely identifies this capability
:lastUpdated: The time at which this capability was last updated, in an ISO-like format

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Cache-Control: no-cache, no-store, max-age=0, must-revalidate
	Content-Type: application/json
	Date: Thu, 01 Nov 2018 14:53:58 GMT
	Server: Mojolicious (Perl)
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: CDz5DUJFoL5DgfnCcvitPmnKJAG5VENhNN6wz2YNqqW1n5HQzSci+NsU5SqfhKnTwnKfSy7PYl9hQhrUKO6KCQ==
	Content-Length: 209

	{ "alerts": [
		{
			"level": "success",
			"text": "API-Capability mapping was created."
		}
	],
	"response": {
		"httpMethod": "PATCH",
		"lastUpdated": null,
		"httpRoute": "/api/1.1/api_capabilities",
		"id": 273,
		"capability": "types-write"
	}}
