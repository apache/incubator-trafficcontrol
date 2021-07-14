/*
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package org.apache.traffic_control.traffic_router.core.router;

import org.apache.traffic_control.traffic_router.configuration.ConfigurationListener;
import org.apache.traffic_control.traffic_router.core.dns.DNSAccessRecord;
import org.apache.traffic_control.traffic_router.core.dns.ZoneManager;
import org.apache.traffic_control.traffic_router.core.ds.DeliveryService;
import org.apache.traffic_control.traffic_router.core.ds.Steering;
import org.apache.traffic_control.traffic_router.core.ds.SteeringGeolocationComparator;
import org.apache.traffic_control.traffic_router.core.ds.SteeringRegistry;
import org.apache.traffic_control.traffic_router.core.ds.SteeringResult;
import org.apache.traffic_control.traffic_router.core.ds.SteeringTarget;
import org.apache.traffic_control.traffic_router.core.edge.Cache;
import org.apache.traffic_control.traffic_router.core.edge.CacheLocation;
import org.apache.traffic_control.traffic_router.core.edge.CacheLocation.LocalizationMethod;
import org.apache.traffic_control.traffic_router.core.edge.CacheRegister;
import org.apache.traffic_control.traffic_router.core.edge.InetRecord;
import org.apache.traffic_control.traffic_router.core.edge.Location;
import org.apache.traffic_control.traffic_router.core.edge.Node;
import org.apache.traffic_control.traffic_router.core.edge.Node.IPVersions;
import org.apache.traffic_control.traffic_router.core.edge.TrafficRouterLocation;
import org.apache.traffic_control.traffic_router.core.hash.ConsistentHasher;
import org.apache.traffic_control.traffic_router.core.loc.AnonymousIp;
import org.apache.traffic_control.traffic_router.core.loc.AnonymousIpDatabaseService;
import org.apache.traffic_control.traffic_router.core.loc.FederationRegistry;
import org.apache.traffic_control.traffic_router.core.loc.MaxmindGeolocationService;
import org.apache.traffic_control.traffic_router.core.loc.NetworkNode;
import org.apache.traffic_control.traffic_router.core.loc.NetworkNodeException;
import org.apache.traffic_control.traffic_router.core.loc.RegionalGeo;
import org.apache.traffic_control.traffic_router.core.request.DNSRequest;
import org.apache.traffic_control.traffic_router.core.request.HTTPRequest;
import org.apache.traffic_control.traffic_router.core.request.Request;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track.ResultDetails;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track.ResultType;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track.RouteType;
import org.apache.traffic_control.traffic_router.core.util.CidrAddress;
import org.apache.traffic_control.traffic_router.core.util.JsonUtils;
import org.apache.traffic_control.traffic_router.core.util.TrafficOpsUtils;
import org.apache.traffic_control.traffic_router.geolocation.Geolocation;
import org.apache.traffic_control.traffic_router.geolocation.GeolocationException;
import org.apache.traffic_control.traffic_router.geolocation.GeolocationService;
import com.fasterxml.jackson.databind.JsonNode;
import org.apache.log4j.Logger;
import org.springframework.beans.BeansException;
import org.springframework.context.ApplicationContext;
import org.xbill.DNS.Name;
import org.xbill.DNS.Type;
import org.xbill.DNS.Zone;

import java.io.IOException;
import java.net.InetAddress;
import java.net.MalformedURLException;
import java.net.URL;
import java.net.UnknownHostException;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Collections;
import java.util.Comparator;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Random;
import java.util.Set;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

/**
 * TrafficRouter is the main router class that handles Traffic Router logic.
 */
@SuppressWarnings({"PMD.TooManyFields", "PMD.ExcessivePublicCount"})
public class TrafficRouter {
	public static final Logger LOGGER = Logger.getLogger(TrafficRouter.class);

	/**
	 * This is an HTTP Header the value of which, if present in a client HTTP request, should be
	 * the XMLID of a Delivery Service to use as an explicit target in CLIENT_STEERING (thus
	 * bypassing normal steering logic).
	 */
	public static final String XTC_STEERING_OPTION = "x-tc-steering-option";

	/**
	 * This is the key of a JSON object that is a configuration option that may be present in
	 * "CRConfig" Snapshots. When this option is present, and is 'true', more Edge-Tier cache
	 * servers will be provided in responses to steering requests (known as "Client Steering Forced
	 * Diversity").
	 */
	public static final String CLIENT_STEERING_DIVERSITY = "client.steering.forced.diversity";
	public static final String DNSSEC_ENABLED = "dnssec.enabled";
	public static final String DNSSEC_ZONE_DIFFING = "dnssec.zone.diffing.enabled";
	public static final String DNSSEC_RRSIG_CACHE_ENABLED = "dnssec.rrsig.cache.enabled";
	private static final long DEFAULT_EDGE_NS_TTL = 3600;
	private static final int DEFAULT_EDGE_TR_LIMIT = 4;

	private final CacheRegister cacheRegister;
	private final ZoneManager zoneManager;
	private final GeolocationService geolocationService;
	private final GeolocationService geolocationService6;
	private final AnonymousIpDatabaseService anonymousIpService;
	private final FederationRegistry federationRegistry;
	private final boolean consistentDNSRouting;
	private final boolean clientSteeringDiversityEnabled;
	private final boolean dnssecZoneDiffingEnabled;
	private final boolean edgeDNSRouting;
	private final boolean edgeHTTPRouting;
	private final long edgeNSttl; // 1 hour default
	private final int edgeDNSRoutingLimit;
	private final int edgeHTTPRoutingLimit;

	private final Random random = new Random(System.nanoTime());
	private Set<String> requestHeaders = new HashSet<String>();
	private static final Geolocation GEO_ZERO_ZERO = new Geolocation(0, 0);
	private ApplicationContext applicationContext;

	private final ConsistentHasher consistentHasher = new ConsistentHasher();
	private SteeringRegistry steeringRegistry;

	private final Map<String, Geolocation> defaultGeolocationsOverride = new HashMap<String, Geolocation>();

	/**
	 * When instantiated, Traffic Router will try to read all of its various configuration files.
	 *
	 * @throws IOException when an error occurs reading in a configuration file.
	 */
	public TrafficRouter(final CacheRegister cr,
			final GeolocationService geolocationService,
			final GeolocationService geolocationService6,
			final AnonymousIpDatabaseService anonymousIpService,
			final StatTracker statTracker,
			final TrafficOpsUtils trafficOpsUtils,
			final FederationRegistry federationRegistry,
			final TrafficRouterManager trafficRouterManager) throws IOException {
		this.cacheRegister = cr;
		this.geolocationService = geolocationService;
		this.geolocationService6 = geolocationService6;
		this.anonymousIpService = anonymousIpService;
		this.federationRegistry = federationRegistry;
		this.clientSteeringDiversityEnabled = JsonUtils.optBoolean(cr.getConfig(), CLIENT_STEERING_DIVERSITY);
		this.dnssecZoneDiffingEnabled = JsonUtils.optBoolean(cr.getConfig(), DNSSEC_ENABLED) && JsonUtils.optBoolean(cr.getConfig(), DNSSEC_ZONE_DIFFING);
		this.consistentDNSRouting = JsonUtils.optBoolean(cr.getConfig(), "consistent.dns.routing"); // previous/default behavior
		this.edgeDNSRouting =  JsonUtils.optBoolean(cr.getConfig(), "edge.dns.routing") && cr.hasEdgeTrafficRouters();
		this.edgeHTTPRouting = JsonUtils.optBoolean(cr.getConfig(), "edge.http.routing") && cr.hasEdgeTrafficRouters();

		if (cr.getConfig() != null) {
			// maxmindDefaultOverride: {countryCode: , lat: , long: }
			final JsonNode geolocations = cr.getConfig().get("maxmindDefaultOverride");
			if (geolocations != null) {
				for (final JsonNode geolocation : geolocations) {
					final String countryCode = JsonUtils.optString(geolocation, "countryCode");
					final double lat = JsonUtils.optDouble(geolocation, "lat");
					final double longitude = JsonUtils.optDouble(geolocation, "long");
					defaultGeolocationsOverride.put(countryCode, new Geolocation(lat, longitude));
				}
			}
		}

		final JsonNode ttls = cacheRegister.getConfig().get("ttls");

		if (ttls != null && ttls.has("NS")) {
			this.edgeNSttl = JsonUtils.optLong(ttls, "NS");
		} else {
			this.edgeNSttl = DEFAULT_EDGE_NS_TTL;
		}

		this.edgeDNSRoutingLimit = JsonUtils.optInt(cr.getConfig(), "edge.dns.limit", DEFAULT_EDGE_TR_LIMIT);
		this.edgeHTTPRoutingLimit = JsonUtils.optInt(cr.getConfig(), "edge.http.limit", DEFAULT_EDGE_TR_LIMIT); // NOTE: this can be overridden per-DS via maxDnsAnswers
		this.zoneManager = new ZoneManager(this, statTracker, trafficOpsUtils, trafficRouterManager);
	}

	public ZoneManager getZoneManager() {
		return zoneManager;
	}

	/**
	 * Returns a {@link List} of all of the online {@link Cache}s that support the specified
	 * {@link DeliveryService}. If no online caches are found to support the specified
	 * DeliveryService an empty list is returned.
	 *
	 * @param ds
	 *            the DeliveryService to check
	 * @return collection of supported caches
	 */
	public List<Cache> getSupportingCaches(final List<Cache> caches, final DeliveryService ds, final IPVersions requestVersion) {
		final List<Cache> supportingCaches = new ArrayList<Cache>();

		for (final Cache cache : caches) {
			if (!cache.hasDeliveryService(ds.getId())) {
				continue;
			}

			if (!cache.hasAuthority() || (cache.isAvailable(requestVersion))) {
				supportingCaches.add(cache);
			}
		}

		return supportingCaches;
	}

	public CacheRegister getCacheRegister() {
		return cacheRegister;
	}

	/**
	 * Selects a Delivery Service to service a request.
	 *
	 * @param request The request being served
	 * @return A Delivery Service to use when servicing the request.
	 */
	protected DeliveryService selectDeliveryService(final Request request) {
		if (cacheRegister == null) {
			LOGGER.warn("no caches yet");
			return null;
		}

		final DeliveryService deliveryService = cacheRegister.getDeliveryService(request);

		if (LOGGER.isDebugEnabled()) {
			LOGGER.debug("Selected DeliveryService: " + deliveryService);
		}
		return deliveryService;
	}

