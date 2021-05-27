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

.. _to-api-v2-cachegroupparameters:

************************
``cachegroupparameters``
************************

``GET``
=======
Extract information about the :ref:`cache-group-parameters` associated with :term:`Cache Groups`.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Object

Response Structure
------------------
.. table:: Request Query Parameters

	+-------------+----------+--------------------------------------------------------------------------------------------------------------------------------------+
	| Name        | Required | Description                                                                                                                          |
	+=============+==========+======================================================================================================================================+
	| cachegroup  | no       | Show only the :ref:`cache-group-parameters` with the :term:`Cache Group` identified by this integral, unique identifier              |
	+-------------+----------+--------------------------------------------------------------------------------------------------------------------------------------+
	| parameter   | no       | Show only the :ref:`cache-group-parameters` with the :term:`Parameter` identified by this integral, unique identifier                |
	+-------------+----------+--------------------------------------------------------------------------------------------------------------------------------------+
	| orderby     | no       | Choose the ordering of the results - must be the name of one of the fields of the objects in the ``response`` array                  |
	+-------------+----------+--------------------------------------------------------------------------------------------------------------------------------------+
	| sortOrder   | no       | Changes the order of sorting. Either ascending (default or "asc") or descending ("desc")                                             |
	+-------------+----------+--------------------------------------------------------------------------------------------------------------------------------------+
	| limit       | no       | Choose the maximum number of results to return                                                                                       |
	+-------------+----------+--------------------------------------------------------------------------------------------------------------------------------------+
	| offset      | no       | The number of results to skip before beginning to return results. Must use in conjunction with limit                                 |
	+-------------+----------+--------------------------------------------------------------------------------------------------------------------------------------+
	| page        | no       | Return the n\ :sup:`th` page of results, where "n" is the value of this parameter, pages are ``limit`` long and the first page is 1. |
	|             |          | If ``offset`` was defined, this query parameter has no effect. ``limit`` must be defined to make use of ``page``.                    |
	+-------------+----------+--------------------------------------------------------------------------------------------------------------------------------------+

Response Structure
------------------
:cachegroupParameters: An array of identifying information for the :ref:`cache-group-parameters` of :term:`Cache Groups`

	:cachegroup:   A string containing the :ref:`cache-group-name` of the :term:`Cache Group`
	:last_updated: Date and time of last modification in an ISO-like format
	:parameter:    An integer that is the :term:`Parameter`'s :ref:`parameter-id`

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Cache-Control: no-cache, no-store, max-age=0, must-revalidate
	Content-Type: application/json
	Date: Wed, 14 Nov 2018 18:24:12 GMT
	X-Server-Name: traffic_ops_golang/
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: PZyh09NeYYy4sXSv+Bfov0v32EuEk/1y7/B+4fyvhbcPxWQ650NXBDpAe8IsmYZQYVRB03xlBtc33bo3Ixunbg==
	Content-Length: 124

	{ "response": {
		"cachegroupParameters": [
			{
				"parameter": 124,
				"last_updated": "2018-11-14 18:23:40.488853+00",
				"cachegroup": "test"
			}
		]
	}}

``POST``
========
Assign :term:`Parameter`\ (s) to :term:`Cache Group`\ (s).

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Array

Request Structure
-----------------
This endpoint accepts two formats for the request payload:

Single Object Format
	For assigning a single :term:`Parameter` to a single :term:`Cache Group`
Array Format
	For making multiple assignments of :term:`Parameters` to :term:`Cache Groups` simultaneously

Single Object Format
""""""""""""""""""""

:cacheGroupId: An integer that is the :ref:`cache-group-id` of the :term:`Cache Group` to which a :term:`Parameter` is being assigned
:parameterId:  An integer that is the :ref:`parameter-id` of the :term:`Parameter` being assigned

.. code-block:: http
	:caption: Request Example

	POST /api/2.0/cachegroupparameters HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 39
	Content-Type: application/json

	{
		"cacheGroupId": 8,
		"parameterId": 124
	}

Array Format
""""""""""""

:cacheGroupId: An integer that is the :ref:`cache-group-id` of the :term:`Cache Group` to which a :term:`Parameter` is being assigned
:parameterId:  An integer that is the :ref:`parameter-id` of the :term:`Parameter` being assigned

.. code-block:: http
	:caption: Request Example

	POST /api/2.0/cachegroupparameters HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 39
	Content-Type: application/json

	[{
		"cacheGroupId": 8,
		"parameterId": 124
	},
	{
		"cacheGroupId": 8,
		"parameterId": 125
	}]

Response Structure
------------------
:cacheGroupId: An integer that is the :ref:`cache-group-id` of the :term:`Cache Group` to which a :term:`Parameter` has been assigned
:parameterId:  An integer that is the :ref:`parameter-id` of the :term:`Parameter` which has been assigned

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Cache-Control: no-cache, no-store, max-age=0, must-revalidate
	Content-Type: application/json
	Date: Wed, 14 Nov 2018 15:47:49 GMT
	X-Server-Name: traffic_ops_golang/
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: wCv388wFaSjgFLCnI9dchlcyGxaVr8IhBAG25F+rpI2/azCswEYTcVBSlYOg6NxTQRzGkluMvn67jI6rV+vNsQ==
	Content-Length: 136

	{ "alerts": [
		{
			"level": "success",
			"text": "Cachegroup parameter associations were created."
		}
	],
	"response": [
		{
			"cacheGroupId": 8,
			"parameterId": 124
		}
	]}
