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

import org.apache.traffic_control.traffic_router.core.request.HTTPRequest;
import org.apache.traffic_control.traffic_router.geolocation.Geolocation;
import org.apache.traffic_control.traffic_router.core.router.StatTracker;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track.ResultType;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track.ResultDetails;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.invocation.InvocationOnMock;
import org.mockito.stubbing.Answer;
import org.powermock.core.classloader.annotations.PrepareForTest;
import org.powermock.modules.junit4.PowerMockRunner;

import javax.servlet.http.HttpServletRequest;
import java.net.URL;
import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.containsString;
import static org.hamcrest.Matchers.equalTo;
import static org.hamcrest.Matchers.not;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;
import static org.powermock.api.mockito.PowerMockito.mockStatic;
import static org.powermock.api.mockito.PowerMockito.whenNew;

@RunWith(PowerMockRunner.class)
@PrepareForTest({Date.class, HTTPAccessEventBuilder.class, HTTPAccessRecord.class, System.class})
public class HTTPAccessEventBuilderTest {
    private HttpServletRequest request;

    @Before
    public void before() throws Exception {
        mockStatic(Date.class);
        Date startDate = mock(Date.class);
        when(startDate.getTime()).thenReturn(144140678000L);
        whenNew(Date.class).withArguments(anyLong()).thenReturn(startDate);

        Date finishDate = mock(Date.class);
        when(finishDate.getTime()).thenReturn(144140678125L);
        whenNew(Date.class).withNoArguments().thenReturn(finishDate);

        request = mock(HttpServletRequest.class);
        when(request.getRequestURL()).thenReturn(new StringBuffer("http://example.com/index.html?foo=bar"));
        when(request.getMethod()).thenReturn("GET");
        when(request.getProtocol()).thenReturn("HTTP/1.1");
        when(request.getRemoteAddr()).thenReturn("192.168.7.6");

        mockStatic(System.class);
    }

    @Test
    public void itGeneratesAccessEvents() throws Exception {
        HTTPAccessRecord.Builder builder = new HTTPAccessRecord.Builder(new Date(144140678000L), request);
        HTTPAccessRecord httpAccessRecord = builder.build();

        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);

