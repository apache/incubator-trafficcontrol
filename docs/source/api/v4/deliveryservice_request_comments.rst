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

.. _to-api-deliveryservice_request_comments:

************************************
``deliveryservice_request_comments``
************************************

``GET``
=======
Gets delivery service request comments.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array

Request Structure
-----------------

.. table:: Request Query Parameters

	+--------------------------+----------+-----------------------------------------------------------------------------------------------------------------------------------------------------+
	| Name                     | Required | Description                                                                                                                                         |
	+==========================+==========+=====================================================================================================================================================+
	| author                   | no       | Filter for :ref:`Delivery Service Request <ds_requests>` comments submitted by the user identified by this username                                 |
	+--------------------------+----------+-----------------------------------------------------------------------------------------------------------------------------------------------------+
	| authorId                 | no       | Filter for :ref:`Delivery Service Request <ds_requests>` comments submitted by the user identified by this integral, unique identifier              |
	+--------------------------+----------+-----------------------------------------------------------------------------------------------------------------------------------------------------+
	| deliveryServiceRequestId | no       | Filter for :ref:`Delivery Service Request <ds_requests>` comments submitted for the delivery service identified by this integral, unique identifier |
	+--------------------------+----------+-----------------------------------------------------------------------------------------------------------------------------------------------------+
	| id                       | no       | Filter for the :ref:`Delivery Service Request <ds_requests>` comment identified by this integral, unique identifier                                 |
	+--------------------------+----------+-----------------------------------------------------------------------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/4.0/deliveryservice_request_comments HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...

Response Structure
------------------
:author:                        The username of the user who created the comment.
:authorId:                      The integral, unique identifier of the user who created the comment.
:deliveryServiceRequestId:      The integral, unique identifier of the :term:`Delivery Service Request` that the comment was posted on.
:id:                            The integral, unique identifier of the :term:`DSR` comment.
:lastUpdated:                   The date and time at which the user was last modified, in an ISO-like format
:value:                         The text of the comment that was posted.
:xmlId:                         This is the ``xmlId`` value that you provided in the request.

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 24 Feb 2020 21:00:26 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: RaJZS1XFJ4oIxVKyyDjTuoQY7gPOmm5EuIL4AgHpyQpuaaNviw0XhGC4V/AKf/Ws6zXLgIUc4OyvMsTxnrilww==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 24 Feb 2020 20:00:26 GMT
	Content-Length: 207

	{
		"response": [
			{
				"authorId": 2,
				"author": "admin",
				"deliveryServiceRequestId": 2,
				"id": 3,
				"lastUpdated": "2020-02-24 19:59:46+00",
				"value": "Changing to a different origin for now.",
				"xmlId": "demo1"
			},
			{
				"authorId": 2,
				"author": "admin",
				"deliveryServiceRequestId": 2,
				"id": 4,
				"lastUpdated": "2020-02-24 19:59:55+00",
				"value": "Using HTTPS.",
				"xmlId": "demo1"
			}
		]
	}

``POST``
========
Allows user to create a :term:`Delivery Service Request` comment.

:Auth. Required: Yes
:Roles Required: "admin", "Federation", "operations", "Portal", or "Steering"
:Response Type:  Object

Request Structure
-----------------
:deliveryServiceRequestId:      The integral, unique identifier of the delivery service that you are commenting on.
:value:                         The comment text itself.
:xmlId:                         This can be any string. It is not validated or used, though it is returned in the response.

.. code-block:: http
	:caption: Request Example

	POST /api/4.0/deliveryservice_request_comments HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...
	Content-Length: 111

	{
		"deliveryServiceRequestId": 2,
		"value": "Does anyone have time to review my delivery service request?"
	}