	/**
	 * Sets the cache server and Delivery Service "states" based on input JSON.
	 * <p>
	 * The input {@code states} is expected to be an object with (at least) two keys:
	 * "caches" and "deliveryServices", which contain the states of the cache servers and
	 * Delivery Services, respectively. @see #setDsStates(JsonNode) and
	 *  {@link #setCacheStates(JsonNode)} for the expected format of those objects themselves.
	 * </p>
	 * @return Always returns {@code true} when successful. On failure, throws.
	 */
	boolean setState(final JsonNode states) throws UnknownHostException {
		setCacheStates(states.get("caches"));
		setDsStates(states.get("deliveryServices"));
		return true;
	}

	/**
	 * Sets Delivery Service states based on the input JSON.
	 * <p>
	 * Delivery Services present in the input which aren't registered are ignored.
	 * </p>
	 * @param dsStates The input JSON object. Expected to be a map of Delivery Service XMLIDs to
	 * "state" strings.
	 * @return {@code false} iff dsStates was {@code null}, otherwise {@code true}.
	 */
	private boolean setDsStates(final JsonNode dsStates) {
		if(dsStates == null) {
			return false;
		}
		final Map<String, DeliveryService> dsMap = cacheRegister.getDeliveryServices();
		for (final String dsName : dsMap.keySet()) {
			dsMap.get(dsName).setState(dsStates.get(dsName));
		}
		return true;
	}

	/**
	 * Sets {@link Cache} states based on the input JSON.
	 * <p>
	 * Caches present in the input which are not registered are ignored.
	 * </p>
	 * @param cacheStates The input JSON object. Expected to be a map of identifying Cache names
	 * to "state" strings.
	 * @return {@code false} iff cacheStates was {@code null}, otherwise {@code true}.
	 */
	private boolean setCacheStates(final JsonNode cacheStates) {
		if(cacheStates == null) {
			return false;
		}
		final Map<String, Cache> cacheMap = cacheRegister.getCacheMap();
		if (cacheMap == null) {
			return false;
		}
		for (final String cacheName : cacheMap.keySet()) {
			final String monitorCacheName = cacheName.replaceFirst("@.*", "");
			final JsonNode state = cacheStates.get(monitorCacheName);
			cacheMap.get(cacheName).setState(state);
		}
		return true;
	}

	protected static final String UNABLE_TO_ROUTE_REQUEST = "Unable to route request.";
	protected static final String URL_ERR_STR = "Unable to create URL.";

	public GeolocationService getGeolocationService() {
		return geolocationService;
	}

	public AnonymousIpDatabaseService getAnonymousIpDatabaseService() {
		return anonymousIpService;
	}

	/**
	 * Geo-locates the client returning a physical location for routing purposes.
	 *
	 * @param clientIP The client's network location - as a {@link String}. This should ideally be
	 * an IP address, but trailing port number specifications are stripped.
	 * @throws GeolocationException if the client could not be located.
	 */
	public Geolocation getLocation(final String clientIP) throws GeolocationException {
		return clientIP.contains(":") ? geolocationService6.location(clientIP) : geolocationService.location(clientIP);
	}

	/**
	 * Retrieves a service for geo-locating clients for a specific Delivery Service.
	 *
	 * @param geolocationProvider The name of the provider for geo-location information (currently
	 * only "Maxmind" and "Neustar" are supported)
	 * @param deliveryServiceId Currently only used for logging error information, should be an
	 * identifier for a Delivery Service
	 * @return A {@link GeolocationService} that can be used to geo-locate clients <em>or</em>
	 * {@code null} if an error occurs.
	 */
	private GeolocationService getGeolocationService(final String geolocationProvider, final String deliveryServiceId) {
		if (applicationContext == null) {
			LOGGER.error("ApplicationContext not set unable to use custom geolocation service providers");
			return null;
		}

		if (geolocationProvider == null || geolocationProvider.isEmpty()) {
			return null;
		}

		try {
			return (GeolocationService) applicationContext.getBean(geolocationProvider);
		} catch (Exception e) {
			StringBuilder error = new StringBuilder("Failed getting providing class '" + geolocationProvider + "' for geolocation");
			if (deliveryServiceId != null && !deliveryServiceId.isEmpty()) {
				error = error.append(" for delivery service " + deliveryServiceId);
			}
			error = error.append(" falling back to " + MaxmindGeolocationService.class.getSimpleName());
			LOGGER.error(error);
		}

		return null;
	}

	/**
	 * Retrieves a location for a given client being served a given Delivery Service using a
	 * specific provider.
	 * @param clientIP The client's network location - as a {@link String}. This should ideally be
	 * an IP address, but trailing port number specifications are stripped.
	 * @param geolocationProvider The name of the provider for geo-location information (currently
	 * only "Maxmind" and "Neustar" are supported)
	 * @param deliveryServiceId Currently only used for logging error information, should be an
	 * identifier for a Delivery Service
	 * @throws GeolocationException if the client could not be located.
	 */
	public Geolocation getLocation(final String clientIP, final String geolocationProvider, final String deliveryServiceId) throws GeolocationException {
		final GeolocationService customGeolocationService = getGeolocationService(geolocationProvider, deliveryServiceId);
		return customGeolocationService != null ? customGeolocationService.location(clientIP) : getLocation(clientIP);
	}

	/**
	 * Retrieves a location for a given client being served a given Delivery Service.
	 * @param clientIP The client's network location - as a {@link String}. This should ideally be
	 * an IP address, but trailing port number specifications are stripped.
	 * @param deliveryService The Delivery Service being served to the client.
	 * @throws GeolocationException if the client could not be located.
	 */
	public Geolocation getLocation(final String clientIP, final DeliveryService deliveryService) throws GeolocationException {
		return getLocation(clientIP, deliveryService.getGeolocationProvider(), deliveryService.getId());
	}

	/**
	 * Gets a {@link List} of {@link Cache}s that are capabable of serving a given Delivery Service.
	 * <p>
	 * The caches chosen are from the closest, non-empty, cache location to the client's physical
	 * location, up to the Location Limit ({@link DeliveryService#getLocationLimit()}) of the
	 * Delivery Service being served.
	 * </p>
	 * @param ds The Delivery Service being served.
	 * @param clientLocation The physical location of the requesting client.
	 * @param track The {@link Track} object on which a result location shall be set, should one be found
	 * @return A {@link List} of {@link Cache}s that should be used to service a request should such a collection be found, or
	 * {@code null} if the no applicable {@link Cache}s could be found.
	 */
	public List<Cache> getCachesByGeo(final DeliveryService ds, final Geolocation clientLocation, final Track track, final IPVersions requestVersion) throws GeolocationException {
		int locationsTested = 0;

		final int locationLimit = ds.getLocationLimit();
		final List<CacheLocation> geoEnabledCacheLocations = filterEnabledLocations(getCacheRegister().getCacheLocations(), LocalizationMethod.GEO);
		final List<CacheLocation> cacheLocations1 = ds.filterAvailableLocations(geoEnabledCacheLocations);
		@SuppressWarnings("unchecked")
		final List<CacheLocation> cacheLocations = (List<CacheLocation>) orderLocations(cacheLocations1, clientLocation);

		for (final CacheLocation location : cacheLocations) {
			final List<Cache> caches = selectCaches(location, ds, requestVersion);
			if (caches != null) {
				track.setResultLocation(location.getGeolocation());
				if (track.getResultLocation().equals(GEO_ZERO_ZERO)) {
					LOGGER.error("Location " + location.getId() + " has Geolocation " + location.getGeolocation());
				}
				return caches;
			}
			locationsTested++;
			if (locationLimit != 0 && locationsTested >= locationLimit) {
				return null;
			}
		}

		return null;
	}

	/**
	 * Selects {@link Cache}s to serve a request for a Delivery Service.
	 * <p>
	 * This is equivalent to calling
	 * {@link #selectCaches(HTTPRequest, DeliveryService, Track, boolean)} with the "deep" parameter
	 * set to {@code true}.
	 * </p>
	 * @param request The HTTP request made by the client.
	 * @param ds The Delivery Service being served.
	 * @param track The {@link Track} object that tracks how requests are served
	 */
	protected List<Cache> selectCaches(final HTTPRequest request, final DeliveryService ds, final Track track) throws GeolocationException {
		return selectCaches(request, ds, track, true);
	}

	/**
	 * Selects {@link Cache}s to serve a request for a Delivery Service.
	 * @param request The HTTP request made by the client.
	 * @param ds The Delivery Service being served.
	 * @param track The {@link Track} object that tracks how requests are served
	 * @param enableDeep Sets whether or not "Deep Caching" may be used.
	 */
	@SuppressWarnings("PMD.CyclomaticComplexity")
	protected List<Cache> selectCaches(final HTTPRequest request, final DeliveryService ds, final Track track, final boolean enableDeep) throws GeolocationException {
		CacheLocation cacheLocation;
		ResultType result = ResultType.CZ;
		final boolean useDeep = enableDeep && (ds.getDeepCache() == DeliveryService.DeepCachingType.ALWAYS);
		final IPVersions requestVersion = request.getClientIP().contains(":") ? IPVersions.IPV6ONLY : IPVersions.IPV4ONLY;

		if (useDeep) {
			// Deep caching is enabled. See if there are deep caches available
			cacheLocation = getDeepCoverageZoneCacheLocation(request.getClientIP(), ds, requestVersion);
			if (cacheLocation != null && cacheLocation.getCaches().size() != 0) {
				// Found deep caches for this client, and there are caches that might be available there.
				result = ResultType.DEEP_CZ;
			} else {
				// No deep caches for this client, would have used them if there were any. Fallback to regular CZ
				cacheLocation = getCoverageZoneCacheLocation(request.getClientIP(), ds, requestVersion);
			}
		} else {
			// Deep caching not enabled for this Delivery Service; use the regular CZ
			cacheLocation = getCoverageZoneCacheLocation(request.getClientIP(), ds, false, track, requestVersion);
		}

		List<Cache>caches = selectCachesByCZ(ds, cacheLocation, track, result, requestVersion);

		if (caches != null) {
			return caches;
		}

		if (ds.isCoverageZoneOnly()) {
			if (ds.getGeoRedirectUrl() != null) {
				//use the NGB redirect
				caches = enforceGeoRedirect(track, ds, request.getClientIP(), null, requestVersion);
			} else {
				track.setResult(ResultType.MISS);
				track.setResultDetails(ResultDetails.DS_CZ_ONLY);
			}
		} else if (track.continueGeo) {
			// continue Geo can be disabled when backup group is used -- ended up an empty cache list if reach here
			caches = selectCachesByGeo(request.getClientIP(), ds, cacheLocation, track, requestVersion);
		}

		return caches;
	}

