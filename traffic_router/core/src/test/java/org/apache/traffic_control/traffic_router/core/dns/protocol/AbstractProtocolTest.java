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

package org.apache.traffic_control.traffic_router.core.dns.protocol;

import org.apache.traffic_control.traffic_router.core.dns.DNSAccessEventBuilder;
import org.apache.traffic_control.traffic_router.core.dns.DNSAccessRecord;
import org.apache.traffic_control.traffic_router.core.dns.NameServer;
import org.apache.log4j.Logger;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.Mockito;
import org.mockito.invocation.InvocationOnMock;
import org.mockito.stubbing.Answer;
import org.powermock.core.classloader.annotations.PrepareForTest;
import org.powermock.modules.junit4.PowerMockRunner;
import org.xbill.DNS.*;

import java.net.Inet4Address;
import java.net.InetAddress;
import java.util.Random;

import static org.junit.Assert.fail;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.powermock.api.mockito.PowerMockito.*;


@RunWith(PowerMockRunner.class)
@PrepareForTest({AbstractProtocolTest.FakeAbstractProtocol.class, Logger.class, DNSAccessEventBuilder.class, Header.class, NameServer.class, DNSAccessRecord.class})
public class AbstractProtocolTest {
    private static Logger accessLogger = mock(Logger.class);
    private NameServer nameServer;
    private Header header;
    InetAddress client;

    @Before
    public void before() throws Exception {
        // force the xn field in the request
        Random random = mock(Random.class);
        Mockito.when(random.nextInt(0xffff)).thenReturn(65535);
        whenNew(Random.class).withNoArguments().thenReturn(random);

        mockStatic(System.class);
        Answer<Long> nanoTimeAnswer = new Answer<Long>() {
            final long[] nanoTimes = {100000000L, 100000000L + 345123000L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return nanoTimes[index++ % 2];
            }
        };
        when(System.nanoTime()).thenAnswer(nanoTimeAnswer);

        Answer<Long> currentTimeAnswer = new Answer<Long>() {
            final long[] currentTimes = {144140678000L, 144140678345L};
            int index = 0;
            public Long answer(InvocationOnMock invocation) {
                return currentTimes[index++ % 2];
            }
        };
        when(System.currentTimeMillis()).then(currentTimeAnswer);

        mockStatic(Logger.class);
        when(Logger.getLogger("org.apache.traffic_control.traffic_router.core.access")).thenAnswer(invocation -> accessLogger);

        header = new Header();
        header.setID(65535);
        header.setFlag(Flags.QR);

        client = Inet4Address.getByAddress(new byte[]{(byte) 192, (byte) 168, 23, 45});
        nameServer = mock(NameServer.class);
    }

    @Test
    public void itLogsARecordQueries() throws Exception {
        header.setRcode(Rcode.NOERROR);

        Name name = Name.fromString("www.example.com.");
        Record question = Record.newRecord(name, Type.A, DClass.IN, 0L);
        Message query = Message.newQuery(question);

        query.getHeader().getRcode();

        byte[] queryBytes = query.toWire();

        whenNew(Message.class).withArguments(queryBytes).thenReturn(query);

        InetAddress resolvedAddress = Inet4Address.getByName("192.168.8.9");

        Record answer = new ARecord(name, DClass.IN, 3600L, resolvedAddress);
        Record[] answers = new Record[] {answer};

        Message response = mock(Message.class);
        when(response.getHeader()).thenReturn(header);
        when(response.getSectionArray(Section.ANSWER)).thenReturn(answers);
        when(response.getQuestion()).thenReturn(question);

        InetAddress client = Inet4Address.getByName("192.168.23.45");
        when(nameServer.query(any(Message.class), any(InetAddress.class), any(DNSAccessRecord.Builder.class))).thenReturn(response);

        FakeAbstractProtocol abstractProtocol = new FakeAbstractProtocol(client, queryBytes);
        abstractProtocol.setNameServer(nameServer);

        abstractProtocol.run();

        verify(accessLogger).info("144140678.000 qtype=DNS chi=192.168.23.45 rhi=- ttms=345.123 xn=65535 fqdn=www.example.com. type=A class=IN rcode=NOERROR rtype=- rloc=\"-\" rdtl=- rerr=\"-\" ttl=\"3600\" ans=\"192.168.8.9\" svc=\"-\"");
    }

