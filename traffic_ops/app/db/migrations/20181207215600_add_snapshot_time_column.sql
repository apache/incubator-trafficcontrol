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

ALTER TABLE snapshot ADD COLUMN time timestamp with time zone NOT NULL DEFAULT now();

CREATE TABLE deliveryservice_snapshots (deliveryservice text PRIMARY KEY NOT NULL, time timestamp with time zone NOT NULL, last_updated timestamp with time zone NOT NULL DEFAULT now());

INSERT INTO deliveryservice_snapshots (deliveryservice, time) SELECT ds.xml_id as deliveryservice, sn.time FROM deliveryservice ds JOIN cdn c ON c.id = ds.cdn_id JOIN snapshot sn ON sn.cdn = c.name;

CREATE TRIGGER on_update_current_timestamp BEFORE UPDATE ON deliveryservice_snapshots FOR EACH ROW EXECUTE PROCEDURE on_update_current_timestamp_last_updated();

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE deliveryservice_snapshots;
