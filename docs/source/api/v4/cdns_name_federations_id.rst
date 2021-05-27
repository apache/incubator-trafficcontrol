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

.. _to-api-cdns-name-federations-id:

************************************
``cdns/{{name}}/federations/{{ID}}``
************************************

``PUT``
=======
Updates a federation.

:Auth. Required: Yes
:Roles Required: "admin"
:Response Type:  Object

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+-------------------------------------------------------------------------------------+
	| Name | Description                                                                         |
	+======+=====================================================================================+
	| name | The name of the CDN for which the federation identified by ``ID`` will be inspected |
	+------+-------------------------------------------------------------------------------------+
	|  ID  | An integral, unique identifier for the federation to be inspected                   |
	+------+-------------------------------------------------------------------------------------+

:cname: The Canonical Name (CNAME) used by the federation

	.. note:: The CNAME must end with a "``.``"

:description: An optional description of the federation
:ttl:         Time to Live (TTL) for the name record used for ``cname``

.. code-block:: http
	:caption: Request Example

	PUT /api/4.0/cdns/CDN-in-a-Box/federations/1 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.62.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 33
	Content-Type: application/json

	{
		"cname": "foo.bar.",
		"ttl": 48
	}


Response Structure
------------------
:cname:       The Canonical Name (CNAME) used by the federation
:description: An optionally-present field containing a description of the field

	.. note:: This key will only be present if the description was provided when the federation was created

:lastUpdated: The date and time at which this federation was last modified, in an ISO-like format
:ttl:         Time to Live (TTL) for the ``cname``, in hours


.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	access-control-allow-credentials: true
	access-control-allow-headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	access-control-allow-methods: POST,GET,OPTIONS,PUT,DELETE
	access-control-allow-origin: *
	content-type: application/json
	set-cookie: mojolicious=...; Path=/; HttpOnly
	whole-content-sha512: qcjfQ+gDjNxYQ1aq+dlddgrkFWnkFYxsFF+SHDqqH0uVHBVksmU0aTFgltozek/u6wbrGoR1LFf9Fr1C1SbigA==
	x-server-name: traffic_ops_golang/
	content-length: 174
	date: Wed, 05 Dec 2018 01:03:40 GMT

	{ "alerts": [
		{
			"text": "cdnfederation was updated.",
			"level": "success"
		}
	],
	"response": {
		"id": 1,
		"cname": "foo.bar.",
		"ttl": 48,
		"description": null,
		"lastUpdated": "2018-12-05 01:03:40+00"
	}}


``DELETE``
==========
Deletes a specific federation.

:Auth. Required: Yes
:Roles Required: "admin"
:Response Type:  ``undefined``

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+-------------------------------------------------------------------------------------+
	| Name | Description                                                                         |
	+======+=====================================================================================+
	| name | The name of the CDN for which the federation identified by ``ID`` will be inspected |
	+------+-------------------------------------------------------------------------------------+
	|  ID  | An integral, unique identifier for the federation to be inspected                   |
	+------+-------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	DELETE /api/4.0/cdns/CDN-in-a-Box/federations/1 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.62.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	access-control-allow-credentials: true
	access-control-allow-headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	access-control-allow-methods: POST,GET,OPTIONS,PUT,DELETE
	access-control-allow-origin: *
	content-type: application/json
	set-cookie: mojolicious=...; Path=/; HttpOnly
	whole-content-sha512: Cnkfj6dmzTD3if9oiDq33tqf7CnAflKK/SPgqJyfu6HUfOjLJOgKIZvkcs2wWY6EjLVdw5qsatsd4FPoCyjvcw==
	x-server-name: traffic_ops_golang/
	content-length: 68
	date: Wed, 05 Dec 2018 01:17:24 GMT

	{ "alerts": [
		{
			"text": "cdnfederation was deleted.",
			"level": "success"
		}
	]}
