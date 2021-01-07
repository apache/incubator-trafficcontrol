-- syntax:postgresql
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

-- Revert traffic_ops:deliveryservice-active from pg

BEGIN;

ALTER TABLE deliveryservice
ADD COLUMN active_flag boolean DEFAULT FALSE NOT NULL;

UPDATE deliveryservice
SET active_flag = FALSE
WHERE active IS 'PRIMED' OR active IS 'INACTIVE';

UPDATE deliveryservice
SET active_flag = TRUE
WHERE active IS 'ACTIVE';

ALTER TABLE deliveryservice DROP COLUMN active_state;
ALTER TABLE deliveryservice RENAME COLUMN active_flag TO active;
DROP TYPE ds_active_state;

COMMIT;