	/**
	 * Returns whether or not a Delivery Service has a valid miss location.
	 * @param deliveryService The Delivery Service being served.
	 */
	public boolean isValidMissLocation(final DeliveryService deliveryService) {
		if (deliveryService.getMissLocation() != null && deliveryService.getMissLocation().getLatitude() != 0.0 && deliveryService.getMissLocation().getLongitude() != 0.0) {
			return true;
		}
		return false;
	}

	/**
	 * Selects {@link Cache}s to serve a request for a Delivery Service based on a given location.
	 * @param clientIp The requesting client's IP address - as a String.
	 * @param deliveryService The Delivery Service being served.
	 * @param cacheLocation A selected {@link CacheLocation} from which {@link Cache}s will be
	 * extracted based on the client's location.
	 * @param track The {@link Track} object that tracks how requests are served
	 */
	public List<Cache> selectCachesByGeo(final String clientIp, final DeliveryService deliveryService, final CacheLocation cacheLocation, final Track track, final IPVersions requestVersion) throws GeolocationException {
		Geolocation clientLocation = null;

		try {
			clientLocation = getClientLocation(clientIp, deliveryService, cacheLocation, track);
		} catch (GeolocationException e) {
			LOGGER.warn("Failed looking up Client GeoLocation: " + e.getMessage());
		}

		if (clientLocation == null) {
			if (deliveryService.getGeoRedirectUrl() != null) {
				//will use the NGB redirect
				LOGGER.debug(String
						.format("client is blocked by geolimit, use the NGB redirect url: %s",
								deliveryService.getGeoRedirectUrl()));
				return enforceGeoRedirect(track, deliveryService, clientIp, track.getClientGeolocation(), requestVersion);
			} else {
				track.setResultDetails(ResultDetails.DS_CLIENT_GEO_UNSUPPORTED);
				return null;
			}
		}

		track.setResult(ResultType.GEO);
		if (clientLocation.isDefaultLocation() && getDefaultGeoLocationsOverride().containsKey(clientLocation.getCountryCode())) {
			if (isValidMissLocation(deliveryService)) {
				clientLocation = deliveryService.getMissLocation();
				track.setResult(ResultType.GEO_DS);
			} else {
				clientLocation = getDefaultGeoLocationsOverride().get(clientLocation.getCountryCode());
			}
		}

		final List<Cache> caches = getCachesByGeo(deliveryService, clientLocation, track, requestVersion);

		if (caches == null || caches.isEmpty()) {
			track.setResultDetails(ResultDetails.GEO_NO_CACHE_FOUND);
		}
		return caches;
	}

	/**
	 * Routes a single DNS request.
	 * @param request The client request being routed.
	 * @param track A "tracker" that tracks the results of routing.
	 * @return The final result of routing.
	 */
	public DNSRouteResult route(final DNSRequest request, final Track track) throws GeolocationException {
		final DeliveryService ds = selectDeliveryService(request);

		track.setRouteType(RouteType.DNS, request.getHostname());

		// TODO: getHostname or getName -- !ds.getRoutingName().equalsIgnoreCase(request.getHostname().split("\\.")[0]))
		if (ds != null && ds.isDns() && request.getName().toString().toLowerCase().matches(ds.getRoutingName().toLowerCase() + "\\..*")) {
			return getEdgeCaches(request, ds, track);
		} else {
			return getEdgeTrafficRouters(request, ds, track);
		}
	}

	private DNSRouteResult getEdgeTrafficRouters(final DNSRequest request, final DeliveryService ds, final Track track) throws GeolocationException {
		final DNSRouteResult result = new DNSRouteResult();

		result.setDeliveryService(ds);
		result.setAddresses(selectTrafficRouters(request, ds, track));

		return result;
	}

	private List<InetRecord> selectTrafficRouters(final DNSRequest request, final DeliveryService ds) throws GeolocationException {
		return selectTrafficRouters(request, ds, null);
	}

	private List<InetRecord> selectTrafficRouters(final DNSRequest request, final DeliveryService ds, final Track track) throws GeolocationException {
		final List<InetRecord> result = new ArrayList<>();
		ResultType resultType = null;

		if (track != null) {
			track.setResultDetails(ResultDetails.LOCALIZED_DNS);
		}

		Geolocation clientGeolocation = null;

		final NetworkNode networkNode = getNetworkNode(request.getClientIP());

		if (networkNode != null && networkNode.getGeolocation() != null) {
			clientGeolocation = networkNode.getGeolocation();
			resultType = ResultType.CZ;
		} else {
			clientGeolocation = getClientGeolocation(request.getClientIP(), track, ds);
			resultType = ResultType.GEO;
		}

		if (clientGeolocation == null) {
			result.addAll(selectTrafficRoutersMiss(request.getZoneName(), ds));
			resultType = ResultType.MISS;
		} else {
			result.addAll(selectTrafficRoutersLocalized(clientGeolocation, request.getZoneName(), ds, track, request.getQueryType()));

			if (track != null) {
				track.setClientGeolocation(clientGeolocation);
			}
		}

		if (track != null) {
			track.setResult(resultType);
		}

		return result;
	}

	@SuppressWarnings({"PMD.CyclomaticComplexity", "PMD.NPathComplexity"})
	public List<InetRecord> selectTrafficRoutersMiss(final String zoneName, final DeliveryService ds) throws GeolocationException {
		final List<InetRecord> trafficRouterRecords = new ArrayList<>();

		if (!isEdgeDNSRouting() && !isEdgeHTTPRouting()) {
			return trafficRouterRecords;
		}

		final List<TrafficRouterLocation> trafficRouterLocations = getCacheRegister().getEdgeTrafficRouterLocations();
		final List<Node> edgeTrafficRouters = new ArrayList<>();
		final Map<String, List<Node>> orderedNodes = new HashMap<>();

		int limit = (getEdgeDNSRoutingLimit() > getEdgeHTTPRoutingLimit(ds)) ? getEdgeDNSRoutingLimit() : getEdgeHTTPRoutingLimit(ds);
		int index = 0;
		boolean exhausted = false;

		// if limits don't exist, or do exist and are higher than the number of edge TRs, use the number of edge TRs as the limit
		if (limit == 0 || limit > getCacheRegister().getEdgeTrafficRouterCount()) {
			limit = getCacheRegister().getEdgeTrafficRouterCount();
		}

		// grab one TR per location until the limit is reached
		while (edgeTrafficRouters.size() < limit && !exhausted) {
			final int initialCount = edgeTrafficRouters.size();

			for (final TrafficRouterLocation location : trafficRouterLocations) {
			    if (edgeTrafficRouters.size() >= limit) {
					break;
				}

				if (!orderedNodes.containsKey(location.getId())) {
					orderedNodes.put(location.getId(), consistentHasher.selectHashables(location.getTrafficRouters(), zoneName));
				}

				final List<Node> trafficRouters = orderedNodes.get(location.getId());

				if (trafficRouters == null || trafficRouters.isEmpty() || index >= trafficRouters.size()) {
					continue;
				}

				edgeTrafficRouters.add(trafficRouters.get(index));
			}

			/*
			 * we iterated through every location and attempted to add edge TR at index, but none were added....
			 * normally, these values would never match unless we ran out of options...
			 * if so, we've exhausted our options so we need to break out of the while loop
			 */
			if (initialCount == edgeTrafficRouters.size()) {
				exhausted = true;
			}

			index++;
		}

		if (!edgeTrafficRouters.isEmpty()) {
			if (isEdgeDNSRouting()) {
				trafficRouterRecords.addAll(nsRecordsFromNodes(ds, edgeTrafficRouters));
			}

			if (ds != null && !ds.isDns() && isEdgeHTTPRouting()) { // only generate edge routing records for HTTP DSs when necessary
				trafficRouterRecords.addAll(inetRecordsFromNodes(ds, edgeTrafficRouters));
			}
		}

		return trafficRouterRecords;
	}

	public List<InetRecord> selectTrafficRoutersLocalized(final Geolocation clientGeolocation, final String name, final DeliveryService ds) throws GeolocationException {
		return selectTrafficRoutersLocalized(clientGeolocation, name, ds, null, 0);
	}

	@SuppressWarnings({"PMD.CyclomaticComplexity", "PMD.NPathComplexity"})
	public List<InetRecord> selectTrafficRoutersLocalized(final Geolocation clientGeolocation, final String zoneName, final DeliveryService ds, final Track track, final int queryType) throws GeolocationException {
		final List<InetRecord> trafficRouterRecords = new ArrayList<>();

		if (!isEdgeDNSRouting() && !isEdgeHTTPRouting()) {
			return trafficRouterRecords;
		}

		final List<TrafficRouterLocation> trafficRouterLocations = (List<TrafficRouterLocation>) orderLocations(getCacheRegister().getEdgeTrafficRouterLocations(), clientGeolocation);

		for (final TrafficRouterLocation location : trafficRouterLocations) {
			final List<Node> trafficRouters = consistentHasher.selectHashables(location.getTrafficRouters(), zoneName);

			if (trafficRouters == null || trafficRouters.isEmpty()) {
				continue;
			}

			if (isEdgeDNSRouting()) {
				trafficRouterRecords.addAll(nsRecordsFromNodes(ds, trafficRouters));
			}

			if (ds != null && !ds.isDns() && isEdgeHTTPRouting()) { // only generate edge routing records for HTTP DSs when necessary
				trafficRouterRecords.addAll(inetRecordsFromNodes(ds, trafficRouters));
			}

			if (track != null) {
				track.setResultLocation(location.getGeolocation());
			}

			break;
		}


		return trafficRouterRecords;
	}

