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

.. _to-api-v1-types-id:

****************
``types/{{ID}}``
****************

``GET``
=======
.. deprecated:: ATCv4
	This method of this endpoint is deprecated, developers/administrators should use the ``GET`` method of :ref:`to-api-types` with the ``id`` query parameter instead.

:Auth. Required: Yes
:Roles Required: None
:Response Type: Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+-------------------------------------------------------------+
	| Name | Description                                                 |
	+======+=============================================================+
	|  ID  | The integral, unique identifier of the type being inspected |
	+------+-------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/1.5/types/50 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
:description: A short description of this type
:id:          An integral, unique identifier for this type
:lastUpdated: The date and time at which this type was last updated, in an ISO-like format
:name:        The name of this type
:useInTable:  The name of the Traffic Ops database table that contains objects which are grouped, identified, or described by this type

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Tue, 17 Mar 2020 17:29:33 GMT; Max-Age=3600; HttpOnly
	X-Server-Name: traffic_ops_golang/
	Date: Tue, 17 Mar 2020 16:29:33 GMT
	Content-Length: 233

	{ "alerts": [
		{
			"text": "This endpoint is deprecated, please use GET /types with the 'id' query parameter instead",
			"level": "warning"
		}
	],
	"response": [
		{
			"id": 50,
			"lastUpdated": "2020-03-17 16:26:38+00",
			"name": "GRAFANA",
			"description": "Grafana Service",
			"useInTable": "server"
		}
	]}

=======

``PUT``
=======
Updates a type

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+-----------------------------------------------------------+
	| Name | Description                                               |
	+======+===========================================================+
	|  ID  | The integral, unique identifier of the type being updated |
	+------+-----------------------------------------------------------+

:description: A short description of this type
:name:        The name of this type
:useInTable:  The name of the Traffic Ops database table that contains objects which are grouped, identified, or described by this type.

.. note:: Only types with useInTable set to 'server' are allowed to be updated.

.. code-block:: http
	:caption: Request Structure

	PUT /api/2.0/type/3004 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 68
	Content-Type: application/json

	{
		"name": "Example01",
		"description": "Example2",
		"useInTable": "server"
	}

Response Structure
------------------
:description: A short description of this type
:id:          An integral, unique identifier for this type
:lastUpdated: The date and time at which this type was last updated, in an ISO-like format
:name:        The name of this type
:useInTable:  The name of the Traffic Ops database table that contains objects which are grouped, identified, or described by this type

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: EH8jo8OrCu79Tz9xpgT3YRyKJ/p2NcTmbS3huwtqRByHz9H6qZLQjA59RIPaVSq3ZxsU6QhTaox5nBkQ9LPSAA==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 26 Feb 2020 18:58:41 GMT
	Content-Length: 172

	{
		"alerts": [
		{
			"text": "type was updated.",
			"level": "success"
		}],
		"response": [
		{
			"id": 3004,
			"lastUpdated": "2020-02-26 18:58:41+00",
			"name": "Example02",
			"description": "Example"
			"useInTable": "server"
		}]
	}

``DELETE``
==========
Deletes a type

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type: Object


Request Structure
-----------------
.. table:: Request Path Parameters

	+------+-----------------------------------------------------------+
	| Name | Description                                               |
	+======+===========================================================+
	|  ID  | The integral, unique identifier of the type being deleted |
	+------+-----------------------------------------------------------+

.. note:: Only types with useInTable set to "server" are allowed to be deleted.

.. code-block:: http
	:caption: Request Structure

	DELETE /api/2.0/type/3004 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 0

Response Structure
------------------
.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: EH8jo8OrCu79Tz9xpgT3YRyKJ/p2NcTmbS3huwtqRByHz9H6qZLQjA59RIPaVSq3ZxsU6QhTaox5nBkQ9LPSAA==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 26 Feb 2020 18:58:41 GMT
	Content-Length: 84

	{
		"alerts": [
		{
			"text": "type was deleted.",
			"level": "success"
		}],
	}
