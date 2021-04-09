#!/usr/bin/python3
#
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
This script checks to see if DNSSEC keys need to be re-generated, and then
submits a request to do so if they do.

Usage: ToAutorenewCerts.py [-h] [-l LOG_LEVEL] -c CONFIG [-k] [-v]

-h, --help                           show this help message and exit
-l LOG_LEVEL, --log-level LOG_LEVEL  Set the log level of the script. 1 or lower for informational
                                     output, 2 for debug output, 3 or higher for printing stack
                                     traces
-c CONFIG, --config CONFIG           Either the literal configuration or the location of a file
                                     containing the configuration
-k, --insecure                       Do not verify SSL certificate chains in requests
-v, --version                        show program's version number and exit
"""

import argparse
import json
import logging
import os
import ssl
import sys
import urllib

from http import cookiejar

__version__ = "0.0.1"

#: This is the custom log-level used for
TRACE_LEVEL = 5

#: This is used during config parsing to output helpful error messages
MISSING_KEY_TEMPLATE = 'Config missing "{}" key, this script now requires "base_url", "user" and' +\
	' "pass", as the endpoint used by this script now requires admin-level authentication.'

class Config:
	"""
	Config represents configuration for the check extension.
	"""

	#: Whether or not to verify SSL certificates


	def __init__(self, base_url: str, user: str, password: str, insecure: bool = False):
		self.base_url = base_url.rstrip('/')
		self.username = user
		self.password = password
		self.insecure = insecure

	def __repr__(self) -> str:
		props = ", ".join((
		f"base_url='{self.base_url}'",
		f"username='{self.username}'",
		f"password='{self.password}'",
		f"insecure={self.insecure}"
		))
		return f"Config({props})"

	def payload(self) -> bytes:
		"""
		Returns an HTTP request payload containing the JSON-encoded Traffic Ops login credentials.

		>>> Config("", "foo", "bar").payload()
		b'{"u":"foo","p":"bar"}'
		"""
		return f'{{"u":"{self.username}","p":"{self.password}"}}'.encode()

def unmarshal_config(dct: dict, insecure: bool) -> Config:
	"""
	Constructs a new Config object from the passed dict.

	>>> unmarshal_config({"base_url": "testquest", "user": "foo", "pass": "bar"}, False)
	Config(base_url='testquest', username='foo', password='bar', insecure=False)
	"""
	if "user" not in dct or not dct["user"]:
		raise ValueError(MISSING_KEY_TEMPLATE.format("user"))
	if "pass" not in dct or not dct["pass"]:
		raise ValueError(MISSING_KEY_TEMPLATE.format("pass"))
	if "base_url" not in dct or not dct["base_url"]:
		raise ValueError(MISSING_KEY_TEMPLATE.format("base_url"))

	return Config(dct["base_url"], dct["user"], dct["pass"], insecure)

def get_cookie(cfg: Config) -> urllib.request.URLopener:
	"""
	Authenticates with Traffic Ops as directed by the passed configuration, and returns a URLopener
	that stores and passes the mojolicious cookie, so that it doesn't need to be manually managed.
	"""
	login_url = f"{cfg.base_url}/api/2.0/user/login"
	logging.log(TRACE_LEVEL, "posting %s", login_url)
	headers = {"Content-Type": "application/json"}
	req = urllib.request.Request(login_url, headers=headers, data=cfg.payload(), method="POST")
	ctx = ssl.SSLContext()
	ctx.check_hostname = False
	ctx.verify_mode = ssl.CERT_NONE if cfg.insecure else ssl.CERT_REQUIRED

	opener = urllib.request.build_opener(
		urllib.request.HTTPSHandler(context=ctx),
		urllib.request.HTTPCookieProcessor(cookiejar.CookieJar())
	)
	resp = opener.open(req)
	code = resp.getcode()
	logging.log(TRACE_LEVEL, "%s", resp.read())
	if code not in range(200, 300):
		raise ConnectionError(f"authentication with Traffic Ops failed with code {code}")

	return opener

def renew_certs(cfg: Config, opener: urllib.request.URLopener):
	"""
	Performs the request to automatically renew certificates.
	"""
	url = f"{cfg.base_url}/api/2.0/letsencrypt/autorenew"
	logging.log(TRACE_LEVEL, "getting %s", url) # We're actually POSTing
	req = urllib.request.Request(url, method="POST", headers={"Content-Type": "application/json"})
	resp = opener.open(req)
	code = resp.getcode()
	body = resp.read()
	logging.log(TRACE_LEVEL, "%s", body)
	if code not in range(200, 300):
		raise ConnectionError(f"response was {code} {resp.reason}")

	logging.debug("Successfully refreshed keys response was %s", body)

def main(log_level: int, cfg_or_file: str, insecure: bool) -> int:
	"""
	The main routine of the script.
	"""
	logging.addLevelName("TRACE", TRACE_LEVEL)

	if log_level <= 1:
		logging.getLogger().setLevel(logging.INFO)
	elif log_level == 2:
		logging.getLogger().setLevel(logging.DEBUG)
	else:
		logging.getLogger().setLevel(TRACE_LEVEL)

	logging.debug("Including DEBUG messages in output. Config is '%s'", cfg_or_file)
	logging.log(TRACE_LEVEL, "Including TRACE messages in output. Config is '%s'", cfg_or_file)

	config = None
	try:
		config = json.loads(cfg_or_file)
	except json.JSONDecodeError as e:
		if os.path.isfile(cfg_or_file):
			try:
				with open(cfg_or_file) as cfg_file:
					config = json.load(cfg_file)
			except (OSError, json.JSONDecodeError) as file_error:
				logging.fatal("Reading configuration file '%s': %s", cfg_or_file, file_error)
				logging.log(TRACE_LEVEL, "", stack_info=True, exc_info=True)
				return 1
		else:
			logging.error("Bad json config: %s", e)
			return 1

	logging.log(TRACE_LEVEL, "%r", config)

	try:
		config = unmarshal_config(config, insecure)
	except ValueError as e:
		logging.error("%s", e)
		logging.log(TRACE_LEVEL, "", stack_info=True, exc_info=True)
		return 1

	lwp = None
	try:
		lwp = get_cookie(config)
	except (ConnectionError, urllib.error.HTTPError) as e:
		logging.error("Error trying to update keys: %s", e)
		logging.log(TRACE_LEVEL, "", stack_info=True, exc_info=True)
		return 1

	try:
		renew_certs(config, lwp)
	except (ConnectionError, urllib.error.HTTPError) as e:
		logging.error("Error trying to update keys: %s", e)
		logging.log(TRACE_LEVEL, "", stack_info=True, exc_info=True)
		return 1 #Perl returned zero here, I refuse to follow suit.
	return 0


if __name__ == '__main__':
	PARSER = argparse.ArgumentParser()
	PARSER.add_argument(
		"-l",
		"--log-level",
		help="Set the log level of the script. 1 or lower for informational output, 2 for debug" +
		" output, 3 or higher for printing stack traces",
		type=int,
		default=1
	)
	PARSER.add_argument(
		"-c",
		"--config",
		help="Either the literal configuration or the location of a file containing the " +
		"configuration",
		required=True
	)
	PARSER.add_argument(
		"-k",
		"--insecure",
		help="Do not verify SSL certificate chains in requests",
		action="store_true"
	)
	PARSER.add_argument(
		"-v",
		"--version",
		action="version",
		version=__version__
	)
	args = PARSER.parse_args()
	try:
		sys.exit(main(args.log_level, args.config, args.insecure))
	except KeyboardInterrupt:
		sys.exit(1)
	finally:
		logging.shutdown()
