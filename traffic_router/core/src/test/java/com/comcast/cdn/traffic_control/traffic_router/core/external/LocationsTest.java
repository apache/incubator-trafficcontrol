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

package com.comcast.cdn.traffic_control.traffic_router.core.external;

import com.comcast.cdn.traffic_control.traffic_router.core.util.ExternalTest;
import com.fasterxml.jackson.core.JsonFactory;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.catalina.LifecycleException;
import org.apache.http.client.methods.CloseableHttpResponse;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.methods.HttpHead;
import org.apache.http.impl.client.CloseableHttpClient;
import org.apache.http.impl.client.HttpClientBuilder;
import org.apache.http.util.EntityUtils;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.junit.experimental.categories.Category;

import java.util.ArrayList;
import java.util.List;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.core.AnyOf.anyOf;
import static org.hamcrest.core.IsEqual.equalTo;
import static org.hamcrest.core.IsNot.not;
import static org.junit.Assert.fail;

@Category(ExternalTest.class)
public class LocationsTest {
	private CloseableHttpClient closeableHttpClient;

	@Before
	public void before() throws LifecycleException {
		closeableHttpClient = HttpClientBuilder.create().build();
	}

	@After
	public void after() throws Exception{
		if (closeableHttpClient != null) closeableHttpClient.close();
	}

	@Test
	public void itGetsAListOfLocations() throws Exception {
		HttpGet httpGet = new HttpGet("http://localhost:3333/crs/locations");

		CloseableHttpResponse response = null;
		try {
			response = closeableHttpClient.execute(httpGet);
			assertThat(response.getStatusLine().getStatusCode(), equalTo(200));

			ObjectMapper objectMapper = new ObjectMapper(new JsonFactory());
			JsonNode jsonNode = objectMapper.readTree(EntityUtils.toString(response.getEntity()));
			assertThat(jsonNode.get("locations").get(0).asText(), not(equalTo("")));
		} finally {
			if (response != null) response.close();
		}
	}
	
	@Test
	public void itGetsAListOfCaches() throws Exception {
		HttpGet httpGet = new HttpGet("http://localhost:3333/crs/locations/caches");
		
		CloseableHttpResponse response = null;
		try {
			response = closeableHttpClient.execute(httpGet);

			ObjectMapper objectMapper = new ObjectMapper(new JsonFactory());
			JsonNode jsonNode = objectMapper.readTree(EntityUtils.toString(response.getEntity()));
			String locationName = jsonNode.get("locations").fieldNames().next();
			JsonNode cacheNode = jsonNode.get("locations").get(locationName).get(0);

			assertThat(cacheNode.get("cacheId").asText(), not(equalTo("")));
			assertThat(cacheNode.get("fqdn").asText(), not(equalTo("")));

			assertThat(cacheNode.get("ipAddresses").isArray(), equalTo(true));
			assertThat(cacheNode.has("adminStatus"), equalTo(true));

			assertThat(cacheNode.get("port").asInt(-123456), not(equalTo(-123456)));
			assertThat(cacheNode.get("lastUpdateTime").asInt(-123456), not(equalTo(-123456)));
			assertThat(cacheNode.get("connections").asInt(-123456), not(equalTo(-123456)));
			assertThat(cacheNode.get("currentBW").asInt(-123456), not(equalTo(-123456)));
			assertThat(cacheNode.get("availBW").asInt(-123456), not(equalTo(-123456)));

			assertThat(cacheNode.get("cacheOnline").asText(), anyOf(equalTo("true"), equalTo("false")) );
			assertThat(cacheNode.get("lastUpdateHealthy").asText(), anyOf(equalTo("true"), equalTo("false")) );
		} finally {
			if (response != null) response.close();
		}
	}

	@Test
	public void itGetsCachesForALocation() throws Exception {
		HttpGet httpGet = new HttpGet("http://localhost:3333/crs/locations");

		CloseableHttpResponse response = null;
		try {
			response = closeableHttpClient.execute(httpGet);

			ObjectMapper objectMapper = new ObjectMapper(new JsonFactory());
			JsonNode jsonNode = objectMapper.readTree(EntityUtils.toString(response.getEntity()));

			String location = jsonNode.get("locations").get(0).asText();

			httpGet = new HttpGet("http://localhost:3333/crs/locations/" + location + "/caches");

			response = closeableHttpClient.execute(httpGet);

			jsonNode = objectMapper.readTree(EntityUtils.toString(response.getEntity()));

			assertThat(jsonNode.get("caches").isArray(), equalTo(true));
			JsonNode cacheNode = jsonNode.get("caches").get(0);

			assertThat(cacheNode.get("cacheId").asText(), not(equalTo("")));
			assertThat(cacheNode.get("fqdn").asText(), not(equalTo("")));

			assertThat(cacheNode.get("ipAddresses").isArray(), equalTo(true));
			assertThat(cacheNode.has("adminStatus"), equalTo(true));

			assertThat(cacheNode.get("port").asInt(-123456), not(equalTo(-123456)));
			assertThat(cacheNode.get("lastUpdateTime").asInt(-123456), not(equalTo(-123456)));
			assertThat(cacheNode.get("connections").asInt(-123456), not(equalTo(-123456)));
			assertThat(cacheNode.get("currentBW").asInt(-123456), not(equalTo(-123456)));
			assertThat(cacheNode.get("availBW").asInt(-123456), not(equalTo(-123456)));

			assertThat(cacheNode.get("cacheOnline").asText(), anyOf(equalTo("true"), equalTo("false")) );
			assertThat(cacheNode.get("lastUpdateHealthy").asText(), anyOf(equalTo("true"), equalTo("false")) );

		} finally {
			if (response != null) response.close();
		}
	}

	@Test
	public void itHandlesHeadRequests() throws Exception {
		final List<String> paths = new ArrayList<String>();
		paths.add("http://localhost:3333/crs/locations");
		paths.add("http://localhost:3333/crs/locations/caches");

		CloseableHttpResponse response = null;

		try {
			final HttpGet httpGet = new HttpGet("http://localhost:3333/crs/locations");
			response = closeableHttpClient.execute(httpGet);

			ObjectMapper objectMapper = new ObjectMapper(new JsonFactory());
			JsonNode jsonNode = objectMapper.readTree(EntityUtils.toString(response.getEntity()));

			String location = jsonNode.get("locations").get(0).asText();
			paths.add("http://localhost:3333/crs/locations/" + location + "/caches");
		} catch (Exception e) {
			fail(e.getMessage());
		} finally {
			if (response != null) response.close();
		}

		for (final String path : paths) {
			final HttpHead httpHead = new HttpHead(path);
			try {
				response = closeableHttpClient.execute(httpHead);
				assertThat(response.getStatusLine().getStatusCode(), equalTo(200));
			} finally {
				if (response != null) response.close();
			}
		}
	}
}
