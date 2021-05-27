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


.. _to-api-cdn-notifications:

*********************
``cdn_notifications``
*********************

``GET``
=======
List CDN notifications.

:Auth. Required: Yes
:Roles Required: Read-Only
:Response Type: Array

Request Structure
-----------------
.. table:: Request Query Parameters

	+------------+----------+-----------------------------------------------------------------------------------------------------+
	| Parameter  | Required | Description                                                                                         |
	+============+==========+=====================================================================================================+
	| cdn        | no       | The CDN name of the notifications you wish to retrieve.                                             |
	+------------+----------+-----------------------------------------------------------------------------------------------------+
	| id         | no       | The integral, unique identifier of the notification you wish to retrieve.                           |
	+------------+----------+-----------------------------------------------------------------------------------------------------+
	| user       | no       | The username of the user responsible for creating the CDN notifications.                            |
	+------------+----------+-----------------------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/4.0/cdn_notifications HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...

Response Structure
------------------
:id:           The integral, unique identifier of the notification
:cdn:          The name of the CDN to which the notification belongs to
:lastUpdated:  The time and date this server entry was last updated in an ISO-like format
:notification: The content of the notification
:user:         The user responsible for creating the notification

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 02 Dec 2019 22:51:14 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: F2NmDbTpXqrIQDX7IBKH9+1drtTL4XedSfJv6klMgLEZwbLCkddIXuSLpmgVCID6kTVqy3fTKjZS3U+HJ3YUEQ==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 02 Dec 2019 21:51:14 GMT
	Content-Length: 128

	{ "response": [
		{
			"id": 42,
			"cdn": "cdn1",
			"lastUpdated": "2019-12-02 21:49:08+00",
			"notification": "the content of the notification",
			"user": "username123",
		}
	]}

``POST``
========
Creates a notification for a specific CDN.

.. note:: Currently only one notification per CDN is supported.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type: Object

Request Structure
-----------------
:cdn:          The name of the CDN to which the notification shall belong
:notification: The content of the notification

.. code-block:: http
	:caption: Request Example

	POST /api/4.0/cdn_notifications HTTP/1.1
	User-Agent: python-requests/2.22.0
	Accept-Encoding: gzip, deflate
	Accept: */*
	Connection: keep-alive
	Cookie: mojolicious=...
	Content-Length: 29

	{"cdn": "cdn1", "notification": "the content of the notification"}


Response Structure
------------------
:id:           The integral, unique identifier of the notification
:cdn:          The name of the CDN to which the notification belongs to
:lastUpdated:  The time and date this server entry was last updated in an ISO-like format
:notification: The content of the notification
:user:         The user responsible for creating the notification

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Encoding: gzip
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 02 Dec 2019 22:49:08 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: mx8b2GTYojz4QtMxXCMoQyZogCB504vs0yv6WGly4dwM81W3XiejWNuUwchRBYYi8QHaWsMZ3DaiGGfQi/8Giw==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 02 Dec 2019 21:49:08 GMT
	Content-Length: 150

	{
	"alerts":
		[
			{
				"text": "notification was created.",
				"level": "success"
			}
		],
	"response":
		{
			"id": 42,
			"cdn": "cdn1",
			"lastUpdated": "2019-12-02 21:49:08+00",
			"notification": "the content of the notification",
			"user": "username123",
		}
	}

``DELETE``
----------
Deletes an existing CDN notification.

:Auth. Required: Yes
:Roles Required: "admin" or "operations"
:Response Type: ``undefined``

Request Structure
-----------------
.. table:: Request Query Parameters

	+------------+----------+-----------------------------------------------------------------------------------------------------+
	| Parameter  | Required | Description                                                                                         |
	+============+==========+=====================================================================================================+
	| id         | yes      | The integral, unique identifier of the notification you wish to delete.                             |
	+------------+----------+-----------------------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	DELETE /api/4.0/cdn_notifications?id=42 HTTP/1.1
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
	Set-Cookie: mojolicious=...; Path=/; Expires=Tue, 25 Feb 2020 08:27:33 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: Woz8NSHIYVpX4V5X4xZWZIX1hvGL2uian7nUhjZ8F23Nb9RWQRMIg/cc+1vXEzkT/ehKV9t11FKRLX+avSae0g==
	X-Server-Name: traffic_ops_golang/
	Date: Tue, 25 Feb 2020 07:27:33 GMT
	Content-Length: 83

	{
		"alerts": [
			{
				"text": "notification was deleted.",
				"level": "success"
			}
		]
	}
