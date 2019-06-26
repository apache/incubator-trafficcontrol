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

.. _to-api-phys_locations-id:

*************************
``phys_locations/{{ID}}``
*************************

``GET``
=======
Retrieves information about a specific physical location

.. deprecated:: 1.1
	Use the ``id`` query parameter of the :ref:`to-api-phys_locations` endpoint instead

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+--------------------------------------------------------------------------+
	| Name |                Description                                               |
	+======+==========================================================================+
	|  ID  | The integral, unique identifier of the physical location being retrieved |
	+------+--------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/1.4/phys_locations/2 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------
:address:     The physical location's street address
:city:        The name of the city in which the physical location lies
:comments:    Any and all human-readable comments
:email:       The email address of the physical location's ``poc``
:id:          An integral, unique identifier for the physical location
:lastUpdated: The date and time at which the physical location was last updated, in ISO format
:name:        The name of the physical location
:phone:       A phone number where the the physical location's ``poc`` might be reached
:poc:         The name of a "point of contact" for the physical location
:region:      The name of the region within which the physical location lies
:regionId:    An integral, unique identifier for the region within which the physical location lies
:shortName:   An abbreviation of the ``name``
:state:       An abbreviation of the name of the state or province within which this physical location lies
:zip:         The zip code of the physical location

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: 0g4b3W1AwXytCnBo8TReQQij2v9oHAl7MG9KuwMig5V4sFcMM5qP8dgPsFTunFr00DPI20c7BpUbZsvJtsYTEQ==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 05 Dec 2018 22:45:47 GMT
	Content-Length: 275

	{ "response": [
		{
			"address": "1600 Pennsylvania Avenue NW",
			"city": "Washington",
			"comments": "",
			"email": "",
			"id": 2,
			"lastUpdated": "2018-12-05 17:50:58+00",
			"name": "CDN_in_a_Box",
			"phone": "",
			"poc": "",
			"regionId": 1,
			"region": "Washington, D.C",
			"shortName": "ciab",
			"state": "DC",
			"zip": "20500"
		}
	]}

``PUT``
=======
Updates a physical location

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+--------------------------------------------------------------------------+
	| Name |                Description                                               |
	+======+==========================================================================+
	|  ID  | The integral, unique identifier of the physical location being modified  |
	+------+--------------------------------------------------------------------------+

:address:   The physical location's street address
:city:      The name of the city in which the physical location lies
:comments:  An optional string for containing any and all human-readable comments
:email:     An optional string containing email address of the physical location's ``poc``
:name:      An optional name of the physical location
:phone:     An optional string containing the phone number where the the physical location's ``poc`` might be reached
:poc:       The name of a "point of contact" for the physical location
:region:    An optional string naming the region that contains this physical location\ [1]_
:regionId:  An integral, unique identifier for the region within which the physical location lies\ [1]_
:shortName: An abbreviation of the ``name``
:state:     An abbreviation of the name of the state or province within which this physical location lies
:zip:       The zip code of the physical location

.. code-block:: http
	:caption: Request Structure

	PUT /api/1.4/phys_locations/2 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 268
	Content-Type: application/json

	{
		"address": "1600 Pennsylvania Avenue NW",
		"city": "Washington",
		"comments": "The White House",
		"email": "the@white.house",
		"name": "CDN_in_a_Box",
		"phone": "1-202-456-1414",
		"poc": "Donald J. Trump",
		"regionId": 2,
		"shortName": "ciab",
		"state": "DC",
		"zip": "20500"
	}

Response Structure
------------------
:address:     The physical location's street address
:city:        The name of the city in which the physical location lies
:comments:    Any and all human-readable comments
:email:       The email address of the physical location's ``poc``
:id:          An integral, unique identifier for the physical location
:lastUpdated: The date and time at which the physical location was last updated, in ISO format
:name:        The name of the physical location
:phone:       A phone number where the the physical location's ``poc`` might be reached
:poc:         The name of a "point of contact" for the physical location
:region:      The name of the region within which the physical location lies
:regionId:    An integral, unique identifier for the region within which the physical location lies
:shortName:   An abbreviation of the ``name``
:state:       An abbreviation of the name of the state or province within which this physical location lies
:zip:         The zip code of the physical location

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: qnMe6OqxjSU8H1njlh00HWNR20YnVlOCufqCTdMBcdC1322jk2ICFQsQQ3XuOOR0WSb7h7OHCfXqDC1/jA1xjA==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 05 Dec 2018 23:39:17 GMT
	Content-Length: 385

	{ "alerts": [
		{
			"text": "physLocation was updated.",
			"level": "success"
		}
	],
	"response": {
		"address": "1600 Pennsylvania Avenue NW",
		"city": "Washington",
		"comments": "The White House",
		"email": "the@white.house",
		"id": 2,
		"lastUpdated": "2018-12-05 23:39:17+00",
		"name": "CDN_in_a_Box",
		"phone": "1-202-456-1414",
		"poc": "Donald J. Trump",
		"regionId": 2,
		"region": "Washington, D.C.",
		"shortName": "ciab",
		"state": "DC",
		"zip": "20500"
	}}

``DELETE``
==========
Deletes a physical location

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  ``undefined``

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+--------------------------------------------------------------------------+
	| Name |                Description                                               |
	+======+==========================================================================+
	|  ID  | The integral, unique identifier of the physical location being modified  |
	+------+--------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	DELETE /api/1.4/phys_locations/3 HTTP/1.1
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
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: KeW/tEmICwpCGC8F0YMTqHdeR9J6W6Z3w/U+HOSbeCGyaEheCIhIsWlngT3dyfH1tiu8UyzaPB6QrJyXdybBkw==
	X-Server-Name: traffic_ops_golang/
	Date: Thu, 06 Dec 2018 00:28:48 GMT
	Content-Length: 67

	{ "alerts": [
		{
			"text": "physLocation was deleted.",
			"level": "success"
		}
	]}
