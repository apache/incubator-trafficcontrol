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
import org.xbill.DNS.DClass;
import org.xbill.DNS.Message;
import org.xbill.DNS.Rcode;
import org.xbill.DNS.Record;
import org.xbill.DNS.Section;
import org.xbill.DNS.Type;
import org.xbill.DNS.WireParseException;

import java.math.RoundingMode;
import java.text.DecimalFormat;

public class DNSAccessEventBuilder {

    @SuppressWarnings("PMD.UseStringBufferForStringAppends")
    public static String create(final DNSAccessRecord dnsAccessRecord) {
        final String event = createEvent(dnsAccessRecord);
        String rType = "-";
        String rdtl = "-";
        String rloc = "-";

        if (dnsAccessRecord.getResultType() != null) {
            rType = dnsAccessRecord.getResultType().toString();
            if (dnsAccessRecord.getResultDetails() != null) {
                rdtl = dnsAccessRecord.getResultDetails().toString();
            }
        }

        if (dnsAccessRecord.getResultLocation() != null) {
            final Geolocation resultLocation = dnsAccessRecord.getResultLocation();

            final DecimalFormat decimalFormat = new DecimalFormat(".##");
            decimalFormat.setRoundingMode(RoundingMode.DOWN);
            rloc = decimalFormat.format(resultLocation.getLatitude()) + "," + decimalFormat.format(resultLocation.getLongitude());
        }

        final String routingInfo = "rtype=" + rType + " rloc=\"" + rloc +  "\" rdtl=" + rdtl + " rerr=\"-\"";
        String answer = "ans=\"-\"";
        final String dsString = dnsAccessRecord.getDeliveryServiceXmlIds() == null ? "-" : dnsAccessRecord.getDeliveryServiceXmlIds();
        final String dsInfo = "svc=\"" + dsString + "\"";

        if (dnsAccessRecord.getDnsMessage() != null) {
            answer = createTTLandAnswer(dnsAccessRecord.getDnsMessage());
        }
        return event + " " + routingInfo + " " + answer + " " + dsInfo;
    }

    private static String createEvent(final DNSAccessRecord dnsAccessRecord) {
        final String timeString = String.format("%d.%03d", dnsAccessRecord.getQueryInstant() / 1000, dnsAccessRecord.getQueryInstant() % 1000);

        final double ttms = (System.nanoTime() - dnsAccessRecord.getRequestNanoTime()) / 1000000.0;

        final String clientAddressString = dnsAccessRecord.getClient().getHostAddress();
        final String resolverAddressString = dnsAccessRecord.getResolver().getHostAddress();

        final StringBuilder stringBuilder = new StringBuilder(timeString).append(" qtype=DNS chi=").append(clientAddressString).append(" rhi=");

        if (!clientAddressString.equals(resolverAddressString)) {
            stringBuilder.append(resolverAddressString);
        } else {
            stringBuilder.append('-');
        }

        stringBuilder.append(" ttms=").append(String.format("%.03f", ttms));

        if (dnsAccessRecord.getDnsMessage() == null) {
            return stringBuilder.append(" xn=- fqdn=- type=- class=- rcode=-").toString();
        }

        final String messageHeader = createDnsMessageHeader(dnsAccessRecord.getDnsMessage());
        return stringBuilder.append(messageHeader).toString();
    }

    private static String createDnsMessageHeader(final Message dnsMessage) {
        final String queryHeader = " xn=" + dnsMessage.getHeader().getID();
        final String query = " " + createQuery(dnsMessage.getQuestion());
        final String rcode = " rcode=" + Rcode.string(dnsMessage.getHeader().getRcode());
        return new StringBuilder(queryHeader).append(query).append(rcode).toString();
    }

    private static String createTTLandAnswer(final Message dnsMessage) {
        if (dnsMessage.getSectionArray(Section.ANSWER) == null || dnsMessage.getSectionArray(Section.ANSWER).length == 0) {
            return "ttl=\"-\" ans=\"-\"";
        }

        final StringBuilder answerStringBuilder = new StringBuilder();
        final StringBuilder ttlStringBuilder = new StringBuilder();
        for (final Record record : dnsMessage.getSectionArray(Section.ANSWER)) {
            final String s = record.rdataToString() + " ";
            final String ttl = record.getTTL() + " ";
            answerStringBuilder.append(s);
            ttlStringBuilder.append(ttl);
        }

        return "ttl=\"" + ttlStringBuilder.toString().trim() + "\" ans=\"" + answerStringBuilder.toString().trim() + "\"";
    }

    public static String create(final DNSAccessRecord dnsAccessRecord, final WireParseException wireParseException) {
        final String event = createEvent(dnsAccessRecord);
        final String rerr = "Bad Request:" + wireParseException.getClass().getSimpleName() + ":" + wireParseException.getMessage();
        final String dsID = dnsAccessRecord.getDeliveryServiceXmlIds() == null ? "-" : dnsAccessRecord.getDeliveryServiceXmlIds();
        return new StringBuilder(event)
                .append(" rtype=-")
                .append(" rloc=\"-\"")
                .append(" rdtl=-")
                .append(" rerr=\"")
                .append(rerr)
                .append("\"")
                .append(" ttl=\"-\"")
                .append(" ans=\"-\"")
                .append(" svc=\"")
                .append(dsID)
                .append("\"")
                .toString();
    }

    public static String create(final DNSAccessRecord dnsAccessRecord, final Exception exception) {
        final Message dnsMessage = dnsAccessRecord.getDnsMessage();
        dnsMessage.getHeader().setRcode(Rcode.SERVFAIL);
        final String event = createEvent(dnsAccessRecord);

        final String rerr = "Server Error:" + exception.getClass().getSimpleName() + ":" + exception.getMessage();
        final String dsID = dnsAccessRecord.getDeliveryServiceXmlIds() == null ? "-" : dnsAccessRecord.getDeliveryServiceXmlIds();

        return new StringBuilder(event)
                .append(" rtype=-")
                .append(" rloc=\"-\"")
                .append(" rdtl=-")
                .append(" rerr=\"")
                .append(rerr)
                .append("\"")
                .append(" ttl=\"-\"")
                .append(" ans=\"-\"")
                .append(" svc=\"")
                .append(dsID)
                .append("\"").toString();
    }

    private static String createQuery(final Record query) {
        if (query != null && query.getName() != null) {
            final String qname = query.getName().toString();
            final String qtype = Type.string(query.getType());
            final String qclass = DClass.string(query.getDClass());

            return new StringBuilder()
                    .append("fqdn=").append(qname)
                    .append(" type=").append(qtype)
                    .append(" class=").append(qclass)
                    .toString();
        }
        return "";
    }
}
