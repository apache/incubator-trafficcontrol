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

CREATE TYPE profile_type AS ENUM ('ATS_PROFILE', 'TR_PROFILE', 'TM_PROFILE', 'TS_PROFILE', 'TP_PROFILE', 'INFLUXDB_PROFILE', 
  'RIAK_PROFILE', 'SPLUNK_PROFILE', 'DS_PROFILE', 'ORG_PROFILE', 'KAFKA_PROFILE', 'LOGSTASH_PROFILE', 'ES_PROFILE', 'UNK_PROFILE');
ALTER TABLE public.profile ADD COLUMN type profile_type;
UPDATE public.profile SET type='UNK_PROFILE'; -- So we don't get any NULL, these should be checked.
UPDATE public.profile SET type='TR_PROFILE' WHERE name like 'CCR_%' OR name like 'TR_%';
UPDATE public.profile SET type='TM_PROFILE' WHERE name like 'RASCAL_%' OR name like 'TM_%';
UPDATE public.profile SET type='TS_PROFILE' WHERE name like 'TRAFFIC_STATS%';
UPDATE public.profile SET type='TP_PROFILE' WHERE name like 'TRAFFIC_PORTAL%';
UPDATE public.profile SET type='INFLUXDB_PROFILE' WHERE name like 'INFLUXDB%';
UPDATE public.profile SET type='RIAK_PROFILE' WHERE name like 'RIAK%';
UPDATE public.profile SET type='SPLUNK_PROFILE' WHERE name like 'SPLUNK%';
UPDATE public.profile SET type='ORG_PROFILE' WHERE name like '%ORG%' or name like 'MSO%' or name like '%ORIGIN%';
UPDATE public.profile SET type='KAFKA_PROFILE' WHERE name like 'KAFKA%';
UPDATE public.profile SET type='LOGSTASH_PROFILE' WHERE name like 'LOGSTASH_%';
UPDATE public.profile SET type='ES_PROFILE' WHERE name like 'ELASTICSEARCH%';
UPDATE public.profile SET type='ATS_PROFILE' WHERE name like 'EDGE%' or name like 'MID%';

ALTER TABLE public.profile ALTER type SET NOT NULL;

ALTER TABLE public.cdn ADD COLUMN domain_name text;
UPDATE cdn SET domain_name=domainlist.value
  FROM (SELECT distinct cdn_id,value FROM server,parameter WHERE type=(SELECT id FROM type WHERE name='EDGE') 
    AND parameter.id in (select parameter from profile_parameter WHERE profile_parameter.profile=server.profile) 
    AND parameter.name='domain_name' 
    AND config_file='CRConfig.json') AS domainlist
WHERE id = domainlist.cdn_id;
UPDATE public.cdn SET domain_name='-' WHERE name='ALL';
ALTER TABLE public.cdn ALTER COLUMN domain_name SET NOT NULL;

ALTER TABLE public.profile ADD COLUMN cdn bigint;

ALTER TABLE public.profile
  ADD CONSTRAINT fk_cdn1 FOREIGN KEY (cdn)
      REFERENCES public.cdn (id) MATCH SIMPLE
      ON UPDATE RESTRICT ON DELETE RESTRICT;
CREATE INDEX idx_181818_fk_cdn1
  ON public.profile
  USING btree
  (cdn);

UPDATE profile set cdn=domainlist.cdn_id
  FROM (SELECT distinct profile.id AS profile_id, value AS profile_domain_name, cdn.id cdn_id 
    FROM profile, parameter, cdn, profile_parameter
    WHERE parameter.name='domain_name'
    AND parameter.config_file='CRConfig.json'
    AND parameter.value = cdn.domain_name
    AND parameter.id in (select parameter from profile_parameter where profile=profile.id)) as domainlist
WHERE id = domainlist.profile_id;

ALTER TABLE deliveryservice ALTER profile DROP NOT NULL;

ALTER TABLE deliveryservice DROP COLUMN multi_site_origin_algorithm;


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE public.cdn DROP COLUMN domain_name;

ALTER TABLE public.profile DROP COLUMN cdn;

ALTER TABLE deliveryservice ALTER profile SET NOT NULL;

ALTER TABLE public.profile DROP COLUMN type;

DROP type profile_type;
