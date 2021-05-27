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

.. _to-api-v1-api_capabilities-id:

***************************
``api_capabilities/{{ID}}``
***************************
Manages a specific API capability.

``GET``
=======
.. deprecated:: ATCv4

Get an API-capability mapping by id.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+-------------+----------+---------+-----------------------------------------+
	|    Name     | Required |  Type   |         Description                     |
	+=============+==========+=========+=========================================+
	|     ID      |   yes    | integer | A unique identifier for this capability |
	+-------------+----------+---------+-----------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/1.1/api_capabilities/273 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...


Response Structure
------------------
:capability: Capability name
:httpMethod: An HTTP request method, practically one of:

	* ``GET``
	* ``POST``
	* ``PUT``
	* ``PATCH``
	* ``DELETE``

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
	Date: Thu, 01 Nov 2018 16:14:09 GMT
	Server: Mojolicious (Perl)
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: SMSFHOcD6VvfJKmcHmHBQcN+jkRRCmFzx1jyBWhJeyPg04YHPSvUcjZzslWlJqyjwWeoXeNVwxhRkBwl8TQX/g==
	Content-Length: 162

	{
		"alerts": [
			{
				"level": "warning",
				"text": "This endpoint is deprecated, please use 'GET /api_capabilities' instead"
			}
		],
		"response": [
			{
				"httpMethod": "PATCH",
				"lastUpdated": "2018-11-01 14:53:58.853356+00",
				"httpRoute": "/api/1.1/api_capabilities",
				"id": 273,
				"capability": "types-write"
			}
	]}

``PUT``
=======
.. deprecated:: 1.1
	This endpoint does not have an alternative. API Capabilities can only be modified at the database seeding level.

Edit an API-capability mapping.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
:capability: Capability name
:httpMethod: An HTTP request method, practically one of:

	* ``GET``
	* ``POST``
	* ``PUT``
	* ``PATCH``
	* ``DELETE``

:httpRoute:   The request route for which this capability applies - relative to the Traffic Ops server's URL

.. table:: Request Path Parameters

	+-------------+----------+---------+-----------------------------------------+
	|    Name     | Required |  Type   |         Description                     |
	+=============+==========+=========+=========================================+
	|     id      |   yes    | integer | A unique identifier for this capability |
	+-------------+----------+---------+-----------------------------------------+

.. code-block:: http
	:caption: Request Example

	PUT /api/1.1/api_capabilities/273 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 98
	Content-Type: application/x-www-form-urlencoded

	{
		"capability": "types-write",
		"httpRoute": "/api/1.1/api_capabilities/*",
		"httpMethod": "PATCH"
	}

Response Structure
------------------
:capability:  Capability name
:httpMethod:  An HTTP request method, practically one of:

	* ``GET``
	* ``POST``
	* ``PUT``
	* ``PATCH``
	* ``DELETE``

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
	Date: Thu, 01 Nov 2018 18:28:38 GMT
	Server: Mojolicious (Perl)
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: zQuDrqpJt02Fh2fNZ6K7/XmVJ49ZqGTnSbsaR7nOyoxbkmLM17XJV1rtef/SAows2M4j4YjcDbEP4WM/hjCFtw==
	Content-Length: 241

	{
		"alerts": [
			{
				"level": "success",
				"text": "API-Capability mapping was updated."
			},
			{
				"level": "warning",
				"text": "This endpoint is deprecated, please use '[NO ALTERNATE - See https://traffic-control-cdn.readthedocs.io/en/latest/api/api_capabilities_id.html#put]' instead"
			}
		],
		"response": {
			"httpMethod": "PATCH",
			"lastUpdated": "2018-11-01 18:28:10.38317+00",
			"httpRoute": "/api/1.1/api_capabilities/*",
			"id": 273,
			"capability": "types-write"
		}
	}

``DELETE``
==========
.. deprecated:: 1.1
	This endpoint does not have an alternative. API Capabilities can only be deleted at the database seeding level.

Delete a capability.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  ``undefined``

Request Structure
-----------------
.. table:: Request Path Parameters

	+-------------+----------+---------+-----------------------------------------+
	|    Name     | Required |  Type   |         Description                     |
	+=============+==========+=========+=========================================+
	|     id      |   yes    | integer | A unique identifier for this capability |
	+-------------+----------+---------+-----------------------------------------+

.. code-block:: http
	:caption: Request Example

	DELETE /api/1.1/api_capabilities/273 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...


Response Structure
------------------
.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Cache-Control: no-cache, no-store, max-age=0, must-revalidate
	Content-Type: application/json
	Date: Wed, 07 Nov 2018 15:44:14 GMT
	Server: Mojolicious (Perl)
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: eTFJkB2Bh8SCT2A29e21e8qoEdNzFGfuT5a3tDG7u8vwz/JHntQRRR8554a1i65733uWojlWKM65bLSDNmmNqQ==
	Content-Length: 73

	{ "alerts": [
		{
			"level": "success",
			"text": "API-capability mapping deleted."
		},
		{
			"level": "warning",
			"text": "This endpoint is deprecated, please use '[NO ALTERNATE - See https://traffic-control-cdn.readthedocs.io/en/latest/api/api_capabilities_id.html#delete]' instead"
		}
	]}
