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

import org.apache.traffic_control.traffic_router.geolocation.Geolocation;
import org.apache.traffic_control.traffic_router.core.loc.RegionalGeoResult;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track.ResultDetails;
import org.apache.traffic_control.traffic_router.core.router.StatTracker.Track.ResultType;

import javax.servlet.http.HttpServletRequest;
import java.net.URL;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

// Using Josh Bloch Builder pattern so suppress these warnings.
@SuppressWarnings({"PMD.MissingStaticMethodInNonInstantiatableClass",
        "PMD.AccessorClassGeneration",
        "PMD.NPathComplexity",
        "PMD.IfStmtsMustUseBraces",
        "PMD.CyclomaticComplexity"})
public class HTTPAccessRecord {
    private final Date requestDate;
    private final long requestNanoTime;
    private final HttpServletRequest request;
    private final URL responseURL;
    private final List<URL> responseURLs;
    private final int responseCode;
    private final ResultType resultType;
    private final String rerr;
    private final ResultDetails resultDetails;
    private final Geolocation resultLocation;
    private final Map<String, String> requestHeaders;
    private final RegionalGeoResult regionalGeoResult;
    private final String deliveryServiceXmlIds;

    public Date getRequestDate() {
        return requestDate;
    }

    public HttpServletRequest getRequest() {
        return request;
    }

    public int getResponseCode() {
        return responseCode;
    }

    public URL getResponseURL() {
        return responseURL;
    }

    public List<URL> getResponseURLs() {
        return responseURLs;
    }

    public ResultType getResultType() {
        return resultType;
    }

    public String getRerr() {
        return rerr;
    }

    public ResultDetails getResultDetails() {
        return resultDetails;
    }

    public Geolocation getResultLocation() {
        return resultLocation;
    }

    public Map<String, String> getRequestHeaders() {
        return requestHeaders;
    }

    public RegionalGeoResult getRegionalGeoResult() {
        return regionalGeoResult;
    }

    public long getRequestNanoTime() {
        return requestNanoTime;
    }

    public String getDeliveryServiceXmlIds() {
        return deliveryServiceXmlIds;
    }

    public static class Builder {
        private final Date requestDate;
        private final HttpServletRequest request;
        private int responseCode = -1;
        private URL responseURL;
        private List<URL> responseURLs;
        private ResultType resultType;
        private String rerr;
        private ResultDetails resultDetails;
        private Geolocation resultLocation;
        private Map<String, String> requestHeaders = new HashMap<String, String>();
        private RegionalGeoResult regionalGeoResult;
        private final long requestNanoTime;
        private String deliveryServiceXmlIds;

        public Builder(final Date requestDate, final HttpServletRequest request) {
            this.requestDate = requestDate;
            this.request = request;
            this.requestNanoTime = System.nanoTime();
        }

        public Builder(final long requestNanoTime, final HttpServletRequest request) {
            this.requestDate =new Date();
            this.request = request;
            this.requestNanoTime = requestNanoTime;
        }

        public Builder(final HTTPAccessRecord httpAccessRecord) {
            this.requestDate = httpAccessRecord.requestDate;
            this.request = httpAccessRecord.request;
            this.responseURL = httpAccessRecord.responseURL;
            this.responseURLs = httpAccessRecord.responseURLs;
            this.responseCode = httpAccessRecord.responseCode;
            this.requestNanoTime = httpAccessRecord.requestNanoTime;
        }

        public Builder responseCode(final int responseCode) {
            this.responseCode = responseCode;
            return this;
        }

        public Builder responseURL(final URL responseURL) {
            this.responseURL = responseURL;
            return this;
        }

        public Builder responseURLs(final List<URL> responseURLs) {
            this.responseURLs = responseURLs;
            return this;
        }

        public Builder resultType(final ResultType resultType) {
            this.resultType = resultType;
            return this;
        }

        public Builder rerr(final String rerr) {
            this.rerr = rerr;
            return this;
        }

        public Builder resultDetails(final ResultDetails resultDetails) {
            this.resultDetails = resultDetails;
            return this;
        }

