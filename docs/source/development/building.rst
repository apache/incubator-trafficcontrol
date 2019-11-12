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

.. _dev-building:

************************
Building Traffic Control
************************
The build steps for Traffic Control components are all pretty much the same, despite that they are written in a variety of different languages and frameworks. This is accomplished by using Docker.

.. note:: Currently, both listed methods of building Traffic Control components will produce ``*.rpm`` files, meaning that the support of these components is limited to RedHat-based distributions - and none of them are currently tested (or guaranteed to work) outside of CentOS7, specifically.

.. _dev-building-downloading:

Downloading Traffic Control
===========================
If any local work on Traffic Monitor, Traffic Router Golang, Grove or Traffic Ops is to be done, it is **highly** recommended that `the Traffic Control repository <https://github.com/apache/trafficcontrol>`_ be downloaded inside the ``$GOPATH`` directory. Specifically, the best location is ``$GOPATH/src/github.com/apache/trafficcontrol``. Cloning the repository outside of this location will require either linking the actual directory to that point, or moving/copying it there.

.. seealso:: The Golang project's ``GOPATH`` `wiki page <https://github.com/golang/go/wiki/GOPATH>`_

.. _pkg:

Build Using ``pkg``
===================
This is the easiest way to build all the components of Traffic Control; all requirements are automatically loaded into the image used to build each component.

Requirements
------------
- `Docker <https://docs.docker.com/engine/installation/>`_
- `Docker Compose <https://docs.docker.com/compose/install/>`_\ [1]_


Usage
-----
``./pkg [options] [projects]``

.. note:: The ``pkg`` script often needs to be run as ``sudo``, as certain privileges are required to run Docker containers

Options

-q      Quiet mode. Supresses output.
-v      Verbose mode. Lists all build output.
-l      List available projects.

If present, ``projects`` should be one or more project names. When no specific project or project list is given the default projects will be built. Valid projects:

- docs
- grove_build\ [2]_
- grovetccfg_build
- source\ [2]_
- traffic_monitor_build\ [2]_
- traffic_ops_build\ [2]_
- traffic_portal_build\ [2]_
- traffic_router_build\ [2]_
- traffic_stats_build\ [2]_
- weasel

Output :file:`{component}-{version}.rpm` files, build logs and source tarballs will be output to the ``dist/`` directory at the root of the Traffic Control repository directory.

.. [1] This is optional, but recommended. If a ``docker-compose`` executable is not available the ``pkg`` script will automatically download and run it using a container. This is noticeably slower than running it natively.
.. [2] This is a default project, which will be built if ``pkg`` is run with no ``projects`` argument

.. _build-with-dc:

Build Using ``docker-compose``
==============================
If the ``pkg`` script fails, ``docker-compose`` can still be used to build the projects directly. The compose file can be found at ``infrastructure/docker/build/docker-compose.yml`` under the repository's root directory. It can be passed directly to ``docker-compose``, either from the ``infrastructure/docker/build/`` directory or by explicitly passing a path to the ``infrastructure/docker/build/docker-compose.yml`` file via ``-f``. It is recommended that between builds ``docker-compose down -v`` is run to prevent caching of old build steps. The service names are the same as the project names described above in `Usage`_, and similar to the ``pkg`` script, the build results, logs and source tarballs may all be found in the ``dist`` directory after completion.

.. note:: Calling ``docker-compose`` in the way described above will build _all_ projects, not just the default projects.

.. seealso:: `The Docker Compose command line reference <https://docs.docker.com/compose/reference/overview/>`_

Building Individual Components
==============================
Each Traffic Control component can be individually built, and the instructions for doing so may be found in their respective component's development documentation.

.. _docs-build:

Building This Documentation
---------------------------
This documentation uses the `Sphinx documentation build system <http://www.sphinx-doc.org/en/master/>`_, and as such requires a Python3 version that is at least 3.4.1. It also has dependency on Sphinx, and Sphinx extensions and themes. All of these can be easily installed using `pip` by referencing the requirements file like so:

.. code-block:: shell
	:caption: Run from the Repository's Root Directory

	python3 -m pip install --user -r docs/source/requirements.txt

Once all dependencies have been satisfied, build using the Makefile at ``docs/Makefile``.

Alternatively, it is also possible to :ref:`pkg` or to :ref:`build-with-dc`, both of which will output a documentation "tarball" to ``dist/``.
