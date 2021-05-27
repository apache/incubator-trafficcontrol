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
.. _delivery-service-requests:

*************************
Delivery Service Requests
*************************
A :abbr:`DSR (Delivery Service Request)` is a request to create a new :term:`Delivery Service`, delete an existing :term:`Delivery Service`, or modify an existing :term:`Delivery Service`. The model for a :abbr:`DSR (Delivery Service Request)` is, therefore, somewhat nebulous as it depends on the model of a :term:`Delivery Service`. This concept is not to be confused with :ref:`to-api-deliveryservices-request`.

.. seealso:: :ref:`ds_requests` for information about how to use :abbr:`DSR (Delivery Service Request)`\ s in

.. seealso:: The API reference for Delivery Service-related endpoints such as :ref:`to-api-deliveryservice-requests` contains definitions of the Delivery Service object(s) returned and/or accepted by those endpoints.

.. seealso:: The :atc-godoc:`lib/go-tc.DeliveryServiceRequestV30` Go structure documentation.

The model for a :abbr:`DSR (Delivery Service Request)` in the most recent version of the :ref:`to-api` is given in :ref:`dsr-interface` as a Typescript interface.

.. _dsr-interface:

.. code-block:: typescript
	:caption: Delivery Service Request as a Typescript Interface

	interface DeliveryServiceRequest {
		assignee: string | null;
		author: string;
		changeType: 'create' | 'delete' | 'update';
		createdAt: Date; // RFC3339 string - response-only field
		id?: number; // response-only field
		lastEditedBy: string; // response-only field
		lastUpdated: Date; // RFC3339 string - response-only field
		original?: DeliveryService;
		requested?: DeliveryService;
		status: 'draft' | 'pending' | 'submitted' | 'rejected' | 'complete';
	}
	// Specifically, every DSR will be one of the following more "concrete" types.
	interface CreateDSR extends DeliveryServiceRequest {
		changeType: 'create';
		original: undefined;
		requested: DeliveryService;
	}
	interface DeleteDSR extends DeliveryServiceRequest {
		changeType: 'delete';
		original: DeliveryService;
		requested: undefined;
	}
	interface CreateDSR extends DeliveryServiceRequest {
		changeType: 'update';
		original: DeliveryService;
		requested: DeliveryService;
	}
.. _dsr-assignee:

Assignee
--------
Assignee is the username of the user to whom the :abbr:`DSR (Delivery Service Request)` is assigned. It may be null-typed if there is no assignee for a given :abbr:`DSR (Delivery Service Request)`.

.. table:: Aliases/Synonyms

	+------------+--------------------------------------------------------+------------------+
	| Name       | Use(s)                                                 | Type             |
	+============+========================================================+==================+
	| assigneeId | older API versions, internally in Traffic Control code | unsigned integer |
	+------------+--------------------------------------------------------+------------------+

Author
------
Author is the username of the user who created the :abbr:`DSR (Delivery Service Request)`.

.. table:: Aliases/Synonyms

	+----------+--------------------------------------------------------+------------------+
	| Name     | Use(s)                                                 | Type             |
	+==========+========================================================+==================+
	| authorId | older API versions, internally in Traffic Control code | unsigned integer |
	+----------+--------------------------------------------------------+------------------+

Change Type
-----------
This string indicates the action that will be taken in the event that the :abbr:`DSR (Delivery Service Request)` is fulfilled. It can be one of the following values:

create
	A new :term:`Delivery Service` will be created
delete
	An existing :term:`Delivery Service` will be deleted
update
	An existing :term:`Delivery Service` will be modified

Created At
----------
This is the date and time at which the :abbr:`DSR (Delivery Service Request)` was created. In the context of the :ref:`to-api`, it is formatted as an :rfc:`3339` date string except where otherwise noted.

ID
--
An integral, unique identifier for the :abbr:`DSR (Delivery Service Request)`.

Last Edited By
--------------
This is the username of the user by whom the :abbr:`DSR (Delivery Service Request)` was last edited.
Author is the username of the user who created the :abbr:`DSR (Delivery Service Request)`.

