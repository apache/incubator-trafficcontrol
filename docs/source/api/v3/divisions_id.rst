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

.. _to-api-v3-divisions-id:

********************
``divisions/{{ID}}``
********************

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

	PUT /api/3.0/divisions/3 HTTP/1.1
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
