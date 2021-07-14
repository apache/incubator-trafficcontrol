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

package org.apache.traffic_control.traffic_router.core.dns;

import org.apache.traffic_control.traffic_router.geolocation.Geolocation;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track.ResultType;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track.ResultDetails;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.invocation.InvocationOnMock;
import org.mockito.stubbing.Answer;
import org.powermock.core.classloader.annotations.PrepareForTest;
import org.powermock.modules.junit4.PowerMockRunner;
import org.xbill.DNS.*;

import java.net.Inet4Address;
import java.net.InetAddress;
import java.util.Random;

import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.equalTo;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;
import static org.powermock.api.mockito.PowerMockito.mockStatic;
import static org.powermock.api.mockito.PowerMockito.spy;
import static org.powermock.api.mockito.PowerMockito.whenNew;

@RunWith(PowerMockRunner.class)
@PrepareForTest({Random.class, Header.class, DNSAccessEventBuilder.class, System.class, DNSAccessRecord.class})
public class DNSAccessEventBuilderTest {

    private InetAddress client;
    private InetAddress resolver;

    @Before
    public void before() throws Exception {
        mockStatic(System.class);

        Random random = mock(Random.class);
        when(random.nextInt(0xffff)).thenReturn(65535);
        whenNew(Random.class).withNoArguments().thenReturn(random);

        client = mock(InetAddress.class);
        when(client.getHostAddress()).thenReturn("192.168.10.11");

        resolver = mock(InetAddress.class);
        when(resolver.getHostAddress()).thenReturn("10.0.0.211");
    }

    @Test
    public void itCreatesRequestErrorData() throws Exception {
        Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {100000000L,889000000L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 2];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);
        when(System.currentTimeMillis()).thenAnswer(invocation -> 144140678789L);

        DNSAccessRecord dnsAccessRecord = new DNSAccessRecord.Builder(144140678000L, client).build();

