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

.. _to-api-stats-summary:

*****************
``stats_summary``
*****************

``GET``
=======
Either retrieve a list of summary stats or the timestamp of the latest recorded stats summary.

What is returned is driven by the query parameter ``lastSummaryDate``.

If the parameter is set it will return an object with the latest timestamp, else an array of summary stats will be returned.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Array or Object

Request Structure
-----------------

Summary Stats
"""""""""""""

.. table:: Request Query Parameters

	+---------------------+----------+-------------------------------------------------------------------------------------------------------+
	| Name                | Required | Description                                                                                           |
	+=====================+==========+=======================================================================================================+
	| deliveryServiceName | no       | Return only summary stats that were reported for :term:`Delivery Service` with the given display name |
	+---------------------+----------+-------------------------------------------------------------------------------------------------------+
	| cdnName             | no       | Return only summary stats that were reported for CDN with the given name                              |
	+---------------------+----------+-------------------------------------------------------------------------------------------------------+
	| statName            | no       | Return only summary stats that were reported for given stat name                                      |
	+---------------------+----------+-------------------------------------------------------------------------------------------------------+
	| orderby             | no       | Choose the ordering of the results - can only be one of deliveryServiceName, statName or cdnName      |
	+---------------------+----------+-------------------------------------------------------------------------------------------------------+
	| sortOrder           | no       | Changes the order of sorting. Either ascending (default or "asc") or                                  |
	|                     |          | descending ("desc")                                                                                   |
	+---------------------+----------+-------------------------------------------------------------------------------------------------------+
	| limit               | no       | Choose the maximum number of results to return                                                        |
	+---------------------+----------+-------------------------------------------------------------------------------------------------------+
	| offset              | no       | The number of results to skip before beginning to return results. Must use in conjunction with limit  |
	+---------------------+----------+-------------------------------------------------------------------------------------------------------+
	| page                | no       | Return the n\ :sup:`th` page of results, where "n" is the value of this parameter, pages are          |
	|                     |          | ``limit`` long and the first page is 1. If ``offset`` was defined, this query parameter has no        |
	|                     |          | effect. ``limit`` must be defined to make use of ``page``.                                            |
	+---------------------+----------+-------------------------------------------------------------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/4.0/stats_summary HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Last Updated Summary Stat
""""""""""""""""""""""""""

.. table:: Request Query Parameters

	+-----------------+----------+---------------------------------------------------+
	| Name            | Required | Description                                       |
	+=================+==========+===================================================+
	| statName        | no       | Get lastest updated date for the given stat       |
	+-----------------+----------+---------------------------------------------------+
	| lastSummaryDate | yes      | Tells route to get only lastest updated timestamp |
	+-----------------+----------+---------------------------------------------------+

.. code-block:: http
	:caption: Request Example

	GET /api/4.0/stats_summary?lastSummaryDate=true HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...

Response Structure
------------------

Summary Stats
"""""""""""""

:cdnName:             The CDN name for which the summary stat was taken for

	.. note:: If the ``cdn`` is equal to ``all`` it represents summary_stats across all delivery services across all CDNs

:deliveryServiceName: The :term:`Delivery Service` display name for which the summary stat was taken for

	.. note:: If the ``deliveryServiceName`` is equal to ``all`` it represents summary_stats across all delivery services within the given CDN

:statName:            Stat name summary stat represents
:statValue:           Summary stat value
:summaryTime:         Timestamp of summary, in an ISO-like format
:statDate:            Date stat was taken, in ``YYYY-MM-DD`` format

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: dHNip9kpTGGS1w39/fWcFehNktgmXZus8XaufnmDpv0PyG/3fK/KfoCO3ZOj9V74/CCffps7doEygWeL/xRtKA==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 10 Dec 2018 20:56:59 GMT
	Content-Length: 150

	{ "response": [
		{
			"cdnName": "CDN-in-a-Box",
			"deliveryServiceName": "all",
			"statName": "daily_maxgbps",
			"statValue": 5,
			"summaryTime": "2019-11-19 00:04:06+00",
			"statDate": "2019-11-19"
		},
		{
			"cdnName": "CDN-in-a-Box",
			"deliveryServiceName": "all",
			"statName": "daily_maxgbps",
			"statValue": 3,
			"summaryTime": "2019-11-18 00:04:06+00",
			"statDate": "2019-11-18"
		},
		{
			"cdnName": "CDN-in-a-Box",
			"deliveryServiceName": "all",
			"statName": "daily_bytesserved",
			"statValue": 1000,
			"summaryTime": "2019-11-19 00:04:06+00",
			"statDate": "2019-11-19"
		}
	]}

Last Updated Summary Stat
"""""""""""""""""""""""""

:summaryTime: Timestamp of the last updated summary, in an ISO-like format

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: dHNip9kpTGGS1w39/fWcFehNktgmXZus8XaufnmDpv0PyG/3fK/KfoCO3ZOj9V74/CCffps7doEygWeL/xRtKA==
	X-Server-Name: traffic_ops_golang/
	Date: Mon, 10 Dec 2018 20:56:59 GMT
	Content-Length: 150

	{ "response": {
		"summaryTime": "2019-11-19 00:04:06+00"
	}}

``POST``
========

Post a stats summary for a given stat.

:Auth. Required: Yes
:Roles Required: None
:Response Type: Object

Request Structure
-----------------
:cdnName:             The CDN name for which the summary stat was taken for

	.. note:: If the ``cdn`` is equal to ``all`` it represents summary_stats across all delivery services across all CDNs

:deliveryServiceName: The :term:`Delivery Service` display name for which the summary stat was taken for

	.. note:: If the ``deliveryServiceName`` is equal to ``all`` it represents summary_stats across all delivery services within the given CDN

:statName:            Stat name summary stat represents
:statValue:           Summary stat value
:summaryTime:         Timestamp of summary, in an ISO-like format
:statDate:            Date stat was taken, in ``YYYY-MM-DD`` format

.. note:: ``statName``, ``statValue`` and ``summaryTime`` are required. If ``cdnName`` and ``deliveryServiceName`` are not given they will default to ``all``.

.. code-block:: http
	:caption: Request Example

	POST /api/4.0/stats_summary HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 113
	Content-Type: application/json

	{
		"cdnName": "CDN-in-a-Box",
		"deliveryServiceName": "all",
		"statName": "daily_maxgbps",
		"statValue": 10,
		"summaryTime": "2019-12-05 00:03:57+00",
		"statDate": "2019-12-05"
	}

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
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: ezxk+iP7o7KE7zpWmGc0j8nz5k+1wAzY0HiNiA2xswTQrt+N+6CgQqUV2r9G1HAsPNr0HF2PhYs/Xr7DrYOw0A==
	X-Server-Name: traffic_ops_golang/
	Date: Thu, 06 Dec 2018 02:14:45 GMT
	Content-Length: 97

	{ "alerts": [
		{
			"text": "Stats Summary was successfully created",
			"level": "success"
		}]
	}