Response Structure
------------------
:author:                        The username of the user who created the comment.
:authorId:                      The integral, unique identifier of the user who created the comment.
:deliveryServiceRequestId:      The integral, unique identifier of the :term:`Delivery Service Request` that the comment was posted on.
:id:                            The integral, unique identifier of the :term:`DSR` comment.
:lastUpdated:                   The date and time at which the user was last modified, in an ISO-like format
:value:                         The text of the comment that was posted.
:xmlId:                         This is the ``xmlId`` value that you provided in the request.

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 24 Feb 2020 21:02:20 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: LiakFP6L7PrnFO5kLXftx7WQoKn3bGpIJT5N15PvNG2sHridRMV3k23eRJM66ET0LcRfMOrQgRiydE+XgA8h0A==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 24 Feb 2020 20:02:20 GMT
	Content-Length: 223

	{
		"alerts": [
			{
				"text": "deliveryservice_request_comment was created.",
				"level": "success"
			}
		],
		"response": {
			"authorId": 2,
			"author": null,
			"deliveryServiceRequestId": 2,
			"id": 6,
			"lastUpdated": "2020-02-24 20:02:20+00",
			"value": "Does anyone have time to review my delivery service request?",
			"xmlId": null
		}
	}

``PUT``
=======
Updates a delivery service request comment.

:Auth. Required: Yes
:Roles Required: "admin", "Federation", "operations", "Portal", or "Steering"
:Response Type:  Object


Request Structure
-----------------
:deliveryServiceRequestId:      The integral, unique identifier of the :term:`Delivery Service Request` that the comment was posted on.
:value:                         The comment text itself.
:xmlId:                         This can be any string. It is not validated or used, though it is returned in the response.

.. table:: Request Query Parameters

	+-----------+----------+-----------------------------------------------------------------------------------+
	| Parameter | Required | Description                                                                       |
	+===========+==========+===================================================================================+
	| id        | yes      | The integral, unique identifier of the :term:`Delivery Service Request` comment   |
	|           |          | that you wish to update.                                                          |
	+-----------+----------+-----------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	PUT /api/4.0/deliveryservice_request_comments?id=6 HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...
	Content-Length: 166

	{
		"deliveryServiceRequestId": 2,
		"value": "Update: We no longer need this, feel free to reject.\n\nDoes anyone have time to review my delivery service request?"
	}

Response Structure
------------------
:author:                        The username of the user who created the comment.
:authorId:                      The integral, unique identifier of the user who created the comment.
:deliveryServiceRequestId:      The integral, unique identifier of the :term:`Delivery Service Request` that the comment was posted on.
:id:                            The integral, unique identifier of the :term:`DSR` comment.
:lastUpdated:                   The date and time at which the user was last modified, in an ISO-like format
:value:                         The text of the comment that was posted.
:xmlId:                         This is the ``xmlId`` value that you provided in the request.

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 24 Feb 2020 21:05:46 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: RalS34imPw7c42nlnu5eTuv6FSxuGcAvxEdeIyNma1zpE3ZojAMFbhj8qi1s+hOVDYybfFPzMz82c+xc1qrMHg==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 24 Feb 2020 20:05:46 GMT
	Content-Length: 255

	{
		"alerts": [
			{
				"text": "deliveryservice_request_comment was updated.",
				"level": "success"
			}
		],
		"response": {
			"authorId": null,
			"author": null,
			"deliveryServiceRequestId": 2,
			"id": 6,
			"lastUpdated": "2020-02-24 20:05:46+00",
			"value": "Update: We no longer need this, feel free to reject.\n\nDoes anyone have time to review my delivery service request?",
			"xmlId": null
		}
	}

``DELETE``
==========
Deletes a delivery service request comment.

:Auth. Required: Yes
:Roles Required: "admin", "Federation", "operations", "Portal", or "Steering"
:Response Type:  ``undefined``

Request Structure
-----------------

.. table:: Request Query Parameters

	+-----------+----------+-----------------------------------------------------------------------------------+
	| Parameter | Required | Description                                                                       |
	+===========+==========+===================================================================================+
	| id        | yes      | The integral, unique identifier of the :term:`Delivery Service Request` comment   |
	|           |          | that you wish to delete.                                                          |
	+-----------+----------+-----------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	DELETE /api/4.0/deliveryservice_request_comments?id=6 HTTP/1.1
	User-Agent: python-requests/2.22.0
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
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 24 Feb 2020 21:07:40 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: lOpGzqeIh/1JAx85mz3MI/5A1i1g5beTSLtfvgcfQmCjNKQvOMs/4TLviuVzOCRrEIPmNcjy35tmvfxwlv7RMQ==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 24 Feb 2020 20:07:40 GMT
	Content-Length: 101

	{
		"alerts": [
			{
				"text": "deliveryservice_request_comment was deleted.",
				"level": "success"
			}
		]
	}
