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

import com.comcast.cdn.traffic_control.traffic_router.core.http.RouterFilter;
import com.comcast.cdn.traffic_control.traffic_router.core.util.ExternalTest;
import com.fasterxml.jackson.core.JsonFactory;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.catalina.LifecycleException;
import org.apache.http.Header;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.methods.CloseableHttpResponse;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.methods.HttpHead;
import org.apache.http.client.methods.HttpRequestBase;
import org.apache.http.impl.client.CloseableHttpClient;
import org.apache.http.impl.client.HttpClientBuilder;
import org.apache.http.util.EntityUtils;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.junit.experimental.categories.Category;

import java.io.IOException;
import java.net.URLEncoder;
import java.util.ArrayList;
import java.util.List;

import static org.hamcrest.Matchers.notNullValue;
import static org.hamcrest.Matchers.nullValue;
import static org.hamcrest.core.IsEqual.equalTo;
import static org.junit.Assert.assertThat;

@Category(ExternalTest.class)
public class BufferedResponseTest {
	private String routerHttpPort = System.getProperty("routerHttpPort", "8888");
	private CloseableHttpClient httpClient;

	@Before
	public void before() throws LifecycleException {
		httpClient = HttpClientBuilder.create().build();
	}

	@After
	public void after() throws Exception {
		if (httpClient != null) httpClient.close();
	}

	@Test
	public void itSetsContentLengthHeaderFor404() throws IOException {
		final String encodedUrl = URLEncoder.encode("http://trafficrouter01.somedeliveryservice.somecdn.domain.foo/stuff", "utf-8");
		final HttpGet httpGet = new HttpGet("http://localhost:3333/crs/deliveryservices?url=" + encodedUrl);
		CloseableHttpResponse response = null;

		try {
			response = httpClient.execute(httpGet);
			assertThat(response.getStatusLine().getStatusCode(), equalTo(404));
			assertThat(response.getFirstHeader("Transfer-Encoding"), nullValue());
			assertThat(response.getFirstHeader("Content-Length"), notNullValue());
		} finally {
			if (response != null) response.close();
		}
	}

	@Test
	public void itSetsTheSameContentLengthForHeadAndGet() throws IOException {
		final List<String> paths = new ArrayList<>();
		paths.add("http://localhost:3333/crs/stats");
		paths.add("http://localhost:3333/crs/locations/caches");
		paths.add("http://localhost:3333/crs/consistenthash/deliveryservice?deliveryServiceId=csd-target-1&requestPath=/");

		for (final String path : paths) {
			final List<HttpRequestBase> requests = new ArrayList<>();
			requests.add(new HttpHead(path));
			requests.add(new HttpGet(path));

			final List<Integer> contentLengths = new ArrayList<>();

			for (final HttpRequestBase request : requests) {
				CloseableHttpResponse response = null;

				try {
					response = httpClient.execute(request);
					final Header contentLengthHeader = response.getFirstHeader("Content-Length");

					assertThat(response.getFirstHeader("Transfer-Encoding"), nullValue());
					assertThat(contentLengthHeader, notNullValue());
					contentLengths.add(Integer.parseInt(contentLengthHeader.getValue()));
				} finally {
					if (response != null) response.close();
				}
			}

			assertThat(contentLengths.size(), equalTo(2));
			assertThat("Expected HEAD and GET requests for " + path + "to have the same Content-Length", contentLengths.get(0), equalTo(contentLengths.get(1)));
		}
	}

	@Test
	public void itSetsAnAccurateContentLengthForGet() throws IOException {
		final List<String> paths = new ArrayList<>();
		paths.add("http://localhost:3333/crs/stats");
		paths.add("http://localhost:3333/crs/locations/caches");
		paths.add("http://localhost:3333/crs/consistenthash/deliveryservice?deliveryServiceId=csd-target-1&requestPath=/");

		for (final String path : paths) {
			CloseableHttpResponse response = null;

			try {
				final HttpGet httpGet = new HttpGet(path);
				response = httpClient.execute(httpGet);

				final ObjectMapper objectMapper = new ObjectMapper(new JsonFactory());
				final String json = EntityUtils.toString(response.getEntity());
				final Header contentLengthHeader = response.getFirstHeader("Content-Length");

				/* If the content length is too low and cuts off the response
				 * body, objectMapper.readTree(json) will likely throw a
				 * JsonProcessingException.
				 */
				objectMapper.readTree(json);
				assertThat(response.getFirstHeader("Transfer-Encoding"), nullValue());
				assertThat(contentLengthHeader, notNullValue());
				assertThat(Integer.parseInt(contentLengthHeader.getValue()), equalTo(json.length()));
			} finally {
				if (response != null) response.close();
			}
		}
	}

	@Test
	public void itSetsContentLengthHeaderForDeliveryServiceSteering() throws IOException {
		final String testHostName = "tr.client-steering-test-1.thecdn.example.com";
		final RequestConfig config = RequestConfig.custom().setRedirectsEnabled(false).build();
		final List<String> paths = new ArrayList<>();
		paths.add("/qwerytuiop/asdfghjkl?fakeClientIpAddress=12.34.56.78");
		paths.add("/qwerytuiop/asdfghjkl?fakeClientIpAddress=12.34.56.78&format=json");
		paths.add("/qwerytuiop/asdfghjkl?fakeClientIpAddress=12.34.56.78&" + RouterFilter.REDIRECT_QUERY_PARAM + "=false");
		paths.add("/qwerytuiop/asdfghjkl?fakeClientIpAddress=12.34.56.78&" + RouterFilter.REDIRECT_QUERY_PARAM + "=true");


		for (final String path : paths) {
			final List<HttpRequestBase> requests = new ArrayList<>();
			requests.add(new HttpHead("http://localhost:" + routerHttpPort + path));
			requests.add(new HttpGet("http://localhost:" + routerHttpPort + path));

			for (final HttpRequestBase request : requests) {
				CloseableHttpResponse response = null;
				request.setConfig(config);
				request.addHeader("Host", testHostName);

				try {
					response = httpClient.execute(request);

					assertThat(response.getFirstHeader("Transfer-Encoding"), nullValue());
					assertThat(response.getFirstHeader("Content-Length"), notNullValue());
				} finally {
					if (response != null) response.close();
				}
			}
		}
	}
}
