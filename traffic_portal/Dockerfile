#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
#
FROM node:4-onbuild
#FROM buildpack-deps:jessie

RUN apt-get update -y && apt-get install libffi-dev ruby-dev rubygems vim -y

# replace this with your application's default port
RUN gem update --system && gem install --no-rdoc --no-ri compass && gem install --no-rdoc --no-ri sass -v 3.4.22
RUN npm install -g
RUN cd /usr/src/app && /usr/local/bin/grunt dist
