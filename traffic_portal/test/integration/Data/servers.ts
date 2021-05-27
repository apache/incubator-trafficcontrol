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
export const servers = {
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
							queryValue: "servertds1",
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
							queryValue: "servertdsop1",
							replace: "route"
						}
					]
				}
			]
		},
		{
			action: "DeleteServerCapabilities",
			route: "/server_capabilities",
			method: "delete",
			data: [
				{
					route: "/server_capabilities?name=servertestcap1"
				},
				{
					route: "/server_capabilities?name=servertestcapop1"
				}
			]
		},
		{
			action: "DeleteServers",
			route: "/servers",
			method: "delete",
			data: [
				{
					route: "/servers/",
					getRequest: [
						{
							route: "/servers",
							queryKey: "hostName",
							queryValue: "servertestcreate2",
							replace: "route"
						}
					]
				},
				{
					route: "/servers/",
					getRequest: [
						{
							route: "/servers",
							queryKey: "hostName",
							queryValue: "servertestcreateop2",
							replace: "route"
						}
					]
				},
				{
					route: "/servers/",
					getRequest: [
						{
							route: "/servers",
							queryKey: "hostName",
							queryValue: "servertestremove3",
							replace: "route"
						}
					]
				},
				{
					route: "/servers/",
					getRequest: [
						{
							route: "/servers",
							queryKey: "hostName",
							queryValue: "servertestremoveop3",
							replace: "route"
						}
					]
				}
			]
		},
		{
			action: "DeleteProfile",
			route: "/profiles",
			method: "delete",
			data: [
				{
					route: "/profiles/",
					getRequest: [
						{
							route: "/profiles",
							queryKey: "name",
							queryValue: "servertestprofiles1",
							replace: "route"
						}
					]
				}
			]
		},
		{
			action: "DeletePhysLocations",
			route: "/phys_locations",
			method: "delete",
			data: [
				{
					route: "/phys_locations/",
					getRequest: [
						{
							route: "/phys_locations",
							queryKey: "name",
							queryValue: "TPPhysLocation2",
							replace: "route"
						}
					]
				}
			]
		},
		{
			action: "DeleteRegions",
			route: "/regions",
			method: "delete",
			data: [
				{
					route: "/regions?name=PhysTest"
				},
				{
					route: "/regions?name=PhysTest2"
				}
			]
		},
		{
			action: "DeleteDivisions",
			route: "/divisions",
			method: "delete",
			data: [
				{
					route: "/divisions/",
					getRequest: [
						{
							route: "/divisions",
							queryKey: "name",
							queryValue: "PhysTest",
							replace: "route"
						}
					]
				}
			]
		},
		{
			action: "DeleteCDN",
			route: "/cdns",
			method: "delete",
			data: [
				{
					route: "/cdns/",
					getRequest: [
						{
							route: "/cdns",
							queryKey: "name",
							queryValue: "servertestcdn1",
							replace: "route"
						}
					]
				}
			]
		}
	],
	setup: [
		{
			action: "CreateDivisions",
			route: "/divisions",
			method: "post",
			data: [
				{
					name: "PhysTest"
				}
			]
		},
		{
			action: "CreateRegions",
			route: "/regions",
			method: "post",
			data: [
				{
					name: "PhysTest",
					division: "4",
					divisionName: "PhysTest",
					getRequest: [
						{
							route: "/divisions",
							queryKey: "name",
							queryValue: "PhysTest",
							replace: "division"
						}
					]
				},
				{
					name: "PhysTest2",
					division: "4",
					divisionName: "PhysTest",
					getRequest: [
						{
							route: "/divisions",
							queryKey: "name",
							queryValue: "PhysTest",
							replace: "division"
						}
					]
				}
			]
		},
		{
			action: "CreatePhysLocation",
			route: "/phys_locations",
			method: "post",
			data: [
				{
					address: "Buckingham Palace",
					city: "London",
					comments: "Buckingham Palace",
					email: "steve.kingstone@royal.gsx.gov.uk",
					name: "TPPhysLocation2",
					phone: "0-843-816-6276",
					poc: "Her Majesty The Queen Elizabeth Alexandra Mary Windsor II",
					regionId: 3,
					shortName: "tpphys2",
					state: "NA",
					zip: "99999",
					getRequest: [
						{
							route: "/regions",
							queryKey: "name",
							queryValue: "PhysTest",
							replace: "regionId"
						}
					]
				}
			]
		},
		{
			action: "CreateServers",
			route: "/servers",
			method: "post",
			data: [
				{
					cachegroupId: 0,
					cdnId: 0,
					domainName: "test.net",
					hostName: "servertestremove2",
					httpsPort: 443,
					iloIpAddress: "",
					iloIpGateway: "",
					iloIpNetmask: "",
					iloPassword: "",
					iloUsername: "",
					interfaces: [
						{
							ipAddresses: [
								{
									address: "::1",
									gateway: "::2",
									serviceAddress: true
								}
							],
							maxBandwidth: null,
							monitor: true,
							mtu: 1500,
							name: "eth0"
						}
					],
					interfaceMtu: 1500,
					interfaceName: "eth0",
					ip6Address: "::1",
					ip6Gateway: "::2",
					ipAddress: "0.0.0.1",
					ipGateway: "0.0.0.2",
					ipNetmask: "255.255.255.0",
					mgmtIpAddress: "",
					mgmtIpGateway: "",
					mgmtIpNetmask: "",
					offlineReason: "",
					physLocationId: 0,
					profileId: 0,
					routerHostName: "",
					routerPortName: "",
					statusId: 3,
					tcpPort: 80,
					typeId: 12,
					updPending: false,
					getRequest: [
						{
							route: "/phys_locations",
							queryKey: "name",
							queryValue: "TPPhysLocation2",
							replace: "physLocationId"
						},
						{
							route: "/cdns",
							queryKey: "name",
							queryValue: "dummycdn",
							replace: "cdnId"
						},
						{
							route: "/cachegroups",
							queryKey: "name",
							queryValue: "testCG",
							replace: "cachegroupId"
						},
						{
							route: "/profiles",
							queryKey: "name",
							queryValue: "testProfile",
							replace: "profileId"
						}
					]
				},
				{
					cachegroupId: 0,
					cdnId: 0,
					domainName: "test.net",
					hostName: "servertestremoveop2",
					httpsPort: 443,
					iloIpAddress: "",
					iloIpGateway: "",
					iloIpNetmask: "",
					iloPassword: "",
					iloUsername: "",
					interfaces: [
						{
							ipAddresses: [
								{
									address: "::1",
									gateway: "::2",
									serviceAddress: true
								}
							],
							maxBandwidth: null,
							monitor: true,
							mtu: 1500,
							name: "eth0"
						}
					],
					interfaceMtu: 1500,
					interfaceName: "eth0",
					ip6Address: "::1",
					ip6Gateway: "::2",
					ipAddress: "0.0.0.1",
					ipGateway: "0.0.0.2",
					ipNetmask: "255.255.255.0",
					mgmtIpAddress: "",
					mgmtIpGateway: "",
					mgmtIpNetmask: "",
					offlineReason: "",
					physLocationId: 0,
					profileId: 0,
					routerHostName: "",
					routerPortName: "",
					statusId: 3,
					tcpPort: 80,
					typeId: 12,
					updPending: false,
					getRequest: [
						{
							route: "/phys_locations",
							queryKey: "name",
							queryValue: "TPPhysLocation2",
							replace: "physLocationId"
						},
						{
							route: "/cdns",
							queryKey: "name",
							queryValue: "dummycdn",
							replace: "cdnId"
						},
						{
							route: "/cachegroups",
							queryKey: "name",
							queryValue: "testCG",
							replace: "cachegroupId"
						},
						{
							route: "/profiles",
							queryKey: "name",
							queryValue: "testProfile",
							replace: "profileId"
						}
					]
				},
				{
					cachegroupId: 0,
					cdnId: 0,
					domainName: "test.net",
					hostName: "servertestremove3",
					httpsPort: 443,
					iloIpAddress: "",
					iloIpGateway: "",
					iloIpNetmask: "",
					iloPassword: "",
					iloUsername: "",
					interfaces: [
						{
							ipAddresses: [
								{
									address: "::1",
									gateway: "::2",
									serviceAddress: true
								}
							],
							maxBandwidth: null,
							monitor: true,
							mtu: 1500,
							name: "eth0"
						}
					],
					interfaceMtu: 1500,
					interfaceName: "eth0",
					ip6Address: "::1",
					ip6Gateway: "::2",
					ipAddress: "0.0.0.1",
					ipGateway: "0.0.0.2",
					ipNetmask: "255.255.255.0",
					mgmtIpAddress: "",
					mgmtIpGateway: "",
					mgmtIpNetmask: "",
					offlineReason: "",
					physLocationId: 0,
					profileId: 0,
					routerHostName: "",
					routerPortName: "",
					statusId: 3,
					tcpPort: 80,
					typeId: 11,
					updPending: false,
					getRequest: [
						{
							route: "/phys_locations",
							queryKey: "name",
							queryValue: "TPPhysLocation2",
							replace: "physLocationId"
						},
						{
							route: "/cdns",
							queryKey: "name",
							queryValue: "dummycdn",
							replace: "cdnId"
						},
						{
							route: "/cachegroups",
							queryKey: "name",
							queryValue: "testCG",
							replace: "cachegroupId"
						},
						{
							route: "/profiles",
							queryKey: "name",
							queryValue: "testProfile",
							replace: "profileId"
						}
					]
				},
				{
					cachegroupId: 0,
					cdnId: 0,
					domainName: "test.net",
					hostName: "servertestremoveop3",
					httpsPort: 443,
					iloIpAddress: "",
					iloIpGateway: "",
					iloIpNetmask: "",
					iloPassword: "",
					iloUsername: "",
					interfaces: [
						{
							ipAddresses: [
								{
									address: "::1",
									gateway: "::2",
									serviceAddress: true
								}
							],
							maxBandwidth: null,
							monitor: true,
							mtu: 1500,
							name: "eth0"
						}
					],
					interfaceMtu: 1500,
					interfaceName: "eth0",
					ip6Address: "::1",
					ip6Gateway: "::2",
					ipAddress: "0.0.0.1",
					ipGateway: "0.0.0.2",
					ipNetmask: "255.255.255.0",
					mgmtIpAddress: "",
					mgmtIpGateway: "",
					mgmtIpNetmask: "",
					offlineReason: "",
					physLocationId: 0,
					profileId: 0,
					routerHostName: "",
					routerPortName: "",
					statusId: 3,
					tcpPort: 80,
					typeId: 11,
					updPending: false,
					getRequest: [
						{
							route: "/phys_locations",
							queryKey: "name",
							queryValue: "TPPhysLocation2",
							replace: "physLocationId"
						},
						{
							route: "/cdns",
							queryKey: "name",
							queryValue: "dummycdn",
							replace: "cdnId"
						},
						{
							route: "/cachegroups",
							queryKey: "name",
							queryValue: "testCG",
							replace: "cachegroupId"
						},
						{
							route: "/profiles",
							queryKey: "name",
							queryValue: "testProfile",
							replace: "profileId"
						}
					]
				}
			]
		},
		{
			action: "CreateServerCapabilities",
			route: "/server_capabilities",
			method: "post",
			data: [
				{
					name: "servertestcap1"
				},
				{
					name: "servertestcapop1"
				}
			]
		},
		{
			action: "CreateServerServerCapabilities",
			route: "/server_server_capabilities",
			method: "post",
			data: [
				{
					serverId: 0,
					serverCapability: "servertestcap1",
					getRequest: [
						{
							route: "/servers",
							queryKey: "hostName",
							queryValue: "servertestremove2",
							replace: "serverId"
						}
					]
				},
				{
					serverId: 0,
					serverCapability: "servertestcapop1",
					getRequest: [
						{
							route: "/servers",
							queryKey: "hostName",
							queryValue: "servertestremoveop2",
							replace: "serverId"
						}
					]
				}
			]
		},
		{
			action: "CreateDeliveryServices",
			route: "/deliveryservices",
			method: "post",
			data: [
				{
					active: "ACTIVE",
					cdnId: 0,
					displayName: "servertestds1",
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
					tenantId: 0,
					typeId: 1,
					xmlId: "servertds1",
					getRequest: [
						{
							route: "/tenants",
							queryKey: "name",
							queryValue: "tenantSame",
							replace: "tenantId"
						},
						{
							route: "/cdns",
							queryKey: "name",
							queryValue: "dummycdn",
							replace: "cdnId"
						}
					]
				},
				{
					active: "ACTIVE",
					cdnId: 0,
					displayName: "servertestdsop1",
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
					tenantId: 0,
					typeId: 1,
					xmlId: "servertdsop1",
					getRequest: [
						{
							route: "/tenants",
							queryKey: "name",
							queryValue: "tenantSame",
							replace: "tenantId"
						},
						{
							route: "/cdns",
							queryKey: "name",
							queryValue: "dummycdn",
							replace: "cdnId"
						}
					]
				}
			]
		},
		{
			action: "CreateDeliveryServiceServer",
			route: "/deliveryserviceserver",
			method: "post",
			data: [
				{
					dsId: 120,
					replace: true,
					servers: [],
					getRequest: [
						{
							route: "/servers",
							queryKey: "hostName",
							queryValue: "servertestremove3",
							replace: "servers",
							isArray: true
						},
						{
							route: "/deliveryservices",
							queryKey: "xmlId",
							queryValue: "servertds1",
							replace: "dsId"
						}
					]
				},
				{
					dsId: 120,
					replace: true,
					servers: [],
					getRequest: [
						{
							route: "/servers",
							queryKey: "hostName",
							queryValue: "servertestremoveop3",
							replace: "servers",
							isArray: true
						},
						{
							route: "/deliveryservices",
							queryKey: "xmlId",
							queryValue: "servertdsop1",
							replace: "dsId"
						}
					]
				}
			]
		},
		{
			action: "CreateCDN",
			route: "/cdns",
			method: "post",
			data: [
				{
					name: "servertestcdn1",
					domainName: "svtestcdn1",
					dnssecEnabled: false
				}
			]
		},
		{
			action: "CreateProfile",
			route: "/profiles",
			method: "post",
			data: [
				{
					name: "servertestprofiles1",
					description: "A test profile for API examples",
					cdn: 2,
					type: "UNK_PROFILE",
					routingDisabled: true,
					getRequest: [
						{
							route: "/cdns",
							queryKey: "name",
							queryValue: "servertestcdn1",
							replace: "cdn"
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
					description: "create a Server",
					Status: "ONLINE",
					Hostname: "servertestcreate1",
					Domainname: "test.com",
					CDN: "dummycdn",
					CacheGroup: "testCG",
					Type: "EDGE",
					Profile: "testProfile",
					PhysLocation: "TPPhysLocation2",
					InterfaceName: "test",
					validationMessage: "Server created"
				},
				{
					description: "create multiple Server",
					Status: "ONLINE",
					Hostname: "servertestcreate2",
					Domainname: "test.com",
					CDN: "dummycdn",
					CacheGroup: "testCG",
					Type: "EDGE",
					Profile: "testProfile",
					PhysLocation: "TPPhysLocation2",
					InterfaceName: "test",
					validationMessage: "Server created"
				}
			],
			update: [
				{
					description: "Validate cannot change the cdn of a Server when it is currently assigned to a Delivery Services in different CDN",
					Name: "servertestremove3",
					CDN: "servertestcdn1",
					Profile: "servertestprofiles1",
					validationMessage: "server cdn can not be updated when it is currently assigned to delivery services"
				},
				{
					description: "change the cdn of a Server when it is currently not assign to any delivery service",
					Name: "servertestcreate1",
					CDN: "servertestcdn1",
					Profile: "servertestprofiles1",
					validationMessage: "Server updated"
				}
			],
			remove: [
				{
					description: "delete a Server",
					Name: "servertestcreate1",
					validationMessage: "Server deleted"
				},
				{
					description: "delete a Server with Server Capabilities assigned",
					Name: "servertestremove2",
					validationMessage: "Server deleted"
				}
			]
		},
		{
			logins: [
				{
					description: "ReadOnly Role",
					username: "TPReadOnly",
					password: "pa$$word"
				}
			],
			add: [
				{
					description: "create a Server",
					Status: "ONLINE",
					Hostname: "servertcreatero",
					Domainname: "test.com",
					CDN: "dummycdn",
					CacheGroup: "testCG",
					Type: "EDGE",
					Profile: "testProfile",
					PhysLocation: "TPPhysLocation2",
					InterfaceName: "test",
					validationMessage: "Forbidden."
				}
			],
			update: [
				{
					description: "change the cdn of a Server",
					Name: "servertestcreate2",
					CDN: "servertestcdn1",
					Profile: "servertestprofiles1",
					validationMessage: "Forbidden."
				}
			],
			remove: [
				{
					description: "delete a Server",
					Name: "servertestcreate2",
					validationMessage: "Forbidden."
				}
			]
		},
		{
			logins: [
				{
					description: "Operator Role",
					username: "TPOperator",
					password: "pa$$word"
				}
			],
			add: [
				{
					description: "create a Server",
					Status: "ONLINE",
					Hostname: "servertestcreateop1",
					Domainname: "test.com",
					CDN: "dummycdn",
					CacheGroup: "testCG",
					Type: "EDGE",
					Profile: "testProfile",
					PhysLocation: "TPPhysLocation2",
					InterfaceName: "test",
					validationMessage: "Server created"
				},
				{
					description: "create multiple Server",
					Status: "ONLINE",
					Hostname: "servertestcreateop2",
					Domainname: "test.com",
					CDN: "dummycdn",
					CacheGroup: "testCG",
					Type: "EDGE",
					Profile: "testProfile",
					PhysLocation: "TPPhysLocation2",
					InterfaceName: "test",
					validationMessage: "Server created"
				}
			],
			update: [
				{
					description: "Validate cannot change the cdn of a Server when it is currently assigned to a Delivery Services in different CDN",
					Name: "servertestremoveop3",
					CDN: "servertestcdn1",
					Profile: "servertestprofiles1",
					validationMessage: "server cdn can not be updated when it is currently assigned to delivery services"
				},
				{
					description: "change the cdn of a Server when it is currently not assign to any delivery service",
					Name: "servertestcreateop1",
					CDN: "servertestcdn1",
					Profile: "servertestprofiles1",
					validationMessage: "Server updated"
				}
			],
			remove: [
				{
					description: "delete a Server",
					Name: "servertestcreateop1",
					validationMessage: "Server deleted"
				},
				{
					description: "delete a Server with Server Capabilities assigned",
					Name: "servertestremoveop2",
					validationMessage: "Server deleted"
				}
			]
		}
	]
};
