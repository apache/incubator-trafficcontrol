--  Licensed to the Apache Software Foundation (ASF) under one
--  or more contributor license agreements.  See the NOTICE file
--  distributed with this work for additional information
--  regarding copyright ownership.  The ASF licenses this file
--  to you under the Apache License, Version 2.0 (the
--  "License"); you may not use this file except in compliance
--  with the License.  You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
--  Unless required by applicable law or agreed to in writing, software
--  distributed under the License is distributed on an "AS IS" BASIS,
--  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
--  See the License for the specific language governing permissions and
--  limitations under the License.

package.path = package.path .. ";/opt/trafficserver/script/uuid.lua"
local uuid = require("uuid")
local QUERY_STRING_PARAMS = ""

function __init__(argtb)
    local unique_id = tostring( {} ):sub(8) -- memory address of blank table
    local x = unique_id + os.time() -- plus current time
    uuid.randomseed(x) -- equals seed to random cookie generator
    QUERY_STRING_PARAMS = argtb -- field(s) in query string to use as cookie (can be blank)
    return 0
end

function send_request()
    ts.server_request.header['Cookie'] = ts.ctx['cookie']
    return 0
end

function send_response()
    ts.client_response.header['Set-Cookie'] = ts.ctx['set-cookie']
    return 0
end

function do_remap()
--  Usage: in remap.config:
--      remap a.domain.com b.origin.com @plugin=ts_lua.so @pparam=sst.lua @pparam=param1 @pparam=param2...
--  This function executes whenever a request is received after the URL is remapped to the origin.
--  It takes as input the query string parameters from the request URL and looks for the parameters configured as @pparam above.
--  If any subset of the configured parameters is found, a Cookie is added to the upstream request containing the parameters and their values
--  and a Set-Cookie header is returned to the downstream client also containing those parameters plus HttpOnly and a Domain header
--  corresponding to the original request's domain (delivery service domain).
--
--  If none of the parameters are found (or if none are configured) then a random UUID will be set as the Cookie and Set-Cookie headers
--  with the key "uuid".

    local _,_,domain = string.find(ts.client_request.header['Host'], ".-%.(.*)") -- get FQDN, extract domain

    local new_cookie_string = ""
    local uri_args = ts.client_request.get_uri_args()
    local cookie_table = {}

    if uri_args then
        uri_args = "&" .. uri_args
        for k,v in pairs(QUERY_STRING_PARAMS)
        do
            if k > 0 then
                local cookie_value = ""
                local uri_arg_pattern = "&(" .. v .. ")=([^&]*)"
                i,j,param,value = string.find(uri_args, uri_arg_pattern)
                if i then cookie_table[param] = value end
            end
        end
    end

--  sort the cookie table lexicographically
    local sorted_cookie_table = {}
    for k in pairs(cookie_table) do table.insert(sorted_cookie_table, k) end
    table.sort(sorted_cookie_table)

    for _,k in ipairs(sorted_cookie_table)
    do
        new_cookie_string = new_cookie_string .. k .. ":" .. cookie_table[k] .. "&"
    end

    if new_cookie_string ~= "" then -- query string exists, overwrite old cookie if it's there
        new_cookie_string = new_cookie_string:sub(1,-2)
        ts.ctx['cookie'] = new_cookie_string
        ts.client_request.header['Cookie'] = "omd_cookie=" .. new_cookie_string
        ts.ctx['set-cookie'] = "omd_cookie=" .. ts.ctx['cookie'] .. "; " ..  "HttpOnly; " ..  "Domain=" .. domain
    else -- query string doesn't exist, either use old cookie (if exists) or generate a random UUID
        ts.ctx['cookie'] = ts.client_request.header['Cookie'] or ("uuid:" .. uuid())
        if ts.client_request.header['Cookie'] == nil then -- only set a new cookie if there is no old one
            ts.ctx['set-cookie'] = "omd_cookie=" .. ts.ctx['cookie'] .. "; " ..  "HttpOnly; " ..  "Domain=" .. domain
            ts.client_request.header['Cookie'] = "omd_cookie=" .. ts.ctx['cookie']
        end
    end

--  Remove double quotes from the cookie
    ts.ctx['cookie'] = string.gsub(ts.ctx['cookie'],"\"","%%22")

    ts.hook(TS_LUA_HOOK_SEND_REQUEST_HDR, send_request)
    ts.hook(TS_LUA_HOOK_SEND_RESPONSE_HDR, send_response)
    return 0
end