	@SuppressWarnings({"PMD.CyclomaticComplexity", "PMD.NPathComplexity"})
	private DNSRouteResult getEdgeCaches(final DNSRequest request, final DeliveryService ds, final Track track) throws GeolocationException {
		final DNSRouteResult result = new DNSRouteResult();
		result.setDeliveryService(ds);

		if (ds == null) {
			track.setResult(ResultType.STATIC_ROUTE);
			track.setResultDetails(ResultDetails.DS_NOT_FOUND);
			return null;
		}

		if (!ds.isAvailable()) {
			result.setAddresses(ds.getFailureDnsResponse(request, track));
			result.addAddresses(selectTrafficRouters(request, ds));
			return result;
		}

		final IPVersions requestVersion = request.getQueryType() == Type.AAAA ? IPVersions.IPV6ONLY : IPVersions.IPV4ONLY;
		final CacheLocation cacheLocation = getCoverageZoneCacheLocation(request.getClientIP(), ds, false, track, requestVersion);
		List<Cache> caches = selectCachesByCZ(ds, cacheLocation, track, requestVersion);

		if (caches != null) {
			track.setResult(ResultType.CZ);
			track.setClientGeolocation(cacheLocation.getGeolocation());
			result.setAddresses(inetRecordsFromCaches(ds, caches, request));
			result.addAddresses(selectTrafficRouters(request, ds));
			return result;
		}

		if (ds.isCoverageZoneOnly()) {
			track.setResult(ResultType.MISS);
			track.setResultDetails(ResultDetails.DS_CZ_ONLY);
			result.setAddresses(ds.getFailureDnsResponse(request, track));
			result.addAddresses(selectTrafficRouters(request, ds));
			return result;
		}

		try {
			final List<InetRecord> inetRecords = federationRegistry.findInetRecords(ds.getId(), CidrAddress.fromString(request.getClientIP()));

			if (inetRecords != null && !inetRecords.isEmpty()) {
				result.setAddresses(inetRecords);
				track.setResult(ResultType.FED);
				return result;
			}
		} catch (NetworkNodeException e) {
			LOGGER.error("Bad client address: '" + request.getClientIP() + "'");
		}

		if (track.continueGeo) {
			caches = selectCachesByGeo(request.getClientIP(), ds, cacheLocation, track, requestVersion);
		}

		if (caches != null) {
			track.setResult(ResultType.GEO);
			result.setAddresses(inetRecordsFromCaches(ds, caches, request));
		} else {
			track.setResult(ResultType.MISS);
			result.setAddresses(ds.getFailureDnsResponse(request, track));
		}

		result.addAddresses(selectTrafficRouters(request, ds));

		return result;
	}

	private List<InetRecord> nsRecordsFromNodes(final DeliveryService ds, final List<Node> nodes) {
		final List<InetRecord> nsRecords = new ArrayList<>();
		final int limit = (getEdgeDNSRoutingLimit() > nodes.size()) ? nodes.size() : getEdgeDNSRoutingLimit();

		long ttl = getEdgeNSttl();

		if (ds != null && ds.getTtls().has("NS")) {
			ttl = JsonUtils.optLong(ds.getTtls(), "NS"); // no exception
		}

		for (int i = 0; i < limit; i++) {
			final Node node = nodes.get(i);
			nsRecords.add(new InetRecord(node.getFqdn(), ttl, Type.NS));
		}

		return nsRecords;
	}

	public List<InetRecord> inetRecordsFromNodes(final DeliveryService ds, final List<Node> nodes) {
		final List<InetRecord> addresses = new ArrayList<>();
		final int limit = (getEdgeHTTPRoutingLimit(ds) > nodes.size()) ? nodes.size() : getEdgeHTTPRoutingLimit(ds);

		if (ds == null) {
			return addresses;
		}

		final JsonNode ttls = ds.getTtls();

		for (int i = 0; i < limit; i++) {
			final Node node = nodes.get(i);

			if (node.getIp4() != null) {
				addresses.add(new InetRecord(node.getIp4(), JsonUtils.optLong(ttls, "A")));
			}

			if (node.getIp6() != null && ds.isIp6RoutingEnabled()) {
				addresses.add(new InetRecord(node.getIp6(), JsonUtils.optLong(ttls,"AAAA")));
			}
		}

		return addresses;
	}

	/**
	 * Extracts the IP Addresses from a set of caches based on a Delivery Service's configuration
	 * @param ds The Delivery Service being served. If this DS does not have "IPv6 routing enabled",
	 * then the IPAddresses returned will not include IPv6 addresses.
	 * @param caches The list of caches chosen to serve ds. If the length of this list is greater
	 * than the maximum allowed IP addresses in a DNS response by the
	 * {@link DeliveryService#getMaxDnsIps()()} of the requested Delivery Service, the maximum
	 * allowed number will be chosen from the list at random.
	 * @param request The request being served - used for consistent hashing when caches must be
	 * chosen at random
	 * @return The IP Addresses of the passed caches. In general, these may be IPv4 or IPv6.
	 */
	public List<InetRecord> inetRecordsFromCaches(final DeliveryService ds, final List<Cache> caches, final Request request) {
		final List<InetRecord> addresses = new ArrayList<>();
		final int maxDnsIps = ds.getMaxDnsIps();
		List<Cache> selectedCaches;

		if (maxDnsIps > 0 && isConsistentDNSRouting()) { // only consistent hash if we must
			selectedCaches = (List<Cache>) consistentHasher.selectHashables(caches, ds.getDispersion(), request.getHostname());
		} else if (maxDnsIps > 0) {
			/*
			 * We also shuffle in NameServer when adding Records to the Message prior
			 * to sending it out, as the Records are sorted later when we fill the
			 * dynamic zone if DNSSEC is enabled. We shuffle here prior to pruning
			 * for maxDnsIps so that we ensure we are spreading load across all caches
			 * assigned to this delivery service.
			*/
			Collections.shuffle(caches, random);

			selectedCaches = new ArrayList<Cache>();

			for (final Cache cache : caches) {
				selectedCaches.add(cache);

				if (selectedCaches.size() >= maxDnsIps) {
					break;
				}
			}
		} else {
			selectedCaches = caches;
		}

		for (final Cache cache : selectedCaches) {
			addresses.addAll(cache.getIpAddresses(ds.getTtls(), ds.isIp6RoutingEnabled()));
		}

		return addresses;
	}

	/**
	 * Geo-locates the client based on their IP address and the Delivery Service they requested.
	 * <p>
	 * This is optimized over {@link #getLocation(String, DeliveryService)} because
	 * @param clientIp The IP Address of the requesting client.
	 * @param track A state-tracking object, it will be notified of the calculated client location
	 * for optimization of future queries.
	 * @param deliveryService The Delivery Service being served. Currently only used for logging
	 * error information.
	 * @return The client's calculated geographic location
	 * @throws GeolocationException
	 */
	public Geolocation getClientGeolocation(final String clientIp, final Track track, final DeliveryService deliveryService) throws GeolocationException {
		if (track != null && track.isClientGeolocationQueried()) {
			return track.getClientGeolocation();
		}

		final Geolocation clientGeolocation;

		if (deliveryService != null) {
			clientGeolocation = getLocation(clientIp, deliveryService);
		} else {
			clientGeolocation = getLocation(clientIp);
		}

		if (track != null) {
			track.setClientGeolocation(clientGeolocation);
			track.setClientGeolocationQueried(true);
		}

		return clientGeolocation;
	}

	/**
	 * Geo-locates the client based on their IP address and the Delivery Service they requested.
	 * @param clientIp The IP Address of the requesting client.
	 * @param ds The Delivery Service being served. If the client's location is blocked by this
	 * Delivery Service, the returned location will instead be the appropriate fallback/miss
	 * location.
	 * @param cacheLocation If this is not 'null', its location will be used in lieu of calculating
	 * one for the client.
	 * @return The client's calculated geographic location (or the appropriate fallback/miss
	 * location).
	 */
	public Geolocation getClientLocation(final String clientIp, final DeliveryService ds, final Location cacheLocation, final Track track) throws GeolocationException {
		if (cacheLocation != null) {
			return cacheLocation.getGeolocation();
		}

		final Geolocation clientGeolocation = getClientGeolocation(clientIp, track, ds);
		return ds.supportLocation(clientGeolocation);
	}

	/**
	 * Selects caches to service requests for a Delivery Service from a cache location based on
	 * Coverage Zone configuration.
	 * <p>
	 * This is equivalent to calling {@link #selectCachesByCZ(DeliveryService, CacheLocation, Track, IPVersions)}
	 * with a 'null' "track" argument.
	 * </p>
	 * @param ds The Delivery Service being served.
	 * @param cacheLocation The location from which caches will be selected.
	 * @return All of the caches in the given location capable of serving ds.
	 */
	public List<Cache> selectCachesByCZ(final DeliveryService ds, final CacheLocation cacheLocation, final IPVersions requestVersion) {
		return selectCachesByCZ(ds, cacheLocation, null, requestVersion);
	}

	/**
	 * Selects caches to service requests for a Delivery Service from a cache location based on
	 * Coverage Zone Configuration.
	 * @param deliveryServiceId An identifier for the {@link DeliveryService} being served.
	 * @param cacheLocationId An identifier for the {@link CacheLocation} from which caches will be
	 * selected.
	 * @return All of the caches in the given location capable of serving the identified Delivery
	 * Service.
	 */
	public List<Cache> selectCachesByCZ(final String deliveryServiceId, final String cacheLocationId, final Track track, final IPVersions requestVersion) {
		return selectCachesByCZ(cacheRegister.getDeliveryService(deliveryServiceId), cacheRegister.getCacheLocation(cacheLocationId), track, requestVersion);
	}

	/**
	 * Selects caches to service requests for a Delivery Service from a cache location based on
	 * Coverage Zone Configuration.
	 * <p>
	 * This is equivalent to calling {@link #selectCachesByCZ(DeliveryService, CacheLocation, Track, ResultType, IPVersions)}
	 * with the "result" argument set to {@link ResultType#CZ}.
	 * </p>
	 * @param ds The Delivery Service being served.
	 * @param cacheLocation The location from which caches will be selected
	 * @return All of the caches in the given location capable of serving ds.
	 */
	private List<Cache> selectCachesByCZ(final DeliveryService ds, final CacheLocation cacheLocation, final Track track, final IPVersions requestVersion) {
		return selectCachesByCZ(ds, cacheLocation, track, ResultType.CZ, requestVersion); // ResultType.CZ was the original default before DDC
	}

	/**
	 * Selects caches to service requests for a Delivery Service from a cache location based on
	 * Coverage Zone Configuration.
	 * <p>
	 * Obviously, at this point, the location from which to select caches must already be known.
	 * So it's totally possible that that decision wasn't made based on Coverage Zones at all,
	 * that's just the default routing result chosen by a common caller of this method
	 * ({@link #selectCachesByCZ(DeliveryService, CacheLocation, Track, IPVersions)}).
	 * </p>
	 * @param ds The Delivery Service being served.
	 * @param cacheLocation The location from which caches will be selected.
	 * @param result The type of routing result that resulted in the returned caches being selected.
	 * This is used for tracking routing results.
	 * @return All of the caches in the given location capable of serving ds.
	 */
	private List<Cache> selectCachesByCZ(final DeliveryService ds, final CacheLocation cacheLocation, final Track track, final ResultType result, final IPVersions requestVersion) {
		if (cacheLocation == null || ds == null || !ds.isLocationAvailable(cacheLocation)) {
			return null;
		}

		final List<Cache> caches = selectCaches(cacheLocation, ds, requestVersion);

		if (caches != null && track != null) {
			track.setResult(result);
			if (track.isFromBackupCzGroup()) {
				track.setResultDetails(ResultDetails.DS_CZ_BACKUP_CG);
			}
			track.setResultLocation(cacheLocation.getGeolocation());
		}

		return caches;
	}

