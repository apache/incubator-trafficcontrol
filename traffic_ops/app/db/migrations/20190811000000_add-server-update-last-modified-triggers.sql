/*
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

-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE OR REPLACE FUNCTION update_cdn_last_updated_old_server() RETURNS trigger AS $$
BEGIN
  UPDATE cdn SET last_updated=NOW() WHERE id = (select cdn_id from server where id = OLD.server);
  RETURN old;
END;
$$ language plpgsql;

CREATE OR REPLACE FUNCTION update_server_last_updated_new_server() RETURNS trigger AS $$
BEGIN
  UPDATE server SET last_updated=NOW() WHERE id = NEW.server;
  RETURN old;
END;
$$ language plpgsql;

CREATE TRIGGER dss_delete_update_server_modified AFTER DELETE ON deliveryservice_server FOR EACH ROW EXECUTE PROCEDURE update_cdn_last_updated_old_server();
CREATE TRIGGER dss_update_update_server_modified AFTER UPDATE ON deliveryservice_server FOR EACH ROW EXECUTE PROCEDURE update_server_last_updated_new_server();
CREATE TRIGGER dss_insert_update_server_modified AFTER INSERT ON deliveryservice_server FOR EACH ROW EXECUTE PROCEDURE update_server_last_updated_new_server();

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled oback

DROP TRIGGER IF EXISTS dss_delete_update_server_modified ON deliveryservice_server;
DROP TRIGGER IF EXISTS dss_update_update_server_modified ON deliveryservice_server;
DROP TRIGGER IF EXISTS dss_insert_update_server_modified ON deliveryservice_server;

DROP FUNCTION IF EXISTS update_cdn_last_updated_old_server();
DROP FUNCTION IF EXISTS update_server_last_updated_new_server();