    @Test
    public void itLogsOtherQueries() throws Exception {
        header.setRcode(Rcode.REFUSED);

        Name name = Name.fromString("John Wayne.");
        Record question = Record.newRecord(name, 65530, 43210);

        Message query = Message.newQuery(question);

        Message response = mock(Message.class);
        when(response.getHeader()).thenReturn(header);
        when(response.getSectionArray(Section.ANSWER)).thenReturn(null);
        when(response.getQuestion()).thenReturn(question);

        when(nameServer.query(any(Message.class), any(InetAddress.class), any(DNSAccessRecord.Builder.class))).thenReturn(response);

        FakeAbstractProtocol abstractProtocol = new FakeAbstractProtocol(client, query.toWire());
        abstractProtocol.setNameServer(nameServer);
        abstractProtocol.run();

        verify(accessLogger).info("144140678.000 qtype=DNS chi=192.168.23.45 rhi=- ttms=345.123 xn=65535 fqdn=John\\032Wayne. type=TYPE65530 class=CLASS43210 rcode=REFUSED rtype=- rloc=\"-\" rdtl=- rerr=\"-\" ttl=\"-\" ans=\"-\" svc=\"-\"");
    }

    @Test
    public void itLogsBadClientRequests() throws Exception {
        FakeAbstractProtocol abstractProtocol = new FakeAbstractProtocol(client, new byte[] {1,2,3,4,5,6,7});
        abstractProtocol.setNameServer(nameServer);
        abstractProtocol.run();
        verify(accessLogger).info("144140678.000 qtype=DNS chi=192.168.23.45 rhi=- ttms=345.123 xn=- fqdn=- type=- class=- rcode=- rtype=- rloc=\"-\" rdtl=- rerr=\"Bad Request:WireParseException:end of input\" ttl=\"-\" ans=\"-\" svc=\"-\"");
    }

    @Test
    public void itLogsServerErrors() throws Exception {
        header.setRcode(Rcode.REFUSED);

        Name name = Name.fromString("John Wayne.");
        Record question = Record.newRecord(name, 65530, 43210);

        Message query = Message.newQuery(question);

        Message response = mock(Message.class);
        when(response.getHeader()).thenReturn(header);
        when(response.getSectionArray(Section.ANSWER)).thenReturn(null);
        when(response.getQuestion()).thenReturn(question);

        when(nameServer.query(any(Message.class), any(InetAddress.class), any(DNSAccessRecord.Builder.class))).thenThrow(new RuntimeException("Aw snap!"));

        FakeAbstractProtocol abstractProtocol = new FakeAbstractProtocol(client, query.toWire());
        abstractProtocol.setNameServer(nameServer);
        abstractProtocol.run();

        verify(accessLogger).info("144140678.000 qtype=DNS chi=192.168.23.45 rhi=- ttms=345.123 xn=65535 fqdn=John\\032Wayne. type=TYPE65530 class=CLASS43210 rcode=SERVFAIL rtype=- rloc=\"-\" rdtl=- rerr=\"Server Error:RuntimeException:Aw snap!\" ttl=\"-\" ans=\"-\" svc=\"-\"");

    }

    public class FakeAbstractProtocol extends AbstractProtocol {

        private final InetAddress inetAddress;
        private final byte[] request;

        public FakeAbstractProtocol(InetAddress inetAddress, byte[] request) {
            this.inetAddress = inetAddress;
            this.request = request;
        }

        @Override
        protected int getMaxResponseLength(Message request) {
            return Integer.MAX_VALUE;
        }

        @Override
        public void run() {
            try {
                query(inetAddress, request);
            } catch (WireParseException e) {
                // Ignore it
            }
        }

    }
}