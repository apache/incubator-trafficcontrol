/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */
export const origins = {
	cleanup: [
		{
			action: "DeleteDeliveryServices",
			route: "/deliveryservices",
			method: "delete",
			data: [
				{
					route: "/deliveryservices/",
					getRequest: [
						{
							route: "/deliveryservices",
							queryKey: "xmlId",
							queryValue: "ds1",
							replace: "route"
						}
					]
				},
				{
					route: "/deliveryservices/",
					getRequest: [
						{
							route: "/deliveryservices",
							queryKey: "xmlId",
							queryValue: "ds2",
							replace: "route"
						}
					]
				},
				{
					route: "/deliveryservices/",
					getRequest: [
						{
							route: "/deliveryservices",
							queryKey: "xmlId",
							queryValue: "ds3",
							replace: "route"
						}
					]
				},
				{
					route: "/deliveryservices/",
					getRequest: [
						{
							route: "/deliveryservices",
							queryKey: "xmlId",
							queryValue: "ds4",
							replace: "route"
						}
					]
				}
			]
		}
	],
	setup: [
		{
			action: "CreateDeliveryServices",
			route: "/deliveryservices",
			method: "post",
			data: [
				{
					active: "ACTIVE",
					cdnId: 2,
					displayName: "ds1",
					dscp: 0,
					geoLimit: 0,
					geoProvider: 0,
					initialDispersion: 1,
					ipv6RoutingEnabled: true,
					logsEnabled: false,
					missLat: 41.881944,
					missLong: -87.627778,
					multiSiteOrigin: false,
					orgServerFqdn: "http://origin.infra.ciab.test",
					protocol: 0,
					qstringIgnore: 0,
					rangeRequestHandling: 0,
					regionalGeoBlocking: false,
					tenantId: 6,
					typeId: 1,
					xmlId: "ds1",
					getRequest: [
						{
							route: "/tenants",
							queryKey: "name",
							queryValue: "tenantSame",
							replace: "tenantId"
						}
					]
				},
				{
					active: "ACTIVE",
					cdnId: 2,
					displayName: "ds2",
					dscp: 0,
					geoLimit: 0,
					geoProvider: 0,
					initialDispersion: 1,
					ipv6RoutingEnabled: true,
					logsEnabled: false,
					missLat: 41.881944,
					missLong: -87.627778,
					multiSiteOrigin: false,
					orgServerFqdn: "http://origin.infra.ciab.test",
					protocol: 0,
					qstringIgnore: 0,
					rangeRequestHandling: 0,
					regionalGeoBlocking: false,
					tenantId: 6,
					typeId: 1,
					xmlId: "ds2",
					getRequest: [
						{
							route: "/tenants",
							queryKey: "name",
							queryValue: "tenantParent",
							replace: "tenantId"
						}
					]
				},
				{
					active: "ACTIVE",
					cdnId: 2,
					displayName: "ds3",
					dscp: 0,
					geoLimit: 0,
					geoProvider: 0,
					initialDispersion: 1,
					ipv6RoutingEnabled: true,
					logsEnabled: false,
					missLat: 41.881944,
					missLong: -87.627778,
					multiSiteOrigin: false,
					orgServerFqdn: "http://origin.infra.ciab.test",
					protocol: 0,
					qstringIgnore: 0,
					rangeRequestHandling: 0,
					regionalGeoBlocking: false,
					tenantId: 6,
					typeId: 1,
					xmlId: "ds3",
					getRequest: [
						{
							route: "/tenants",
							queryKey: "name",
							queryValue: "tenantChild",
							replace: "tenantId"
						}
					]
				},
				{
					active: "ACTIVE",
					cdnId: 2,
					displayName: "ds4",
					dscp: 0,
					geoLimit: 0,
					geoProvider: 0,
					initialDispersion: 1,
					ipv6RoutingEnabled: true,
					logsEnabled: false,
					missLat: 41.881944,
					missLong: -87.627778,
					multiSiteOrigin: false,
					orgServerFqdn: "http://origin.infra.ciab.test",
					protocol: 0,
					qstringIgnore: 0,
					rangeRequestHandling: 0,
					regionalGeoBlocking: false,
					tenantId: 6,
					typeId: 1,
					xmlId: "ds4",
					getRequest: [
						{
							route: "/tenants",
							queryKey: "name",
							queryValue: "tenantDifferent",
							replace: "tenantId"
						}
					]
				}
			]
		}
	],
	tests: [
		{
			logins: [
				{
					description: "Admin Role",
					username: "TPAdmin",
					password: "pa$$word"
				}
			],
			add: [
				{
					description: "create an Origins",
					Name: "TP_Origins1",
					Tenant: "- tenantSame",
					FQDN: "0",
					Protocol: "https",
					deliveryServiceId: "ds1",
					validationMessage: "origin was created."
				},
				{
					description: "create multiple Origins with the same Delivery Service",
					Name: "TP_Origins2",
					Tenant: "- tenantSame",
					FQDN: "0",
					Protocol: "https",
					deliveryServiceId: "ds1",
					validationMessage: "origin was created."
				},
				{
					description: "create multiple Origins with child tenant Delivery Service",
					Name: "TP_Origins3",
					Tenant: "- tenantSame",
					FQDN: "0",
					Protocol: "https",
					deliveryServiceId: "ds3",
					validationMessage: "origin was created."
				}
			],
			update: [
				{
					description: "update Origin Delivery Service",
					Name: "TP_Origins1",
					NewDeliveryService: "ds3",
					validationMessage: "origin was updated."
				},
				{
					description: "Validate cannot change current Origin's Delivery Service to Delivery Service in tenant parent",
					Name: "TP_Origins2",
					NewDeliveryService: "ds2"
				},
				{
					description: "Validate cannot change current Origin's Delivery Service to Delivery Service in tenant different",
					Name: "TP_Origins2",
					NewDeliveryService: "ds4"
				}
			],
			remove: [
				{
					description: "delete an Origins",
					Name: "TP_Origins1",
					validationMessage: "origin was deleted."
				},
				{
					description: "delete an Origins",
					Name: "TP_Origins2",
					validationMessage: "origin was deleted."
				}
			]
		},
		{
			logins: [
				{
					description: "Operation Role",
					username: "TPOperator",
					password: "pa$$word"
				}
			],
			add: [
				{
					description: "create an Origins",
					Name: "TP_Origins1",
					Tenant: "- tenantSame",
					FQDN: "0",
					Protocol: "https",
					deliveryServiceId: "ds1",
					validationMessage: "origin was created."
				},
				{
					description: "create multiple Origins with the same Delivery Service",
					Name: "TP_Origins2",
					Tenant: "- tenantSame",
					FQDN: "0",
					Protocol: "https",
					deliveryServiceId: "ds1",
					validationMessage: "origin was created."
				}
			],
			update: [
				{
					description: "update Origin Delivery Service",
					Name: "TP_Origins1",
					NewDeliveryService: "ds3",
					validationMessage: "origin was updated."
				},
				{
					description: "Validate cannot change current Origin's Delivery Service to Delivery Service in tenant parent",
					Name: "TP_Origins2",
					NewDeliveryService: "ds2"
				},
				{
					description: "Validate cannot change current Origin's Delivery Service to Delivery Service in tenant different",
					Name: "TP_Origins2",
					NewDeliveryService: "ds4"
				}
			],
			remove: [
				{
					description: "delete an Origins",
					Name: "TP_Origins1",
					validationMessage: "origin was deleted."
				},
				{
					description: "delete an Origins",
					Name: "TP_Origins2",
					validationMessage: "origin was deleted."
				}
			]
		},
		{
			logins: [
				{
					description: "Read Only Role",
					username: "TPReadOnly",
					password: "pa$$word"
				}
			],
			add: [
				{
					description: "create an Origins",
					Name: "TP_Origins1",
					Tenant: "- tenantSame",
					FQDN: "0",
					Protocol: "https",
					deliveryServiceId: "ds1",
					validationMessage: "Forbidden."
				}
			],
			update: [
				{
					description: "update Origin Delivery Service",
					Name: "TP_Origins3",
					NewDeliveryService: "ds1",
					validationMessage: "Forbidden."
				}
			],
			remove: [
				{
					description: "delete an Origins",
					Name: "TP_Origins3",
					validationMessage: "Forbidden."
				}
			]
		}
	]
}
