--  Licensed to the Apache Software Foundation (ASF) under one
--  or more contributor license agreements.  See the NOTICE file
--  distributed with this work for additional information
--  regarding copyright ownership.  The ASF licenses this file
--  to you under the Apache License, Version 2.0 (the
--  "License"); you may not use this file except in compliance
--  with the License.  You may obtain a copy of the License at
--
--  http://www.apache.org/licenses/LICENSE-2.0
--
--  Unless required by applicable law or agreed to in writing, software
--  distributed under the License is distributed on an "AS IS" BASIS,
--  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
--  See the License for the specific language governing permissions and
--  limitations under the License.

_G.ts = {
        client_request = {
            header = {
                Host = "ats.ds.cdn.domain.com"
            },
        },
        ctx = {}
}
_G.TS_LUA_REMAP_DID_REMAP = 1

describe("Busted unit testing framework", function()
  describe("script for ATS Lua Plugin", function()

    it("tests if there's no query string and no cookie", function()

      stub(ts.client_request, "get_uri_args")
      stub(ts, "hook")

      require("sst")
      local result = __init__({})
      local result = do_remap()
      assert(string.find(ts.ctx['cookie'],"omd_cookie=uuid:%x%x%x%x%x%x%x%x%-%x%x%x%x%-%x%x%x%x%-%x%x%x%x%-%x%x%x%x%x%x%x%x%x%x%x%x"))

    end)

    it("tests if there's no query string and already a cookie", function()

      stub(ts.client_request, "get_uri_args")
      stub(ts, "hook")

      _G.ts.client_request.header['Cookie'] = "param3=5" -- set up the cookie for this test
      require("sst")
      local result = __init__({})
      local result = do_remap()
      assert(string.find(ts.ctx['cookie'],"omd_cookie=param3:5"))

    end)

    it("tests if there's a query string and already a cookie", function()

      ts.client_request.get_uri_args = function() return "?param1=1234&param2=asdf" end
      stub(ts, "hook")

      _G.ts.client_request.header['Cookie'] = "param3=5" -- set up the cookie for this test
      require("sst")
      local result = __init__({"sst.lua","param1","param2"})
      local result = do_remap()
      assert(string.find(ts.ctx['cookie'],"omd_cookie=param1:1234&param2:asdf"))

    end)

  end)
end)