.. table:: Aliases/Synonyms

	+----------------+--------------------------------------------------------+------------------+
	| Name           | Use(s)                                                 | Type             |
	+================+========================================================+==================+
	| lastEditedById | older API versions, internally in Traffic Control code | unsigned integer |
	+----------------+--------------------------------------------------------+------------------+

Original
--------
If this property of a :abbr:`DSR (Delivery Service Request)` exists, then it represents the original :term:`Delivery Service` before the :abbr:`DSR (Delivery Service Request)` was/would have been/is fulfilled. This property only exists on :abbr:`DSR (Delivery Service Request)`\ s that have a `Change Type`_ of "update" or "delete". This is a full representation of a :term:`Delivery Service`, and so in the context of :ref:`to-api` has the same structure as a request to or response from the :ref:`to-api-deliveryservices` endpoint, as appropriate for its `Change Type`_.

.. table:: Aliases/Synonyms

	+-----------------+--------------------------------------------------------------------------------------------+-----------------------------------------------------+
	| Name            | Use(s)                                                                                     | Type                                                |
	+=================+============================================================================================+=====================================================+
	| deliveryservice | older API versions combined the concepts of Original and Requested_ into this single field | unchanged (:term:`Delivery Service` representation) |
	+-----------------+--------------------------------------------------------------------------------------------+-----------------------------------------------------+

Requested
---------
If this property of a :abbr:`DSR (Delivery Service Request)` exists, then it is represents the :term:`Delivery Service` the creator wishes to exist - possibly in place of an existing :term:`Delivery Service` that shares its identifying properties. This property only exists on :abbr:`DSR (Delivery Service Request)`\ s that have a `Change Type`_ of "update" or "create". This is a full representation of a :term:`Delivery Service`, and so in the context of :ref:`to-api` has the same structure as a request to or response from the :ref:`to-api-deliveryservices` endpoint, as appropriate for its `Change Type`_.

.. table:: Aliases/Synonyms

	+-----------------+--------------------------------------------------------------------------------------------+-----------------------------------------------------+
	| Name            | Use(s)                                                                                     | Type                                                |
	+=================+============================================================================================+=====================================================+
	| deliveryservice | older API versions combined the concepts of Original_ and Requested into this single field | unchanged (:term:`Delivery Service` representation) |
	+-----------------+--------------------------------------------------------------------------------------------+-----------------------------------------------------+

.. _dsr-status:

Status
------
Status is a string that indicates the point in the :abbr:`DSR (Delivery Service Request)` workflow lifecycle at which a given :abbr:`DSR (Delivery Service Request)` is. Generally a :abbr:`DSR (Delivery Service Request)` may be either "open" - meaning that it is available to be modified, reviewed, and possibly either completed or rejected - or "closed" - meaning that it has been completed or rejected. More specifically, "open" :abbr:`DSR (Delivery Service Request)`\ s have one of the following Statuses:

draft
	The :abbr:`DSR (Delivery Service Request)` is not yet ready for completion or review that might result in rejection, as it is still being actively worked on.
submitted
	The :abbr:`DSR (Delivery Service Request)` has been submitted for review, but has not yet been reviewed.

... while a "closed" :abbr:`DSR (Delivery Service Request)` has one of these Statuses:

complete
	The :abbr:`DSR (Delivery Service Request)` was approved and its declared action was taken.
pending
	The :abbr:`DSR (Delivery Service Request)` was approved and the changes are applied, but the new configuration is not yet disseminated to other :abbr:`ATC (Apache Traffic Control)` components - usually meaning that it cannot be considered truly complete until a :term:`Snapshot` is taken or a :term:`Queue Updates` performed.
rejected
	The :abbr:`DSR (Delivery Service Request)` was rejected and closed; it cannot be completed.

A "closed" :abbr:`DSR (Delivery Service Request)` cannot be edited - except to change a "pending" Status to "complete" or "rejected".
