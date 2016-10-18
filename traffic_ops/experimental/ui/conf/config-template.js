/*

 Copyright 2015 Comcast Cable Communications Management, LLC

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.

 */

// this is the config for /opt/traffic_ops/server/server.js and is consumed when 'sudo service traffic_ops start'
module.exports = {
	timeout: '120s',
	useSSL: false, // set to true if you plan to use https (self-signed or trusted certs).
	port: 8080,
	sslPort: 8443,
	proxyPort: 8009,
	// if useSSL is true, generate ssl certs and provide the proper locations.
	ssl: {
		key:    '/path/to/ssl.key',
		cert:   '/path/to/ssl.crt',
		ca:     [
			'/path/to/ssl-bundle.crt'
		]
	},
	// set api 'base_url' to the traffic ops api (all api calls made from the traffic ops ui will be proxied to the api base_url)
	// enter value for api 'key' if you want to append ?API_KEY=value to all api calls. It is suggested to leave blank.
	api: {
		base_url: 'http(s)://where-traffic-ops-api-is.com/api/',
		key: ''
	},
	// default files location (this is where the traffic ops html, css and javascript was installed)
	files: {
		static: '/opt/traffic_ops/public'
	},
	// default log location (this is where traffic_ops logs are written)
	log: {
		stream: '/var/log/traffic_ops/access.log'
	},
	reject_unauthorized: 0 // 0 if using self-signed certs, 1 if trusted certs
};