        assertThat(httpAccessEvent, equalTo("144140678.000 qtype=HTTP chi=192.168.7.6 rhi=- url=\"http://example.com/index.html?foo=bar\" cqhm=GET cqhv=HTTP/1.1 rtype=- rloc=\"-\" rdtl=- rerr=\"-\" rgb=\"-\" rurl=\"-\" rurls=\"-\" uas=\"null\" svc=\"-\" rh=\"-\""));
    }

    @Test
    public void itAddsResponseData() throws Exception {
        Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {100111001L, 225111001L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 2];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);

        StatTracker.Track track = new StatTracker.Track();
        HTTPAccessRecord.Builder builder = new HTTPAccessRecord.Builder(new Date(144140633999L), request)
            .resultType(track.getResult())
            .resultLocation(new Geolocation(39.7528,-104.9997))
            .responseCode(302)
            .responseURL(new URL("http://example.com/hereitis/index.html?foo=bar"));

        HTTPAccessRecord httpAccessRecord = builder.resultType(ResultType.CZ).build();
        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);

        assertThat(httpAccessEvent, equalTo("144140678.000 qtype=HTTP chi=192.168.7.6 rhi=- url=\"http://example.com/index.html?foo=bar\" cqhm=GET cqhv=HTTP/1.1 rtype=CZ rloc=\"39.75,-104.99\" rdtl=- rerr=\"-\" rgb=\"-\" pssc=302 ttms=125.000 rurl=\"http://example.com/hereitis/index.html?foo=bar\" rurls=\"-\" uas=\"null\" svc=\"-\" rh=\"-\""));
    }

    @Test
    public void itAddsMuiltiResponseData() throws Exception {
        Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {100111001L, 225111001L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 2];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);

        List<URL> urls = new ArrayList<URL>();
        urls.add(new URL("http://example.com/hereitis/index.html?foo=bar"));
        urls.add(new URL("http://example.com/thereitis/index.html?boo=baz"));

        StatTracker.Track track = new StatTracker.Track();
        HTTPAccessRecord.Builder builder = new HTTPAccessRecord.Builder(new Date(144140633999L), request)
            .resultType(track.getResult())
            .resultLocation(new Geolocation(39.7528,-104.9997))
            .responseCode(302)
            .responseURL(new URL("http://example.com/hereitis/index.html?foo=bar"))
            .responseURLs(urls);

        HTTPAccessRecord httpAccessRecord = builder.resultType(ResultType.CZ).build();
        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);

        assertThat(httpAccessEvent, equalTo("144140678.000 qtype=HTTP chi=192.168.7.6 rhi=- url=\"http://example.com/index.html?foo=bar\" cqhm=GET cqhv=HTTP/1.1 rtype=CZ rloc=\"39.75,-104.99\" rdtl=- rerr=\"-\" rgb=\"-\" pssc=302 ttms=125.000 rurl=\"http://example.com/hereitis/index.html?foo=bar\" rurls=\"[http://example.com/hereitis/index.html?foo=bar, http://example.com/thereitis/index.html?boo=baz]\" uas=\"null\" svc=\"-\" rh=\"-\""));
    }

    @Test
    public void itRoundsUpToNearestMicroSecond() throws Exception {
        Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {100111001L, 100234999L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 2];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);

        Date fastFinishDate = mock(Date.class);
        when(fastFinishDate.getTime()).thenReturn(144140678000L);
        whenNew(Date.class).withNoArguments().thenReturn(fastFinishDate);

        StatTracker.Track track = new StatTracker.Track();
        HTTPAccessRecord.Builder builder = new HTTPAccessRecord.Builder(new Date(144140633999L), request)
                .resultType(track.getResult())
                .responseCode(302)
                .responseURL(new URL("http://example.com/hereitis/index.html?foo=bar"));

        HTTPAccessRecord httpAccessRecord = builder.build();
        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);

        assertThat(httpAccessEvent, equalTo("144140678.000 qtype=HTTP chi=192.168.7.6 rhi=- url=\"http://example.com/index.html?foo=bar\" cqhm=GET cqhv=HTTP/1.1 rtype=ERROR rloc=\"-\" rdtl=- rerr=\"-\" rgb=\"-\" pssc=302 ttms=0.124 rurl=\"http://example.com/hereitis/index.html?foo=bar\" rurls=\"-\" uas=\"null\" svc=\"-\" rh=\"-\""));
    }


    @Test
    public void itRecordsTrafficRouterErrors() throws Exception {
        Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {111001L, 567002L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 2];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);

        Date fastFinishDate = mock(Date.class);
        when(fastFinishDate.getTime()).thenReturn(144140678000L);
        whenNew(Date.class).withNoArguments().thenReturn(fastFinishDate);

        StatTracker.Track track = new StatTracker.Track();
        HTTPAccessRecord.Builder builder = new HTTPAccessRecord.Builder(new Date(144140633999L), request)
                .resultType(track.getResult())
                .responseCode(302)
                .rerr("RuntimeException: you're doing it wrong")
                .responseURL(new URL("http://example.com/hereitis/index.html?foo=bar"));

        HTTPAccessRecord httpAccessRecord = builder.build();
        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);

        assertThat(httpAccessEvent, equalTo("144140678.000 qtype=HTTP chi=192.168.7.6 rhi=- url=\"http://example.com/index.html?foo=bar\" cqhm=GET cqhv=HTTP/1.1 rtype=ERROR rloc=\"-\" rdtl=- rerr=\"RuntimeException: you're doing it wrong\" rgb=\"-\" pssc=302 ttms=0.456 rurl=\"http://example.com/hereitis/index.html?foo=bar\" rurls=\"-\" uas=\"null\" svc=\"-\" rh=\"-\""));
    }
    
    @Test
    public void itRecordsMissResultDetails() throws Exception {
        Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {100000101L, 100789000L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 2];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);

        Date fastFinishDate = mock(Date.class);
        when(fastFinishDate.getTime()).thenReturn(144140678000L);
        whenNew(Date.class).withNoArguments().thenReturn(fastFinishDate);

        HTTPAccessRecord.Builder builder = new HTTPAccessRecord.Builder(new Date(144140633999L), request)
                .resultType(ResultType.MISS)
                .resultDetails(ResultDetails.DS_NO_BYPASS)
                .responseCode(503);

        HTTPAccessRecord httpAccessRecord = builder.build();
        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);

        assertThat(httpAccessEvent, equalTo("144140678.000 qtype=HTTP chi=192.168.7.6 rhi=- url=\"http://example.com/index.html?foo=bar\" cqhm=GET cqhv=HTTP/1.1 rtype=MISS rloc=\"-\" rdtl=DS_NO_BYPASS rerr=\"-\" rgb=\"-\" pssc=503 ttms=0.789 rurl=\"-\" rurls=\"-\" uas=\"null\" svc=\"-\" rh=\"-\""));
    }

    @Test
    public void itRecordsRequestHeaders() throws Exception {
        Map<String, String> httpAccessRequestHeaders = new HashMap<String, String>();
        httpAccessRequestHeaders.put("If-Modified-Since", "Thurs, 15 July 2010 12:00:00 UTC");
        httpAccessRequestHeaders.put("Accept", "text/*, text/html, text/html;level=1, */*");
        httpAccessRequestHeaders.put("Arbitrary", "The cow says \"moo\"");

        StatTracker.Track track = new StatTracker.Track();
        HTTPAccessRecord.Builder builder = new HTTPAccessRecord.Builder(new Date(144140633999L), request)
            .resultType(track.getResult())
            .resultLocation(new Geolocation(39.7528,-104.9997))
            .responseCode(302)
            .responseURL(new URL("http://example.com/hereitis/index.html?foo=bar"))
            .requestHeaders(httpAccessRequestHeaders);

        HTTPAccessRecord httpAccessRecord = builder.resultType(ResultType.CZ).build();
        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);


        assertThat(httpAccessEvent, not(containsString(" rh=\"-\"")));
        assertThat(httpAccessEvent, containsString("rh=\"If-Modified-Since: Thurs, 15 July 2010 12:00:00 UTC\""));
        assertThat(httpAccessEvent, containsString("rh=\"Accept: text/*, text/html, text/html;level=1, */*\""));
        assertThat(httpAccessEvent, containsString("rh=\"Arbitrary: The cow says 'moo'"));
    }

    @Test
    public void itUsesXMmClientIpHeaderForChi() throws Exception {
        when(request.getHeader(HTTPRequest.X_MM_CLIENT_IP)).thenReturn("192.168.100.100");
        when(request.getRemoteAddr()).thenReturn("12.34.56.78");

        HTTPAccessRecord httpAccessRecord = new HTTPAccessRecord.Builder(new Date(144140678000L), request).build();
        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);

        assertThat(httpAccessEvent, equalTo("144140678.000 qtype=HTTP chi=192.168.100.100 rhi=12.34.56.78 url=\"http://example.com/index.html?foo=bar\" cqhm=GET cqhv=HTTP/1.1 rtype=- rloc=\"-\" rdtl=- rerr=\"-\" rgb=\"-\" rurl=\"-\" rurls=\"-\" uas=\"null\" svc=\"-\" rh=\"-\""));
    }

    @Test
    public void itUsesFakeIpParameterForChi() throws Exception {
        when(request.getParameter("fakeClientIpAddress")).thenReturn("192.168.123.123");

        HTTPAccessRecord httpAccessRecord = new HTTPAccessRecord.Builder(new Date(144140678000L), request).build();
        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);

        assertThat(httpAccessEvent, equalTo("144140678.000 qtype=HTTP chi=192.168.123.123 rhi=192.168.7.6 url=\"http://example.com/index.html?foo=bar\" cqhm=GET cqhv=HTTP/1.1 rtype=- rloc=\"-\" rdtl=- rerr=\"-\" rgb=\"-\" rurl=\"-\" rurls=\"-\" uas=\"null\" svc=\"-\" rh=\"-\""));
    }

    @Test
    public void itUsesXMmClientIpHeaderOverFakeIpParameterForChi() throws Exception {
        when(request.getParameter("fakeClientIpAddress")).thenReturn("192.168.123.123");
        when(request.getHeader(HTTPRequest.X_MM_CLIENT_IP)).thenReturn("192.168.100.100");

        HTTPAccessRecord httpAccessRecord = new HTTPAccessRecord.Builder(new Date(144140678000L), request).build();
        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);

        assertThat(httpAccessEvent, equalTo("144140678.000 qtype=HTTP chi=192.168.100.100 rhi=192.168.7.6 url=\"http://example.com/index.html?foo=bar\" cqhm=GET cqhv=HTTP/1.1 rtype=- rloc=\"-\" rdtl=- rerr=\"-\" rgb=\"-\" rurl=\"-\" rurls=\"-\" uas=\"null\" svc=\"-\" rh=\"-\""));
    }

    @Test
    public void itUsesUserAgentHeaderString() throws Exception {
        when(request.getHeader("User-Agent")).thenReturn("Mozilla/5.0 Gecko/20100101 Firefox/68.0");

        HTTPAccessRecord httpAccessRecord = new HTTPAccessRecord.Builder(new Date(144140678000L), request).build();
        String httpAccessEvent = HTTPAccessEventBuilder.create(httpAccessRecord);

        assertThat(httpAccessEvent, containsString("uas=\"Mozilla/5.0 Gecko/20100101 Firefox/68.0\""));
    }
}
