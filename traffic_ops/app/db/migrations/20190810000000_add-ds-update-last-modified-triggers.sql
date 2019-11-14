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

ALTER TABLE deliveryservice_consistent_hash_query_param ADD COLUMN IF NOT EXISTS last_updated timestamp with time zone NOT NULL DEFAULT NOW();

CREATE OR REPLACE FUNCTION update_ds_last_updated_old_deliveryservice() RETURNS trigger AS $$
BEGIN
  UPDATE deliveryservice SET last_updated=NOW() WHERE id = OLD.deliveryservice;
  RETURN old;
END;
$$ language plpgsql;

CREATE OR REPLACE FUNCTION update_ds_last_updated_new_deliveryservice() RETURNS trigger AS $$
BEGIN
  UPDATE deliveryservice SET last_updated=NOW() WHERE id = NEW.deliveryservice;
  RETURN old;
END;
$$ language plpgsql;

CREATE OR REPLACE FUNCTION update_ds_last_updated_old_deliveryservice_id() RETURNS trigger AS $$
BEGIN
  UPDATE deliveryservice SET last_updated=NOW() WHERE id = OLD.deliveryservice_id;
  RETURN old;
END;
$$ language plpgsql;

CREATE OR REPLACE FUNCTION update_ds_last_updated_new_deliveryservice_id() RETURNS trigger AS $$
BEGIN
  UPDATE deliveryservice SET last_updated=NOW() WHERE id = NEW.deliveryservice_id;
  RETURN old;
END;
$$ language plpgsql;

CREATE TRIGGER origin_delete_update_ds_modified AFTER DELETE ON origin FOR EACH ROW EXECUTE PROCEDURE update_ds_last_updated_old_deliveryservice();
CREATE TRIGGER origin_update_update_ds_modified AFTER UPDATE ON origin FOR EACH ROW EXECUTE PROCEDURE update_ds_last_updated_new_deliveryservice();
CREATE TRIGGER origin_insert_update_ds_modified AFTER INSERT ON origin FOR EACH ROW EXECUTE PROCEDURE update_ds_last_updated_new_deliveryservice();

CREATE TRIGGER dsc_delete_update_ds_modified AFTER DELETE on deliveryservice_consistent_hash_query_param FOR EACH ROW EXECUTE PROCEDURE update_ds_last_updated_old_deliveryservice_id();
CREATE TRIGGER dsc_update_update_ds_modified AFTER UPDATE on deliveryservice_consistent_hash_query_param FOR EACH ROW EXECUTE PROCEDURE update_ds_last_updated_new_deliveryservice_id();
CREATE TRIGGER dsc_insert_update_ds_modified AFTER INSERT on deliveryservice_consistent_hash_query_param FOR EACH ROW EXECUTE PROCEDURE update_ds_last_updated_new_deliveryservice_id();

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TRIGGER IF EXISTS origin_delete_update_ds_modified ON origin;
DROP TRIGGER IF EXISTS origin_update_update_ds_modified ON origin;
DROP TRIGGER IF EXISTS origin_insert_update_ds_modified ON origin;

DROP TRIGGER IF EXISTS dsc_delete_update_ds_modified ON deliveryservice_consistent_hash_query_param;
DROP TRIGGER IF EXISTS dsc_update_update_ds_modified ON deliveryservice_consistent_hash_query_param;
DROP TRIGGER IF EXISTS dsc_insert_update_ds_modified ON deliveryservice_consistent_hash_query_param;

DROP FUNCTION IF EXISTS update_ds_last_updated_old_deliveryservice();
DROP FUNCTION IF EXISTS update_ds_last_updated_new_deliveryservice();
DROP FUNCTION IF EXISTS update_ds_last_updated_old_deliveryservice_id();
DROP FUNCTION IF EXISTS update_ds_last_updated_new_deliveryservice_id();

ALTER TABLE deliveryservice_consistent_hash_query_param DROP COLUMN IF EXISTS last_updated;
