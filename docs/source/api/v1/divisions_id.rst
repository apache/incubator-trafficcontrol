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

.. _to-api-v1-divisions-id:

********************
``divisions/{{ID}}``
********************

``GET``
=======
Get a specific Division.

.. deprecated:: ATCv4
	Use the ``GET`` method of :ref:`to-api-v1-divisions` with the ``id`` query parameter instead.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Query Parameters

	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| Name      | Required | Description                                                                                                   |
	+===========+==========+===============================================================================================================+
	| orderby   | no       | Choose the ordering of the results - must be the name of one of the fields of the objects in the ``response`` |
	|           |          | array                                                                                                         |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| sortOrder | no       | Changes the order of sorting. Either ascending (default or "asc") or descending ("desc")                      |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| limit     | no       | Choose the maximum number of results to return                                                                |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| offset    | no       | The number of results to skip before beginning to return results. Must use in conjunction with limit          |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+
	| page      | no       | Return the n\ :sup:`th` page of results, where "n" is the value of this parameter, pages are ``limit`` long   |
	|           |          | and the first page is 1. If ``offset`` was defined, this query parameter has no effect. ``limit`` must be     |
	|           |          | defined to make use of ``page``.                                                                              |
	+-----------+----------+---------------------------------------------------------------------------------------------------------------+

.. table:: Request Path Parameters

	+------+-----------------------------------------------------------+
	| Name | Description                                               |
	+======+===========================================================+
	|  ID  | The integral, unique identifier of the requested Division |
	+------+-----------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/1.4/divisions/1 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
:id:          An integral, unique identifier for this Division
:lastUpdated: The date and time at which this Division was last modified, in an ISO-like format
:name:        The Division name

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: JTBi9pskjuUAg+MSex6ObeWKE/GyIuRNVy2YXo6AVe+x1nFyvvC3iEVXZkmjiSXg2OUXGeSCkA1LcFouQFSs3A==
	X-Server-Name: traffic_ops_golang/
	Date: Thu, 29 Nov 2018 19:59:57 GMT
	Content-Length: 78

	{ "alerts": [{
			"text": "This endpoint is deprecated, please use GET /divisions with the 'id' parameter instead",
			"level": "warning"
		}],
		"response": [{
			"id": 1,
			"lastUpdated": "2018-11-29 18:38:28+00",
			"name": "Quebec"
		}]
	}

``PUT``
=======
Updates a specific Division

:Auth. Required: Yes
:Roles Required: "admin" or "operations"


Request Structure
-----------------
.. table:: Request Path Parameters

	+------+-----------------------------------------------------------+
	| Name | Description                                               |
	+======+===========================================================+
	|  ID  | The integral, unique identifier of the requested Division |
	+------+-----------------------------------------------------------+


:name: The new name of the Division

.. code-block:: http
	:caption: Request Example

	PUT /api/1.4/divisions/3 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 17
	Content-Type: application/json

	{"name": "quest"}

Response Structure
------------------
:id:          An integral, unique identifier for this Division
:lastUpdated: The date and time at which this Division was last modified, in an ISO-like format
:name:        The Division name

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: yBd8MzBR/Qbc/xts44WEIFRTrqeMKZwUe2ufpm6JH6frh1UjFmYRs3/B7E5FTruFWRTuvEIlx5EpDmp3f9LjzA==
	X-Server-Name: traffic_ops_golang/
	Date: Thu, 29 Nov 2018 20:10:36 GMT
	Content-Length: 137

	{ "alerts": [
		{
			"text": "division was updated.",
			"level": "success"
		}
	],
	"response": {
		"id": 3,
		"lastUpdated": "2018-11-29 20:10:36+00",
		"name": "quest"
	}}
