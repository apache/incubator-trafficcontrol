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

.. _to-api-v2-regions-id:

******************
``regions/{{ID}}``
******************

``PUT``
=======
Updates a :term:`Region`.

:Auth. Required: Yes
:Role(s) Required: "admin" or "operator"
:Response Type: Object

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+---------------------------------------------------------+
	| Name |                Description                              |
	+======+=========================================================+
	|  ID  | The integral, unique identifier of the region to update |
	+------+---------------------------------------------------------+

:division:     The new integral, unique identifier of the division which shall contain the region\ [1]_
:divisionName: The new name of the division which shall contain the region\ [1]_
:name:         The new name of the region

.. code-block:: http
	:caption: Request Example

	PUT /api/2.0/regions/5 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 60
	Content-Type: application/json

	{
		"name": "Leeds",
		"division": 3,
		"divisionName": "England"
	}

.. [1] The only "division" key that actually matters in the request body is ``division``; ``divisionName`` is not validated and has no effect - particularly not the effect of re-naming the division - beyond changing the name in the API response to this request. Subsequent requests will reveal the true name of the division. Note that if ``divisionName`` is not present in the request body it will be ``null`` in the response, but again further requests will show the true division name (provided it has been assigned to a division).


Response Structure
------------------
:divisionName: The name of the division which contains this region
:divisionId:   The integral, unique identifier of the division which contains this region
:id:           An integral, unique identifier for this region
:lastUpdated:  The date and time at which this region was last updated, in an ISO-like format
:name:         The region name

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: 7SVj4q7dtSTNQEJlBApEwlad28WBVFnpdHaatoIpNfeLltfcpcdVTcOKB4JXQv7rlSD2p/TxBQC6EXpxwYTnKQ==
	X-Server-Name: traffic_ops_golang/
	Date: Thu, 06 Dec 2018 02:23:40 GMT
	Content-Length: 173

	{ "alerts": [
		{
			"text": "region was updated.",
			"level": "success"
		}
	],
	"response": {
		"divisionName": "England",
		"division": 3,
		"id": 5,
		"lastUpdated": "2018-12-06 02:23:40+00",
		"name": "Leeds"
	}}
