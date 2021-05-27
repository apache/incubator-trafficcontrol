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

.. _to-api-v1-divisions:

*************
``divisions``
*************

``GET``
=======
Returns a JSON representation of all configured :term:`Divisions`.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Query Parameters

	+-----------+---------------------------------------------------------------------------------------------------------------+
	| Name      | Description                                                                                                   |
	+===========+===============================================================================================================+
	| id        | Filter for :term:`Divisions` having this integral, unique identifier                                          |
	+-----------+---------------------------------------------------------------------------------------------------------------+
	| name      | Filter for :term:`Divisions` with this name                                                                   |
	+-----------+---------------------------------------------------------------------------------------------------------------+
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
	Whole-Content-Sha512: SLKi9RHa67sGoSz62IDcQsk7KZjTXKfonqMoCUFPXGcNUdhBssvUjc1G7KkWK8X1Ny16geMx2BN8Hm/3dQ75GA==
	X-Server-Name: traffic_ops_golang/
	Date: Thu, 29 Nov 2018 19:44:03 GMT
	Content-Length: 139

	{ "response": [
		{
			"id": 1,
			"lastUpdated": "2018-11-29 18:38:28+00",
			"name": "Quebec"
		},
		{
			"id": 2,
			"lastUpdated": "2018-11-29 18:38:28+00",
			"name": "USA"
		}
	]}


``POST``
========
Creates a new Division.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
:name: The name of the new Division

.. code-block:: http
	:caption: Request Example

	POST /api/1.4/divisions HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 16
	Content-Type: application/json

	{"name": "test"}

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
	Whole-Content-Sha512: +pJm4c3O+JTaSXNt+LP+u240Ba/SsvSSDOQ4rDc6hcyZ0FIL+iY/WWrMHhpLulRGKGY88bM4YPCMaxGn3FZ9yQ==
	X-Server-Name: traffic_ops_golang/
	Date: Thu, 29 Nov 2018 19:52:06 GMT
	Content-Length: 136

	{ "alerts": [
		{
			"text": "division was created.",
			"level": "success"
		}
	],
	"response": {
		"id": 3,
		"lastUpdated": "2018-11-29 19:52:06+00",
		"name": "test"
	}}
