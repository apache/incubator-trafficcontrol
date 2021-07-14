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

import javax.servlet.http.HttpServletRequest;
import java.math.RoundingMode;
import java.text.DecimalFormat;
import java.util.Map;

public class HTTPAccessEventBuilder {
    private static String formatRequest(final HttpServletRequest request) {
        String url = formatObject(request.getRequestURL());

        if ("-".equals(url)) {
            return url;
        }

        if (request.getQueryString() != null && !request.getQueryString().isEmpty()) {
            final String queryString = "?" + request.getQueryString();
            final StringBuilder stringBuilder = new StringBuilder(url);
            stringBuilder.append(queryString);
            url = stringBuilder.toString();
        }

        return url;
    }

    private static String formatObject(final Object o) {
        return (o == null) ? "-" : o.toString();
    }

    private static String formatRequestHeaders(final Map<String, String> requestHeaders) {
        if (requestHeaders == null || requestHeaders.isEmpty()) {
            return "rh=\"-\"";
        }

        final StringBuilder stringBuilder = new StringBuilder();
        boolean first = true;
        for (final Map.Entry<String, String> entry : requestHeaders.entrySet()) {
            if (entry.getValue() == null || entry.getValue().isEmpty()) {
                continue;
            }

            if (!first) {
                stringBuilder.append(' ');
            }
            else {
                first = false;
            }

            stringBuilder.append("rh=\"");
            stringBuilder.append(entry.getKey()).append(": ");
            stringBuilder.append(entry.getValue().replaceAll("\"", "'"));
            stringBuilder.append('"');
        }

        return stringBuilder.toString();
    }

    @SuppressWarnings({"PMD.UseStringBufferForStringAppends", "PMD.NPathComplexity"})
    public static String create(final HTTPAccessRecord httpAccessRecord) {
        final long start = httpAccessRecord.getRequestDate().getTime();
        final String timeString = String.format("%d.%03d", start / 1000, start % 1000);

        final HttpServletRequest httpServletRequest = httpAccessRecord.getRequest();

        String chi = formatObject(httpServletRequest.getRemoteAddr());
        final String url = formatRequest(httpServletRequest);
        final String cqhm = formatObject(httpServletRequest.getMethod());
        final String cqhv = formatObject(httpServletRequest.getProtocol());

        final String resultType = formatObject(httpAccessRecord.getResultType());
        final String rerr = formatObject(httpAccessRecord.getRerr());

        String resultDetails = "-";
        if (!"-".equals(resultType)) {
            resultDetails = formatObject(httpAccessRecord.getResultDetails());
        }

        String rloc = "-";
        final Geolocation resultLocation = httpAccessRecord.getResultLocation();

        if (resultLocation != null) {
            final DecimalFormat decimalFormat = new DecimalFormat("0.00");
            decimalFormat.setRoundingMode(RoundingMode.DOWN);
            rloc = decimalFormat.format(resultLocation.getLatitude()) + "," + decimalFormat.format(resultLocation.getLongitude());
        }

        final String xMmClientIpHeader = httpServletRequest.getHeader(HTTPRequest.X_MM_CLIENT_IP);
        final String fakeIpParameter = httpServletRequest.getParameter(HTTPRequest.FAKE_IP);

        final String remoteIp = chi;
        if (xMmClientIpHeader != null) {
            chi = xMmClientIpHeader;
        } else if (fakeIpParameter != null) {
            chi = fakeIpParameter;
        }

        final String rgb = formatObject(httpAccessRecord.getRegionalGeoResult());

        final StringBuilder stringBuilder = new StringBuilder(timeString)
            .append(" qtype=HTTP chi=")
            .append(chi)
            .append(" rhi=");

        if (!remoteIp.equals(chi)) {
            stringBuilder.append(remoteIp);
        } else {
            stringBuilder.append('-');
        }

        stringBuilder.append(" url=\"").append(url)
            .append("\" cqhm=").append(cqhm)
            .append(" cqhv=").append(cqhv)
            .append(" rtype=").append(resultType)
            .append(" rloc=\"").append(rloc)
            .append("\" rdtl=").append(resultDetails)
            .append(" rerr=\"").append(rerr)
            .append("\" rgb=\"").append(rgb).append('"');

        if (httpAccessRecord.getResponseCode() != -1) {
            final String pssc = formatObject(httpAccessRecord.getResponseCode());
            final double ttms = (System.nanoTime() - httpAccessRecord.getRequestNanoTime()) / 1000000.0;
            stringBuilder.append(" pssc=").append(pssc).append(" ttms=").append(String.format("%.03f",ttms));
        }

        final String respurl = " rurl=\"" + formatObject(httpAccessRecord.getResponseURL()) + "\"";
        stringBuilder.append(respurl);

        final String respurls = " rurls=\"" + formatObject(httpAccessRecord.getResponseURLs()) + "\"";
        stringBuilder.append(respurls);

        final String userAgent = httpServletRequest.getHeader("User-Agent") + "\"";
        stringBuilder.append(" uas=\"").append(userAgent);

        final String fmtDSID = formatObject(httpAccessRecord.getDeliveryServiceXmlIds());
        final String deliveryServiceId = fmtDSID == null ? "-" : fmtDSID;
        stringBuilder.append(" svc=\"").append(deliveryServiceId).append("\" ");

        stringBuilder.append(formatRequestHeaders(httpAccessRecord.getRequestHeaders()));
        return stringBuilder.toString();
    }
}
