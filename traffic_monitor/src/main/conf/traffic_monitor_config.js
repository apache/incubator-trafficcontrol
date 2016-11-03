/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 * 
 *   http://www.apache.org/licenses/LICENSE-2.0
 * 
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

{
	"traffic_monitor_config": {
		"health.polling.interval": "5000",
		"tm.polling.interval": "10000",
		"tm.hostname": "",
		"tm.healthParams.polling.url": "https://${tmHostname}/health/${cdnName}",
		"hack.ttl": "30",
		"cdnName": "",
		"peers.polling.url": "http://${hostname}/publish/CrStates?raw",
		"health.timepad": "20",
		"health.event-count": "200",
		"tm.dataServer.polling.url": "https://${tmHostname}/dataserver/orderby/id",
		"tm.auth.url": "https://${tmHostname}/login",
		"tm.auth.username": "",
		"tm.auth.password": "",
		"tm.crConfig.json.polling.url": "https://${tmHostname}/CRConfig-Snapshots/${cdnName}/CRConfig.json"
	}
}