	/**
	 * Gets multiple routes for STEERING Delivery Services
	 *
	 * @param request The client's HTTP Request
	 * @param track A {@link Track} object used to track routing statistics
	 * @return The list of routes available to service the client's request.
	 */
	@SuppressWarnings({"PMD.CyclomaticComplexity", "PMD.NPathComplexity"})
	public HTTPRouteResult multiRoute(final HTTPRequest request, final Track track) throws MalformedURLException, GeolocationException {
		final DeliveryService entryDeliveryService = cacheRegister.getDeliveryService(request);

		final List<SteeringResult> steeringResults = getSteeringResults(request, track, entryDeliveryService);

		if (steeringResults == null) {
			return null;
		}

		final HTTPRouteResult routeResult = new HTTPRouteResult(true);
		routeResult.setDeliveryService(entryDeliveryService);

		if (entryDeliveryService.isRegionalGeoEnabled()) {
		    RegionalGeo.enforce(this, request, entryDeliveryService, null, routeResult, track);
		    if (routeResult.getUrl() != null) {
		        return routeResult;
		    }
		}

		final List<SteeringResult> resultsToRemove = new ArrayList<>();

		final Set<Cache> selectedCaches = new HashSet<>();

		// Pattern based consistent hashing - use consistentHashRegex from steering DS instead of targets
		final String steeringHash = buildPatternBasedHashString(entryDeliveryService.getConsistentHashRegex(), request.getPath());
		for (final SteeringResult steeringResult : steeringResults) {
			final DeliveryService ds = steeringResult.getDeliveryService();
			List<Cache> caches = selectCaches(request, ds, track);

			// child Delivery Services can use their query parameters
			final String pathToHash = steeringHash + ds.extractSignificantQueryParams(request);

			if (caches != null && !caches.isEmpty()) {
				if (isClientSteeringDiversityEnabled()) {
					List<Cache> tryCaches = new ArrayList<>(caches);
					tryCaches.removeAll(selectedCaches);
					if (!tryCaches.isEmpty()) {
						caches = tryCaches;
					} else if (track.result == ResultType.DEEP_CZ) {
						// deep caches have been selected already, try non-deep selection
						tryCaches = selectCaches(request, ds, track, false);
						track.setResult(ResultType.DEEP_CZ); // request should still be tracked as a DEEP_CZ hit
						tryCaches.removeAll(selectedCaches);
						if (!tryCaches.isEmpty()) {
							caches = tryCaches;
						}
					}
				}
				final Cache cache = consistentHasher.selectHashable(caches, ds.getDispersion(), pathToHash);
				steeringResult.setCache(cache);
				selectedCaches.add(cache);
			} else {
				resultsToRemove.add(steeringResult);
			}
		}
		steeringResults.removeAll(resultsToRemove);

		geoSortSteeringResults(steeringResults, request.getClientIP(), entryDeliveryService);

		for (final SteeringResult steeringResult: steeringResults) {
			routeResult.addUrl(new URL(steeringResult.getDeliveryService().createURIString(request, steeringResult.getCache())));
			routeResult.addDeliveryService(steeringResult.getDeliveryService());
		}

		if (routeResult.getUrls().isEmpty()) {
			routeResult.addUrl(entryDeliveryService.getFailureHttpResponse(request, track));
		}

		return routeResult;
	}

	/**
	 * Creates a string to be used in consistent hashing.
	 *<p>
	 * This uses simply the request path by default, but will consider any and all Query Parameters
	 * that are in deliveryService's {@link DeliveryService} consistentHashQueryParam set as well.
	 * It will also fall back on the request path if the query parameters are not UTF-8-encoded.
	 *</p>
	 * @param deliveryService The {@link DeliveryService} being requested
	 * @param request An {@link HTTPRequest} representing the client's request.
	 * @return A string appropriate to use for consistent hashing to service the request
	*/
	@SuppressWarnings({"PMD.CyclomaticComplexity"})
	public String buildPatternBasedHashString(final DeliveryService deliveryService, final HTTPRequest request) {
		final String requestPath = request.getPath();
		final StringBuilder hashString = new StringBuilder("");
		if (deliveryService.getConsistentHashRegex() != null && !requestPath.isEmpty()) {
			hashString.append(buildPatternBasedHashString(deliveryService.getConsistentHashRegex(), requestPath));
		}

		hashString.append(deliveryService.extractSignificantQueryParams(request));

		return hashString.toString();
	}

	/**
	 * Constructs a string to be used in consistent hashing
	 * <p>
	 * If {@code regex} is {@code null} or empty - or if an error occurs applying it -, returns
	 * {@code requestPath} unaltered.
	 * </p>
	 * @param regex A regular expression matched against the client's request path to extract
	 * information important to consistent hashing
	 * @param requestPath The client's request path - e.g. {@code /some/path} from
	 * {@code https://example.com/some/path}
	 * @return The parts of requestPath that matched regex
	 */
	public String buildPatternBasedHashString(final String regex, final String requestPath) {
		if (regex == null || regex.isEmpty()) {
			return requestPath;
		}

		try {
			final Pattern pattern = Pattern.compile(regex);
			final Matcher matcher = pattern.matcher(requestPath);

			final StringBuilder sb = new StringBuilder("");
			if (matcher.find() && matcher.groupCount() > 0) {
				for (int i = 1; i <= matcher.groupCount(); i++) {
					final String text = matcher.group(i);
					sb.append(text);
				}
				return sb.toString();
			}
		} catch (final Exception e) {
			final StringBuilder error = new StringBuilder("Failed to construct hash string using regular expression: '");
			error.append(regex);
			error.append("' against request path: '");
			error.append(requestPath);
			error.append("' Exception: ");
			error.append(e.toString());
			LOGGER.error(error.toString());
		}
		return requestPath;
	}

	/**
	 * Routes an HTTP request.
	 * @param request The request being routed.
	 * @return The result of routing the HTTP request.
	 * @throws MalformedURLException
	 * @throws GeolocationException
	 */
	@SuppressWarnings({"PMD.CyclomaticComplexity", "PMD.NPathComplexity"})
	public HTTPRouteResult route(final HTTPRequest request, final Track track) throws MalformedURLException, GeolocationException {
		track.setRouteType(RouteType.HTTP, request.getHostname());

		if (isMultiRouteRequest(request)) {
			return multiRoute(request, track);
		} else {
			return singleRoute(request, track);
		}
	}

	/**
	 * Routes an HTTP request that isn't for a CLIENT_STEERING-type Delivery Service.
	 * @param request The request being routed.
	 * @return The result of routing this HTTP request.
	 * @throws MalformedURLException if a URL cannot be constructed to return to the client
	 */
	@SuppressWarnings({ "PMD.CyclomaticComplexity", "PMD.NPathComplexity" })
	public HTTPRouteResult singleRoute(final HTTPRequest request, final Track track) throws MalformedURLException, GeolocationException {
		final DeliveryService deliveryService = getDeliveryService(request, track);

		if (deliveryService == null) {
			return null;
		}

		final HTTPRouteResult routeResult = new HTTPRouteResult(false);

		if (!deliveryService.isAvailable()) {
			routeResult.setUrl(deliveryService.getFailureHttpResponse(request, track));
			return routeResult;
		}

		routeResult.setDeliveryService(deliveryService);

		final List<Cache> caches = selectCaches(request, deliveryService, track);
		if (caches == null || caches.isEmpty()) {
			if (track.getResult() == ResultType.GEO_REDIRECT) {
				routeResult.setUrl(new URL(deliveryService.getGeoRedirectUrl()));
				LOGGER.debug(String.format("NGB redirect to url: %s for request: %s", deliveryService.getGeoRedirectUrl()
						, request.getRequestedUrl()));
				return routeResult;
			}

			routeResult.setUrl(deliveryService.getFailureHttpResponse(request, track));
			return routeResult;
		}

		// Pattern based consistent hashing
		final String pathToHash = buildPatternBasedHashString(deliveryService, request);
		final Cache cache = consistentHasher.selectHashable(caches, deliveryService.getDispersion(), pathToHash);

		// Enforce anonymous IP blocking if a DS has anonymous blocking enabled
		// and the feature is enabled
		if (deliveryService.isAnonymousIpEnabled() && AnonymousIp.getCurrentConfig().enabled) {
			AnonymousIp.enforce(this, request, deliveryService, cache, routeResult, track);

			if (routeResult.getResponseCode() == AnonymousIp.BLOCK_CODE) {
				return routeResult;
			}
		}

		if (deliveryService.isRegionalGeoEnabled()) {
			RegionalGeo.enforce(this, request, deliveryService, cache, routeResult, track);
			return routeResult;
		}

		final String uriString = deliveryService.createURIString(request, cache);
		routeResult.setUrl(new URL(uriString));

		return routeResult;
	}

	/**
	 * Gets all the possible steering results for a request to a Delivery Service.
	 * @param request The client HTTP request.
	 * @param entryDeliveryService The steering Delivery Service being served.
	 * @return All of the possible steering results for routing request through entryDeliveryService.
	 */
	@SuppressWarnings({"PMD.NPathComplexity"})
	private List<SteeringResult> getSteeringResults(final HTTPRequest request, final Track track, final DeliveryService entryDeliveryService) {

		if (isTlsMismatch(request, entryDeliveryService)) {
			track.setResult(ResultType.ERROR);
			track.setResultDetails(ResultDetails.DS_TLS_MISMATCH);
			return null;
		}

		final List<SteeringResult> steeringResults = consistentHashMultiDeliveryService(entryDeliveryService, request);

		if (steeringResults == null || steeringResults.isEmpty()) {
			track.setResult(ResultType.DS_MISS);
			track.setResultDetails(ResultDetails.DS_NOT_FOUND);
			return null;
		}

		final List<SteeringResult> toBeRemoved = new ArrayList<>();
		for (final SteeringResult steeringResult : steeringResults) {
			final DeliveryService ds = steeringResult.getDeliveryService();
			if (isTlsMismatch(request, ds)) {
				track.setResult(ResultType.ERROR);
				track.setResultDetails(ResultDetails.DS_TLS_MISMATCH);
				return null;
			}
			if (!ds.isAvailable()) {
				toBeRemoved.add(steeringResult);
			}

		}

		steeringResults.removeAll(toBeRemoved);
		return steeringResults.isEmpty() ? null : steeringResults;
	}

