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

package com.comcast.cdn.traffic_control.traffic_router.api.controllers;

import com.comcast.cdn.traffic_control.traffic_router.core.status.model.CacheModel;
import com.comcast.cdn.traffic_control.traffic_router.core.util.DataExporter;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestMethod;
import org.springframework.web.bind.annotation.ResponseBody;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Controller
@RequestMapping("/locations")
public class LocationController {

	@Autowired
	private DataExporter dataExporter;

	@RequestMapping(value = "/{locID}/caches", method = {RequestMethod.GET, RequestMethod.HEAD})
	public @ResponseBody
	Map<String, List<CacheModel>> getCaches(@PathVariable("locID") final String locId) {
		final Map<String, List<CacheModel>> map = new HashMap<String, List<CacheModel>>();
		map.put("caches", dataExporter.getCaches(locId));
		return map;
	}

	@RequestMapping(value = "", method = {RequestMethod.GET, RequestMethod.HEAD})
	public @ResponseBody
	Map<String, List<String>> getLocations() {
		final Map<String, List<String>> locations = new HashMap<String, List<String>>();
		locations.put("locations", dataExporter.getLocations());
		return locations;
	}

	@RequestMapping(value = "/caches", method = {RequestMethod.GET, RequestMethod.HEAD})
	public @ResponseBody
	Map<String, Map<String, List<CacheModel>>> getCaches() {
		final Map<String, Map<String, List<CacheModel>>> map = new HashMap<String, Map<String, List<CacheModel>>>();
		final Map<String, List<CacheModel>> innerMap = new HashMap<String, List<CacheModel>>();

		for (final String location : dataExporter.getLocations()) {
			innerMap.put(location, dataExporter.getCaches(location));
		}

		map.put("locations", innerMap);
		return map;
	}
}
