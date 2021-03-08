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

package com.comcast.cdn.traffic_control.traffic_router.core.util;

import com.comcast.cdn.traffic_control.traffic_router.core.TestBase;
import com.comcast.cdn.traffic_control.traffic_router.core.ds.SteeringRegistry;
import com.comcast.cdn.traffic_control.traffic_router.core.ds.SteeringWatcher;
import com.comcast.cdn.traffic_control.traffic_router.core.edge.CacheRegister;
import com.comcast.cdn.traffic_control.traffic_router.core.loc.FederationsWatcher;
import com.comcast.cdn.traffic_control.traffic_router.core.router.TrafficRouter;
import com.comcast.cdn.traffic_control.traffic_router.core.router.TrafficRouterManager;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.node.ObjectNode;
import org.apache.log4j.Logger;
import org.junit.After;
import org.junit.Before;
import org.junit.BeforeClass;
import org.junit.Test;
import org.junit.experimental.categories.Category;
import org.springframework.context.ApplicationContext;

import java.nio.file.Files;
import java.nio.file.Paths;

import static com.comcast.cdn.traffic_control.traffic_router.core.ds.SteeringWatcher.DEFAULT_STEERING_DATA_URL;
import static com.comcast.cdn.traffic_control.traffic_router.core.loc.FederationsWatcher.DEFAULT_FEDERATION_DATA_URL;
import static org.hamcrest.MatcherAssert.assertThat;
import static org.hamcrest.Matchers.endsWith;
import static org.hamcrest.Matchers.equalTo;
import static org.hamcrest.Matchers.greaterThan;
import static org.hamcrest.Matchers.not;


@Category(IntegrationTest.class)
public class AbstractResourceWatcherTest {

    private static final Logger LOGGER = Logger.getLogger(com.comcast.cdn.traffic_control.traffic_router.core.util.AbstractResourceWatcherTest.class);

    private FederationsWatcher federationsWatcher;
    private SteeringWatcher steeringWatcher;
    private TrafficRouterManager trafficRouterManager;
    private SteeringRegistry steeringRegistry;
    private static ApplicationContext context;
    private String oldFedUrl;

    @BeforeClass
    public static void setUpBeforeClass() {
        assertThat("Copy core/src/main/conf/traffic_monitor.properties to core/src/test/conf and set 'traffic_monitor.bootstrap.hosts' to a real traffic monitor", Files.exists(Paths.get(TestBase.monitorPropertiesPath)), equalTo(true));
        context = TestBase.getContext();
    }

    @Before
    public void setUp() throws InterruptedException {
        federationsWatcher = (FederationsWatcher) context.getBean("federationsWatcher");
        steeringWatcher = (SteeringWatcher) context.getBean("steeringWatcher");
        steeringRegistry = (SteeringRegistry) context.getBean("steeringRegistry");
        trafficRouterManager = (TrafficRouterManager) context.getBean("trafficRouterManager");
        trafficRouterManager.getTrafficRouter().setApplicationContext(context);

        TrafficRouter trafficRouter = trafficRouterManager.getTrafficRouter();
        CacheRegister cacheRegister = trafficRouter.getCacheRegister();
        JsonNode config = cacheRegister.getConfig();

        if(config.get(federationsWatcher.getWatcherConfigPrefix() + ".polling.url") != null) {
            oldFedUrl = config.get(federationsWatcher.getWatcherConfigPrefix() + ".polling.url").asText();
            config = ((ObjectNode) config).remove(federationsWatcher.getWatcherConfigPrefix() + ".polling.url");
            federationsWatcher.trafficOpsUtils.setConfig(config);
            federationsWatcher.configure(config);
        }

        while (!federationsWatcher.isLoaded()) {
            LOGGER.info("Waiting for a valid federations database before proceeding");
            Thread.sleep(1000);
        }

        while (!steeringWatcher.isLoaded()) {
            LOGGER.info("Waiting for a valid steering database before proceeding");
            Thread.sleep(1000);
        }

    }

    @After
    public void tearDown() {
        TrafficRouter trafficRouter = trafficRouterManager.getTrafficRouter();
        CacheRegister cacheRegister = trafficRouter.getCacheRegister();
        JsonNode config = cacheRegister.getConfig();
        if (oldFedUrl != null && !oldFedUrl.isEmpty()) {
            config = ((ObjectNode) config).put(federationsWatcher.getWatcherConfigPrefix() + ".polling.url", oldFedUrl);
        } else {
            config = ((ObjectNode) config).remove(federationsWatcher.getWatcherConfigPrefix() + ".polling.url");
        }
        federationsWatcher.trafficOpsUtils.setConfig(config);
        federationsWatcher.configure(config);
        assertThat(federationsWatcher.getDataBaseURL(), endsWith(DEFAULT_FEDERATION_DATA_URL.split("api")[1]));
    }

    @Test
    public void testWatchers() {
        TrafficRouter trafficRouter = trafficRouterManager.getTrafficRouter();
        CacheRegister cacheRegister = trafficRouter.getCacheRegister();
        JsonNode config = cacheRegister.getConfig();
        JsonNode pollingUrlNode = config.get(federationsWatcher.getWatcherConfigPrefix() + ".polling.url");
        if (pollingUrlNode != null) {
            assertThat(pollingUrlNode.asText(), not(endsWith("api/3.0/notAFederationsEndpoint")));
        }
        assertThat(federationsWatcher.getDataBaseURL(), endsWith(DEFAULT_FEDERATION_DATA_URL.split("api")[1]));
        assertThat(steeringWatcher.getDataBaseURL(), endsWith(DEFAULT_STEERING_DATA_URL.split("api")[1]));

        String newFedsUrl = "https://${toHostname}/api/3.0/notAFederationsEndpoint";
        config = ((ObjectNode) config).put(federationsWatcher.getWatcherConfigPrefix() + ".polling.url", newFedsUrl);
        federationsWatcher.trafficOpsUtils.setConfig(config);
        federationsWatcher.configure(config);
        config = cacheRegister.getConfig();
        assertThat(config.get(federationsWatcher.getWatcherConfigPrefix() + ".polling.url").asText(), endsWith("api/3.0/notAFederationsEndpoint"));
        assertThat(federationsWatcher.getDataBaseURL(), endsWith("api/3.0/notAFederationsEndpoint"));
    }
}
