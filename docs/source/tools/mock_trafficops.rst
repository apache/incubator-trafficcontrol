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

.. _mock-to:

***************************
The Mock Traffic Ops Server
***************************
.. versionadded:: 4.0.0

.. warning:: At the time of this writing, the Traffic Ops Mock Server is still under active development. As such, it should not be considered stable and/or built into critical procedures until it has reached maturity.

Developers who write clients that consume the :ref:`to-api` may want to write tests for said clients. Currently, this is most typically done by defining a static "data set" for Traffic Ops and loading it into the Traffic Ops Database. This method has two flaws:

* It still requires an authentication handshake at startup to test any given route (though in most cases testing that would also be desired).
* The :ref:`to-api` produces non-deterministic output, especially for configuration files\ [#configfiles_issue]_. Because of this, no two requests to the same endpoint with the same data in the Traffic Ops Database can be guaranteed to produce the same output.

If these factors are important to the tests for a client of the :ref:`to-api`, it is suggested that they instead utilize the Traffic Ops Mock server. The source code for this binary can be found in :file:`traffic_ops/client_tests`.

Behavior
========
The Traffic Ops mock server **SHALL**

* Serve all of the API routes supported by the :ref:`to-api`\ [#implemented_routes_disclaimer]_
* Serve static responses that never change in structure or content
* Identify itself through the ``Server`` HTTP header as ``Traffic Ops/{{version}} (Mock)``
* Support all methods supported by the :ref:`to-api` (the behavior of the server when the client uses erroneous request methods, request paths or request parameters is not defined)
* Faithfully reproduce syntactically valid - and self-consistent - responses of the real Traffic Ops

The Traffic Ops mock server **SHALL NOT**

* Require proper authentication, despite that a real server would (because testing one route shouldn't have extraneous dependencies)
* Modify any of its pre-determined responses during runtime. This means that a client may attempt to e.g. create a server object by submitting a ``POST`` request to :ref:`to-api-servers` - which should receive a response indicating the operation was successful (assuming it was syntactically valid and consistent with the static data set) - the new server object will not appear in the response of a subsequent ``GET`` request to the same endpoint

Building
========
.. note:: The Traffic Ops Mock Server has no prerequisites beyond those required by the Traffic Control Go client, and has the same Go language version requirement as that client library.

To build the Traffic Ops Mock Server, first follow the instructions in :ref:`dev-building-downloading` to properly clone the repository for Go development. It is then suggested that the server be built to produce a binary with the name "mock" to distinguish it from any other/future client testing tools.

.. code-block:: bash
	:caption: Suggested Build Example

	# Assuming we start from the repository root
	pushd traffic_ops/client_tests
	go build -o mock .
	popd

.. program:: mock

Usage
=====
``mock [-v] [-h] [-c CERT_PATH] [-k KEY_PATH] [-l LISTEN_ADDR] [-p PORT]``

.. note:: This assumes that in the `Building`_ step, the output binary was explicitly named :program:`mock`. If this was not done, the default name that Go will choose is ``client_tests``.

.. option:: -c CERT_PATH, --cert-path CERT_PATH

	Specify a path to an SSL certificate for the :program:`mock` server to use (Default: :file:`./localhost.crt`)

	.. note:: :program:`mock` **only** serves HTTPS traffic, and as such there MUST be both a certificate and a key, either generated to match the default paths or specified using :option:`-c` and :option:`-k`.

.. option:: -h, --help

	Print usage information and exit

.. option:: -k KEY_PATH, --key-path KEY_PATH

	Specify a path to an SSL private key for the :program:`mock` server to use (Default: :file:`./localhost.key`)

	.. note:: :program:`mock` **only** serves HTTPS traffic, and as such there MUST be both a certificate and a key, either generated to match the default paths or specified using :option:`-c` and :option:`-k`.

.. option:: -l LISTEN_ADDR, --listen LISTEN_ADDR

	Choose the address or hostname on which the :program:`mock` server will listen (Default: '' i.e. "all")

.. option:: -p PORT, --port PORT

	Choose the port on which the :program:`mock` server will listen for connections (Default: 443)

.. option:: -v, --version

	Print version information and exit


.. [#configfiles_issue] Refer to `GitHub Issue #3106 <https://github.com/apache/trafficcontrol/issues/3106>`_
.. [#implemented_routes_disclaimer] At the time of this writing, the vast majority of :ref:`to-api` routes have not been mocked, and even those that have typically support an extremely limited set of functionality.
