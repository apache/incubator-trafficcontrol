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

import org.apache.traffic_control.traffic_router.core.edge.Cache;
import org.apache.traffic_control.traffic_router.core.edge.CacheLocation;
import org.apache.traffic_control.traffic_router.core.edge.CacheRegister;
import org.apache.traffic_control.traffic_router.core.edge.InetRecord;
import org.apache.traffic_control.traffic_router.core.edge.Node.IPVersions;
import org.apache.traffic_control.traffic_router.core.ds.DeliveryService;
import org.apache.traffic_control.traffic_router.core.ds.Dispersion;
import org.apache.traffic_control.traffic_router.core.ds.SteeringRegistry;
import org.apache.traffic_control.traffic_router.core.hash.ConsistentHasher;
import org.apache.traffic_control.traffic_router.core.loc.FederationRegistry;
import org.apache.traffic_control.traffic_router.geolocation.Geolocation;
import org.apache.traffic_control.traffic_router.core.request.DNSRequest;
import org.apache.traffic_control.traffic_router.core.request.HTTPRequest;
import org.apache.traffic_control.traffic_router.core.request.Request;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track;
import org.apache.traffic_control.traffic_router.core.util.CidrAddress;
import org.junit.Before;
import org.junit.Test;
import org.xbill.DNS.Name;
import org.xbill.DNS.Type;

import java.net.MalformedURLException;
import java.net.URL;
import java.util.ArrayList;
import java.util.Collection;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.Map;
import java.util.HashMap;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.containsInAnyOrder;
import static org.hamcrest.Matchers.equalTo;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyBoolean;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.doCallRealMethod;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.spy;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.powermock.reflect.Whitebox.setInternalState;

public class TrafficRouterTest {
    private ConsistentHasher consistentHasher;
    private TrafficRouter trafficRouter;

    private DeliveryService deliveryService;
    private FederationRegistry federationRegistry;

    @Before
    public void before() throws Exception {
        deliveryService = mock(DeliveryService.class);
        when(deliveryService.isAvailable()).thenReturn(true);
        when(deliveryService.isCoverageZoneOnly()).thenReturn(false);
        when(deliveryService.getDispersion()).thenReturn(mock(Dispersion.class));
        when(deliveryService.isAcceptHttp()).thenReturn(true);

        consistentHasher = mock(ConsistentHasher.class);

        when(deliveryService.createURIString(any(HTTPRequest.class), any())).thenReturn("http://atscache.kabletown.net/index.html");

        List<InetRecord> inetRecords = new ArrayList<InetRecord>();
        InetRecord inetRecord = new InetRecord("cname1", 12345);
        inetRecords.add(inetRecord);

        federationRegistry = mock(FederationRegistry.class);
        when(federationRegistry.findInetRecords(any(), any(CidrAddress.class))).thenReturn(inetRecords);

        trafficRouter = mock(TrafficRouter.class);

        CacheRegister cacheRegister = mock(CacheRegister.class);
        when(cacheRegister.getDeliveryService(any(HTTPRequest.class))).thenReturn(deliveryService);

        setInternalState(trafficRouter, "cacheRegister", cacheRegister);
        setInternalState(trafficRouter, "federationRegistry", federationRegistry);
        setInternalState(trafficRouter, "consistentHasher", consistentHasher);
        setInternalState(trafficRouter, "steeringRegistry", mock(SteeringRegistry.class));

        when(trafficRouter.route(any(DNSRequest.class), any(Track.class))).thenCallRealMethod();
        when(trafficRouter.route(any(HTTPRequest.class), any(Track.class))).thenCallRealMethod();
        when(trafficRouter.singleRoute(any(HTTPRequest.class), any(Track.class))).thenCallRealMethod();
        when(trafficRouter.selectDeliveryService(any(Request.class))).thenReturn(deliveryService);
        when(trafficRouter.consistentHashDeliveryService(any(DeliveryService.class), any(HTTPRequest.class), any())).thenCallRealMethod();
        doCallRealMethod().when(trafficRouter).stripSpecialQueryParams(any(HTTPRouteResult.class));
    }

