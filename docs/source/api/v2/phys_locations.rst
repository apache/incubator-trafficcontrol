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

.. _to-api-v2-phys_locations:

******************
``phys_locations``
******************

``GET``
=======
Retrieves :term:`Physical Locations`

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Query Parameters

	+-----------+---------------------------------------------------------------------------------------------------------------+
	| Name      | Description                                                                                                   |
	+===========+===============================================================================================================+
	| id        | Filter by integral, unique identifier                                                                         |
	+-----------+---------------------------------------------------------------------------------------------------------------+
	| region    | Filter by integral, unique identifier of containing :term:`Region`                                            |
	+-----------+---------------------------------------------------------------------------------------------------------------+
	| name      | Filter by name                                                                                                |
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

.. code-block:: http
	:caption: Request Example

	GET /api/2.0/phys_locations?name=CDN_in_a_Box HTTP/1.1
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
:lastUpdated: The date and time at which the physical location was last updated, in an ISO-like format
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
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: 0g4b3W1AwXytCnBo8TReQQij2v9oHAl7MG9KuwMig5V4sFcMM5qP8dgPsFTunFr00DPI20c7BpUbZsvJtsYTEQ==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 05 Dec 2018 22:19:52 GMT
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

``POST``
========
Creates a new physical location

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
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
	:caption: Request Example

	POST /api/2.0/phys_locations HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 326
	Content-Type: application/json

	{
		"address": "Buckingham Palace",
		"city": "London",
		"comments": "Buckingham Palace",
		"email": "steve.kingstone@royal.gsx.gov.uk",
		"name": "Great_Britain",
		"phone": "0-843-816-6276",
		"poc": "Her Majesty The Queen Elizabeth Alexandra Mary Windsor II",
		"regionId": 3,
		"shortName": "uk",
		"state": "Westminster",
		"zip": "SW1A 1AA"
	}

.. [1] The only "region" key that actually matters in the request body is ``regionId``; ``region`` is not validated and has no effect - particularly not the effect of re-naming the region - beyond changing the name in the API response to this request. Subsequent requests will reveal the true name of the region. Note that if ``region`` is not present in the request body it will be ``null`` in the response, but again further requests will show the true region name.

Response Structure
------------------
:address:     The physical location's street address
:city:        The name of the city in which the physical location lies
:comments:    Any and all human-readable comments
:email:       The email address of the physical location's ``poc``
:id:          An integral, unique identifier for the physical location
:lastUpdated: The date and time at which the physical location was last updated, in an ISO-like format
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
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: GZ/BC+AgGpOQNfd9oiZy19jtsD8MPOdeyi7PVdz+9YSiLYP44gmn5K+Xi1yS0l59yjHf7O+C1loVQPSlIeP9fg==
	X-Server-Name: traffic_ops_golang/
	Date: Thu, 06 Dec 2018 00:14:47 GMT
	Content-Length: 443

	{ "alerts": [
		{
			"text": "physLocation was created.",
			"level": "success"
		}
	],
	"response": {
		"address": "Buckingham Palace",
		"city": "London",
		"comments": "Buckingham Palace",
		"email": "steve.kingstone@royal.gsx.gov.uk",
		"id": 3,
		"lastUpdated": "2018-12-06 00:14:47+00",
		"name": "Great_Britain",
		"phone": "0-843-816-6276",
		"poc": "Her Majesty The Queen Elizabeth Alexandra Mary Windsor II",
		"regionId": 3,
		"region": null,
		"shortName": "uk",
		"state": "Westminster",
		"zip": "SW1A 1AA"
	}}
