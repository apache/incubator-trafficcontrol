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


.. _to-api-asns-id:

***************
``asns/{{id}}``
***************
.. seealso:: `The Autonomous System Wikipedia page <https://en.wikipedia.org/wiki/Autonomous_system_%28Internet%29>` for an explanation of what an ASN actually is.

``GET``
=======
Deal with a specific Autonomous System Number (ASN).
:Auth. Required: Yes
:Roles Required: None
:Response Type: Array

Request Structure
-----------------
.. table:: Request Path Parameters

	+-------------------+----------+----------------------------------------------------+
	| Name              | Required |                 Description                        |
	+===================+==========+====================================================+
	| id                | yes      | The integral, unique identifier of the desired ASN |
	+-------------------+----------+----------------------------------------------------+

Response Structure
------------------
:asn:          Autonomous System Numbers per APNIC for identifying a service provider
:cachegroup:   Related Cache Group name
:cachegroupId: Related Cache Group ID
:id:           An integer which uniquely identifies the ASN
:lastUpdated:  The time and date at which this server entry was last updated in an ISO-like format

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: oeifOX6ImU1KVyCmyswh4uddhbNPZqxliMuNw+lNea1q/SJOYKXpaKnYqVjRm9QqJ7gH3vqeBxCftMLtb3sAWg==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 07 Nov 2018 18:44:31 GMT
	Content-Length: 120

	{ "response": [
		{
			"lastUpdated": "2012-09-17 21:41:22",
			"id": "28",
			"asn": "7016",
			"cachegroup": "us-pa-pittsburgh",
			"cachegroupId": "13"
		}
	]}

``PUT``
=======
Allows user to edit an existing Autonomous System Number (ASN).

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type: Object

Request Structure
-----------------
:asn:          The value of the new ASN
:cachegroupId: The integral, unique identifier of a Cache Group to which this ASN will be assigned
:cachegroup:   An optional field which, if present, specifies the name of a Cache Group to which this ASN will be assigned

.. note:: While this endpoint accepts the ``cachegroup`` field, sending this in the request payload has no effect except that the response will (erroneously) name the Cache Group to which the ASN was assigned. Any subsequent requests will reveal that, in fact, the Cache Group name is set by the ``cachegroupId`` field.

.. table:: Request Path Parameters

	+-------------------+----------+----------------------------------------------------+
	| Name              | Required |                 Description                        |
	+===================+==========+====================================================+
	| id                | yes      | The integral, unique identifier of the desired ASN |
	+-------------------+----------+----------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	PUT /api/1.1/asns/1 HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 29
	Content-Type: application/x-www-form-urlencoded

	{"asn": 2, "cachegroupId": 1}

Response Structure
------------------
:asn:          Autonomous System Numbers per APNIC for identifying a service provider
:cachegroup:   Related Cache Group name
:cachegroupId: Related Cache Group ID
:id:           An integer which uniquely identifies the ASN
:lastUpdated:  The date and time at which this server entry was last updated in an ISO-like format

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; HttpOnly
	Whole-Content-Sha512: /83P4LJVsrQx9BKHFxo5pbhQMlB4o3a9v3PpkspyOJcpNx1S/GJhCPpiANBki547sbY+0vTq76IriHZ4GYp8bA==
	X-Server-Name: traffic_ops_golang/
	Date: Thu, 08 Nov 2018 14:37:39 GMT
	Content-Length: 160

	{ "alerts": [
		{
			"text": "asn was updated.",
			"level": "success"
		}
	],
	"response": {
		"asn": 2,
		"cachegroup": "CDN_in_a_Box_Mid",
		"cachegroupId": 1,
		"id": 1,
		"lastUpdated": "2018-11-08 14:37:39+00"
	}}

``DELETE``
==========
Deletes an Autonomous System Number (ASN).

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  ``undefined``

Request Structure
-----------------
.. table:: Request Path Parameters

	+------+----------+----------------------------------------------------+
	| Name | Required |                 Description                        |
	+======+==========+====================================================+
	| id   | yes      | The integral, unique identifier of the desired ASN |
	+------+----------+----------------------------------------------------+

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
	Whole-Content-Sha512: 6t3WA+DOcfPJB5UnvDpzEVx5ySfmJgEV9wgkO71U5k32L1VXpxcaTdDVLNGgDDl9sdNftmYnKXf5jpfWUuFYJQ==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 07 Nov 2018 19:14:08 GMT
	Content-Length: 58

	{ "alerts": [
		{
			"text": "asn was deleted.",
			"level": "success"
		}
	]}
