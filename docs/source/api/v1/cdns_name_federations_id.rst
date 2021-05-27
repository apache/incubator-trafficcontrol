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

.. _to-api-v1-cdns-name-federations-id:

************************************
``cdns/{{name}}/federations/{{ID}}``
************************************

``GET``
=======
.. deprecated:: ATCv4
	Use the ``GET`` method of :ref:`to-api-v1-cdns-name-federations` with the query parameter ``id`` instead.

Retrieves a specific federation used within a specific CDN.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+-----------+-------------------------------------------------------------------------------------+
	| Name      | Description                                                                         |
	+===========+=====================================================================================+
	| name      | The name of the CDN for which the federation identified by ``ID`` will be inspected |
	+-----------+-------------------------------------------------------------------------------------+
	| ID        | An integral, unique identifier for the federation to be inspected                   |
	+-----------+-------------------------------------------------------------------------------------+

.. table:: Request Query Parameters

	+-----------+---------------------------------------------------------------------------------------------------------------+
	| Name      | Description                                                                                                   |
	+===========+===============================================================================================================+
	| orderby   | Choose the ordering of the results - must be the name of one of the fields of the objects in the ``response`` |
	|           | array                                                                                                         |
	+-----------+---------------------------------------------------------------------------------------------------------------+
	| sortOrder | Changes the order of sorting. Either ascending (default or "asc") or descending ("desc")                      |
	+-----------+---------------------------------------------------------------------------------------------------------------+
	| limit     | Choose the maximum number of results to return                                                                |
	+-----------+---------------------------------------------------------------------------------------------------------------+
	| offset    | The number of results to skip before beginning to return results. Must use in conjunction with limit          |
	+-----------+---------------------------------------------------------------------------------------------------------------+
	| page      | Return the n\ :sup:`th` page of results, where "n" is the value of this parameter, pages are ``limit`` long   |
	|           | and the first page is 1. If ``offset`` was defined, this query parameter has no effect. ``limit`` must be     |
	|           | defined to make use of ``page``.                                                                              |
	+-----------+---------------------------------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/1.4/cdns/CDN-in-a-Box/federations/1 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.62.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
:cname:           The Canonical Name (CNAME) used by the federation
:deliveryService: An object with keys that provide identifying information for the :term:`Delivery Service` using this federation

	:id:    The integral, unique identifer for the :term:`Delivery Service`
	:xmlId: The :term:`Delivery Service`'s uniquely identifying 'xml_id'

:description: An optionally-present field containing a description of the field

	.. note:: This key will only be present if the description was provided when the federation was created. Refer to the ``POST`` method of the :ref:`to-api-v1-cdns-name-federations` endpoint to see how federations can be created.

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
	whole-content-sha512: SJA7G+7G5KcOfCtnE3Dq5DCobWtGRUKSppiDkfLZoG5+paq4E1aZGqUb6vGVsd+TpPg75MLlhyqfdfCHnhLX/g==
	x-server-name: traffic_ops_golang/
	content-length: 170
	date: Wed, 05 Dec 2018 00:36:57 GMT

	{ "response": [
		{
			"id": 1,
			"cname": "test.quest.",
			"ttl": 48,
			"description": "A test federation",
			"lastUpdated": "2018-12-05 00:05:16+00",
			"deliveryService": {
				"id": 1,
				"xmlId": "demo1"
			}
		}
	],
	"alerts": [
		{
			"text": "This endpoint is deprecated, please use GET /cdns/{name}/federations with query parameter id instead",
			"level": "warning"
		}
	]}


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

	PUT /api/1.4/cdns/CDN-in-a-Box/federations/1 HTTP/1.1
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

	DELETE /api/1.4/cdns/CDN-in-a-Box/federations/1 HTTP/1.1
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
