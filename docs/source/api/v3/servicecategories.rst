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

.. _to-api-v3-service-categories:

**********************
``service_categories``
**********************


``GET``
=======
Get all requested :term:`Service Categories`.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------
.. table:: Request Query Parameters

    +-----------+---------------------------------------------------------------------------------------------------------------+
    | Name      | Description                                                                                                   |
    +===========+===============================================================================================================+
    | name      | Filter for :term:`Service Categories` with this name                                                          |
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

    GET /api/3.0/service_categories?name=SERVICE_CATEGORY_NAME HTTP/1.1
    Host: trafficops.infra.ciab.test
    User-Agent: curl/7.47.0
    Accept: */*
    Cookie: mojolicious=...

Response Structure
------------------
:name:        This :term:`Service Category`'s name
:lastUpdated: The date and time at which this :term:`Service Category` was last modified, in an ISO-like format

.. code-block:: http
    :caption: Response Example

    HTTP/1.1 200 OK
    Access-Control-Allow-Credentials: true
    Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
    Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
    Access-Control-Allow-Origin: *
    Content-Type: application/json
    Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
    Whole-Content-Sha512: Yzr6TfhxgpZ3pbbrr4TRG4wC3PlnHDDzgs2igtz/1ppLSy2MzugqaGW4y5yzwzl5T3+7q6HWej7GQZt1XIVeZQ==
    X-Server-Name: traffic_ops_golang/
    Date: Wed, 11 Mar 2020 20:02:47 GMT
    Content-Length: 102

    {
        "response": [
            {
                "lastUpdated": "2020-03-04 15:46:20-07",
                "name": "SERVICE_CATEGORY_NAME"
            }
        ]
    }

``POST``
========
Create a new service category.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  Object

Request Structure
-----------------
:name:        This :term:`Service Category`'s name

.. code-block:: http
    :caption: Request Example

    POST /api/3.0/service_categories HTTP/1.1
    Host: trafficops.infra.ciab.test
    User-Agent: curl/7.47.0
    Accept: */*
    Cookie: mojolicious=...
    Content-Length: 48
    Content-Type: application/json

    {
        "name": "SERVICE_CATEGORY_NAME",
    }

Response Structure
------------------
:name:        This :term:`Service Category`'s name
:lastUpdated: The date and time at which this :term:`Service Category` was last modified, in an ISO-like format

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
    Date: Wed, 11 Mar 2020 20:12:20 GMT
    Content-Length: 154

    {
        "alerts": [
            {
                "text": "serviceCategory was created.",
                "level": "success"
            }
        ],
        "response": {
            "lastUpdated": "2020-03-11 14:12:20-06",
            "name": "SERVICE_CATEGORY_NAME"
        }
    }

``DELETE``
==========
Deletes a specific :term:`Service Category`.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type:  ``undefined``


Request Structure
-----------------

.. code-block:: http
	:caption: Request Example

	DELETE /api/3.0/service_categories/my-service-category HTTP/1.1
	User-Agent: python-requests/2.23.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
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
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 17 Aug 2020 16:13:31 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: yErJobzG9IA0khvqZQK+Yi7X4pFVvOqxn6PjrdzN5DnKVm/K8Kka3REul1XmKJnMXVRY8RayoEVGDm16mBFe4Q==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 17 Aug 2020 15:13:31 GMT
	Content-Length: 93

	{
		"alerts": [
			{
				"text": "serviceCategory was deleted.",
				"level": "success"
			}
		]
	}
