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
export const jobs = {
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
							queryValue: "dstestjob1",
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
					active: true,
					cdnId: 0,
					displayName: "DSJobTest",
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
					xmlId: "dstestjob1",
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
					description: "create an invalidation request",
                    DeliveryService: "dstestjob1",
                    Regex: "/test",
                    Ttl: "1",
					validationMessage: "Invalidation request created"
				}
			],
		}
    ] 
}