        public Builder resultLocation(final Geolocation resultLocation) {
            this.resultLocation = resultLocation;
            return this;
        }

        public Builder requestHeaders(final Map<String,String> requestHeaders) {
            this.requestHeaders = requestHeaders;
            return this;
        }

        public Builder regionalGeoResult(final RegionalGeoResult regionalGeoResult) {
            this.regionalGeoResult = regionalGeoResult;
            return this;
        }

        public HTTPAccessRecord build() {
            return new HTTPAccessRecord(this);
        }

        public Builder deliveryServiceIds(final String deliveryServiceIds) {
            this.deliveryServiceXmlIds = deliveryServiceIds;
            return this;
        }
    }

    private HTTPAccessRecord(final Builder builder) {
        requestDate = builder.requestDate;
        request = builder.request;
        responseCode = builder.responseCode;
        responseURL = builder.responseURL;
        responseURLs = builder.responseURLs;
        resultType = builder.resultType;
        rerr = builder.rerr;
        resultDetails = builder.resultDetails;
        resultLocation = builder.resultLocation;
        requestHeaders = builder.requestHeaders;
        regionalGeoResult = builder.regionalGeoResult;
        requestNanoTime = builder.requestNanoTime;
        deliveryServiceXmlIds = builder.deliveryServiceXmlIds;
    }

    @Override
    public boolean equals(final Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;

        final HTTPAccessRecord that = (HTTPAccessRecord) o;

        if (requestNanoTime != that.requestNanoTime) return false;
        if (responseCode != that.responseCode) return false;
        if (requestDate != null ? !requestDate.equals(that.requestDate) : that.requestDate != null) return false;
        if (request != null ? !request.equals(that.request) : that.request != null) return false;
        if (responseURL != null ? !responseURL.equals(that.responseURL) : that.responseURL != null) return false;
        if (responseURLs != null ? !responseURLs.equals(that.responseURLs) : that.responseURLs != null) return false;
        if (resultType != that.resultType) return false;
        if (rerr != null ? !rerr.equals(that.rerr) : that.rerr != null) return false;
        if (resultDetails != that.resultDetails) return false;
        if (resultLocation != null ? !resultLocation.equals(that.resultLocation) : that.resultLocation != null)
            return false;
        if (requestHeaders != null ? !requestHeaders.equals(that.requestHeaders) : that.requestHeaders != null)
            return false;
        return regionalGeoResult != null ? regionalGeoResult.equals(that.regionalGeoResult) : that.regionalGeoResult == null;

    }

    @Override
    public int hashCode() {
        int result = requestDate != null ? requestDate.hashCode() : 0;
        result = 31 * result + (int) (requestNanoTime ^ (requestNanoTime >>> 32));
        result = 31 * result + (request != null ? request.hashCode() : 0);
        result = 31 * result + (responseURL != null ? responseURL.hashCode() : 0);
        result = 31 * result + (responseURLs != null ? responseURLs.hashCode() : 0);
        result = 31 * result + responseCode;
        result = 31 * result + (resultType != null ? resultType.hashCode() : 0);
        result = 31 * result + (rerr != null ? rerr.hashCode() : 0);
        result = 31 * result + (resultDetails != null ? resultDetails.hashCode() : 0);
        result = 31 * result + (resultLocation != null ? resultLocation.hashCode() : 0);
        result = 31 * result + (requestHeaders != null ? requestHeaders.hashCode() : 0);
        result = 31 * result + (regionalGeoResult != null ? regionalGeoResult.hashCode() : 0);
        return result;
    }

    @Override
    public String toString() {
        return "HTTPAccessRecord{" +
                "requestDate=" + requestDate +
                ", request=" + request +
                ", responseURL=" + responseURL +
                ", responseURLs=" + responseURLs +
                ", responseCode=" + responseCode +
                ", resultType=" + resultType +
                ", rerr='" + rerr + '\'' +
                ", resultDetails=" + resultDetails +
                ", rgb=" + regionalGeoResult +
                ", requestNanoTime=" + requestNanoTime +
                '}';
    }
}

