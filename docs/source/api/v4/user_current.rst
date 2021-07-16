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

.. _to-api-user-current:

****************
``user/current``
****************

``GET``
=======
.. caution:: As a username is needed to log in, any administrator or application must necessarily know the current username at any given time. Thus it's generally better to use the ``username`` query parameter of a ``GET`` request to :ref:`to-api-users` instead.

Retrieves the details of the authenticated user.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Object

Request Structure
-----------------
No parameters available.

Response Structure
------------------
:addressLine1:      The user's address - including street name and number
:addressLine2:      An additional address field for e.g. apartment number
:city:              The name of the city wherein the user resides
:company:           The name of the company for which the user works
:country:           The name of the country wherein the user resides
:email:             The user's email address
:fullName:          The user's full name, e.g. "John Quincy Adams"
:gid:               A deprecated field only kept for legacy compatibility reasons that used to contain the UNIX group ID of the user
:id:                An integral, unique identifier for this user
:lastAuthenticated: The date and time at which the user was last authenticated, in :rfc:`3339`
:lastUpdated:       The date and time at which the user was last modified, in :ref:`non-rfc-datetime`
:newUser:           A meta field with no apparent purpose that is usually ``null`` unless explicitly set during creation or modification of a user via some API endpoint
:phoneNumber:       The user's phone number
:postalCode:        The postal code of the area in which the user resides
:publicSshKey:      The user's public key used for the SSH protocol
:registrationSent:  If the user was created using the :ref:`to-api-users-register` endpoint, this will be the date and time at which the registration email was sent - otherwise it will be ``null``
:role:              The integral, unique identifier of the highest-privilege :term:`Role` assigned to this user
:rolename:          The name of the highest-privilege :term:`Role` assigned to this user
:stateOrProvince:   The name of the state or province where this user resides
:tenant:            The name of the :term:`Tenant` to which this user belongs
:tenantId:          The integral, unique identifier of the :term:`Tenant` to which this user belongs
:uid:               A deprecated field only kept for legacy compatibility reasons that used to contain the UNIX user ID of the user
:username:          The user's username

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Whole-Content-Sha512: HQwu9FxFyinXSVFK5+wpEhSxU60KbqXuokFbMZ3OoerOoM5ZpWpglsHz7mRch8VAw0dzwsJzpPJivj07RiKaJg==
	X-Server-Name: traffic_ops_golang/
	Date: Thu, 13 Dec 2018 15:14:45 GMT
	Content-Length: 382

	{ "response": {
		"username": "admin",
		"localUser": true,
		"addressLine1": null,
		"addressLine2": null,
		"city": null,
		"company": null,
		"country": null,
		"email": null,
		"fullName": "admin",
		"gid": null,
		"id": 2,
		"newUser": false,
		"phoneNumber": null,
		"postalCode": null,
		"publicSshKey": null,
		"role": 1,
		"rolename": "admin",
		"stateOrProvince": null,
		"tenant": "root",
		"tenantId": 1,
		"uid": null,
		"lastUpdated": "2018-12-12 16:26:32+00",
		"lastAuthenticated": "2021-07-09T14:44:10.371708-06:00"
	}}

``PUT``
=======
.. warning:: Assuming the current user's integral, unique identifier is known, it's generally better to use the ``PUT`` method of the :ref:`to-api-users` instead.

.. warning:: Users that login via LDAP pass-back cannot be modified

Updates the date for the authenticated user.

:Auth. Required: Yes
:Roles Required: None
:Response Type:  Object