	/**
	 * Gets the Delivery Service that matches the client HTTP request.
	 * @param request The client HTTP request.
	 * @return The Delivery Service corresponding to the request if one can be found, otherwise
	 * {@code null}.
	 */
	private DeliveryService getDeliveryService(final HTTPRequest request, final Track track) {
		final String xtcSteeringOption = request.getHeaderValue(XTC_STEERING_OPTION);
		final DeliveryService deliveryService = consistentHashDeliveryService(cacheRegister.getDeliveryService(request), request, xtcSteeringOption);

		if (deliveryService == null) {
			track.setResult(ResultType.DS_MISS);
			track.setResultDetails(ResultDetails.DS_NOT_FOUND);
			return null;
		}

		if (isTlsMismatch(request, deliveryService)) {
			track.setResult(ResultType.ERROR);
			track.setResultDetails(ResultDetails.DS_TLS_MISMATCH);
			return null;
		}

		return deliveryService;
	}

	/**
	 * Checks if the TLS settings on the client HTTP request match those of the Delivery Service
	 * it's requesting.
	 * @param request The client HTTP request.
	 * @param deliveryService The Delivery Service being served.
	 */
	private boolean isTlsMismatch(final HTTPRequest request, final DeliveryService deliveryService) {
		if (request.isSecure() && !deliveryService.isSslEnabled()) {
			return true;
		}

		if (!request.isSecure() && !deliveryService.isAcceptHttp()) {
			return true;
		}

		return false;
	}

	/**
	 * Finds a network subnet for the given IP address based on Deep Coverage Zone configuration.
	 * @param ip The IP address to look up.
	 * @return A network subnet  capable of serving requests for the given IP, or {@code null} if
	 * one couldn't be found.
	 */
	protected NetworkNode getDeepNetworkNode(final String ip) {
		try {
			return NetworkNode.getDeepInstance().getNetwork(ip);
		} catch (NetworkNodeException e) {
			LOGGER.warn(e);
		}
		return null;
	}

	/**
	 * Finds a network subnet for the given IP address based on Coverage Zone configuration.
	 * @param ip The IP address to look up.
	 * @return A network subnet capable of serving requests for the given IP, or {@code null} if
	 * one couldn't be found.
	 */
	protected NetworkNode getNetworkNode(final String ip) {
		try {
			return NetworkNode.getInstance().getNetwork(ip);
		} catch (NetworkNodeException e) {
			LOGGER.warn(e);
		}
		return null;
	}

	public CacheLocation getCoverageZoneCacheLocation(final String ip, final String deliveryServiceId, final IPVersions requestVersion) {
		return getCoverageZoneCacheLocation(ip, deliveryServiceId, false, null, requestVersion); // default is not deep
	}

	/**
	 * Finds the deep coverage zone location information for a give IP address.
	 * @param ip
	 * @return deep coverage zone location
	 */
	public CacheLocation getDeepCoverageZoneLocationByIP(final String ip) {
		final NetworkNode networkNode = getDeepNetworkNode(ip);

		if (networkNode == null) {
			return null;
		}

		final CacheLocation cacheLocation = (CacheLocation) networkNode.getLocation();

		if (cacheLocation != null) {
			cacheLocation.loadDeepCaches(networkNode.getDeepCacheNames(), cacheRegister);
		}

		return cacheLocation;
	}

	@SuppressWarnings({"PMD.CyclomaticComplexity", "PMD.NPathComplexity"})
	public CacheLocation getCoverageZoneCacheLocation(final String ip, final String deliveryServiceId, final boolean useDeep, final Track track, final IPVersions requestVersion) {
		final NetworkNode networkNode = useDeep ? getDeepNetworkNode(ip) : getNetworkNode(ip);
		final LocalizationMethod localizationMethod = useDeep ? LocalizationMethod.DEEP_CZ : LocalizationMethod.CZ;

		if (networkNode == null) {
			return null;
		}

		final DeliveryService deliveryService = cacheRegister.getDeliveryService(deliveryServiceId);
		CacheLocation cacheLocation = (CacheLocation) networkNode.getLocation();

		if (useDeep && cacheLocation != null) {
			// lazily load deep Caches into the deep CacheLocation
			cacheLocation.loadDeepCaches(networkNode.getDeepCacheNames(), cacheRegister);
		}

		if (cacheLocation != null && !cacheLocation.isEnabledFor(localizationMethod)) {
			return null;
		}

		if (cacheLocation != null && !getSupportingCaches(cacheLocation.getCaches(), deliveryService, requestVersion).isEmpty()) {
			return cacheLocation;
		}

		if (useDeep) {
			// there were no available deep caches in the deep CZF
			return null;
		}

		if (networkNode.getLoc() == null) {
			return null;
		}

		// find CacheLocation
		cacheLocation = getCacheRegister().getCacheLocationById(networkNode.getLoc());
		if (cacheLocation != null && !cacheLocation.isEnabledFor(localizationMethod)) {
			track.continueGeo = false; // hit in the CZF but the cachegroup doesn't allow CZ-localization, don't fall back to GEO
			return null;
		}

		if (cacheLocation != null && !getSupportingCaches(cacheLocation.getCaches(), deliveryService, requestVersion).isEmpty()) {
			// lazy loading in case a CacheLocation has not yet been associated with this NetworkNode
			networkNode.setLocation(cacheLocation);
			return cacheLocation;
		}

		if (cacheLocation != null && cacheLocation.getBackupCacheGroups() != null) {
			for (final String cacheGroup : cacheLocation.getBackupCacheGroups()) {
				final CacheLocation bkCacheLocation = getCacheRegister().getCacheLocationById(cacheGroup);
				if (bkCacheLocation != null && !bkCacheLocation.isEnabledFor(localizationMethod)) {
					continue;
				}
				if (bkCacheLocation != null && !getSupportingCaches(bkCacheLocation.getCaches(), deliveryService, requestVersion).isEmpty()) {
					LOGGER.debug("Got backup CZ cache group " + bkCacheLocation.getId() + " for " + ip + ", ds " + deliveryServiceId);
					if (track != null) {
						track.setFromBackupCzGroup(true);
					}
					return bkCacheLocation;
				}
			}
			// track.continueGeo
			// will become to false only when backups are configured and (primary group's) fallbackToClosedGeo is configured (non-empty list) to false
			// False signals subsequent cacheSelection routine to stop geo based selection.
			if (!cacheLocation.isUseClosestGeoLoc()) {
			    track.continueGeo = false;
			    return null;
			}
		}

		// We had a hit in the CZF but the name does not match a known cache location.
		// Check whether the CZF entry has a geolocation and use it if so.
		List<CacheLocation> availableLocations = cacheRegister.filterAvailableCacheLocations(deliveryServiceId);
		availableLocations = filterEnabledLocations(availableLocations, localizationMethod);
		final CacheLocation closestCacheLocation = getClosestCacheLocation(availableLocations, networkNode.getGeolocation(), cacheRegister.getDeliveryService(deliveryServiceId), requestVersion);

		if (closestCacheLocation != null) {
			LOGGER.debug("Got closest CZ cache group " + closestCacheLocation.getId() + " for " + ip + ", ds " + deliveryServiceId);
			if (track != null) {
				track.setFromBackupCzGroup(true);
			}
		}
		return closestCacheLocation;
	}

	public List<CacheLocation> filterEnabledLocations(final Collection<CacheLocation> locations, final LocalizationMethod localizationMethod) {
		return locations.stream()
				.filter(loc -> loc.isEnabledFor(localizationMethod))
				.collect(Collectors.toList());
	}

	public CacheLocation getDeepCoverageZoneCacheLocation(final String ip, final DeliveryService deliveryService, final IPVersions requestVersion) {
		return getCoverageZoneCacheLocation(ip, deliveryService, true, null, requestVersion);
	}

	public CacheLocation getCoverageZoneCacheLocation(final String ip, final DeliveryService deliveryService, final boolean useDeep, final Track track, final IPVersions requestVersion) {
		return getCoverageZoneCacheLocation(ip, deliveryService.getId(), useDeep, track, requestVersion);
	}

	public CacheLocation getCoverageZoneCacheLocation(final String ip, final DeliveryService deliveryService, final IPVersions requestVersion) {
		return getCoverageZoneCacheLocation(ip, deliveryService.getId(), requestVersion);
	}

	/**
	 * Chooses a {@link Cache} for a Delivery Service based on the Coverage Zone File given a
	 * client's IP and request *path*.
	 * @param ip The client's IP address
	 * @param deliveryServiceId The "xml_id" of a Delivery Service being routed
	 * @param requestPath The client's requested path - e.g.
	 * {@code http://test.example.com/request/path} &rarr; {@code /request/path}
	 * @return A cache object chosen to serve the client's request
	 */
	public Cache consistentHashForCoverageZone(final String ip, final String deliveryServiceId, final String requestPath) {
		return consistentHashForCoverageZone(ip, deliveryServiceId, requestPath, false);
	}

	/**
	 * Chooses a cache for a Delivery Service based on the Coverage Zone File or Deep Coverage Zone
	 * File given a client's IP and request *path*.
	 * @param ip The client's IP address
	 * @param deliveryServiceId The "xml_id" of a Delivery Service being routed
	 * @param requestPath The client's requested path - e.g.
	 * {@code http://test.example.com/request/path} &rarr; {@code /request/path}
	 * @param useDeep if {@code true} will attempt to use Deep Coverage Zones - otherwise will only
	 * use Coverage Zone File
	 * @return A {@link Cache} object chosen to serve the client's request
	 */
	public Cache consistentHashForCoverageZone(final String ip, final String deliveryServiceId, final String requestPath, final boolean useDeep) {
		final HTTPRequest r = new HTTPRequest();
		r.setPath(requestPath);
		r.setQueryString("");
		return consistentHashForCoverageZone(ip, deliveryServiceId, r, useDeep);
	}