        String dnsAccessEvent = DNSAccessEventBuilder.create(dnsAccessRecord, new WireParseException("invalid record length"));
        assertThat(dnsAccessEvent, equalTo("144140678.000 qtype=DNS chi=192.168.10.11 rhi=- ttms=789.000 xn=- fqdn=- type=- class=- rcode=-" +
                " rtype=- rloc=\"-\" rdtl=- rerr=\"Bad Request:WireParseException:invalid record length\" ttl=\"-\" ans=\"-\" svc=\"-\""));
    }

    @Test
    public void itAddsResponseData() throws Exception {
        final Name name = Name.fromString("www.example.com.");

		Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {100000000L, 100000000L + 789123000L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 2];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);

		Answer<Long> currentTimeAnswer = new Answer<Long>() {
            final long[] currentTimes = {144140678789L, 144140678000L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return currentTimes[index++ % 2];
            }
        };
        when(System.currentTimeMillis()).then(currentTimeAnswer);

        final Record question = Record.newRecord(name, Type.A, DClass.IN, 12345L);

        final Message response = spy(Message.newQuery(question));
        response.getHeader().setRcode(Rcode.NOERROR);

        final Record record1 = mock(Record.class);
        when(record1.rdataToString()).thenReturn("foo");
        when(record1.getTTL()).thenReturn(1L);
        final Record record2 = mock(Record.class);
        when(record2.rdataToString()).thenReturn("bar");
        when(record2.getTTL()).thenReturn(2L);
        final Record record3 = mock(Record.class);
        when(record3.rdataToString()).thenReturn("baz");
        when(record3.getTTL()).thenReturn(3L);

        Record[] records = new Record[] {record1, record2, record3};
        when(response.getSectionArray(Section.ANSWER)).thenReturn(records);

        InetAddress answerAddress = Inet4Address.getByName("192.168.1.23");

        ARecord addressRecord = new ARecord(name, DClass.IN, 54321L, answerAddress);
        response.addRecord(addressRecord, Section.ANSWER);

        DNSAccessRecord dnsAccessRecord = new DNSAccessRecord.Builder(144140678000L, client).dnsMessage(response).build();
        String dnsAccessEvent = DNSAccessEventBuilder.create(dnsAccessRecord);

        assertThat(dnsAccessEvent, equalTo("144140678.000 qtype=DNS chi=192.168.10.11 rhi=- ttms=789.123" +
                " xn=65535 fqdn=www.example.com. type=A class=IN" +
                " rcode=NOERROR rtype=- rloc=\"-\" rdtl=- rerr=\"-\" ttl=\"1 2 3\" ans=\"foo bar baz\" svc=\"-\""));


        when(System.nanoTime()).thenAnswer(invocation -> 100000000L + 456000L);
        dnsAccessEvent = DNSAccessEventBuilder.create(dnsAccessRecord);

        assertThat(dnsAccessEvent, equalTo("144140678.000 qtype=DNS chi=192.168.10.11 rhi=- ttms=0.456" +
                " xn=65535 fqdn=www.example.com. type=A class=IN" +
                " rcode=NOERROR rtype=- rloc=\"-\" rdtl=- rerr=\"-\" ttl=\"1 2 3\" ans=\"foo bar baz\" svc=\"-\""));
    }

    @Test
    public void itCreatesServerErrorData() throws Exception {
        Message query = Message.newQuery(Record.newRecord(Name.fromString("www.example.com."), Type.A, DClass.IN, 12345L));
        Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {100000000L, 100000000L + 789876321L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 2];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);
        when(System.currentTimeMillis()).thenAnswer(invocation -> 144140678789L);

        DNSAccessRecord dnsAccessRecord = new DNSAccessRecord.Builder(144140678000L, client).dnsMessage(query).build();
        String dnsAccessEvent = DNSAccessEventBuilder.create(dnsAccessRecord, new RuntimeException("boom it failed"));
        assertThat(dnsAccessEvent, equalTo("144140678.000 qtype=DNS chi=192.168.10.11 rhi=- ttms=789.876" +
                " xn=65535 fqdn=www.example.com. type=A class=IN" +
                " rcode=SERVFAIL rtype=- rloc=\"-\" rdtl=- rerr=\"Server Error:RuntimeException:boom it failed\" ttl=\"-\" ans=\"-\" svc=\"-\""));
    }

    @Test
    public void itAddsResultTypeData() throws Exception {
        final Name name = Name.fromString("www.example.com.");

        Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {100000000L, 100000000L + 789000321L, 100000000L + 123123L, 100000000L + 246001L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 4];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);

        Answer<Long> currentTimeAnswer = new Answer<Long>() {
            final long[] currentTimes = {144140678789L, 144140678000L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return currentTimes[index++ % 2];
            }
        };
        when(System.currentTimeMillis()).then(currentTimeAnswer);

        final Record question = Record.newRecord(name, Type.A, DClass.IN, 12345L);
        final Message response = spy(Message.newQuery(question));
        response.getHeader().setRcode(Rcode.NOERROR);

        final Record record1 = mock(Record.class);
        when(record1.rdataToString()).thenReturn("foo");
        when(record1.getTTL()).thenReturn(1L);
        final Record record2 = mock(Record.class);
        when(record2.rdataToString()).thenReturn("bar");
        when(record2.getTTL()).thenReturn(2L);
        final Record record3 = mock(Record.class);
        when(record3.rdataToString()).thenReturn("baz");
        when(record3.getTTL()).thenReturn(3L);

        Record[] records = new Record[] {record1, record2, record3};
        when(response.getSectionArray(Section.ANSWER)).thenReturn(records);

        InetAddress answerAddress = Inet4Address.getByName("192.168.1.23");

        ARecord addressRecord = new ARecord(name, DClass.IN, 54321L, answerAddress);
        response.addRecord(addressRecord, Section.ANSWER);

        Geolocation resultLocation = new Geolocation(39.7528,-104.9997);
        ResultType resultType = ResultType.CZ;
        final DNSAccessRecord.Builder builder = new DNSAccessRecord.Builder(144140678000L, client)
            .dnsMessage(response).resultType(resultType).resultLocation(resultLocation);

        DNSAccessRecord dnsAccessRecord = builder.build();
        String dnsAccessEvent = DNSAccessEventBuilder.create(dnsAccessRecord);

        assertThat(dnsAccessEvent, equalTo("144140678.000 qtype=DNS chi=192.168.10.11 rhi=- ttms=789.000" +
                " xn=65535 fqdn=www.example.com. type=A class=IN" +
                " rcode=NOERROR rtype=CZ rloc=\"39.75,-104.99\" rdtl=- rerr=\"-\" ttl=\"1 2 3\" ans=\"foo bar baz\" svc=\"-\""));

        dnsAccessRecord = builder.resultType(ResultType.GEO).build();
        dnsAccessEvent = DNSAccessEventBuilder.create(dnsAccessRecord);

        assertThat(dnsAccessEvent, equalTo("144140678.000 qtype=DNS chi=192.168.10.11 rhi=- ttms=0.123" +
                " xn=65535 fqdn=www.example.com. type=A class=IN" +
                " rcode=NOERROR rtype=GEO rloc=\"39.75,-104.99\" rdtl=- rerr=\"-\" ttl=\"1 2 3\" ans=\"foo bar baz\" svc=\"-\""));

        dnsAccessRecord = builder.resultType(ResultType.MISS).resultDetails(ResultDetails.DS_NOT_FOUND).build();
        dnsAccessEvent = DNSAccessEventBuilder.create(dnsAccessRecord);

        assertThat(dnsAccessEvent, equalTo("144140678.000 qtype=DNS chi=192.168.10.11 rhi=- ttms=0.246" +
                " xn=65535 fqdn=www.example.com. type=A class=IN" +
                " rcode=NOERROR rtype=MISS rloc=\"39.75,-104.99\" rdtl=DS_NOT_FOUND rerr=\"-\" ttl=\"1 2 3\" ans=\"foo bar baz\" svc=\"-\""));
    }

    @Test
    public void itLogsResolverAndClient() throws Exception {
        final Name name = Name.fromString("www.example.com.");

        Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {100000000L, 100000000L + 789000321L, 100000000L + 123123L, 100000000L + 246001L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 4];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);

        Answer<Long> currentTimeAnswer = new Answer<Long>() {
            final long[] currentTimes = {144140678789L, 144140678000L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return currentTimes[index++ % 2];
            }
        };
        when(System.currentTimeMillis()).then(currentTimeAnswer);

        final Record question = Record.newRecord(name, Type.A, DClass.IN, 12345L);
        final Message response = spy(Message.newQuery(question));
        response.getHeader().setRcode(Rcode.NOERROR);

        final Record record1 = mock(Record.class);
        when(record1.rdataToString()).thenReturn("foo");
        when(record1.getTTL()).thenReturn(1L);

        Record[] records = new Record[] {record1};
        when(response.getSectionArray(Section.ANSWER)).thenReturn(records);

        InetAddress answerAddress = Inet4Address.getByName("192.168.1.23");

        ARecord addressRecord = new ARecord(name, DClass.IN, 54321L, answerAddress);
        response.addRecord(addressRecord, Section.ANSWER);

        Geolocation resultLocation = new Geolocation(39.7528,-104.9997);
        ResultType resultType = ResultType.CZ;
        final DNSAccessRecord.Builder builder = new DNSAccessRecord.Builder(144140678000L, resolver)
                .dnsMessage(response).resultType(resultType).resultLocation(resultLocation).client(client).deliveryServiceXmlIds("test");

        DNSAccessRecord dnsAccessRecord = builder.build();
        String dnsAccessEvent = DNSAccessEventBuilder.create(dnsAccessRecord);

        assertThat(dnsAccessEvent, equalTo("144140678.000 qtype=DNS chi=192.168.10.11 rhi=10.0.0.211 ttms=789.000" +
                " xn=65535 fqdn=www.example.com. type=A class=IN" +
                " rcode=NOERROR rtype=CZ rloc=\"39.75,-104.99\" rdtl=- rerr=\"-\" ttl=\"1\" ans=\"foo\" svc=\"test\""));
    }
}