    @Test
    public void itCreatesDnsResultsFromFederationMappingHit() throws Exception {
        final Name name = Name.fromString("edge.example.com");
        DNSRequest request = new DNSRequest("example.com", name, Type.A);
        request.setClientIP("192.168.10.11");
        request.setHostname(name.relativize(Name.root).toString());

        Track track = spy(StatTracker.getTrack());

        when(deliveryService.getRoutingName()).thenReturn("edge");
        when(deliveryService.isDns()).thenReturn(true);

        DNSRouteResult result = trafficRouter.route(request, track);

        assertThat(result.getAddresses(), containsInAnyOrder(new InetRecord("cname1", 12345)));
        verify(track).setRouteType(Track.RouteType.DNS, "edge.example.com");
    }

    @Test
    public void itCreatesHttpResults() throws Exception {
        HTTPRequest httpRequest = new HTTPRequest();
        httpRequest.setClientIP("192.168.10.11");
        httpRequest.setHostname("ccr.example.com");

        Track track = spy(StatTracker.getTrack());

        Cache cache = mock(Cache.class);
        when(cache.hasDeliveryService(anyString())).thenReturn(true);
        CacheLocation cacheLocation = new CacheLocation("", new Geolocation(50,50));

        cacheLocation.addCache(cache);

        Set<CacheLocation> cacheLocationCollection = new HashSet<CacheLocation>();
        cacheLocationCollection.add(cacheLocation);

        CacheRegister cacheRegister = mock(CacheRegister.class);
        when(cacheRegister.getCacheLocations()).thenReturn(cacheLocationCollection);

        when(deliveryService.filterAvailableLocations(any(Collection.class))).thenCallRealMethod();
        when(deliveryService.isLocationAvailable(cacheLocation)).thenReturn(true);

        List<Cache> caches = new ArrayList<Cache>();
        caches.add(cache);
        when(trafficRouter.selectCaches(any(HTTPRequest.class), any(DeliveryService.class), any(Track.class))).thenReturn(caches);
        when(trafficRouter.selectCachesByGeo(any(), any(), any(), any(), any())).thenCallRealMethod();
        when(trafficRouter.getClientLocation(anyString(), any(DeliveryService.class), any(CacheLocation.class), any(Track.class))).thenReturn(new Geolocation(40, -100));
        when(trafficRouter.getCachesByGeo(any(DeliveryService.class), any(Geolocation.class), any(Track.class), any(IPVersions.class))).thenCallRealMethod();
        when(trafficRouter.getCacheRegister()).thenReturn(cacheRegister);
        when(trafficRouter.orderLocations(any(List.class), any(Geolocation.class))).thenCallRealMethod();

        HTTPRouteResult httpRouteResult = trafficRouter.route(httpRequest, track);

        assertThat(httpRouteResult.getUrl().toString(), equalTo("http://atscache.kabletown.net/index.html"));
    }

    @Test
    public void itFiltersByIPAvailability() throws Exception {

        DeliveryService ds = mock(DeliveryService.class);

        Cache cacheIPv4 = mock(Cache.class);
        when(cacheIPv4.hasDeliveryService(any())).thenReturn(true);
        when(cacheIPv4.hasAuthority()).thenReturn(true);
        when(cacheIPv4.isAvailable(any(IPVersions.class))).thenCallRealMethod();
        doCallRealMethod().when(cacheIPv4).setIsAvailable(anyBoolean());
        setInternalState(cacheIPv4, "ipv4Available", true);
        setInternalState(cacheIPv4, "ipv6Available", false);
        cacheIPv4.setIsAvailable(true);
        when(cacheIPv4.getId()).thenReturn("cache IPv4");

        Cache cacheIPv6 = mock(Cache.class);
        when(cacheIPv6.hasDeliveryService(any())).thenReturn(true);
        when(cacheIPv6.hasAuthority()).thenReturn(true);
        when(cacheIPv6.isAvailable(any(IPVersions.class))).thenCallRealMethod();
        doCallRealMethod().when(cacheIPv6).setIsAvailable(anyBoolean());
        setInternalState(cacheIPv6, "ipv4Available", false);
        setInternalState(cacheIPv6, "ipv6Available", true);
        cacheIPv6.setIsAvailable(true);
        when(cacheIPv6.getId()).thenReturn("cache IPv6");

        List<Cache> caches = new ArrayList<Cache>();
        caches.add(cacheIPv4);
        caches.add(cacheIPv6);

        when(trafficRouter.getSupportingCaches(any(), any(), any())).thenCallRealMethod();

        List<Cache> supportingIPv4Caches = trafficRouter.getSupportingCaches(caches, ds, IPVersions.IPV4ONLY);
        assertThat(supportingIPv4Caches.size(), equalTo(1));
        assertThat(supportingIPv4Caches.get(0).getId(), equalTo("cache IPv4"));

        List<Cache> supportingIPv6Caches = trafficRouter.getSupportingCaches(caches, ds, IPVersions.IPV6ONLY);
        assertThat(supportingIPv6Caches.size(), equalTo(1));
        assertThat(supportingIPv6Caches.get(0).getId(), equalTo("cache IPv6"));

        List<Cache> supportingEitherCaches = trafficRouter.getSupportingCaches(caches, ds, IPVersions.ANY);
        assertThat(supportingEitherCaches.size(), equalTo(2));

        cacheIPv6.setIsAvailable(false);
        List<Cache> supportingAvailableCaches = trafficRouter.getSupportingCaches(caches, ds, IPVersions.ANY);
        assertThat(supportingAvailableCaches.size(), equalTo(1));
        assertThat(supportingAvailableCaches.get(0).getId(), equalTo("cache IPv4"));
    }

