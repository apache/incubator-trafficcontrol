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

import java.net.URL;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.StringJoiner;
import java.util.stream.Collectors;

import org.apache.traffic_control.traffic_router.core.ds.DeliveryService;

public class HTTPRouteResult implements RouteResult {
	final private List<URL> urls = new ArrayList<URL>();
	final private List<DeliveryService> deliveryServices = new ArrayList<DeliveryService>();
	final private boolean multiRouteRequest;
	private int responseCode;

	public HTTPRouteResult(final boolean multiRouteRequest) {
		this.multiRouteRequest = multiRouteRequest;
	}

	@Override
	public Object getResult() {
		return getUrls();
	}

	public List<URL> getUrls() {
		return urls;
	}

	public void addUrl(final URL url) {
		urls.add(url);
	}

	public URL getUrl() {
		return !urls.isEmpty() ? urls.get(0) : null;
	}

	public void setUrl(final URL url) {
		urls.clear();
		urls.add(url);
	}

	public List<DeliveryService> getDeliveryServices() {
		return deliveryServices;
	}

	public String getDeliveryServicesLogString() {
		return this.getDeliveryServices().stream().map(DeliveryService::getId).collect(Collectors.joining("|"));
	}

	public void addDeliveryService(final DeliveryService deliveryService) {
		deliveryServices.add(deliveryService);
	}

	public DeliveryService getDeliveryService() {
		return !deliveryServices.isEmpty() ? deliveryServices.get(0) : null;
	}

	public void setDeliveryService(final DeliveryService deliveryService) {
		deliveryServices.clear();
		deliveryServices.add(deliveryService);
	}

	public int getResponseCode() {
		return responseCode;
	}

	public void setResponseCode(final int rc) {
		this.responseCode = rc;
	}

	public String toLocationJSONString() {
		return "{\"location\": \"" + getUrl().toString() + "\" }";
	}

	public String toMultiLocationJSONString() {
		final StringJoiner joiner = new StringJoiner("\",\"");

		for (final URL url : urls) {
			joiner.add(url.toString());
		}

		return "{\"locations\":[\"" + joiner.toString() + "\"]}";
	}

	public Set<String> getRequestHeaders() {
		final Set<String> requestHeaders = new HashSet<String>();

		for (final DeliveryService deliveryService : getDeliveryServices()) {
			requestHeaders.addAll(deliveryService.getRequestHeaders());
		}

		return requestHeaders;
	}

	public boolean isMultiRouteRequest() {
		return multiRouteRequest;
	}
}
