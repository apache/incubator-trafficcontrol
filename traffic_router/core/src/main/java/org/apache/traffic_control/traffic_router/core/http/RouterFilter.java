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

package org.apache.traffic_control.traffic_router.core.http;

import org.apache.traffic_control.traffic_router.core.ds.DeliveryService;
import org.apache.traffic_control.traffic_router.core.request.HTTPRequest;
import org.apache.traffic_control.traffic_router.core.router.HTTPRouteResult;
import org.apache.traffic_control.traffic_router.core.router.StatTracker;
import org.apache.traffic_control.traffic_router.core.router.TrafficRouter;
import org.apache.traffic_control.traffic_router.core.router.TrafficRouterManager;
import org.apache.traffic_control.traffic_router.geolocation.GeolocationException;
import org.apache.log4j.Logger;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.filter.OncePerRequestFilter;

import javax.servlet.FilterChain;
import javax.servlet.ServletException;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.net.URL;
import java.util.Date;
import java.util.List;
import java.util.Map;
import java.util.Set;

public class RouterFilter extends OncePerRequestFilter {
	private static final Logger ACCESS = Logger.getLogger("org.apache.traffic_control.traffic_router.core.access");
	public static final String REDIRECT_QUERY_PARAM = "trred";
	private static final String HEAD = "HEAD";

	@Autowired
	private TrafficRouterManager trafficRouterManager;

	@Autowired
	private StatTracker statTracker;

	private List<String> staticContentWhiteList;

	private boolean doNotLog = false;

	@Override
	public void doFilterInternal(final HttpServletRequest request, final HttpServletResponse response, final FilterChain chain) throws IOException, ServletException {
		final Date requestDate = new Date();

		if (request.getLocalPort() == trafficRouterManager.getApiPort() || request.getLocalPort() == trafficRouterManager.getSecureApiPort()) {
			chain.doFilter(request, response);
			return;
		}

		if (staticContentWhiteList.contains(request.getRequestURI())) {
			chain.doFilter(request, response);

			if (doNotLog) {
				return;
			}

			final HTTPAccessRecord access = new HTTPAccessRecord.Builder(requestDate, request).build();
			ACCESS.info(HTTPAccessEventBuilder.create(access));
			return;
		}

		final HTTPAccessRecord httpAccessRecord = new HTTPAccessRecord.Builder(requestDate, request).build();
		writeHttpResponse(response, request, new HTTPRequest(request), StatTracker.getTrack(), httpAccessRecord);
	}

	@SuppressWarnings("PMD.CyclomaticComplexity")
	private void writeHttpResponse(final HttpServletResponse response, final HttpServletRequest httpServletRequest,
	                               final HTTPRequest request, final StatTracker.Track track, final HTTPAccessRecord httpAccessRecord) throws IOException {
		final HTTPAccessRecord.Builder httpAccessRecordBuilder = new HTTPAccessRecord.Builder(httpAccessRecord);
		HTTPRouteResult routeResult = null;


		try {
			final TrafficRouter trafficRouter = trafficRouterManager.getTrafficRouter();
			routeResult = trafficRouter.route(request, track);

			if (routeResult == null || routeResult.getUrl() == null) {
				setErrorResponseCode(response, httpAccessRecordBuilder, routeResult);
			} else if (routeResult.isMultiRouteRequest()) {
				setMultiResponse(routeResult, httpServletRequest, response, httpAccessRecordBuilder);
			} else {
				setSingleResponse(routeResult, httpServletRequest, response, httpAccessRecordBuilder);
			}
		} catch (final IOException e) {
			httpAccessRecordBuilder.responseCode(-1);
			httpAccessRecordBuilder.responseURL(null);
			httpAccessRecordBuilder.rerr(e.getMessage());
			throw e;
		} catch (final GeolocationException e) {
			httpAccessRecordBuilder.responseCode(-1);
			httpAccessRecordBuilder.responseURL(null);
			httpAccessRecordBuilder.rerr(e.getMessage());
		} finally {
			final Set<String> requestHeaders = trafficRouterManager.getTrafficRouter().getRequestHeaders();

			if (routeResult != null && routeResult.getRequestHeaders() != null) {
				requestHeaders.addAll(routeResult.getRequestHeaders());
			}

			final Map<String,String> accessRequestHeaders = new HttpAccessRequestHeaders().makeMap(httpServletRequest, requestHeaders);

			String deliveryServiceIds = "";
			if (routeResult != null && routeResult.getDeliveryServices().size() > 0) {
				deliveryServiceIds = routeResult.getDeliveryServicesLogString();
			}

			final HTTPAccessRecord access = httpAccessRecordBuilder.resultType(track.getResult())
				.resultDetails(track.getResultDetails())
				.resultLocation(track.getResultLocation())
				.requestHeaders(accessRequestHeaders)
				.regionalGeoResult(track.getRegionalGeoResult())
					.deliveryServiceIds(deliveryServiceIds)
				.build();
			ACCESS.info(HTTPAccessEventBuilder.create(access));
			statTracker.saveTrack(track);
		}
	}

