#! /usr/bin/env python

# -*- coding: utf-8 -*-

#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

"""
Module to help create/retrieve/update/delete data from/to the Traffic Ops API.

Requires Python Version >= 2.7 or >= 3.6
"""

# Core Modules
import logging
import sys

# Third-party Modules
import munch

import requests.exceptions as rex

# Local Modules
import common.restapi as restapi
import common.utils as utils


logger = logging.getLogger(__name__)

__all__ = [u'default_headers', u'TOSession']

# Miscellaneous Constants and/or Variables
default_headers = {u'Content-Type': u'application/json; charset=UTF-8'}


# TOSession Class
class TOSession(restapi.RestApiSession):
	"""
	Traffic Ops Session Class
	Once you login to the Traffic Ops API via the 'login' method, you can call one or more of the methods to retrieve,
	post, put, delete, etc. data to the API.  If you are not logged in, an exception will be thrown if you try
	to call any of the endpoint methods. e.g. get_servers, get_cachegroups, etc.

	This API client is simplistic and lightly structured on purpose but adding support for new end-points
	routinely takes seconds.  Another nice bit of convenience that result data is, by default, wrapped in
	munch.Munch objects, which provide attribute access to the returned dictionaries/hashes.

		e.g. "a_dict['a_key']" with munch becomes "a_dict.a_key" or "a_dict['a_key']"
			 "a_dict['a_key']['b_key']" with munch becomes "a_dict.a_key.b_key" or "a_dict['a_key']['b_key']"

	Also, the lack of rigid structure (loose coupling) means many changes to the Traffic Ops API,
	as it evolves, will probably go un-noticed (usually additions), which means fewer
	future problems to potentially fix in user applications.

	An area of improvement for later is defining classes to represent request data instead
	of loading up dictionaries for request data.

	As of now you can see the following URL for API details:
	   https://traffic-control-cdn.readthedocs.io/en/latest/api/index.html #api for details

	Adding end-point methods: (See "Implemented Direct API URL Endpoint Methods" for actual examples)
		E.g. End-point with no URL parameters and no query parameters:
			given end-point URL: GET api/1.2/cdns
				@restapi.api_request(u'get', u'cdns', (u'1.1', u'1.2',))
				def get_cdns(self):
					pass

		E.g. End-point with URL parameters and no query parameters:
			given end-point URL: GET api/1.2/cdns/{cdn_id:d}
				 @restapi.api_request(u'get', u'cdns/{cdn_id:d}', (u'1.1', u'1.2',))
				 def get_cdn_by_id(self, cdn_id=None):
					 pass

		E.g. End-point with no URL parameters but with query parameters:
			given end-point URL: GET api/1.2/deliveryservices
				 @restapi.api_request(u'get', u'deliveryservices', (u'1.1', u'1.2',))
				 def get_deliveryservices(self, query_params=None):
					 pass

		E.g. End-point with URL parameters and query parameters:
			given end-point URL: GET api/1.2/deliveryservices/xmlId/{xml_id}/sslkeys
				 @restapi.api_request(u'get', u'deliveryservices/xmlId/{xml_id}/sslkeys', (u'1.1', u'1.2',))
				 def get_deliveryservice_ssl_keys_by_xml_id(self, xml_id=None, query_params=None):
					 pass

		E.g. End-point with request data:
			given end-point URL: POST api/1.2/cdns
				 @restapi.api_request(u'post', u'cdns', (u'1.1', u'1.2',))
				 def create_cdn(self, data=None):
					 pass

		E.g. End-point with URL parameters and request data:
			given end-point URL: PUT api/1.2/cdns/{cdn_id:d}
				 @restapi.api_request(u'put', u'cdns', (u'1.1', u'1.2',))
				 def update_cdn_by_id(self, cdn_id=None, data=None):
					 pass

	Calling end-point methods:

		E.g. Using no URL parameters and no query parameters:
			given end-point URL: GET api/1.2/cdns
			get_cdns() -> calls end-point: GET api/1.2/cdns

		E.g. Using no URL parameters but with query parameters:
			given end-point URL: GET api/1.2/types
			get_types(query_params={'useInTable': 'servers'}) -> calls end-point: GET api/1.2/types?useInTable=servers

		E.g. Using URL parameters and query parameters:
		   given end-point URL: GET api/1.2/foo/{id}
		   get_foo_data(id=45, query_params={'sort': 'asc'}) -> calls end-point: GET api/1.2/foo/45?sort=asc

		E.g. Using with required request data:
			given end-point URL: POST api/1.2/cdns/{id:d}/queue_update
			cdns_queue_update(...) -> calls end-point -> POST api/1.2/cdns/{id:d}/queue_update
			cdns_queue_update(id=1, data={'action': 'queue'}) -> calls end-point: POST api/1.2/cdns/1/queue_update
			   with json data '{"action": "queue"}'.

		   So,

		   dict_request = {'action': 'queue'}

		   or

		   Example with a namedtuple:
			   import collections
			   QueueUpdateRequest = collections.namedtuple('QueueUpdateRequest', ['action'])
			   request = QueueUpdateRequest(action='update')

		   Then:
			   cdns_queue_update(id=1, data=vars(request))     # Python 2.x
			   cdns_queue_update(id=1, data=request.asdict())  # Python 3.x
			   cdns_queue_update(id=1, data=dict_request)      # Python 2.x/3.x

		   NOTE: var(request)/request.asdict() transforms the namedtuple into a dictionary which is required
				 by the 'data' argument.

	NOTE: Only a small subset of the API endpoints are implemented.  More can be implemented as needed.
		  See the Traffic Ops API documentation for more detail:
					 https://traffic-control-cdn.readthedocs.io/en/latest/api/index.html #api for details #api
	"""

	def __init__(self, host_ip, host_port=443, api_version=u'1.3', ssl=True, headers=default_headers,
				 verify_cert=True):
		"""
		The class initializer.
		:param host_ip: The dns name or ip address of the Traffic Ops host to use to talk to the API
		:type host_ip: Text
		:param host_port: The port to use when contacting the Traffic Ops API
		:type host_port: int
		:param api_version: The version of the API to use when calling end-points on the Traffic Ops API
		:type api_version: Text
		:param ssl: Should ssl be used? http vs. https
		:type ssl: bool
		:param headers:  The http headers to use when contacting the Traffic Ops API
		:type headers: Dict[Text, Text]
		:param verify_cert: Should the ssl certificates be verified when contacting the Traffic Ops API.
							You may want to set this to False for systems with self-signed certificates.
		:type verify_cert: bool
		"""
		super(TOSession, self).__init__(host_ip=host_ip, api_version=api_version,
										api_base_path=u'api/{api_version}/',
										host_port=host_port, ssl=ssl, headers=headers, verify_cert=verify_cert)

		self._logged_in = False

		msg = u'TOSession instance {0:#0x} initialized: Details: {1}'
		utils.log_with_debug_info(logging.DEBUG, msg.format(id(self), self.__dict__))

	def login(self, username, password):
		"""
		Login to the Traffic Ops API.
		:param username: Traffic Ops User Name
		:type username: Text
		:param password: Traffic Ops User Password
		:type password: Text
		:return: None
		:rtype: None
		:raises: trafficops.restapi.LoginError
		"""
		logging.info("Connecting to Traffic Ops at %s...", self.to_url)

		if not self.is_open:
			self.create()

		logging.info("Connected. Authenticating...")

		self._logged_in = False
		try:
			# Try to login to Traffic Ops
			self.post(u'user/login', data={u'u': username, u'p': password})
			self._logged_in = True
		except rex.SSLError as e:
			logging.debug("%s", e, stack_info=True, exc_info=True)
			self.close()
			msg = (u'{0}.  This system may have a self-signed certificate.  Try creating this TOSession '
				   u'object passing verify_cert=False. e.g. TOSession(..., verify_cert=False). ')
			msg = msg.format(e)
			logging.error(msg)
			logging.warning("disabling certificate verification is not recommended.")
			raise restapi.LoginError(msg) from e
		except restapi.OperationError as e:
			logging.debug("%s", e, exc_info=True, stack_info=True)
			msg = u'Logging in to Traffic Ops has failed. Reason: {0}'.format(e)
			self.close()
			logging.error(msg)
			raise restapi.OperationError(msg) from e

		logging.info("Authenticated.")

	@property
	def to_url(self):
		"""
		The URL without the api portion. (read-only)
		:return: The url should be in the format of
				 '<protocol>://<hostname>[:<port>]'; [] = optional
				 e.g https://to.somedomain.net or https://to.somedomain.net:443
		:rtype: Text
		"""

		return self.server_url

	@property
	def base_url(self):
		"""
		Returns the base url. (read-only)
		:return: The base url should be in the format of
				 '<protocol>://<hostname>[:<port>]/api/<api version>/'; [] = optional
				 e.g https://to.somedomain.net/api/1.2/
		:rtype: Text
		"""

		return self._api_base_url

	@property
	def logged_in(self):
		"""
		Read-only property of boolean to determine if user is logged in to Traffic Ops. (read-only)
		:return: boolean if logged in or not.
		:rtype: bool
		"""

		return self.is_open and self._logged_in

	# Programmatic Endpoint Methods - These can be created when you need to employ "creative
	# methods" to form a correlated composite data set from one or more Traffic Ops API call(s) or
	# employ composite operations against the API.
	# Also, if the API requires you to retrieve the data via paging, these types of methods can be
	# useful to perform that type of work too.
	# These methods need to support similar method signatures as employed by the restapi.api_request decorator
	# method_name argument.
	def get_all_deliveryservice_servers(self, *args, **kwargs):
		"""
		Get all servers attached to all delivery services via the Traffic Ops API.
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""
		result_set = []
		response = None
		limit = 10000
		page = 1

		munchify = True  # Default to True
		if u'munchify' in kwargs:
			munchify = kwargs[u'munchify']

		while True:
			data, response = self.get_deliveryserviceserver(query_params={u'limit': limit, u'page': page},
															munchify=munchify, *args, **kwargs)

			if not data:
				break

			result_set.extend(munch.munchify(data) if munchify else data)
			page += 1

		return result_set, response  # Note: Return last response object received

# 
# PUT ALL API DEFINITIONS BELOW AND UNDER ITS RESPECTIVE PAGE (whether it is 1.2 or 1.3, etc, if its a CDN put it under CDN header and corresponding calls)
#

	#
	#	API Capabilities
	#
	@restapi.api_request(u'get', u'api_capabilities', (u'1.2', u'1.3',))
	def get_api_capabilities(self, query_params=None):
		"""
		Get all API-capability mappings
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/api_capability.html#api-capabilities
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'api_capabilities/{id}', (u'1.2', u'1.3',))
	def get_api_capabilities_by_id(self, id=None):
		"""
		Get an API-capability mapping by ID
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/api_capability.html#api-capabilities
		:param id: The api-capabilities Id
		:type id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# ASN
	#

	# Implemented Direct API URL Endpoint Methods
	# See https://traffic-control-cdn.readthedocs.io/en/latest/api/index.html #api for detail
	@restapi.api_request(u'get', u'asns', (u'1.1', u'1.2', u'1.3',))
	def get_asns(self, query_params=None):
		"""
		Get ASNs.
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""


	#
	# Cache
	#

	@restapi.api_request(u'get', u'caches/stats', (u'1.1', u'1.2', u'1.3',))
	def get_cache_stats(self):
		"""
		Retrieves cache stats from Traffic Monitor. Also includes rows for aggregates
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cache.html#cache
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Cache Group
	#

	@restapi.api_request(u'get', u'cachegroups', (u'1.1', u'1.2', u'1.3',))
	def get_cachegroups(self, query_params=None):
		"""
		Get Cache Groups.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup.html#cache-group
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	# Example of URL parameter substitution via call parameter. You will need to pass the parameter
	# value as a keyword parameter with the proper type to match the str.format specification,
	# e.g. 'cachegroups/{cache_group_id:d}'.  In this case, ':d' specifies a decimal integer.  A specification
	# of 'cachegroups/{cache_group_id}' will try to convert any value passed to a string, which basically does
	# no type checking, unless of course the value cannot be cast to a string.
	# E.g. get_cachegroups_by_id(cache_group_id=23) -> call end-point .../api/1.2/cachegroups/23
	@restapi.api_request(u'get', u'cachegroups/{cache_group_id:d}', (u'1.1', u'1.2', u'1.3',))
	def get_cachegroup_by_id(self, cache_group_id=None):
		"""
		Get a Cache Group by Id.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup.html#cache-group
		:param cache_group_id: The cache group Id
		:type cache_group_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'cachegroups/{cache_group_id:d}/parameters', (u'1.1', u'1.2', u'1.3',))
	def get_cachegroup_parameters(self, cache_group_id=None):
		"""
		Get a cache groups parameters
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup.html#cache-group
		:param cache_group_id: The cache group Id
		:type cache_group_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'cachegroups/{cache_group_id:d}/unassigned_parameters', (u'1.1', u'1.2', u'1.3',))
	def get_cachegroup_unassigned_parameters(self, cache_group_id=None):
		"""
		Get a cache groups unassigned parameters
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup.html#cache-group
		:param cache_group_id: The cache group Id
		:type cache_group_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'cachegroup/{parameter_id:d}/parameter', (u'1.1', u'1.2', u'1.3',))
	def get_cachegroup_parameters_by_id(self, parameter_id=None):
		"""
		Get a cache groups parameter by its ID
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup.html#cache-group
		:param parameter_id: The parameter Id
		:type parameter_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'cachegroupparameters', (u'1.1', u'1.2', u'1.3',))
	def get_all_cachegroup_parameters(self):
		"""
		A collection of all cache group parameters.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup.html#cache-group
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'cachegroups', (u'1.1', u'1.2', u'1.3',))
	def create_cachegroups(self, data=None):
		"""
		Create a Cache Group
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup.html#cache-group
		:param data: The parameter data to use for cachegroup creation.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'cachegroups/{cache_group_id:d}', (u'1.1', u'1.2', u'1.3',))
	def update_cachegroups(self, cache_group_id=None, data=None):
		"""
		Update a cache group
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup.html#cache-group
		:param cache_group_id: The cache group id to update
		:type cache_group_id: Integer
		:param data: The parameter data to use for cachegroup creation.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'cachegroups/{cache_group_id:d}', (u'1.1', u'1.2', u'1.3',))
	def delete_cachegroups(self, cache_group_id=None):
		"""
		Delete a cache group
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup.html#cache-group
		:param cache_group_id: The cache group id to update
		:type cache_group_id: Integer
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'cachegroups/{cache_group_id:d}/queue_update', (u'1.1', u'1.2', u'1.3',))
	def cachegroups_queue_update(self, cache_group_id=None, data=None):
		"""
		Queue Updates by Cache Group ID
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup.html#cache-group
		:param cache_group_id: The Cache Group Id
		:type cache_group_id: int
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Cache Group Parameters
	#

	@restapi.api_request(u'post', u'cachegroupparameters', (u'1.2', u'1.3'))
	def assign_cache_group_parameters(self, data=None):
		"""
		Assign parameter(s) to cache group(s).
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup_parameter.html#cache-group-parameters
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'cachegroupparameters/{cache_group_id:d}/{parameter_id:d}', (u'1.2', u'1.3'))
	def delete_cache_group_parameters(self, cache_group_id=None, parameter_id=None):
		"""
		Delete a cache group parameter association
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup_parameter.html#cache-group-parameters
		:param cache_group_id: The cache group id in which the parameter will be deleted
		:type cache_group_id: int
		:param parameter_id: The parameter id which will be disassociated
		:type parameter_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	# Cache Group Fallback

	@restapi.api_request(u'get', u'cachegroup_fallbacks', (u'1.2', u'1.3'))
	def get_cache_group_fallbacks(self, query_params=None):
		"""
		Retrieve fallback related configurations for a cache group
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup_fallbacks.html#api-1-2-cachegroup-fallbacks
		:param query_params: Either cacheGroupId or fallbackId must be used or can be used simultaneously
		:type query_params: Dict[Text, int]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'cachegroup_fallbacks', (u'1.2', u'1.3'))
	def create_cache_group_fallbacks(self, data=None):
		"""
		Creates fallback configuration for the cache group. New fallbacks can be added only via POST.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup_fallbacks.html#api-1-2-cachegroup-fallbacks
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
 		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'cachegroup_fallbacks', (u'1.2', u'1.3'))
	def update_cache_group_fallbacks(self, data=None):
		"""
		Updates an existing fallback configuration for the cache group. 
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup_fallbacks.html#api-1-2-cachegroup-fallbacks
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'cachegroup_fallbacks', (u'1.2', u'1.3'))
	def delete_cache_group_fallbacks(self, query_params=None):
		"""
		Deletes an existing fallback related configurations for a cache group
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cachegroup_fallbacks.html#api-1-2-cachegroup-fallbacks
		:param query_params: Either cacheGroupId or fallbackId must be used or can be used simultaneously
		:type query_params: Dict[Text, int]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Cache Statistics
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cache_stats.html#cache-statistics
	#
	@restapi.api_request(u'get', u'cache_stats', (u'1.2', u'1.3',))
	def get_cache_stats(self, query_params=None):
		"""
		Retrieves statistics about the CDN.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/cache_stats.html#cache-statistics
		:param query_params: See API page for more information on accepted params
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Capabilities - Not adding for vagueness
	#

	#
	# CDN
	#
	#
	
	@restapi.api_request(u'get', u'cdns', (u'1.1', u'1.2', u'1.3',))
	def get_cdns(self):
		"""
		Get all CDNs.
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'cdns/{cdn_id:d}', (u'1.1', u'1.2', u'1.3',))
	def get_cdn_by_id(self, cdn_id=None):
		"""
		Get a CDN by Id.
		:param cdn_id: The CDN id
		:type cdn_id: Text
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'cdns/name/{cdn_name}', (u'1.1', u'1.2', u'1.3',))
	def get_cdn_by_name(self, cdn_name=None):
		"""
		Get a CDN by name.
		:param cdn_name: The CDN name
		:type cdn_name: Text
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'cdns', (u'1.1', u'1.2', u'1.3',))
	def create_cdn(self, data=None):
		"""
		Create a new CDN.
		:param data: The parameter data to use for cdn creation.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'cdns/{cdn_id:d}', (u'1.1', u'1.2', u'1.3',))
	def update_cdn_by_id(self, cdn_id=None, data=None):
		"""
		Update a CDN by Id.
		:param cdn_id: The CDN id
		:type cdn_id: int
		:param data: The parameter data to use for cdn update.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'cdns/{cdn_id:d}', (u'1.1', u'1.2', u'1.3',))
	def delete_cdn_by_id(self, cdn_id=None):
		"""
		Delete a CDN by Id.
		:param cdn_id: The CDN id
		:type cdn_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'cdns/{cdn_id:d}/queue_update', (u'1.1', u'1.2', u'1.3',))
	def cdns_queue_update(self, cdn_id=None, data=None):
		"""
		Queue Updates by CDN Id.
		:param cdn_id: The CDN Id
		:type cdn_id: int
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""


	#
	# Change Logs
	#https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/changelog.html#change-logs
	#

	@restapi.api_request(u'get', u'logs', (u'1.2', u'1.3',))
	def get_change_logs(self):
		"""
		Retrieve all change logs from traffic ops
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/changelog.html#change-logs
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'logs/{days:d}/days', (u'1.2', u'1.3',))
	def get_change_logs_for_days(self, days=None):
		"""
		Retrieve all change logs from Traffic Ops
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/changelog.html#change-logs
		:param days: The number of days to retrieve change logs
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'logs/newcount', (u'1.2', u'1.3',))
	def get_change_logs_newcount(self):
		"""
		Get amount of new logs from traffic ops
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/changelog.html#change-logs
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Config Files and Config File Metadata
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/configfiles_ats.html#config-files-and-config-file-metadata
	#

	#
	# Delivery Service
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#delivery-service
	#

	@restapi.api_request(u'get', u'deliveryservices', (u'1.1', u'1.2', u'1.3',))
	def get_deliveryservices(self, query_params=None):
		"""
		Retrieves all delivery services (if admin or ops) or all delivery services assigned to user.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#api-1-2-deliveryservices
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'deliveryservices/{delivery_service_id:d}', (u'1.1', u'1.2', u'1.3',))
	def get_deliveryservice_by_id(self, delivery_service_id=None):
		"""
		Retrieves a specific delivery service. If not admin / ops, delivery service must be assigned to user.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#api-1-2-deliveryservices
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'deliveryservices/{delivery_service_id:d}/servers', (u'1.1', u'1.2', u'1.3',))
	def get_deliveryservice_servers(self, delivery_service_id=None):
		"""
		Retrieves properties of CDN EDGE or ORG servers assigned to a delivery service.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#api-1-2-deliveryservices
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'deliveryservices/{delivery_service_id:d}/servers/unassigned', (u'1.1', u'1.2', u'1.3',))
	def get_deliveryservice_unassigned_servers(self, delivery_service_id=None):
		"""
		Retrieves properties of CDN EDGE or ORG servers not assigned to a delivery service. (Currently call does not work)
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#api-1-2-deliveryservices
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'deliveryservices/{delivery_service_id:d}/servers/eligible', (u'1.1', u'1.2', u'1.3',))
	def get_deliveryservice_ineligible_servers(self, delivery_service_id=None):
		"""
		Retrieves properties of CDN EDGE or ORG servers not eligible for assignment to a delivery service.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#api-1-2-deliveryservices
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""


	@restapi.api_request(u'post', u'deliveryservices', (u'1.1', u'1.2', u'1.3',))
	def create_deliveryservice(self, data=None):
		"""
		Allows user to create a delivery service.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#api-1-2-deliveryservices
		:param data: The request data structure for the API request
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'deliveryservices/{delivery_service_id:d}', (u'1.1', u'1.2', u'1.3',))
	def update_deliveryservice_by_id(self, delivery_service_id=None, data=None):
		"""
		Update a Delivery Service by Id.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#api-1-2-deliveryservices
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:param data: The request data structure for the API request
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'deliveryservices/{delivery_service_id:d}/safe', (u'1.1', u'1.2', u'1.3',))
	def update_deliveryservice_safe(self, delivery_service_id=None, data=None):
		"""
		Allows a user to edit limited fields of an assigned delivery service.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#api-1-2-deliveryservices
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:param data: The request data structure for the API request
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""


	@restapi.api_request(u'delete', u'deliveryservices/{delivery_service_id:d}', (u'1.1', u'1.2', u'1.3',))
	def delete_deliveryservice_by_id(self, delivery_service_id=None):
		"""
		Allows user to delete a delivery service.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#api-1-2-deliveryservices
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	# 
	# Delivery Service Health
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#health
	#

	@restapi.api_request(u'get', u'deliveryservices/{delivery_service_id:d}/state', (u'1.1', u'1.2', u'1.3',))
	def get_delivery_service_failover_state(self, delivery_service_id=None):
		"""
		Retrieves the failover state for a delivery service. Delivery service must be assigned to user if user is not admin or operations.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#health
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'deliveryservices/{delivery_service_id:d}/health', (u'1.1', u'1.2', u'1.3',))
	def get_delivery_service_health(self, delivery_service_id=None):
		"""
		Retrieves the health of all locations (cache groups) for a delivery service. Delivery service must be assigned to user if user is not admin or operations.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#health
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'deliveryservices/{delivery_service_id:d}/capacity', (u'1.1', u'1.2', u'1.3',))
	def get_delivery_service_capacity(self, delivery_service_id=None):
		"""
		Retrieves the capacity percentages of a delivery service. Delivery service must be assigned to user if user is not admin or operations.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#health
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'deliveryservices/{delivery_service_id:d}/routing', (u'1.1', u'1.2', u'1.3',))
	def get_delivery_service_routing(self, delivery_service_id=None):
		"""
		Retrieves the routing method percentages of a delivery service. Delivery service must be assigned to user if user is not admin or operations.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#health
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Delivery Service Server
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#delivery-service-server
	#

	@restapi.api_request(u'get', u'deliveryserviceserver', (u'1.1', u'1.2', u'1.3',))
	def get_deliveryserviceserver(self, query_params=None):
		"""
		Retrieves delivery service / server assignments. (Allows pagination and limits)
		:param query_params: The required url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'deliveryserviceserver', (u'1.1', u'1.2', u'1.3',))
	def assign_deliveryservice_servers_by_ids(self, data=None):
		"""
		Assign servers by id to a Delivery Service. (New Method)
		:param data: The required data to create server associations to a delivery service
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'deliveryservices/{xml_id}/servers', (u'1.1', u'1.2', u'1.3',))
	def assign_deliveryservice_servers_by_names(self, xml_id=None, data=None):
		"""
		Assign servers by name to a Delivery Service by xmlId. (Old Method)
		:param xml_id: The XML Id of the delivery service
		:type xml_id: Text
		:param data: The required data to assign servers to a delivery service
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'deliveryservice_server/{delivery_service_id:d}/{server_id:d}', (u'1.1', u'1.2', u'1.3',))
	def delete_deliveryservice_servers_by_id(self, delivery_service_id=None, server_id=None):
		"""
		Removes a server (cache) from a delivery service.
		:param delivery_service_id: The delivery service id 
		:type delivery_service_id: int
		:param server_id: The server id to remove from delivery service
		:type server_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Delivery Service User
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#delivery-service-user
	# (TODO: Add all calls)

	#
	# Delivery Service SSL Keys
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#ssl-keys
	#

	@restapi.api_request(u'get', u'deliveryservices/xmlId/{xml_id}/sslkeys', (u'1.1', u'1.2', u'1.3',))
	def get_deliveryservice_ssl_keys_by_xml_id(self, xml_id=None, query_params=None):
		"""
		Get SSL keys for a Delivery Service by xmlId.
		:param xml_id: The Delivery Service XML id
		:type xml_id: Text
		:param query_params: The url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'deliveryservices/xmlId/{xml_id}/sslkeys/delete', (u'1.1', u'1.2', u'1.3',))
	def delete_deliveryservice_ssl_keys_by_xml_id(self, xml_id=None, query_params=None):
		"""
		Delete SSL keys for a Delivery Service by xmlId.
		:param xml_id: The Delivery Service xmlId
		:type xml_id: Text
		:param query_params: The url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'deliveryservices/sslkeys/generate', (u'1.1', u'1.2', u'1.3',))
	def generate_deliveryservice_ssl_keys(self, data=None):
		"""
		Generate an SSL certificate. (self-signed)
		:param data: The parameter data to use for Delivery Service SSL key generation.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'deliveryservices/sslkeys/add', (u'1.1', u'1.2', u'1.3',))
	def add_ssl_keys_to_deliveryservice(self, data=None):
		"""
		Add SSL keys to a Delivery Service.
		:param data: The parameter data to use for adding SSL keys to a Delivery Service.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Delivery Service URL Sig Keys
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/deliveryservice.html#url-sig-keys
	#

	@restapi.api_request(u'post', u'deliveryservices/xmlId/{xml_id}/urlkeys/generate', (u'1.1', u'1.2', u'1.3',))
	def generate_deliveryservice_url_signature_keys(self, xml_id=None):
		"""
		Generate URL Signature Keys for a Delivery Service by xmlId.
		:param xml_id: The Delivery Service xmlId
		:type xml_id: Text
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Delivery Service Regexes
	#
	#

	@restapi.api_request(u'get', u'deliveryservices_regexes', (u'1.1', u'1.2', u'1.3',))
	def get_deliveryservices_regexes(self):
		"""
		Get RegExes for all Delivery Services.
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'deliveryservices/{delivery_service_id:d}/regexes', (u'1.1', u'1.2', u'1.3',))
	def get_deliveryservice_regexes_by_id(self, delivery_service_id=None):
		"""
		Get RegExes for a Delivery Service by Id.
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'deliveryservices/regexes', (u'1.1', u'1.2', u'1.3',))
	def delete_deliveryservice_regexes(self, data=None):
		"""
		Delete RegExes.
		:param data: The required data to delete delivery service regexes
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'deliveryservices/{delivery_service_id:d}/regexes/{delivery_service_regex_id:d}',
						 (u'1.1', u'1.2', u'1.3',))
	def delete_deliveryservice_regex_by_regex_id(self, delivery_service_id=None, delivery_service_regex_id=None):
		"""
		Delete a RegEx by Id for a Delivery Service by Id.
		:param delivery_service_id: The delivery service Id
		:type delivery_service_id: int
		:param delivery_service_regex_id: The delivery service regex Id
		:type delivery_service_regex_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""




	# Delivery Service Statistics

	# Divisions

	# Federation

	# Federation Delivery Service

	# Federation Federation Resolver

	# Federation Resolver

	# Federation User

	# Hardware Info

	# ISO

	# Jobs

	# Parameter
	@restapi.api_request(u'get', u'parameters', (u'1.1', u'1.2', u'1.3',))
	def get_parameters(self):
		"""
		Get all Profile Parameters.
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'parameters/validate', (u'1.1', u'1.2', u'1.3',))
	def validate_parameter_exists(self, data=None):
		"""
		Validate that a Parameter exists.
		:param data: The parameter data to use for parameter validation.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'parameters/{parameter_id:d}', (u'1.1', u'1.2', u'1.3',))
	def get_parameter_by_id(self, parameter_id=None):
		"""
		Get a Parameter by Id.
		:param parameter_id: The parameter Id
		:type parameter_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'profiles/{id:d}/parameters', (u'1.1', u'1.2', u'1.3',))
	def get_parameters_by_profile_id(self, profile_id=None):
		"""
		Get all Parameters associated with a Profile by Id.
		:param profile_id: The profile Id
		:type profile_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'profiles/name/{profile_name}/parameters', (u'1.1', u'1.2', u'1.3',))
	def get_parameters_by_profile_name(self, profile_name=None):
		"""
		Get all Parameters associated with a Profile by Name.
		:param profile_name: The profile name
		:type profile_name: Text
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'parameters', (u'1.1', u'1.2', u'1.3',))
	def create_parameters(self, data=None):
		"""
		Create Parameters
		:param data: The parameter(s) data to use for parameter creation.
		:type data: Union[Dict[Text, Any], List[Dict[Text, Any]]]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'parameters/{parameter_id:d}/profiles', (u'1.1', u'1.2', u'1.3',))
	def get_associated_profiles_by_parameter_id(self, parameter_id=None):
		"""
		Get all Profiles associated to a Parameter by Id.
		:param parameter_id: The parameter id
		:type parameter_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'profiles/id/{profile_id:d}/parameters', (u'1.1', u'1.2', u'1.3',))
	def associate_parameters_by_profile_id(self, profile_id=None, data=None):
		"""
		Associate Parameters to a Profile by Id.
		:param profile_id: The profile id
		:type profile_id: int
		:param data: The parameter data to associate
		:type data: Union[Dict[Text, Any], List[Dict[Text, Any]]]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'profiles/name/{profile_name}/parameters', (u'1.1', u'1.2', u'1.3',))
	def associate_parameters_by_profile_name(self, profile_name=None, data=None):
		"""
		Associate Parameters to a Profile by Name.
		:param profile_name: The profile name
		:type profile_name: Text
		:param data: The parameter data to associate
		:type data: Union[Dict[Text, Any], List[Dict[Text, Any]]]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'profileparameters/{profile_id:d}/{parameter_id:d}', (u'1.1', u'1.2', u'1.3',))
	def delete_profile_parameter_association_by_id(self, profile_id=None, parameter_id=None):
		"""
		Delete Parameter association by Id for a Profile by Id.
		:param profile_id: The profile id
		:type profile_id: int
		:param parameter_id: The parameter id
		:type parameter_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	# Physical Location
	@restapi.api_request(u'get', u'phys_locations', (u'1.1', u'1.2', u'1.3',))
	def get_physical_locations(self, query_params=None):
		"""
		Get Physical Locations.
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	# Profiles
	@restapi.api_request(u'get', u'profiles', (u'1.1', u'1.2', u'1.3',))
	def get_profiles(self, query_params=None):
		"""
		Get Profiles.
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'profiles/{profile_id:d}', (u'1.1', u'1.2', u'1.3',))
	def get_profile_by_id(self, profile_id=None):
		"""
		Get Profile by Id.
		:param profile_id: The profile Id
		:type profile_id: int
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'profiles/{profile_id:d}', (u'1.1', u'1.2', u'1.3',))
	def update_profile_by_id(self, profile_id=None, data=None):
		"""
		Update Profile by Id.
		:param profile_id: The profile Id
		:type profile_id: int
		:param data: The parameter data to edit
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'profiles/{profile_id:d}', (u'1.1', u'1.2', u'1.3',))
	def delete_profile_by_id(self, profile_id=None):
		"""
		Delete Profile by Id.
		:param profile_id: The profile Id
		:type profile_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	# Profile Parameters

	# InfluxDB

	#
	# Regions
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/region.html#regions
	#

	@restapi.api_request(u'get', u'regions', (u'1.1', u'1.2', u'1.3',))
	def get_regions(self):
		"""
		Get Regions.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/region.html#regions
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'regions/{region_id:d}', (u'1.1', u'1.2', u'1.3',))
	def get_region_by_id(self, region_id=None):
		"""
		Get Region by ID
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/region.html#regions
		:param region_id: The region id of the region to retrieve
		:type region_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'regions/{region_id:d}', (u'1.1', u'1.2', u'1.3',))
	def update_region(self, region_id=None):
		"""
		Update a region
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/tenant.html#tenants
		:parma region_id: The region to update
		:type region_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'divisions/{division_name:s}/regions', (u'1.1', u'1.2', u'1.3',))
	def create_region(self, division_name=None, data=None):
		"""
		Create a region
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/tenant.html#tenants
		:param division_name: The Division name in which region will reside
		:type division_name: String
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""




	#
	# Roles
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/role.html#roles
	#

	@restapi.api_request(u'get', u'roles', (u'1.1', u'1.2', u'1.3',))
	def get_roles(self):
		"""
		Get Roles.
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Server
	#
	#
	@restapi.api_request(u'get', u'servers/hostname/{name}/details', (u'1.1', u'1.2', u'1.3',))
	def get_server_details(self, name=None):
		"""
		#GET /api/1.2/servers/hostname/:name/details
		Get server details from trafficOps
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/server.html
		:param hostname: Server hostname
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'servers', (u'1.1', u'1.2',u'1.3',))
	def get_servers(self, query_params=None):
		"""
		Get Servers.
		:param query_params: The optional url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'servers', (u'1.1', u'1.2', u'1.3',))
	def create_server(self, data=None):
		"""
		Create a new Server.
		:param data: The parameter data to use for server creation
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'servers/{server_id:d}', (u'1.1', u'1.2', u'1.3',))
	def update_server_by_id(self, server_id=None, data=None):
		"""
		Update a Server by Id.
		:param server_id: The server Id
		:type server_id: int
		:param data: The parameter data to edit
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""
	@restapi.api_request(u'put', u'servers/{server_id:d}/status', (u'1.1', u'1.2', u'1.3',))
	def update_server_status_by_id(self, server_id=None, data=None):
		"""
		Update server_status by Id.
		:param server_id: The server Id
		:type server_id: int
		:status: https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/server.html
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'servers/{server_id:d}', (u'1.1', u'1.2', u'1.3',))
	def delete_server_by_id(self, server_id=None):
		"""
		Delete a Server by Id.
		:param server_id: The server Id
		:type server_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'servers/{server_id:d}/queue_update', (u'1.1', u'1.2', u'1.3',))
	def servers_queue_update(self, server_id=None, data=None):
		"""
		Queue Updates by Server Id.
		:param server_id: The server Id
		:type server_id: int
		:param data: The update action.  QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Static DNS Entries
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/static_dns.html#api-1-2-staticdnsentries
	#

	@restapi.api_request(u'get', u'staticdnsentries', (u'1.1', u'1.2', ))
	def get_static_dns_entries(self):
		"""
		Get Static DNS Entries.
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'staticdnsentries', (u'1.1', u'1.2', u'1.3',))
	def get_staticdnsentries(self, query_params=None):
		"""
		Get static DNS entries associated with the delivery service
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/staticdnsentry.html#api-1-3-staticdnsentries
		:param query_params: The optional url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'staticdnsentries', (u'1.3',))
	def create_staticdnsentries(self, data=None):
		"""
		Create static DNS entries associated with the delivery service
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/staticdnsentry.html#api-1-3-staticdnsentries
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'staticdnsentries', (u'1.3',))
	def update_staticdnsentries(self, data=None, query_params=None):
		"""
		Update static DNS entries associated with the delivery service
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/staticdnsentry.html#api-1-3-staticdnsentries
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:param query_params: The optional url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'staticdnsentries', (u'1.3',))
	def delete_staticdnsentries(self, query_params=None):
		"""
		Delete static DNS entries associated with the delivery service
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/staticdnsentry.html#api-1-3-staticdnsentries
		:param query_params: The optional url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Status
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/status.html#api-1-2-statuses
	#

	@restapi.api_request(u'get', u'statuses', (u'1.1', u'1.2', u'1.3',))
	def get_statuses(self):
		"""
		Retrieves a list of the server status codes available.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/status.html#api-1-2-statuses
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'statuses/{status_id:d}', (u'1.1', u'1.2', u'1.3',))
	def get_statuses_by_id(self, status_id=None):
		"""
		Retrieves a server status by ID.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/status.html#api-1-2-statuses
		:param status_id: The status id to retrieve
		:type status_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# System
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/system.html#api-1-1-system
	#

	@restapi.api_request(u'get', u'system/info.json', (u'1.1', u'1.2', u'1.3',))
	def get_system_info(self):
		"""
		Get information on the traffic ops system.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/system.html#api-1-1-system
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""


	#
	# Tenants
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/tenant.html#tenants
	#

	@restapi.api_request(u'get', u'tenants', (u'1.1', u'1.2', u'1.3',))
	def get_tenants(self):
		"""
		Get all tenants.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/tenant.html#tenants
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'tenants/{tenant_id:d}', (u'1.1', u'1.2', u'1.3',))
	def get_tenant_by_id(self, tenant_id=None):
		"""
		Get a tenant by ID.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/tenant.html#tenants
		:param tenant_id: The tenant to retrieve
		:type tenant_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'tenants/{tenant_id:d}', (u'1.1', u'1.2', u'1.3',))
	def update_tenant(self, tenant_id=None):
		"""
		Update a tenant
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/tenant.html#tenants
		:param tenant_id: The tenant to update
		:type tenant_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'tenants', (u'1.1', u'1.2', u'1.3',))
	def create_tenant(self, data=None):
		"""
		Create a tenant
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/tenant.html#tenants
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""


	#
	# TO Extensions
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/to_extension.html#to-extensions
	#
	
	@restapi.api_request(u'get', u'to_extensions.json', (u'1.1', u'1.2', u'1.3',))
	def get_to_extensions(self):
		"""
		Retrieves the list of extensions.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/to_extension.html#to-extensions
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'to_extensions', (u'1.1', u'1.2', u'1.3',))
	def create_to_extension(self, data=None):
		"""
		Creates a Traffic Ops extension.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/to_extension.html#to-extensions
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'to_extensions/{extension_id:d}/delete', (u'1.1', u'1.2', u'1.3',))
	def delete_to_extension(self, extension_id=None):
		"""
		Deletes a Traffic Ops extension.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/to_extension.html#to-extensions
		:param extension_id: The extension id to delete
		:type extension_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Types
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/type.html#api-1-2-types
	#

	@restapi.api_request(u'get', u'types', (u'1.1', u'1.2', u'1.3',))
	def get_types(self, query_params=None):
		"""
		Get Data Types.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/type.html#api-1-2-types
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'types/trimmed', (u'1.1', u'1.2', u'1.3',))
	def get_types_only_names(self):
		"""
		Get Data Types with only the Names
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/type.html#api-1-2-types
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'types/{type_id:d}', (u'1.1', u'1.2', u'1.3',))
	def get_type_by_id(self, type_id=None):
		"""
		Get Data Type with the given type id
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/type.html#api-1-2-types
		:param type_id: The ID of the type to retrieve
		:type type_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Users
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/user.html#api-1-2-users
	#

	@restapi.api_request(u'get', u'users', (u'1.1', u'1.2', u'1.3',))
	def get_users(self):
		"""
		Retrieves all users.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/user.html#api-1-2-users
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'users/{user_id:d}', (u'1.1', u'1.2', u'1.3',))
	def get_user_by_id(self, user_id=None):
		"""
		Retrieves user by ID.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/user.html#api-1-2-users
		:param user_id: The user to retrieve
		:type user_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'users', (u'1.1', u'1.2', u'1.3',))
	def create_user(self, data=None):
		"""
		Create a user.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/user.html#api-1-2-users
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'users/register', (u'1.1', u'1.2', u'1.3',))
	def create_user_with_registration(self, data=None):
		"""
		Register a user and send registration email
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/user.html#api-1-2-users
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'users/{user_id:d}/deliveryservices', (u'1.1', u'1.2', u'1.3',))
	def get_user_delivery_services(self, user_id=None):
		"""
		Retrieves all delivery services assigned to the user.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/user.html#api-1-2-users
		:param user_id: The user to retrieve
		:type user_id: int
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'user/current', (u'1.1', u'1.2', u'1.3',))
	def get_authenticated_user(self):
		"""
		Retrieves the profile for the authenticated user.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/user.html#api-1-2-users
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'user/current/jobs.json', (u'1.1', u'1.2', u'1.3',))
	def get_authenticated_user_jobs(self):
		"""
		Retrieves the user’s list of jobs.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/user.html#api-1-2-users

		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'user/current/jobs', (u'1.1', u'1.2', u'1.3',))
	def create_invalidation_job(self, data=None):
		"""
		Invalidating content on the CDN is sometimes necessary when the origin was mis-configured and something is cached in the CDN that needs to be removed. 
		Given the size of a typical Traffic Control CDN and the amount of content that can be cached in it, removing the content from all the caches may take a long time.
		To speed up content invalidation, Traffic Ops will not try to remove the content from the caches, but it makes the content inaccessible using the regex_revalidate ATS plugin (https://docs.trafficserver.apache.org/en/latest/admin-guide/plugins/regex_revalidate.en.html). 
		This forces a revalidation of the content, rather than a new get..
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/user.html#api-1-2-users
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Snapshot CRConfig
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/topology.html#snapshot-crconfig
	#

	@restapi.api_request(u'get', u'cdns/{cdn_name}/snapshot', (u'1.2', u'1.3',))
	def get_current_snapshot_crconfig(self, cdn_name=None):
		"""
		Retrieves the CURRENT snapshot for a CDN which doesn’t necessarily represent the current state of the CDN. 
		The contents of this snapshot are currently used by Traffic Monitor and Traffic Router.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/topology.html#snapshot-crconfig
		:param cdn_name: The CDN name
		:type cdn_name: Text
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'get', u'cdns/{cdn_name}/snapshot/new', (u'1.2', u'1.3',))
	def get_pending_snapshot_crconfig(self, cdn_name=None):
		"""
		Retrieves a PENDING snapshot for a CDN which represents the current state of the CDN. 
		The contents of this snapshot are NOT currently used by Traffic Monitor and Traffic Router. 
		Once a snapshot is performed, this snapshot will become the CURRENT snapshot and will be used by Traffic Monitor and Traffic Router.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/topology.html#snapshot-crconfig
		:param cdn_name: The CDN name
		:type cdn_name: Text
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'snapshot/{cdn_name}', (u'1.2', u'1.3',))
	def snapshot_crconfig(self, cdn_name=None):
		"""
		Snapshot CRConfig by CDN Name.
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v12/topology.html#snapshot-crconfig
		:param cdn_name: The CDN name
		:type cdn_name: Text
		:rtype: Tuple[Dict[Text, Any], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""



	#
	# Coordinate
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/coordinate.html#coordinate
	#

	@restapi.api_request(u'get', u'coordinates', (u'1.3',))
	def get_coordinates(self, query_params=None):
		"""
		Get all coordinates associated with the cdn
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/coordinate.html#coordinate
		:param query_params: The optional url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'coordinates', (u'1.3'))
	def create_coordinates(self, data=None):
		"""
		Create coordinates
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/coordinate.html#coordinate
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'coordinates', (u'1.3'))
	def update_coordinates(self, query_params=None, data=None):
		"""
		Update coordinates
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/coordinate.html#coordinate
		:param query_params: The optional url query parameters for the call
		:type query_params: Dict[Text, Any]
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'coordinates', (u'1.3'))
	def delete_coordinates(self, query_params=None):
		"""
		Delete coordinates
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/coordinate.html#coordinate
		:param query_params: The optional url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	#
	# Origin
	# https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/origin.html#api-1-3-origins
	#

	@restapi.api_request(u'get', u'origins', (u'1.3',))
	def get_origins(self, query_params=None):
		"""
		Get origins associated with the delivery service
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/origin.html#api-1-3-origins
		:param query_params: The optional url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'post', u'origins', (u'1.3',))
	def create_origins(self, data=None):
		"""
		Creates origins associated with a delivery service
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/origin.html#api-1-3-origins
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'put', u'origins', (u'1.3',))
	def update_origins(self, query_params=None):
		"""
		Updates origins associated with a delivery service
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/origin.html#api-1-3-origins
		:param data: The update action. QueueUpdateRequest() can be used for this argument also.
		:type data: Dict[Text, Any]
		:param query_params: The optional url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	@restapi.api_request(u'delete', u'origins', (u'1.3',))
	def delete_origins(self, query_params=None):
		"""
		Updates origins associated with a delivery service
		https://traffic-control-cdn.readthedocs.io/en/latest/api/v13/origin.html#api-1-3-origins
		:param query_params: The optional url query parameters for the call
		:type query_params: Dict[Text, Any]
		:rtype: Tuple[Union[Dict[Text, Any], List[Dict[Text, Any]]], requests.Response]
		:raises: Union[trafficops.restapi.LoginError, trafficops.restapi.OperationError]
		"""

	# Extra things here

	@restapi.api_request("get", "servers/{servername:s}/configfiles/ats", ("1.3",))
	def getServerConfigFiles(self, servername=None):
		"""
		Fetches the configuration files for a server with the given short hostname
		:param servername: The short hostname of the server
		:returns: The response content and actual response object
		"""

	####################################################################################

	####                          Data Model Overrides                              ####

	####################################################################################

	def __enter__(self):
		"""
		Implements context-management for ToSessions. This will open the session by sending a
		connection request immediately, rather than waiting for login.

		:returns: The constructed object (:meth:`__init__` is called implicitly prior to this method)
		"""
		self.create()
		return self

	def __exit__(self, exc_type, exc_value, traceback):
		"""
		Implements context-management for TOSessions. This will close the underlying socket.
		"""
		self.close()

		if exc_type:
			logging.error("%s", exc_value)
			logging.debug("%s", exc_type, stack_info=traceback)




if __name__ == u'__main__':
	# Sample usages
	import sys
	import operator

	DEBUG = False

	logging.basicConfig(stream=sys.stderr, level=logging.INFO if not DEBUG else logging.DEBUG)

	# TOSession Class Examples
	#     TOSession is a class that allows you to create a session to a Traffic Ops instance
	#     and interact with the Traffic Ops API.

	# Traffic Ops System - for self-signed cert -> turn off cert verification
	tos = TOSession(host_ip=u'to.somedomain.net', verify_cert=True)
	tos.login(u'someuser', u'someuser123')

	# Objects get returned munch-ified by default which means you can access dictionary keys as
	# attributes names but you can still access the entries with keys as well.  E.g. cdn.name == cdn['name']
	cdns, response = tos.get_cdns()
	print(cdns)
	for cdn in cdns:
		print(u'CDN [{0}] has id [{1}]'.format(cdn.name, cdn.id))

	all_types, response = tos.get_types()
	print(u'All Types are (sorted by useInTable, name):')
	print(all_types)
	for atype in sorted(all_types, key=operator.itemgetter(u'useInTable', u'name')):
		print(u'Type [{0}] for table [{1}]'.format(atype.name, atype.useInTable))

	print(u'Getting all cache groups (bulk)...')
	cache_groups, response = tos.get_cachegroups()
	for cache_group in cache_groups:
		print(u'Bulk cache group [{0}] has id [{1}]'.format(cache_group.name, cache_group.id))

		# Example with URL replacement parameters
		# e.g. TOSession.get_cachegroups_by_id() == end-point 'api/1.2/cachegroups/{id}'
		#      See TOSession object for details.
		print(u'    Getting cachegroup by id [{0}] to demonstrate getting by id...'.format(cache_group.id))
		cg_id_list, response = tos.get_cachegroup_by_id(cache_group_id=cache_group.id)  # data returned is always a list
		print(u'    Cache group [{0}] by id [{1}]'.format(cg_id_list[0].name, cg_id_list[0].id))

	# Example with URL query parameters
	server_types, response = tos.get_types(query_params={u'useInTable': u'server'})
	print(u'Server Types are:')
	print(server_types)
	for stype in server_types:
		print(u'Type [{0}] for table [{1}]'.format(stype.name, stype.useInTable))
	tos.close()
	print(u'Done!')