    @Test
    public void itChecksDefaultLocation() throws Exception {
        String ip = "1.2.3.4";
        Track track = new Track();
        Geolocation geolocation = mock(Geolocation.class);
        when(trafficRouter.getClientLocation(ip, deliveryService, null, track)).thenReturn(geolocation);
        when(geolocation.isDefaultLocation()).thenReturn(true);
        when(geolocation.getCountryCode()).thenReturn("US");
        Map<String, Geolocation> map = new HashMap<>();
        Geolocation defaultUSLocation = new Geolocation(37.751,-97.822);
        defaultUSLocation.setCountryCode("US");
        map.put("US", defaultUSLocation);
        when(trafficRouter.getDefaultGeoLocationsOverride()).thenReturn(map);
        Cache cache = mock(Cache.class);
        List<Cache> list = new ArrayList<>();
        list.add(cache);
        when(deliveryService.getMissLocation()).thenReturn(defaultUSLocation);
        when(trafficRouter.getCachesByGeo(deliveryService, deliveryService.getMissLocation(), track, IPVersions.IPV4ONLY)).thenReturn(list);
        when(trafficRouter.selectCachesByGeo(ip, deliveryService, null, track, IPVersions.IPV4ONLY)).thenCallRealMethod();
        when(trafficRouter.isValidMissLocation(deliveryService)).thenCallRealMethod();
        List<Cache> result = trafficRouter.selectCachesByGeo(ip, deliveryService, null, track, IPVersions.IPV4ONLY);
        verify(trafficRouter).getCachesByGeo(deliveryService, deliveryService.getMissLocation(), track, IPVersions.IPV4ONLY);
        assertThat(result.size(), equalTo(1));
        assertThat(result.get(0), equalTo(cache));
        assertThat(track.getResult(), equalTo(Track.ResultType.GEO_DS));
    }

    @Test
    public void itChecksMissLocation() throws Exception {
        Geolocation defaultUSLocation = new Geolocation(37.751,-97.822);
        when(deliveryService.getMissLocation()).thenReturn(defaultUSLocation);
        when(trafficRouter.isValidMissLocation(deliveryService)).thenCallRealMethod();
        boolean result = trafficRouter.isValidMissLocation(deliveryService);
        assertThat(result, equalTo(true));
        defaultUSLocation = new Geolocation(0,0);
        when(deliveryService.getMissLocation()).thenReturn(defaultUSLocation);
        result = trafficRouter.isValidMissLocation(deliveryService);
        assertThat(result, equalTo(false));
    }