	private void setMultiResponse(final HTTPRouteResult routeResult, final HttpServletRequest httpServletRequest, final HttpServletResponse response, final HTTPAccessRecord.Builder httpAccessRecordBuilder) throws IOException {
		if (routeResult.getDeliveryService() != null) {
			final Map<String, String> responseHeaders = routeResult.getDeliveryService().getResponseHeaders();

			for (final String key : responseHeaders.keySet()) {
				// if two DSs append the same header, the last one wins; no way around it unless we enforce unique response headers between subordinate DSs
				response.addHeader(key, responseHeaders.get(key));
			}
		}

		final String redirect = httpServletRequest.getParameter(REDIRECT_QUERY_PARAM);

		if (!HEAD.equals(httpServletRequest.getMethod())) {
			response.setContentType("application/json");
			response.getWriter().println(routeResult.toMultiLocationJSONString());
			httpAccessRecordBuilder.responseURLs(routeResult.getUrls());
		}

		// don't actually parse the boolean value; trred would always be false unless the query param is "true"
		if ("false".equalsIgnoreCase(redirect)) {
			response.setStatus(HttpServletResponse.SC_OK);
			httpAccessRecordBuilder.responseCode(HttpServletResponse.SC_OK);
		} else {
			response.setHeader("Location", routeResult.getUrl().toString());
			response.setStatus(HttpServletResponse.SC_MOVED_TEMPORARILY);
			httpAccessRecordBuilder.responseCode(HttpServletResponse.SC_MOVED_TEMPORARILY);
			httpAccessRecordBuilder.responseURL(routeResult.getUrl());
		}
	}

	private void setSingleResponse(final HTTPRouteResult routeResult, final HttpServletRequest httpServletRequest, final HttpServletResponse response, final HTTPAccessRecord.Builder httpAccessRecordBuilder) throws IOException {
		final String redirect = httpServletRequest.getParameter(REDIRECT_QUERY_PARAM);
		final String format = httpServletRequest.getParameter("format");
		final URL location = routeResult.getUrl();

		if (routeResult.getDeliveryService() != null) {
			final DeliveryService deliveryService = routeResult.getDeliveryService();
			final Map<String, String> responseHeaders = deliveryService.getResponseHeaders();

			for (final String key : responseHeaders.keySet()) {
				response.addHeader(key, responseHeaders.get(key));
			}
		}

		if ("false".equalsIgnoreCase(redirect)) {
			if (!HEAD.equals(httpServletRequest.getMethod())) {
				response.setContentType("application/json");
				response.getWriter().println(routeResult.toMultiLocationJSONString());
				httpAccessRecordBuilder.responseURLs(routeResult.getUrls());
			}

			httpAccessRecordBuilder.responseCode(HttpServletResponse.SC_OK);
		} else if ("json".equals(format)) {
			if (!HEAD.equals(httpServletRequest.getMethod())) {
				response.setContentType("application/json");
				response.getWriter().println(routeResult.toLocationJSONString());
				httpAccessRecordBuilder.responseURL(location);
			}

			httpAccessRecordBuilder.responseCode(HttpServletResponse.SC_OK);
		} else {
			response.sendRedirect(location.toString());
			httpAccessRecordBuilder.responseCode(HttpServletResponse.SC_MOVED_TEMPORARILY);
			httpAccessRecordBuilder.responseURL(location);
		}
	}

	private void setErrorResponseCode(final HttpServletResponse response,
	                                  final HTTPAccessRecord.Builder httpAccessRecordBuilder, final HTTPRouteResult result) throws IOException {

		if (result != null && result.getResponseCode() > 0) {
			httpAccessRecordBuilder.responseCode(result.getResponseCode());
			response.sendError(result.getResponseCode());
			return;
		}

		httpAccessRecordBuilder.responseCode(HttpServletResponse.SC_SERVICE_UNAVAILABLE);
		response.sendError(HttpServletResponse.SC_SERVICE_UNAVAILABLE);
	}

	public void setDoNotLog(final String logAccessString) {
		this.doNotLog = Boolean.valueOf(logAccessString);
	}

	public void setStaticContentWhiteList(final List<String> staticContentWhiteList) {
		this.staticContentWhiteList = staticContentWhiteList;
	}
}
