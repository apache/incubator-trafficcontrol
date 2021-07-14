<!--
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
-->
# Snapshots and Monitoring Separation

## Problem Description
Traffic Monitor uses data partially from CDN Snapshots and partially from its
own Monitoring payloads to determine what to monitor and how. This leads to
race conditions and certain parts of it being out-of-sync with others as one
data set is polled while another grows stale.

## Proposed Change
Traffic Router should receive Snapshots directly from Traffic Ops, while
Traffic Monitor should only fetch data relevant to its operation through the
Monitoring payloads.

## Data Model Impact
The new structures for the Monitoring payloads and Snapshots are stripped down
to only what each service needs, as well as having some properties renamed to
be more consistent with the rest of the API. The proposed structures for each
are given below as TypeScript<sup>[1](#typescript)</sup> interfaces.

### Monitoring Payloads
```typescript
interface IPAddress {
	address: string;
	gateway: string | null;
	serviceAddress: boolean;
}

interface Interface {
	ipAddresses: Array<IPAddress>;
	maxBandwidth: number | null;//integer
	monitor: boolean;
	mtu: number;//integer
	name: string;
}

interface CacheServer {
	deliveryServices: Array<string>;
	httpsPort: number;//integer
	interfaces: Array<Interface>;
	profile: string;
	status: string;
	// formerly "port"
	tcpPort: number;//integer
	type: "EDGE" | "MID";
}

interface Monitoring {
	// This set should only contain MID_LOC and EDGE_LOC cache servers
	cacheGroups: {
		[cacheGroupName: string]: {
			// formerly a top-level property of monitoring payloads named
			// "trafficServers"
			cacheServers: {
				[hostName: string]: CacheServer;
			};
			// latitude and longitude used to be inside a "coordinates"
			// property which only had those two properties
			latitude: number;
			longitude: number;
		}
	};
	config: {
		// formerly 'health.event-count'
		eventCount: number;//integer
		// formerly 'health.timepad'
		healthPad: number;
		// formerly 'health.polling.interval'
		healthPollingInterval: number;
		// formerly 'heartbeat.polling.interval'
		heartbeatPollingInterval: number;
		// formerly 'peers.polling.interval'
		peerPollingInterval: number;
		// formerly 'tm.polling.interval'
		configPollingInterval: number;
	};
	deliveryServices: {
		[xmlID: string]: {
			// formerly totalTpsThreshold
			globalMaxTps: number | null;
			// formerly totalKbpsThreshold
			globalMaxMbps: number | null;
		}
	};
	// This set should only include Profiles for Edge-tier and Mid-tier Cache
	// Servers
	profiles: {
		[profileName: string]: {
			parameters: {
				[parameterName: string]: string;
			};
			thresholds: {
				[thresholdName: string]: {
					value: number;
					comparator: "<" | ">" | "<=" | ">=" | "=" | "!=";
				}
			}
		};
	};
	topologies: {
		[topologyName: string]: {
			nodes: Array<string>;
		};
	};
	trafficMonitors: {
		[hostName: string]: {
			fqdn: string;
			ipv4Address: string | null;
			ipv6Address: string | null;
			port: number;
			profile: string;
			status: string;
		};
	}
}
```

... and what follows is an example payload from the default CDN-in-a-Box data
set.

```json
{ "response": {
	"cacheGroups": {
		"CDN_in_a_Box_Mid-01": {
			"cacheServers": {
				"mid-01": {
					"deliveryServices": [],
					"httpsPort": 443,
					"interfaces": [{
						"ipAddresses": [{
							"address": "172.20.0.5",
							"gateway": "172.20.0.1",
							"serviceAddress": true
						}],
						"maxBandwidth": null,
						"monitor": true,
						"mtu": 1500,
						"name": "eth0"
					}],
					"profile": "ATS_MID_TIER_CACHE",
					"status": "REPORTED",
					"tcpPort": 80,
					"type": "MID"
				}
			},
			"latitude": 38.897663,
			"longitude": -77.036574
		},
		"CDN_in_a_Box_Mid-02": {
			"cacheServers": {
				"mid-02": {
					"httpsPort": 443,
					"interfaces": [{
						"ipAddresses": [{
							"address": "172.20.0.3",
							"gateway": "172.20.0.1",
							"serviceAddress": true
						}],
						"maxBandwidth": null,
						"monitor": true,
						"mtu": 1500,
						"name": "eth0"
					}],
					"profile": "ATS_MID_TIER_CACHE",
					"status": "REPORTED",
					"tcpPort": 80,
					"type": "MID"
				}
			},
			"latitude": 38.897663,
			"longitude": -77.036574
		},
		"CDN_in_a_Box_Edge": {
			"cacheServers": {
				"edge": {
					"httpsPort": 443,
					"interfaces": [{
						"ipAddresses": [{
							"address": "172.20.0.2",
							"gateway": "172.20.0.1",
							"serviceAddress": true
						}],
						"maxBandwidth": null,
						"monitor": true,
						"mtu": 1500,
						"name": "eth0"
					}],
					"profile": "ATS_EDGE_TIER_CACHE",
					"status": "REPORTED",
					"tcpPort": 80,
					"type": "EDGE"
				}
			},
			"latitude": 38.897663,
			"longitude": -77.036574
		}
	},
	"config": {
		"configPollingInterval": 2000,
		"eventCount": 200,
		"healthPad": 0,
		"healthPollingInterval": 6000,
		"heartbeatPollingInterval": 3000,
		"peerPollingInterval": 3000
	},
	"deliveryServices": {
		"demo1": {
			"globalMaxTps": 0,
			"globalMaxMbps": 0
		}
	},
	"profiles": {
		"ATS_MID_TIER_CACHE": {
			"parameters": {
				"health.connection.timeout": "2000",
				"health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
				"health.polling.format": "",
				"health.polling.type": "",
				"history.count": "30",
				"MinFreeKbps": "0"
			},
			"thresholds": {
				"health_threshold": {
					"availableBandwidthInKbps": {
						"value": 1750000,
						"comparator": ">"
					},
					"loadavg": {
						"value": 25,
						"comparator": "<"
					},
					"queryTime": {
						"value": 1000,
						"comparator": "<"
					}
				}
			}
		},
		"ATS_EDGE_TIER_CACHE": {
			"parameters": {
				"health.connection.timeout": "2000",
				"health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
				"health.polling.format": "",
				"health.polling.type": "",
				"history.count": "30",
				"MinFreeKbps": "0"
			},
			"thresholds": {
				"health_threshold": {
					"availableBandwidthInKbps": {
						"value": 1750000,
						"comparator": ">"
					},
					"loadavg": {
						"value": 25,
						"comparator": "<"
					},
					"queryTime": {
						"value": 1000,
						"comparator": "<"
					}
				}
			}
		}
	},
	"topologies": {
		"my-topology": {
			"nodes": ["CDN_in_a_Box_Edge"]
		}
	},
	"trafficMonitors": {
		"trafficmonitor": {
			"fqdn": "trafficmonitor.infra.ciab.test",
			"ipv4Address": "172.20.0.13",
			"ipv6Address": null,
			"port": 80,
			"profile": "RASCAL-Traffic_Monitor",
			"status": "ONLINE"
		}
	}
}}
```

### CDN Snapshots
```typescript
interface MaxmindDefaultOverride {
	countryCode: string;
	latitude: number;
	longitude: number;
}

// All of these properties used to be strings.
interface SOA {
	admin: string;
	expire: number;//integer
	minimum: number;//integer
	refresh: number;//integer
	retry: number;//integer
}

interface DNSBypassDestination {
	cname: string | null;
	ipv4Address: string | null;
	ipv6Address: string | null;
	ttl: number;//integer
}

interface HTTPBypassDestination {
	fqdn: string;
	port: number;//integer
}

interface DNSMatchItem {
	protocol: "DNS";
	// used to be named "matchlist"
	matchList: Array<{
		regex: string,
		// used to be named "match-type"
		type: "HOST"
	}>;
}

interface HTTPMatchItem {
	protocol: "HTTP";
	// used to be named "matchlist"
	matchList: Array<{
		regex: string,
		// used to be named "match-type"
		type: "HOST" | "PATH" | "HEADER"
	}>;
}

interface BaseDeliveryService {
	// used to be a string containing a boolean, instead of just a boolean.
	anonymousBlockingEnabled: boolean;
	// Formerly 'ttl' - now consistent with the DS property it represents.
	ccrDNSTTL: null | number;//integer
	consistentHashQueryParams: Array<string>;
	coverageZoneOnly: boolean;
	deepCachingType: "NEVER" | "ALWAYS";
	// used to be an object: {dispersion: number, shuffle: boolean} - but 'shuffle' was always true
	dispersion: number;//integer
	domains: Array<string>;
	ecsEnabled: boolean;
	// used to be an array of {countryCode: string}, this collapses single-property object into just that property
	// also used to be named "geoEnabled" - this name is consistent with the DS property
	geoLimitCountries: Array<string>;
	geoLimitRedirectURL: string | null;
	// Formerly 'geolocationProvider' - this name is consistent with the DS property.
	// Also used to be 'maxmindGeolocationService' or 'neustarGeolocationService'
	geoProvider: "MaxMind" | "Neustar";
	// used to be named "ip6RoutingEnabled"
	ipv6RoutingEnabled: boolean;
	// formerly "maxDNSIPsForLocation" - this name is consistent with the Delivery Service property name
	maxDNSAnswers: number | null;//integer
	missLocation: {
		// used to be named just "lat"
		latitude: number;
		// used to be named just "lon"
		longitude: number;
	};
	// this used to be expressed as {acceptHttps: boolean, redirectToHttps: boolean}
	// now expresses the same concept in the same format as in the rest of the API, and
	// with the added bonus of being incapable of expressing the invalid scenario:
	// {acceptHttps: false, redirectToHttps: true}
	protocol: "HTTP_ONLY" | "HTTPS_ONLY" | "HTTP_AND_HTTPS" | "HTTP_TO_HTTPS";
	regionalGeoBlocking: boolean;
	requiredCapabilities: Array<string>;
	routingName: string;
	soa: SOA;
	sslEnabled: boolean;
	staticDNSEntries: Array<{
		name: string;
		ttl: number;//integer
		type: string;
		value: string;
	}>;
	topology: string | null;
	// formerly named "requestHeaders" - now consistent with the DS property name
	trRequestHeaders: Array<string>;
	// formerly named "responseHeaders" - now consistent with the DS property name
	trResponseHeaders: Record<string, string>;
	// used to include "A" and "AAAA", but they were hard-coded to the value of ccrDNSTTL, so
	// they were superfluous.
	ttls: {
		NS: number;
		SOA: number;
	}
}

interface DNSDeliveryService extends BaseDeliveryService {
	// this used to be a mapping of "DNS" to a bypass destination.
	bypassDestination: null | DNSBypassDestination;
	// used to be named "matchsets"
	matchSets: Array<DNSMatchItem>;
}

interface HTTPDeliveryService extends BaseDeliveryService {
	// this used to be a mapping of "HTTP" to a bypass destination.
	bypassDestination: null | HTTPBypassDestination;
	// used to be named "matchsets"
	matchSets: Array<HTTPMatchItem>;
}

type DeliveryService = HTTPDeliveryService | DNSDeliveryService;

interface Snapshot {
	config: {
		// Formerly 'coverage.zone.polling.url' and optionally present (but
		// Traffic Router would raise an exception if not present).
		coverageZonePollingURL: string;
		// Formerly 'domain_name' and optionally present (but Traffic Router
		// would raise an exception if not present).
		domainName: string;
		// Formerly 'dnssec.enabled' and optionally present (but Traffic
		// Router would raise an exception if not present). Also used to be a
		// string containing a boolean, instead of a boolean.
		dnssecEnabled: boolean;
		// Formerly 'geolocation.polling.url' and optionally present (but
		// Traffic Router would raise an exception if not present).
		geoLocationPollingURL: string;
		// Formerly 'maxmindDefaultOverride' (singular).
		maxMindDefaultOverrides: Array<MaxmindDefaultOverride>;
		// Container for anything not explicitly required, used to be properties
		// of 'config' itself.
		parameters: {
			[parameterName: string]: string;
		};
		requestHeaders: Array<string>;
		soa: SOA;
		// These all used to be strings containing numbers.
		ttls: {
			A: number;//integer
			AAAA: number;//integer
			DNSKEY: number;//integer
			DS: number;//integer
			NS: number;//integer
			SOA: number;//integer
		}
	};
	deliveryServices: {
		[xmlID: string]: DeliveryService;
	};
	// Formerly "edgeLocations".
	edgeCacheGroups: {
		[cacheGroupName: string]: {
			// This used to be a property of Snapshots themselves, but the first thing
			// Traffic Router does is build this relationship of adding these to their
			// cache groups.
			cacheServers: {
				[hostName: string]: {
					capabilities: Array<string>;
					fqdn: string;
					hashCount: number;//integer
					// Formerly 'hashId'
					hashID: string;
					// Formerly 'port'
					httpsPort: number;//integer
					// Formerly 'ip'
					ipv4Address: string | null;
					// Formerly 'ip6'
					ipv6Address: string | null;
					tcpPort: number;//integer
				};
			};
			// used to be a property of the removed 'backupLocations' object
			fallbacks: Array<string>;
			// used to be a property of the removed 'backupLocations' object
			fallbackToClosest: boolean;
			latitude: number;
			localizationMethods: Array<"CZ" | "DEEP_CZ" | "GEO">;
			longitude: number;
		};
	};
	stats: {
		// Formerly 'CDN_name'
		cdn: string;
		date: number;//Unix epoch timestamp in seconds
		// Formerly 'tm_host'
		toHost: string;
	};
	topologies: {
		[topologyName: string]: {
			nodes: Array<string>;
		};
	};
	// formerly 'monitors'
	trafficMonitors: {
		[hostName: string]: {
			fqdn: string;
			httpsPort: number;//integer
			// Formerly 'ip'
			ipv4Address: null | string;
			// Formerly 'ip6'
			ipv6Address: null | string;
			// Formerly 'port'
			tcpPort: number;//integer
			status: string;
		};
	};
	// Formerly 'trafficRouterLocations'
	trafficRouterCacheGroups: {
		[cacheGroupName: string]: {
			latitude: number;
			longitude: number;
			// This used to be a property of Snapshots themselves, but the first thing
			// Traffic Router does is build this relationship of adding these to their
			// cache groups.
			trafficRouters: {
				[hostName: string]: {
					// Formerly 'api.port'
					apiPort: number;//integer
					fqdn: string;
					httpsPort: number;//integer
					// formerly 'ip'
					ipv4Address: string | null;
					// formerly 'ip6'
					ipv6Address: string | null;
					// Formerly 'secure.api.port'
					secureAPIPort: number;//integer
					// Formerly 'port'
					tcpPort: number;//integer
				};
			}
		};
	};
}
```

... and what follows is an example Snapshot from the default CDN-in-a-Box data set.

```json
{ "response": {
	"config": {
		"coveragezonePollingURL": "https://trafficops.infra.ciab.test:443/coverage-zone.json",
		"domainName": "mycdn.ciab.test",
		"dnssecEnabled": false,
		"geolocationPollingURL": "https://trafficops.infra.ciab.test:443/GeoLite2-City.mmdb.gz",
		"parameters": {
			"api.cache-control.max-age": "10",
			"certificates.polling.interval": "300000",
			"consistent.dns.routing": "true",
			"coveragezone.polling.interval": "3600000",
			"dnssec.dynamic.response.expiration": "300s",
			"federationmapping.polling.interval": "60000",
			"federationmapping.polling.url": "https://${toHostname}/api/2.0/federations/all",
			"geolocation.polling.interval": "86400000",
			"keystore.maintenance.interval": "300",
			"neustar.polling.interval": "86400000",
			"steeringmapping.polling.interval": "60000",
			"neustar.polling.url": "https://trafficops.infra.ciab.test:443/neustar.tar.gz",
			"zonemanager.cache.maintenance.interval": "300",
			"zonemanager.threadpool.scale": "0.50"
		},
		"soa": {
			"admin": "twelve_monkeys",
			"expire": 604800,
			"minimum": 30,
			"refresh": 28800,
			"retry": 7200
		},
		"ttls": {
			"A": 3600,
			"AAAA": 3600,
			"DNSKEY": 30,
			"DS": 30,
			"NS": 3600,
			"SOA": 86400
		}
	},
	"deliveryServices": {
		"demo1": {
			"anonymousBlockingEnabled": false,
			"ccrDNSTTL": null,
			"consistentHashQueryParams": [
				"abc",
				"pdq",
				"xxx",
				"zyx"
			],
			"coverageZoneOnly": false,
			"deepCachingType": "NEVER",
			"dispersion": 1,
			"domains": [
				"demo1.mycdn.ciab.test"
			],
			"ecsEnabled": false,
			"geoProvider": "maxmindGeolocationService",
			"ipv6RoutingEnabled": true,
			"maxDNSAnswers": null,
			"matchsets": [
				{
					"protocol": "HTTP",
					"matchList": [
						{
							"regex": ".*\\.demo1\\..*",
							"type": "HOST"
						}
					]
				}
			],
			"missLocation": {
				"latitude": 42,
				"longitude": -88
			},
			"protocol": "HTTP_AND_HTTPS",
			"regionalGeoBlocking": false,
			"requiredCapabilities": [],
			"routingName": "video",
			"soa": {
				"admin": "traffic_ops",
				"expire": 604800,
				"minimum": 30,
				"refresh": 28800,
				"retry": 7200
			},
			"sslEnabled": true,
			"staticDNSEntries": [],
			"topology": "demo1-top",
			"trRequestHeaders": [],
			"trResponseHeaders": [],
			"ttls": {
				"NS": 3600,
				"SOA": 86400
			}
		}
	},
	"edgeCacheGroups": {
		"CDN_in_a_Box_Edge": {
			"cacheServers": {
				"edge": {
					"capabilities": [],
					"fqdn": "edge.infra.ciab.test",
					"hashCount": 999,
					"hashID": "21dff858-384b-4e47-8c8c-b61f8337aa9b",
					"httpsPort": 443,
					"ipv4Address": "172.19.0.2",
					"ipv6Address": null,
					"tcpPort": 80
				}
			},
			"fallbacks": [],
			"fallbackToClosest": "true",
			"latitude": 38.897663,
			"longitude": -77.036574,
			"localizationMethods": [
				"GEO",
				"CZ",
				"DEEP_CZ"
			]
		}
	},
	"trafficRouterCacheGroups": {
		"CDN_in_a_Box_Edge": {
			"latitude": 38.897663,
			"longitude": -77.036574,
			"trafficRouters": {
				"trafficrouter": {
					"apiPort": 3333,
					"fqdn": "trafficrouter.infra.ciab.test",
					"httpsPort": 443,
					"ipv4Address": "172.19.0.14",
					"ipv6Address": null,
					"secure.api.port": 3443,
					"tcpPort": 80
				}
			}
		}
	},
	"trafficMonitors": {
		"trafficmonitor": {
			"fqdn": "trafficmonitor.infra.ciab.test",
			"httpsPort": 443,
			"ipv4Address": "172.19.0.12",
			"ipv6Address": null,
			"status": "ONLINE",
			"tcpPort": 80
		}
	},
	"stats": {
		"cdn": "CDN-in-a-Box",
		"date": 1607615441,
		"toHost": "trafficops.infra.ciab.test:443"
	},
	"topologies": {
		"demo1-top": {
			"nodes": [
				"CDN_in_a_Box_Edge"
			]
		}
	}
}}
```

## Traffic Portal Impact
Traffic Portal shouldn't _need_ any changes, but it should ideally show
Monitoring payloads in its Snapshot diff page, and there's never a better time
than when big changes are being made to both, to help reduce operaton
confusion.

## Traffic Ops Impact
### API Impact
The CDN Snapshot and Monitoring handlers will need to be updated to output the
new formats, which will necessitate a new major API version. This includes the
following endpoints:

- `/cdns/{{name}}/configs/monitoring`
- `/cdns/{{name}}/snapshot/new`
- `/cdns/{{name}}/snapshot`
- `/snapshot` (this, however, shares most of its logic with `/cdns/{{name}}/snapshot/new`)

New logic will also need to be added to convert from the new snapshot (and
monitoring payload) formats back to the old formats, so that newly stored
Snapshot (and monitoring payload) data can be served in old API versions.

### Client Impact
The APIv4 client will need to change the call signatures of the affected
endpoint methods.

### Database Impact
Existing stored Snapshots (and monitoring payload data) will need to be
converted to the new format via a migration. No schema changes will be
necessary, because Snapshots (and monitoring payload data) are stored as
JSON blobs.

## ORT Impact
None.

## Traffic Monitor Impact
Traffic Monitor will need to be overhauled; Snapshot dependency is ingrained
fairly deep in it. It will need to build out its configuration and stats from
only the monitoring payload. It will also need to keep serving the Snapshots,
however, to support old versions of Traffic Router.

Traffic Monitor's parsing of monitoring payloads will also need to be updated
to handle the new format - but as always, a previous API version must still be
supported.

## Traffic Router Impact
Traffic Router will need to be updated to handle the new Snapshot format, as
well as maintaining compatibility with the old format. It will also need to be
updated to fetch Snapshots from Traffic Ops directly, rather than through
Traffic Monitor.

## Traffic Stats Impact
None.

## Traffic Vault Impact
None.

## Documentation Impact
Documentation for CDN Snapshots and Monitoring payloads will need to be updated
to reflect the new formats. This includes all of the affected endpoints'
documentation, but should also include an overview section for each, now that
the typing is more solidly defined.

## Testing Impact
Existing tests for each affected endpoint will need to be updated. Also, tests
should be written for the logic for converting from the new formats to legacy
formats, as well as for the new parsing logic in Traffic Monitor and Traffic
Ops.

## Performance Impact
Many of the fields of Snapshots and Monitoring payloads have been changed from
strings containing arbitrary values to statically typed properties. This
generally means that Traffic Ops will need to spend some extra time parsing
some properties - although sometimes the reverse is true where TO is currently
transforming booleans into strings representing booleans. That will slightly
increase the load on Traffic Ops.

However, on the other side (Traffic Monitor and Traffic Router), less time will
be spent doing the same parsing in reverse, so some performance gains will be
made. Also, some mappings (most notably from a server's
"cacheGroup"/"locationID" property to an entry in a 'cacheGroups' map) will be
done on the Traffic Ops side, which incurs a negligible cost to TO itself but
saves potentially significant time reproducing those mappings manually on the
Traffic Monitor and Traffic Router side. Overall the expected performance
impact is positive.

## Security Impact
There should be no changes from a security standpoint.

## Upgrade Impact
This should not enforce an upgrade order for components.

## Operations Impact
Operators that deal with CDN Snapshot "diffs" on a regular basis will need to
be made aware of the new property names and structure, as well as possibly the
removed properties. Hopefully the naming scheme should be intuitive, as it
strives to be close as possible to the names of the API object properties one
manipulates to generate different Monitoring and Snapshot configurations.

## Developer Impact
The code should ideally be easier to understand with the more consistent naming
scheme and stronger typing. There should be no impact on development
environments and/or their setup.

## Alternatives
One other alternative that was suggested was eliminating one of the
configuration blobs and collapsing the data it contained into the other.
Ultimately it was decided that this would lead to transferring payloads that
contained far too much unrelated data and/or adding extra functionality to
Traffic Monitor to produce minimal Traffic Router configuration upon request -
which could potentially cause significant resource usage.

## Dependencies
None.

## References
<a name="typescript">1:</a> The syntax should be mostly self-explanatory, but
see their [official documentation for a five-minute introduction](https://www.typescriptlang.org/docs/handbook/typescript-in-5-minutes.html)