Request Structure
-----------------
:user: The entire request must be inside a top-level "user" key for legacy reasons

	:addressLine1:       The user's address - including street name and number
	:addressLine2:       An additional address field for e.g. apartment number
	:city:               The name of the city wherein the user resides
	:company:            The name of the company for which the user works
	:confirmLocalPasswd: An optional 'confirm' field in a new user's password specification. This has no known effect and in fact *doesn't even need to match* ``localPasswd``
	:country:            The name of the country wherein the user resides
	:email:              The user's email address - cannot be an empty string\ [#notnull]_. The given email is validated (circuitously) by `GitHub user asaskevich's regular expression <https://github.com/asaskevich/govalidator/blob/9a090521c4893a35ca9a228628abf8ba93f63108/patterns.go#L7>`_ . Note that it can't actually distinguish a valid, deliverable, email address but merely ensure the email is in a commonly-found format.
	:fullName:           The user's full name, e.g. "John Quincy Adams"
	:gid:                A legacy field only kept for legacy compatibility reasons that used to contain the UNIX group ID of the user - please don't use this
	:id:                 The user's integral, unique, identifier - this cannot be changed\ [#notnull]_
	:localPasswd:        Optionally, the user's password. This should never be given if it will not be changed. An empty string or ``null`` can be used to explicitly specify no change.
	:phoneNumber:        The user's phone number
	:postalCode:         The user's postal code
	:publicSshKey:       The user's public encryption key used for the SSH protocol
	:role:               The integral, unique identifier of the highest permission :term:`Role` which will be permitted to the user - this cannot be altered from the user's current :term:`Role`\ [#notnull]_
	:stateOrProvince:    The state or province in which the user resides
	:tenantId:           The integral, unique identifier of the :term:`Tenant` to which the new user shall belong\ [#tenancy]_\ [#notnull]_
	:uid:                A legacy field only kept for legacy compatibility reasons that used to contain the UNIX user ID of the user - please don't use this
	:username:           The user's new username\ [#notnull]_

.. code-block:: http
	:caption: Request Example

	PUT /api/4.0/user/current HTTP/1.1
	Host: trafficops.infra.ciab.test
	User-Agent: curl/7.47.0
	Accept: */*
	Cookie: mojolicious=...
	Content-Length: 465
	Content-Type: application/json

	{ "user": {
		"addressLine1": null,
		"addressLine2": null,
		"city": null,
		"company": null,
		"country": null,
		"email": "admin@infra.trafficops.ciab.test",
		"fullName": "admin",
		"gid": null,
		"id": 2,
		"phoneNumber": null,
		"postalCode": null,
		"publicSshKey": null,
		"role": 1,
		"stateOrProvince": null,
		"tenantId": 1,
		"uid": null,
		"username": "admin"
	}}

Response Structure
------------------
:addressLine1:     The user's address - including street name and number
:addressLine2:     An additional address field for e.g. apartment number
:city:             The name of the city wherein the user resides
:company:          The name of the company for which the user works
:country:          The name of the country wherein the user resides
:email:            The user's email address validated (circuitously) by `GitHub user asaskevich's regular expression <https://github.com/asaskevich/govalidator/blob/9a090521c4893a35ca9a228628abf8ba93f63108/patterns.go#L7>`_ . Note that it can't actually distinguish a valid, deliverable, email address but merely ensure the email is in a commonly-found format.
:fullName:         The user's full name, e.g. "John Quincy Adams"
:gid:              A legacy field only kept for legacy compatibility reasons that used to contain the UNIX group ID of the user
:id:               An integral, unique identifier for this user
:lastUpdated:      The date and time at which the user was last modified, in :ref:`non-rfc-datetime`
:newUser:          A meta field with no apparent purpose
:phoneNumber:      The user's phone number
:postalCode:       The postal code of the area in which the user resides
:publicSshKey:     The user's public key used for the SSH protocol
:registrationSent: If the user was created using the :ref:`to-api-users-register` endpoint, this will be the date and time at which the registration email was sent - otherwise it will be ``null``
:role:             The integral, unique identifier of the highest-privilege :term:`Role` assigned to this user
:rolename:         The name of the highest-privilege :term:`Role` assigned to this user
:stateOrProvince:  The name of the state or province where this user resides
:tenant:           The name of the :term:`Tenant` to which this user belongs
:tenantId:         The integral, unique identifier of the :term:`Tenant` to which this user belongs
:uid:              A legacy field only kept for legacy compatibility reasons that used to contain the UNIX user ID of the user
:username:         The user's username

.. code-block:: http
	:caption: Response Example

	HTTP/1.1 200 OK
	Access-Control-Allow-Credentials: true
	Access-Control-Allow-Headers: Origin, X-Requested-With, Content-Type, Accept, Set-Cookie, Cookie
	Access-Control-Allow-Methods: POST,GET,OPTIONS,PUT,DELETE
	Access-Control-Allow-Origin: *
	Content-Type: application/json
	Date: Thu, 13 Dec 2018 21:05:49 GMT
	X-Server-Name: traffic_ops_golang/
	Set-Cookie: mojolicious=...; Path=/; Expires=Mon, 18 Nov 2019 17:40:54 GMT; Max-Age=3600; HttpOnly
	Vary: Accept-Encoding
	Whole-Content-Sha512: sHFqZQ4Cv7IIWaIejoAvM2Fr/HSupcX3D16KU/etjw+4jcK9EME3Bq5ohLC+eQ52BDCKW2Ra+AC3TfFtworJww==
	Content-Length: 478

	{ "alerts": [
		{
			"text": "User profile was successfully updated",
			"level": "success"
		}
	],
	"response": {
		"addressLine1": null,
		"addressLine2": null,
		"city": null,
		"company": null,
		"country": null,
		"email": "admin@infra.trafficops.ciab.test",
		"fullName": null,
		"gid": null,
		"id": 2,
		"lastUpdated": "2019-10-08 20:14:25+00",
		"newUser": false,
		"phoneNumber": null,
		"postalCode": null,
		"publicSshKey": null,
		"registrationSent": null,
		"role": 1,
		"roleName": "admin",
		"stateOrProvince": null,
		"tenant": "root",
		"tenantId": 1,
		"uid": null,
		"username": "admin"
	}}

.. [#notnull] This field cannot be ``null``.
.. [#tenancy] This endpoint respects tenancy; a user cannot assign itself to a :term:`Tenant` that is not the same :term:`Tenant` to which it was previously assigned or a descendant thereof.
