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

.. _env-creation:

********************
Environment Creation
********************

Apache Traffic Control is a complex set of components to construct a functional CDN.
To help users get started more quickly, two paths exist to help based on the audience and needs.
These tools are optional and administrators may use whatever means are at their disposal to perform their deployments.

CDN-in-a-Box (CIAB)
===================
Audience for CIAB
-----------------
* Developers who need a local full-stack to aid in development using Docker
* A common platform for developers to write tests against and executed by a CI system
* New users who are learning how a CDN using Apache Traffic Control works
* Functional demonstrations

.. toctree::
	:maxdepth: 2
	:caption: Read more

	quick_howto/ciab.rst

Ansible-based Lab Deployment
============================
Audience for Ansible-based Lab Deployment
-----------------------------------------
* DevOps engineers building more production-like lab environments
* DevOps engineers building an initial greenfield production environment
* Developers who need test environments that cannot be modeled on local resources

.. toctree::
	:maxdepth: 4
	:caption: Read more
	:glob:

	ansible-labs/*