	/**
	 * Chooses a cache for a Delivery Service based on the Coverage Zone File or Deep Coverage Zone
	 * File given a client's IP and request.
	 * @param ip The client's IP address
	 * @param deliveryServiceId The "xml_id" of a Delivery Service being routed
	 * @param request The client's HTTP request
	 * @param useDeep if {@code true} will attempt to use Deep Coverage Zones - otherwise will only
	 * use Coverage Zone File
	 * @return A {@link Cache} object chosen to serve the client's request
	 */
	public Cache consistentHashForCoverageZone(final String ip, final String deliveryServiceId, final HTTPRequest request, final boolean useDeep) {
		final DeliveryService deliveryService = cacheRegister.getDeliveryService(deliveryServiceId);
		if (deliveryService == null) {
			LOGGER.error("Failed getting delivery service from cache register for id '" + deliveryServiceId + "'");
			return null;
		}

		final IPVersions requestVersion = ip.contains(":") ? IPVersions.IPV6ONLY : IPVersions.IPV4ONLY;
		final CacheLocation coverageZoneCacheLocation = getCoverageZoneCacheLocation(ip, deliveryService, useDeep, null, requestVersion);
		final List<Cache> caches = selectCachesByCZ(deliveryService, coverageZoneCacheLocation, requestVersion);

		if (caches == null || caches.isEmpty()) {
			return null;
		}

		final String pathToHash = buildPatternBasedHashString(deliveryService, request);
		return consistentHasher.selectHashable(caches, deliveryService.getDispersion(), pathToHash);
	}

	/**
	 * Chooses a {@link Cache} for a Delivery Service based on GeoLocation given a client's IP and
	 * request *path*.
	 * @param ip The client's IP address
	 * @param deliveryServiceId The "xml_id" of a Delivery Service being routed
	 * @param requestPath The client's requested path - e.g.
	 * {@code http://test.example.com/request/path} &rarr; {@code /request/path}
	 * @return A cache object chosen to serve the client's request
	 */
	public Cache consistentHashForGeolocation(final String ip, final String deliveryServiceId, final String requestPath) {
		final HTTPRequest r = new HTTPRequest();
		r.setPath(requestPath);
		r.setQueryString("");
		return consistentHashForGeolocation(ip, deliveryServiceId, r);
	}

	/**
	 * Chooses a {@link Cache} for a Delivery Service based on GeoLocation given a client's IP and
	 * request.
	 * @param ip The client's IP address
	 * @param deliveryServiceId The "xml_id" of a Delivery Service being routed
	 * @param request The client's HTTP request
	 * @return A cache object chosen to serve the client's request
	 */
	public Cache consistentHashForGeolocation(final String ip, final String deliveryServiceId, final HTTPRequest request) {
		final DeliveryService deliveryService = cacheRegister.getDeliveryService(deliveryServiceId);
		if (deliveryService == null) {
			LOGGER.error("Failed getting delivery service from cache register for id '" + deliveryServiceId + "'");
			return null;
		}

		final IPVersions requestVersion = ip.contains(":") ? IPVersions.IPV6ONLY : IPVersions.IPV4ONLY;
		List<Cache> caches = null;
		if (deliveryService.isCoverageZoneOnly() && deliveryService.getGeoRedirectUrl() != null) {
			//use the NGB redirect
			caches = enforceGeoRedirect(StatTracker.getTrack(), deliveryService, ip, null, requestVersion);
		} else {
			final CacheLocation cacheLocation = getCoverageZoneCacheLocation(ip, deliveryServiceId, requestVersion);

			try {
				caches = selectCachesByGeo(ip, deliveryService, cacheLocation, StatTracker.getTrack(), requestVersion);
			} catch (GeolocationException e) {
				LOGGER.warn("Failed gettting list of caches by geolocation for ip " + ip + " delivery service id '" + deliveryServiceId + "'");
			}
		}

		if (caches == null || caches.isEmpty()) {
			return null;
		}

		final String pathToHash = buildPatternBasedHashString(deliveryService, request);
		return consistentHasher.selectHashable(caches, deliveryService.getDispersion(), pathToHash);
	}

	/**
	 * Builds a string to be used for consistent hashing based on a client's request *path*.
	 * @param deliveryServiceId The "xml_id" of a Delivery Service, the consistent hash settings of
	 * which will be used to build the consistent hashing string.
	 * @param requestPath The client's requested path.
	 * @return A string suitable for using in consistent hashing.
	 */
	public String buildPatternBasedHashStringDeliveryService(final String deliveryServiceId, final String requestPath) {
		final HTTPRequest r = new HTTPRequest();
		r.setPath(requestPath);
		r.setQueryString("");
		return buildPatternBasedHashString(cacheRegister.getDeliveryService(deliveryServiceId), r);
	}

	/**
	 * Returns whether or not the given Delivery Service is of the STEERING or CLIENT_STEERING type.
	 */
	private boolean isSteeringDeliveryService(final DeliveryService deliveryService) {
		return deliveryService != null && steeringRegistry.has(deliveryService.getId());
	}

	/**
	 * Checks whether the given client's HTTP request is for a CLIENT_STEERING Delivery Service.
	 */
	private boolean isMultiRouteRequest(final HTTPRequest request) {
		final DeliveryService deliveryService = cacheRegister.getDeliveryService(request);

		if (deliveryService == null || !isSteeringDeliveryService(deliveryService)) {
			return false;
		}

		return steeringRegistry.get(deliveryService.getId()).isClientSteering();
	}

	/**
	 * Gets a geographic location for the client based on their IP address.
	 * @param clientIP The client's IP address as a string.
	 * @param deliveryService The Delivery Service the client is requesting. This is used to
	 * determine the appropriate location if the client cannot be located, or is blocked by RGB
	 * or Anonymous Blocking rules.
	 * @return The client's calculated geographic location, or {@code null} if they cannot be
	 * geo-located (and deliveryService has no default "miss" location set) or if the client is
	 * blocked by the Delivery Service's settings.
	 */
	protected Geolocation getClientLocationByCoverageZoneOrGeo(final String clientIP, final DeliveryService deliveryService) {
		Geolocation clientLocation;
		final NetworkNode networkNode = getNetworkNode(clientIP);
		if (networkNode != null && networkNode.getGeolocation() != null) {
			clientLocation = networkNode.getGeolocation();
		} else {
			try {
				clientLocation = getLocation(clientIP, deliveryService);
			} catch (GeolocationException e) {
				clientLocation = null;
			}
		}
		return deliveryService.supportLocation(clientLocation);
	}

	/**
	 * Sorts the provided steering results by their geographic proximity to the client and their
	 * configured ordering and weights.
	 * @param steeringResults The results to be sorted. They are sorted "in place" - this modifies
	 * the list directly.
	 * @param clientIP The client's IP address as a string. This is used to calculate their
	 * geographic location.
	 * @param deliveryService The Delivery Service being served. This is used to help geo-locate the
	 * client according to blocking and fallback configuration.
	 */
	protected void geoSortSteeringResults(final List<SteeringResult> steeringResults, final String clientIP, final DeliveryService deliveryService) {
		if (clientIP == null || clientIP.isEmpty()
				|| steeringResults.stream().allMatch(t -> t.getSteeringTarget().getGeolocation() == null)) {
			return;
		}

		final Geolocation clientLocation = getClientLocationByCoverageZoneOrGeo(clientIP, deliveryService);
		if (clientLocation != null) {
			steeringResults.sort(new SteeringGeolocationComparator(clientLocation));
			steeringResults.sort(Comparator.comparingInt(s -> s.getSteeringTarget().getOrder())); // re-sort by order to preserve the ordering done by ConsistentHasher
		}
	}

	public List<SteeringResult> consistentHashMultiDeliveryService(final DeliveryService deliveryService, final HTTPRequest request) {
		if (deliveryService == null) {
			return null;
		}

		final List<SteeringResult> steeringResults = new ArrayList<>();

		if (!isSteeringDeliveryService(deliveryService)) {
			steeringResults.add(new SteeringResult(null, deliveryService));
			return steeringResults;
		}

		final Steering steering = steeringRegistry.get(deliveryService.getId());

		// Pattern based consistent hashing
		final String pathToHash = buildPatternBasedHashString(deliveryService, request);
		final List<SteeringTarget> steeringTargets = consistentHasher.selectHashables(steering.getTargets(), pathToHash);

		for (final SteeringTarget steeringTarget : steeringTargets) {
			final DeliveryService target = cacheRegister.getDeliveryService(steeringTarget.getDeliveryService());

			if (target != null) { // target might not be in CRConfig yet
				steeringResults.add(new SteeringResult(steeringTarget, target));
			}
		}

		return steeringResults;
	}

 	/**
	 * Chooses a {@link Cache} for a Steering Delivery Service target based on the Coverage Zone
	 * File given a clients IP and request *path*.
	 * @param ip The client's IP address
	 * @param deliveryServiceId The "xml_id" of a Delivery Service being routed
	 * @param requestPath The client's requested path - e.g.
	 * {@code http://test.example.com/request/path} &rarr; {@code /request/path}
	 * @return A cache object chosen to serve the client's request
	 */
	public Cache consistentHashSteeringForCoverageZone(final String ip, final String deliveryServiceId, final String requestPath) {
		final HTTPRequest r = new HTTPRequest();
		r.setPath(requestPath);
		r.setQueryString("");
		return consistentHashSteeringForCoverageZone(ip, deliveryServiceId, r);
	}

 	/**
	 * Chooses a {@link Cache} for a Steering Delivery Service target based on the Coverage Zone
	 * File given a clients IP and request.
	 * @param ip The client's IP address
	 * @param deliveryServiceId The "xml_id" of a Delivery Service being routed
	 * @param request The client's HTTP request
	 * @return A cache object chosen to serve the client's request
	 */
	public Cache consistentHashSteeringForCoverageZone(final String ip, final String deliveryServiceId, final HTTPRequest request) {
		final DeliveryService deliveryService = consistentHashDeliveryService(deliveryServiceId, request);
		if (deliveryService == null) {
			LOGGER.error("Failed getting delivery service from cache register for id '" + deliveryServiceId + "'");
			return null;
		}

		final IPVersions requestVersion = ip.contains(":") ? IPVersions.IPV6ONLY : IPVersions.IPV4ONLY;
		final CacheLocation coverageZoneCacheLocation = getCoverageZoneCacheLocation(ip, deliveryService, false, null, requestVersion);
		final List<Cache> caches = selectCachesByCZ(deliveryService, coverageZoneCacheLocation, requestVersion);

		if (caches == null || caches.isEmpty()) {
			return null;
		}

		final String pathToHash = buildPatternBasedHashString(deliveryService, request);
		return consistentHasher.selectHashable(caches, deliveryService.getDispersion(), pathToHash);
	}

