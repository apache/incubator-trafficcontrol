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

.. _to-api-v2-cdns:

********
``cdns``
********
Extract information about all CDNs

``GET``
=======
:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------

.. table:: Request Query Parameters

	+---------------+----------+-----------------------------------------------------------------------------------+
	| Parameter     | Required | Description                                                                       |
	+===============+==========+===================================================================================+
	| domainName    | no       | Return only the CDN that has this domain name                                     |
	+---------------+----------+-----------------------------------------------------------------------------------+
	| dnssecEnabled | no       | Return only the CDNs that are either dnssec enabled or not                        |
	+---------------+----------+-----------------------------------------------------------------------------------+
	| id            | no       | Return only the CDN that has this id                                              |
	+---------------+----------+-----------------------------------------------------------------------------------+
	| name          | no       | Return only the CDN that has this name                                            |
	+---------------+----------+-----------------------------------------------------------------------------------+
	| orderby       | no       | Choose the ordering of the results - must be the name of one of the fields of the |
	|               |          | objects in the ``response`` array                                                 |
	+---------------+----------+-----------------------------------------------------------------------------------+
	| sortOrder     | no       | Changes the order of sorting. Either ascending (default or "asc") or descending   |
	|               |          | ("desc")                                                                          |
	+---------------+----------+-----------------------------------------------------------------------------------+
	| limit         | no       | Choose the maximum number of results to return                                    |
	+---------------+----------+-----------------------------------------------------------------------------------+
	| offset        | no       | The number of results to skip before beginning to return results. Must use in     |
	|               |          | conjunction with limit                                                            |
	+---------------+----------+-----------------------------------------------------------------------------------+
	| page          | no       | Return the n\ :sup:`th` page of results, where "n" is the value of this           |
	|               |          | parameter, pages are ``limit`` long and the first page is 1. If ``offset`` was    |
	|               |          | defined, this query parameter has no effect. ``limit`` must be defined to make    |
	|               |          | use of ``page``.                                                                  |
	+---------------+----------+-----------------------------------------------------------------------------------+

Response Structure
------------------
:dnssecEnabled: ``true`` if DNSSEC is enabled on this CDN, otherwise ``false``
:domainName:    Top Level Domain name within which this CDN operates
:id:            The integral, unique identifier for the CDN
:lastUpdated:   Date and time when the CDN was last modified in an ISO-like format
:name:          The name of the CDN

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: z9P1NkxGebPncUhaChDHtYKYI+XVZfhE6Y84TuwoASZFIMfISELwADLpvpPTN+wwnzBfREksLYn+0313QoBWhA==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 14 Nov 2018 20:46:57 GMT
	Content-Length: 237

	{ "response": [
		{
			"dnssecEnabled": false,
			"domainName": "-",
			"id": 1,
			"lastUpdated": "2018-11-14 18:21:06+00",
			"name": "ALL"
		},
		{
			"dnssecEnabled": false,
			"domainName": "mycdn.ciab.test",
			"id": 2,
			"lastUpdated": "2018-11-14 18:21:14+00",
			"name": "CDN-in-a-Box"
		}
	]}


``POST``
========
Allows user to create a CDN

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
:dnssecEnabled: If ``true``, this CDN will use DNSSEC, if ``false`` it will not
:domainName:    The top-level domain (TLD) belonging to the new CDN
:name:          Name of the new CDN

.. code-block:: http
	:caption: Request Structure

	POST /api/2.0/cdns HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 63
	Content-Type: application/json

	{"name": "test", "domainName": "quest", "dnssecEnabled": false}

Response Structure
------------------
:dnssecEnabled: ``true`` if the CDN uses DNSSEC, ``false`` otherwise
:domainName:    The top-level domain (TLD) assigned to the newly created CDN
:id:            An integral, unique identifier for the newly created CDN
:name:          The newly created CDN's name


.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: 1rZRlOfQioGRrEb4nCfjGGx7y3Ub2h7BZ4z6NbhcY4acPslKSUNM8QLjWTVwLU4WpkfJNxcoyy8NlKULFrY9Bg==
	X-Server-Name: traffic_ops_golang/
	Date: Wed, 14 Nov 2018 20:49:28 GMT
	Content-Length: 174

	{ "alerts": [
		{
			"text": "cdn was created.",
			"level": "success"
		}
	],
	"response": {
		"dnssecEnabled": false,
		"domainName": "quest",
		"id": 3,
		"lastUpdated": "2018-11-14 20:49:28+00",
		"name": "test"
	}}
