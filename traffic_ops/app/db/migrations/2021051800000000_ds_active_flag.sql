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

-- +goose Up
CREATE TYPE public.ds_active_state AS ENUM (
	'ACTIVE',
	'INACTIVE',
	'PRIMED'
);
ALTER TABLE public.deliveryservice
ADD COLUMN active_state ds_active_state NOT NULL DEFAULT 'INACTIVE';

UPDATE public.deliveryservice SET active_state = 'ACTIVE' WHERE active IS TRUE;
UPDATE public.deliveryservice SET active_state = 'PRIMED' WHERE active IS FALSE;

ALTER TABLE public.deliveryservice DROP COLUMN active;
ALTER TABLE public.deliveryservice RENAME COLUMN active_state TO active;

-- +goose Down
ALTER TABLE public.deliveryservice
ADD COLUMN active_flag boolean DEFAULT FALSE NOT NULL;

UPDATE public.deliveryservice
SET active_flag = FALSE
WHERE active IS 'PRIMED' OR active IS 'INACTIVE';

UPDATE public.deliveryservice
SET active_flag = TRUE
WHERE active IS 'ACTIVE';

ALTER TABLE public.deliveryservice DROP COLUMN active;
ALTER TABLE public.deliveryservice RENAME COLUMN active_flag TO active;
DROP TYPE public.ds_active_state;