	/**
	 * Chooses a target Delivery Service of a given Delivery Service to service a given request path
	 *
	 * @param deliveryServiceId The "xml_id" of the Delivery Service being requested
	 * @param requestPath The requested path - e.g.
	 * {@code http://test.example.com/request/path} &rarr; {@code /request/path}
	 * @return The chosen target Delivery Service, or null if one could not be determined.
	*/
	public DeliveryService consistentHashDeliveryService(final String deliveryServiceId, final String requestPath) {
		final HTTPRequest r = new HTTPRequest();
		r.setPath(requestPath);
		r.setQueryString("");
		return consistentHashDeliveryService(deliveryServiceId, r);
	}

	/**
	 * Chooses a target Delivery Service of a given Delivery Service to service a given request.
	 *
	 * @param deliveryServiceId The "xml_id" of the Delivery Service being requested
	 * @param request The client's HTTP request
	 * @return The chosen target Delivery Service, or null if one could not be determined.
	*/
	public DeliveryService consistentHashDeliveryService(final String deliveryServiceId, final HTTPRequest request) {
		return consistentHashDeliveryService(cacheRegister.getDeliveryService(deliveryServiceId), request, "");
	}

	/**
	 * Chooses a target Delivery Service of a given Delivery Service to service a given request and
	 * {@link #XTC_STEERING_OPTION} value.
	 *
	 * @param deliveryService The DeliveryService being requested
	 * @param request The client's HTTP request
	 * @param xtcSteeringOption The value of the client's {@link #XTC_STEERING_OPTION} HTTP Header.
	 * @return The chosen target Delivery Service, or null if one could not be determined.
	*/
	@SuppressWarnings({"PMD.CyclomaticComplexity", "PMD.NPathComplexity"})
	public DeliveryService consistentHashDeliveryService(final DeliveryService deliveryService, final HTTPRequest request, final String xtcSteeringOption) {
		if (deliveryService == null) {
			return null;
		}

		if (!isSteeringDeliveryService(deliveryService)) {
			return deliveryService;
		}

		final Steering steering = steeringRegistry.get(deliveryService.getId());

		if (xtcSteeringOption != null && !xtcSteeringOption.isEmpty()) {
			return steering.hasTarget(xtcSteeringOption) ? cacheRegister.getDeliveryService(xtcSteeringOption) : null;
		}

		final String bypassDeliveryServiceId = steering.getBypassDestination(request.getPath());
		if (bypassDeliveryServiceId != null && !bypassDeliveryServiceId.isEmpty()) {
			final DeliveryService bypass = cacheRegister.getDeliveryService(bypassDeliveryServiceId);
			if (bypass != null) { // bypass DS target might not be in CRConfig yet. Until then, try existing targets
				return bypass;
			}
		}

		// only select from targets in CRConfig
		final List<SteeringTarget> availableTargets = steering.getTargets().stream()
				.filter(target -> cacheRegister.getDeliveryService(target.getDeliveryService()) != null)
				.collect(Collectors.toList());

		// Pattern based consistent hashing
		final String pathToHash = buildPatternBasedHashString(deliveryService, request);
		final SteeringTarget steeringTarget = consistentHasher.selectHashable(availableTargets, deliveryService.getDispersion(), pathToHash);

		// set target.consistentHashRegex from steering DS, if it is set
		final DeliveryService targetDeliveryService = cacheRegister.getDeliveryService(steeringTarget.getDeliveryService());
		if (deliveryService.getConsistentHashRegex() != null && !deliveryService.getConsistentHashRegex().isEmpty()) {
			targetDeliveryService.setConsistentHashRegex(deliveryService.getConsistentHashRegex());
		}
		return targetDeliveryService;
	}

	/**
	 * Returns a list {@link Location}s sorted by distance from the client.
	 * If the client's location could not be determined, then the list is
	 * unsorted.
	 *
	 * @param locations the collection of Locations to order
	 * @return the ordered list of locations
	 */
	public List<? extends Location> orderLocations(final List<? extends Location> locations, final Geolocation clientLocation) {
		Collections.sort(locations, new LocationComparator(clientLocation));
		return locations;
	}

	private CacheLocation getClosestCacheLocation(final List<CacheLocation> cacheLocations, final Geolocation clientLocation, final DeliveryService deliveryService, final IPVersions requestVersion) {
		if (clientLocation == null) {
			return null;
		}

	    final List<CacheLocation> orderedLocations = (List<CacheLocation>) orderLocations(cacheLocations, clientLocation);

		for (final CacheLocation cacheLocation : orderedLocations) {
			if (!getSupportingCaches(cacheLocation.getCaches(), deliveryService, requestVersion).isEmpty()) {
				return cacheLocation;
			}
		}

		return null;
	}

	/**
	 * Selects a {@link Cache} from the {@link CacheLocation} provided.
	 *
	 * @param location
	 *            the caches that will considered
	 * @param ds
	 *            the delivery service for the request
	 * @return the selected cache or null if none can be found
	 */
	private List<Cache> selectCaches(final CacheLocation location, final DeliveryService ds, final IPVersions requestVersion) {
		if (LOGGER.isDebugEnabled()) {
			LOGGER.debug("Trying location: " + location.getId());
		}

		final List<Cache> caches = getSupportingCaches(location.getCaches(), ds, requestVersion);
		if (caches.isEmpty()) {
			if (LOGGER.isDebugEnabled()) {
				LOGGER.debug("No online, supporting caches were found at location: "
						+ location.getId());
			}
			return null;
		}

		return caches;
	}

	/**
	 * Gets a DNS zone that contains a given name.
	 * @param qname The DNS name that the returned zone will contain. This can include wildcards.
	 * @param qtype The
	 * <a href="https://javadoc.io/doc/dnsjava/dnsjava/latest/org/xbill/DNS/Type.html"> of the
	 * record which will be returned for DNS queries for the returned zone.
	 * @param clientAddress The IP address of the client making the DNS request. The zone that is
	 * ultimately returned can depend on blocking configuration for a requested Delivery Service,
	 * if the qname represents a Delivery Service routing name.
	 * @param isDnssecRequest Tells whether or not the request was made using DNSSEC, which will
	 * control whether or not the returned zone is signed.
	 * @param builder Used to build a zone if one has not already been created containing qname.
	 * @return A zone containing records of type qtype that contains qname. This can be null
	 */
	public Zone getZone(final Name qname, final int qtype, final InetAddress clientAddress, final boolean isDnssecRequest, final DNSAccessRecord.Builder builder) {
		return zoneManager.getZone(qname, qtype, clientAddress, isDnssecRequest, builder);
	}

	public void setRequestHeaders(final Set<String> requestHeaders) {
		this.requestHeaders = requestHeaders;
	}

	public Set<String> getRequestHeaders() {
		return requestHeaders;
	}

	public boolean isConsistentDNSRouting() {
		return consistentDNSRouting;
	}

	public boolean isClientSteeringDiversityEnabled() {
		return clientSteeringDiversityEnabled;
	}

	public boolean isDnssecZoneDiffingEnabled() {
		return dnssecZoneDiffingEnabled;
	}

	private List<Cache> enforceGeoRedirect(final Track track, final DeliveryService ds, final String clientIp, final Geolocation queriedClientLocation, final IPVersions requestVersion) {
		final String urlType = ds.getGeoRedirectUrlType();
		track.setResult(ResultType.GEO_REDIRECT);

		if ("NOT_DS_URL".equals(urlType)) {
			// redirect url not belongs to this DS, just redirect it
			return null;
		}

		if (!"DS_URL".equals(urlType)) {
			LOGGER.error("invalid geo redirect url type '" + urlType + "'");
			track.setResult(ResultType.MISS);
			track.setResultDetails(ResultDetails.GEO_NO_CACHE_FOUND);
			return null;
		}

		Geolocation clientLocation = queriedClientLocation;

		//redirect url belongs to this DS, will try return the caches
		if (clientLocation == null) {
			try {
				clientLocation = getLocation(clientIp, ds);
			} catch (GeolocationException e) {
				LOGGER.warn("Failed getting geolocation for client ip " + clientIp + " and delivery service '" + ds.getId() + "'");
			}
		}

		if (clientLocation == null) {
			clientLocation = ds.getMissLocation();
		}

		if (clientLocation == null) {
			LOGGER.error("cannot find a geo location for the client: " + clientIp);
			// particular error was logged in ds.supportLocation
			track.setResult(ResultType.MISS);
			track.setResultDetails(ResultDetails.DS_CLIENT_GEO_UNSUPPORTED);
			return null;
		}

		List<Cache> caches = null;

		try {
			caches = getCachesByGeo(ds, clientLocation, track, requestVersion);
		} catch (GeolocationException e) {
			LOGGER.error("Failed getting caches by geolocation " + e.getMessage());
		}

		if (caches == null) {
			LOGGER.warn(String.format("No Cache found by Geo in NGB redirect"));
			track.setResult(ResultType.MISS);
			track.setResultDetails(ResultDetails.GEO_NO_CACHE_FOUND);
		}

		return caches;
	}

	public void setApplicationContext(final ApplicationContext applicationContext) throws BeansException {
		this.applicationContext = applicationContext;
	}

	public void configurationChanged() {
		if (applicationContext == null) {
			LOGGER.warn("Application Context not yet ready, skipping calling listeners of configuration change");
			return;
		}

		final Map<String, ConfigurationListener> configurationListenerMap = applicationContext.getBeansOfType(ConfigurationListener.class);
		for (final ConfigurationListener configurationListener : configurationListenerMap.values()) {
			configurationListener.configurationChanged();
		}
	}

	public void setSteeringRegistry(final SteeringRegistry steeringRegistry) {
		this.steeringRegistry = steeringRegistry;
	}

	public boolean isEdgeDNSRouting() {
		return edgeDNSRouting;
	}

	public boolean isEdgeHTTPRouting() {
		return edgeHTTPRouting;
	}

	private long getEdgeNSttl() {
		return edgeNSttl;
	}

	private int getEdgeDNSRoutingLimit() {
		return edgeDNSRoutingLimit;
	}

	private int getEdgeHTTPRoutingLimit(final DeliveryService ds) {
		if (ds != null && ds.getMaxDnsIps() != 0 && ds.getMaxDnsIps() != edgeHTTPRoutingLimit) {
			return ds.getMaxDnsIps();
		}

		return edgeHTTPRoutingLimit;
	}

	public Map<String, Geolocation> getDefaultGeoLocationsOverride() {
		return defaultGeolocationsOverride;
	}
}