    @Test
    public void itSetsResultToGeo() throws Exception {
        Cache cache = mock(Cache.class);
        when(cache.hasDeliveryService(any())).thenReturn(true);
        CacheLocation cacheLocation = new CacheLocation("", new Geolocation(50,50));

        cacheLocation.addCache(cache);

        Set<CacheLocation> cacheLocationCollection = new HashSet<CacheLocation>();
        cacheLocationCollection.add(cacheLocation);

        CacheRegister cacheRegister = mock(CacheRegister.class);
        when(cacheRegister.getCacheLocations()).thenReturn(cacheLocationCollection);

        when(trafficRouter.getCacheRegister()).thenReturn(cacheRegister);
        when(deliveryService.isLocationAvailable(cacheLocation)).thenReturn(true);
        when(deliveryService.filterAvailableLocations(any())).thenCallRealMethod();

        when(trafficRouter.selectCaches(any(), any(), any())).thenCallRealMethod();
        when(trafficRouter.selectCaches(any(), any(), any(), anyBoolean())).thenCallRealMethod();
        when(trafficRouter.selectCachesByGeo(any(), any(), any(), any(), any())).thenCallRealMethod();

        Geolocation clientLocation = new Geolocation(40, -100);
        when(trafficRouter.getClientLocation(any(), any(), any(), any())).thenReturn(clientLocation);

        when(trafficRouter.getCachesByGeo(any(), any(), any(), any())).thenCallRealMethod();
        when(trafficRouter.filterEnabledLocations(any(), any())).thenCallRealMethod();
        when(trafficRouter.orderLocations(any(), any())).thenCallRealMethod();
        when(trafficRouter.getSupportingCaches(any(), any(), any())).thenCallRealMethod();

        HTTPRequest httpRequest = new HTTPRequest();
        httpRequest.setClientIP("192.168.10.11");
        httpRequest.setHostname("ccr.example.com");
        httpRequest.setPath("/some/path");

        Track track = spy(StatTracker.getTrack());

        trafficRouter.route(httpRequest, track);

        assertThat(track.getResult(), equalTo(Track.ResultType.GEO));
        assertThat(track.getResultLocation(), equalTo(new Geolocation(50, 50)));

        when(federationRegistry.findInetRecords(any(), any(CidrAddress.class))).thenReturn(null);
        when(deliveryService.getRoutingName()).thenReturn("edge");
        when(deliveryService.isDns()).thenReturn(true);

        final Name name = Name.fromString("edge.example.com");
        DNSRequest dnsRequest = new DNSRequest("example.com", name, Type.A);
        dnsRequest.setClientIP("10.10.10.10");
        dnsRequest.setHostname(name.relativize(Name.root).toString());

        track = StatTracker.getTrack();
        trafficRouter.route(dnsRequest, track);

        assertThat(track.getResult(), equalTo(Track.ResultType.GEO));
        assertThat(track.getResultLocation(), equalTo(new Geolocation(50, 50)));
    }

    @Test
    public void itRetainsPathElementsInURI() throws Exception {
        Cache cache = mock(Cache.class);
        when(cache.getFqdn()).thenReturn("atscache-01.kabletown.net");
        when(cache.getPort()).thenReturn(80);

        when(deliveryService.createURIString(any(HTTPRequest.class), any(Cache.class))).thenCallRealMethod();

        HTTPRequest httpRequest = new HTTPRequest();
        httpRequest.setClientIP("192.168.10.11");
        httpRequest.setHostname("tr.ds.kabletown.net");
        httpRequest.setPath("/782-93d215fcd88b/6b6ce2889-ae4c20a1584.ism/manifest(format=m3u8-aapl).m3u8");
        httpRequest.setUri("/782-93d215fcd88b/6b6ce2889-ae4c20a1584.ism;urlsig=O0U9MTQ1Ojhx74tjchm8yzfdanshdafHMNhv8vNA/manifest(format=m3u8-aapl).m3u8");

        StringBuilder dest = new StringBuilder();
        dest.append("http://");
        dest.append(cache.getFqdn().split("\\.", 2)[0]);
        dest.append(".");
        dest.append(httpRequest.getHostname().split("\\.", 2)[1]);
        dest.append(httpRequest.getUri());

        assertThat(deliveryService.createURIString(httpRequest, cache), equalTo(dest.toString()));
    }

    @Test
    public void itStripsSpecialQueryParameters() throws MalformedURLException {
        HTTPRouteResult result = new HTTPRouteResult(false);
        result.setUrl(new URL("http://example.org/foo?trred=false&fakeClientIpAddress=192.168.0.2"));
        trafficRouter.stripSpecialQueryParams(result);
        assertThat(result.getUrl().toString(), equalTo("http://example.org/foo"));

        result.setUrl(new URL("http://example.org/foo?b=1&trred=false&a=2&asdf=foo&fakeClientIpAddress=192.168.0.2&c=3"));
        trafficRouter.stripSpecialQueryParams(result);
        assertThat(result.getUrl().toString(), equalTo("http://example.org/foo?b=1&a=2&asdf=foo&c=3"));
    }
}
