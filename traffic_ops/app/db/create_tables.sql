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

--
-- PostgreSQL database dump
--

-- Dumped from database version 9.6.19
-- Dumped by pg_dump version 12.5

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner:
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


SET search_path = public, pg_catalog;

SET default_with_oids = false;

DO $$
BEGIN
IF NOT EXISTS (SELECT FROM pg_type WHERE pg_type.typname = 'change_types') THEN
    --
    -- Name: change_types; Type: TYPE; Schema: public; Owner: traffic_ops
    --

    CREATE TYPE change_types AS ENUM (
        'create',
        'update',
        'delete'
    );

    ALTER TYPE change_types OWNER TO traffic_ops;
END IF;

IF NOT EXISTS (SELECT FROM pg_type WHERE pg_type.typname = 'deep_caching_type') THEN
    --
    -- Name: deep_caching_type; Type: TYPE; Schema: public; Owner: traffic_ops
    --

    CREATE TYPE deep_caching_type AS ENUM (
        'NEVER',
        'ALWAYS'
    );

    ALTER TYPE deep_caching_type OWNER TO traffic_ops;
END IF;

IF NOT EXISTS (SELECT FROM pg_type WHERE pg_type.typname = 'deliveryservice_signature_type') THEN
    --
    -- Name: deliveryservice_signature_type; Type: DOMAIN; Schema: public; Owner: traffic_ops
    --

    CREATE DOMAIN public.deliveryservice_signature_type AS text
        CONSTRAINT deliveryservice_signature_type_check CHECK ((VALUE = ANY (ARRAY['url_sig'::text, 'uri_signing'::text])));


    ALTER DOMAIN public.deliveryservice_signature_type OWNER TO traffic_ops;
END IF;

IF NOT EXISTS (SELECT FROM pg_type WHERE pg_type.typname = 'http_method_t') THEN
    --
    -- Name: http_method_t; Type: TYPE; Schema: public; Owner: traffic_ops
    --

    CREATE TYPE public.http_method_t AS ENUM (
        'GET',
        'POST',
        'PUT',
        'PATCH',
        'DELETE'
    );

    ALTER TYPE public.http_method_t OWNER TO traffic_ops;
END IF;



IF NOT EXISTS (SELECT FROM pg_type WHERE pg_type.typname = 'localization_method') THEN
    --
    -- Name: localization_method; Type: TYPE; Schema: public; Owner: traffic_ops
    --

    CREATE TYPE public.localization_method AS ENUM (
        'CZ',
        'DEEP_CZ',
        'GEO'
    );

    ALTER TYPE public.localization_method OWNER TO traffic_ops;
END IF;



IF NOT EXISTS (SELECT FROM pg_type WHERE pg_type.typname = 'origin_protocol') THEN
    --
    -- Name: origin_protocol; Type: TYPE; Schema: public; Owner: traffic_ops
    --

    CREATE TYPE public.origin_protocol AS ENUM (
        'http',
        'https'
    );

    ALTER TYPE public.origin_protocol OWNER TO traffic_ops;
END IF;



IF NOT EXISTS (SELECT FROM pg_type WHERE pg_type.typname = 'profile_type') THEN
    --
    -- Name: profile_type; Type: TYPE; Schema: public; Owner: traffic_ops
    --

    CREATE TYPE public.profile_type AS ENUM (
        'ATS_PROFILE',
        'TR_PROFILE',
        'TM_PROFILE',
        'TS_PROFILE',
        'TP_PROFILE',
        'INFLUXDB_PROFILE',
        'RIAK_PROFILE',
        'SPLUNK_PROFILE',
        'DS_PROFILE',
        'ORG_PROFILE',
        'KAFKA_PROFILE',
        'LOGSTASH_PROFILE',
        'ES_PROFILE',
        'UNK_PROFILE',
        'GROVE_PROFILE'
    );

    ALTER TYPE public.profile_type OWNER TO traffic_ops;
END IF;



IF NOT EXISTS (SELECT FROM pg_type WHERE pg_type.typname = 'server_ip_address') THEN
    --
    -- Name: server_ip_address; Type: TYPE; Schema: public; Owner: traffic_ops
    --

    CREATE TYPE public.server_ip_address AS (
        address inet,
        gateway inet,
        service_address boolean
    );

    ALTER TYPE public.server_ip_address OWNER TO traffic_ops;
END IF;



IF NOT EXISTS (SELECT FROM pg_type WHERE pg_type.typname = 'server_interface') THEN
    --
    -- Name: server_interface; Type: TYPE; Schema: public; Owner: traffic_ops
    --

    CREATE TYPE public.server_interface AS (
        ip_addresses public.server_ip_address[],
        max_bandwidth bigint,
        monitor boolean,
        mtu bigint,
        name text
    );

    ALTER TYPE public.server_interface OWNER TO traffic_ops;
END IF;


IF NOT EXISTS (SELECT FROM pg_type WHERE pg_type.typname = 'workflow_states') THEN
    --
    -- Name: workflow_states; Type: TYPE; Schema: public; Owner: traffic_ops
    --

    CREATE TYPE public.workflow_states AS ENUM (
        'draft',
        'submitted',
        'rejected',
        'pending',
        'complete'
    );

    ALTER TYPE public.workflow_states OWNER TO traffic_ops;
END IF;
END$$;

--
-- Name: before_ip_address_table(); Type: FUNCTION; Schema: public; Owner: traffic_ops
--

CREATE OR REPLACE FUNCTION public.before_ip_address_table() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    server_count   BIGINT;
    server_id      BIGINT;
    server_profile BIGINT;
BEGIN
    WITH server_ips AS (
        SELECT s.id as sid, ip.interface, i.name, ip.address, s.profile, ip.server
        FROM server s
                 JOIN interface i
                     on i.server = s.ID
                 JOIN ip_address ip
                     on ip.Server = s.ID and ip.interface = i.name
        WHERE i.monitor = true
    )
    SELECT count(sip.sid), sip.sid, sip.profile
    INTO server_count, server_id, server_profile
    FROM server_ips sip
             JOIN server_ips sip2 on sip.sid <> sip2.sid
    WHERE (sip.server = NEW.server AND sip.address = NEW.address AND sip.interface = NEW.interface)
      AND sip2.address = sip.address
      AND sip2.profile = sip.profile
    GROUP BY sip.sid, sip.profile;

    IF server_count > 0 THEN
        RAISE EXCEPTION 'ip_address is not unique accross the server [id:%] profile [id:%], [%] conflicts',
            server_id,
            server_profile,
            server_count;
    END IF;
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.before_ip_address_table() OWNER TO traffic_ops;

--
-- Name: before_server_table(); Type: FUNCTION; Schema: public; Owner: traffic_ops
--

CREATE OR REPLACE FUNCTION public.before_server_table() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    server_count BIGINT;
BEGIN
    WITH server_ips AS (
        SELECT s.id, i.name, ip.address, s.profile
        FROM server s
                 JOIN interface i on i.server = s.ID
                 JOIN ip_address ip on ip.Server = s.ID and ip.interface = i.name
        WHERE i.monitor = true
    )
    SELECT count(*)
    INTO server_count
    FROM server_ips sip
             JOIN server_ips sip2 on sip.id <> sip2.id
    WHERE sip.id = NEW.id
      AND sip2.address = sip.address
      AND sip2.profile = sip.profile;

    IF server_count > 0 THEN
        RAISE EXCEPTION 'Server [id:%] does not have a unique ip_address over the profile [id:%], [%] conflicts',
            NEW.id,
            NEW.profile,
            server_count;
    END IF;
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.before_server_table() OWNER TO traffic_ops;

--
-- Name: on_delete_current_timestamp_last_updated(); Type: FUNCTION; Schema: public; Owner: traffic_ops
--

CREATE OR REPLACE FUNCTION public.on_delete_current_timestamp_last_updated() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  update last_deleted set last_updated = now() where table_name = TG_ARGV[0];
  RETURN NEW;
END;
$$;


ALTER FUNCTION public.on_delete_current_timestamp_last_updated() OWNER TO traffic_ops;

--
-- Name: on_update_current_timestamp_last_updated(); Type: FUNCTION; Schema: public; Owner: traffic_ops
--

CREATE OR REPLACE FUNCTION public.on_update_current_timestamp_last_updated() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
  NEW.last_updated = now();
  RETURN NEW;
END;
$$;


ALTER FUNCTION public.on_update_current_timestamp_last_updated() OWNER TO traffic_ops;

SET default_tablespace = '';

--
-- Name: api_capability; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.api_capability (
    id bigint NOT NULL,
    http_method public.http_method_t NOT NULL,
    route text NOT NULL,
    capability text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.api_capability OWNER TO traffic_ops;

--
-- Name: api_capability_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.api_capability_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.api_capability_id_seq OWNER TO traffic_ops;

--
-- Name: api_capability_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.api_capability_id_seq OWNED BY public.api_capability.id;


--
-- Name: asn; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.asn (
    id bigint NOT NULL,
    asn bigint NOT NULL,
    cachegroup bigint DEFAULT '0'::bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.asn OWNER TO traffic_ops;

--
-- Name: asn_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.asn_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.asn_id_seq OWNER TO traffic_ops;

--
-- Name: asn_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.asn_id_seq OWNED BY public.asn.id;


--
-- Name: cachegroup; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.cachegroup (
    id bigint NOT NULL,
    name text NOT NULL,
    short_name text NOT NULL,
    parent_cachegroup_id bigint,
    secondary_parent_cachegroup_id bigint,
    type bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    fallback_to_closest boolean DEFAULT true,
    coordinate bigint
);


ALTER TABLE public.cachegroup OWNER TO traffic_ops;

--
-- Name: cachegroup_fallbacks; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.cachegroup_fallbacks (
    primary_cg bigint NOT NULL,
    backup_cg bigint NOT NULL,
    set_order bigint NOT NULL,
    CONSTRAINT cachegroup_fallbacks_check CHECK ((primary_cg <> backup_cg))
);


ALTER TABLE public.cachegroup_fallbacks OWNER TO traffic_ops;

--
-- Name: cachegroup_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.cachegroup_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.cachegroup_id_seq OWNER TO traffic_ops;

--
-- Name: cachegroup_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.cachegroup_id_seq OWNED BY public.cachegroup.id;


--
-- Name: cachegroup_localization_method; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.cachegroup_localization_method (
    cachegroup bigint NOT NULL,
    method public.localization_method NOT NULL
);


ALTER TABLE public.cachegroup_localization_method OWNER TO traffic_ops;

--
-- Name: cachegroup_parameter; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.cachegroup_parameter (
    cachegroup bigint DEFAULT '0'::bigint NOT NULL,
    parameter bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.cachegroup_parameter OWNER TO traffic_ops;

--
-- Name: capability; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.capability (
    name text NOT NULL,
    description text,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.capability OWNER TO traffic_ops;

--
-- Name: cdn; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.cdn (
    id bigint NOT NULL,
    name text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    dnssec_enabled boolean DEFAULT false NOT NULL,
    domain_name text NOT NULL
);


ALTER TABLE public.cdn OWNER TO traffic_ops;

--
-- Name: cdn_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.cdn_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.cdn_id_seq OWNER TO traffic_ops;

--
-- Name: cdn_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.cdn_id_seq OWNED BY public.cdn.id;


--
-- Name: coordinate; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.coordinate (
    id bigint NOT NULL,
    name text NOT NULL,
    latitude numeric DEFAULT 0.0 NOT NULL,
    longitude numeric DEFAULT 0.0 NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.coordinate OWNER TO traffic_ops;

--
-- Name: coordinate_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.coordinate_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.coordinate_id_seq OWNER TO traffic_ops;

--
-- Name: coordinate_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.coordinate_id_seq OWNED BY public.coordinate.id;


--
-- Name: deliveryservice; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.deliveryservice (
    id bigint NOT NULL,
    xml_id text NOT NULL,
    active boolean DEFAULT false NOT NULL,
    dscp bigint NOT NULL,
    signing_algorithm public.deliveryservice_signature_type,
    qstring_ignore smallint,
    geo_limit smallint DEFAULT '0'::smallint,
    http_bypass_fqdn text,
    dns_bypass_ip text,
    dns_bypass_ip6 text,
    dns_bypass_ttl bigint,
    type bigint NOT NULL,
    profile bigint,
    cdn_id bigint NOT NULL,
    ccr_dns_ttl bigint,
    global_max_mbps bigint,
    global_max_tps bigint,
    long_desc text,
    long_desc_1 text,
    long_desc_2 text,
    max_dns_answers bigint DEFAULT '5'::bigint,
    info_url text,
    miss_lat numeric,
    miss_long numeric,
    check_path text,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    protocol smallint DEFAULT '0'::smallint,
    ssl_key_version bigint DEFAULT '0'::bigint,
    ipv6_routing_enabled boolean DEFAULT false,
    range_request_handling smallint DEFAULT '0'::smallint,
    edge_header_rewrite text,
    origin_shield text,
    mid_header_rewrite text,
    regex_remap text,
    cacheurl text,
    remap_text text,
    multi_site_origin boolean DEFAULT false,
    display_name text NOT NULL,
    tr_response_headers text,
    initial_dispersion bigint DEFAULT '1'::bigint,
    dns_bypass_cname text,
    tr_request_headers text,
    regional_geo_blocking boolean DEFAULT false NOT NULL,
    geo_provider smallint DEFAULT '0'::smallint,
    geo_limit_countries text,
    logs_enabled boolean DEFAULT false,
    multi_site_origin_algorithm smallint,
    geolimit_redirect_url text,
    tenant_id bigint NOT NULL,
    routing_name text DEFAULT 'cdn'::text NOT NULL,
    deep_caching_type public.deep_caching_type DEFAULT 'NEVER'::public.deep_caching_type NOT NULL,
    fq_pacing_rate bigint DEFAULT 0,
    anonymous_blocking_enabled boolean DEFAULT false NOT NULL,
    consistent_hash_regex text,
    max_origin_connections bigint DEFAULT 0 NOT NULL,
    ecs_enabled boolean DEFAULT false NOT NULL,
    range_slice_block_size integer,
    topology text,
    first_header_rewrite text,
    inner_header_rewrite text,
    last_header_rewrite text,
    service_category text,
    CONSTRAINT deliveryservice_max_origin_connections_check CHECK ((max_origin_connections >= 0)),
    CONSTRAINT deliveryservice_range_slice_block_size_check CHECK (((range_slice_block_size >= 262144) AND (range_slice_block_size <= 33554432))),
    CONSTRAINT routing_name_not_empty CHECK ((length(routing_name) > 0))
);


ALTER TABLE public.deliveryservice OWNER TO traffic_ops;

--
-- Name: deliveryservice_consistent_hash_query_param; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.deliveryservice_consistent_hash_query_param (
    name text NOT NULL,
    deliveryservice_id bigint NOT NULL,
    CONSTRAINT name_empty CHECK ((length(name) > 0)),
    CONSTRAINT name_reserved CHECK ((name <> ALL (ARRAY['format'::text, 'trred'::text])))
);


ALTER TABLE public.deliveryservice_consistent_hash_query_param OWNER TO traffic_ops;

--
-- Name: deliveryservice_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.deliveryservice_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.deliveryservice_id_seq OWNER TO traffic_ops;

--
-- Name: deliveryservice_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.deliveryservice_id_seq OWNED BY public.deliveryservice.id;


--
-- Name: deliveryservice_regex; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.deliveryservice_regex (
    deliveryservice bigint NOT NULL,
    regex bigint NOT NULL,
    set_number bigint DEFAULT '0'::bigint,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.deliveryservice_regex OWNER TO traffic_ops;

--
-- Name: deliveryservice_request; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.deliveryservice_request (
    assignee_id bigint,
    author_id bigint NOT NULL,
    change_type public.change_types NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    id bigint NOT NULL,
    last_edited_by_id bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    deliveryservice jsonb NOT NULL,
    status public.workflow_states NOT NULL
);


ALTER TABLE public.deliveryservice_request OWNER TO traffic_ops;

--
-- Name: deliveryservice_request_comment; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.deliveryservice_request_comment (
    author_id bigint NOT NULL,
    deliveryservice_request_id bigint NOT NULL,
    id bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    value text NOT NULL
);


ALTER TABLE public.deliveryservice_request_comment OWNER TO traffic_ops;

--
-- Name: deliveryservice_request_comment_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.deliveryservice_request_comment_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.deliveryservice_request_comment_id_seq OWNER TO traffic_ops;

--
-- Name: deliveryservice_request_comment_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.deliveryservice_request_comment_id_seq OWNED BY public.deliveryservice_request_comment.id;


--
-- Name: deliveryservice_request_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.deliveryservice_request_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.deliveryservice_request_id_seq OWNER TO traffic_ops;

--
-- Name: deliveryservice_request_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.deliveryservice_request_id_seq OWNED BY public.deliveryservice_request.id;


--
-- Name: deliveryservice_server; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.deliveryservice_server (
    deliveryservice bigint NOT NULL,
    server bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.deliveryservice_server OWNER TO traffic_ops;

--
-- Name: deliveryservice_tmuser; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.deliveryservice_tmuser (
    deliveryservice bigint NOT NULL,
    tm_user_id bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.deliveryservice_tmuser OWNER TO traffic_ops;

--
-- Name: deliveryservices_required_capability; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.deliveryservices_required_capability (
    required_capability text NOT NULL,
    deliveryservice_id bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.deliveryservices_required_capability OWNER TO traffic_ops;

--
-- Name: division; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.division (
    id bigint NOT NULL,
    name text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.division OWNER TO traffic_ops;

--
-- Name: division_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.division_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.division_id_seq OWNER TO traffic_ops;

--
-- Name: division_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.division_id_seq OWNED BY public.division.id;


--
-- Name: dnschallenges; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.dnschallenges (
    fqdn text NOT NULL,
    record text NOT NULL
);


ALTER TABLE public.dnschallenges OWNER TO traffic_ops;

--
-- Name: federation; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.federation (
    id bigint NOT NULL,
    cname text NOT NULL,
    description text,
    ttl integer NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.federation OWNER TO traffic_ops;

--
-- Name: federation_deliveryservice; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.federation_deliveryservice (
    federation bigint NOT NULL,
    deliveryservice bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.federation_deliveryservice OWNER TO traffic_ops;

--
-- Name: federation_federation_resolver; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.federation_federation_resolver (
    federation bigint NOT NULL,
    federation_resolver bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.federation_federation_resolver OWNER TO traffic_ops;

--
-- Name: federation_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.federation_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.federation_id_seq OWNER TO traffic_ops;

--
-- Name: federation_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.federation_id_seq OWNED BY public.federation.id;


--
-- Name: federation_resolver; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.federation_resolver (
    id bigint NOT NULL,
    ip_address text NOT NULL,
    type bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.federation_resolver OWNER TO traffic_ops;

--
-- Name: federation_resolver_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.federation_resolver_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.federation_resolver_id_seq OWNER TO traffic_ops;

--
-- Name: federation_resolver_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.federation_resolver_id_seq OWNED BY public.federation_resolver.id;


--
-- Name: federation_tmuser; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.federation_tmuser (
    federation bigint NOT NULL,
    tm_user bigint NOT NULL,
    role bigint,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.federation_tmuser OWNER TO traffic_ops;

--
-- Name: goose_db_version; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.goose_db_version (
    id integer NOT NULL,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp without time zone DEFAULT now()
);


ALTER TABLE public.goose_db_version OWNER TO traffic_ops;

--
-- Name: goose_db_version_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.goose_db_version_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.goose_db_version_id_seq OWNER TO traffic_ops;

--
-- Name: goose_db_version_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.goose_db_version_id_seq OWNED BY public.goose_db_version.id;


--
-- Name: hwinfo; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.hwinfo (
    id bigint NOT NULL,
    serverid bigint NOT NULL,
    description text NOT NULL,
    val text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.hwinfo OWNER TO traffic_ops;

--
-- Name: hwinfo_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.hwinfo_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.hwinfo_id_seq OWNER TO traffic_ops;

--
-- Name: hwinfo_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.hwinfo_id_seq OWNED BY public.hwinfo.id;


--
-- Name: interface; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.interface (
    max_bandwidth bigint,
    monitor boolean NOT NULL,
    mtu bigint DEFAULT 1500,
    name text NOT NULL,
    server bigint NOT NULL,
    CONSTRAINT interface_max_bandwidth_check CHECK (((max_bandwidth IS NULL) OR (max_bandwidth >= 0))),
    CONSTRAINT interface_mtu_check CHECK (((mtu IS NULL) OR (mtu > 1280))),
    CONSTRAINT interface_name_check CHECK ((name <> ''::text))
);


ALTER TABLE public.interface OWNER TO traffic_ops;

--
-- Name: ip_address; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.ip_address (
    address inet NOT NULL,
    gateway inet,
    interface text NOT NULL,
    server bigint NOT NULL,
    service_address boolean DEFAULT false NOT NULL,
    CONSTRAINT ip_address_gateway_check CHECK (((gateway IS NULL) OR ((family(gateway) = 4) AND (masklen(gateway) = 32)) OR ((family(gateway) = 6) AND (masklen(gateway) = 128))))
);


ALTER TABLE public.ip_address OWNER TO traffic_ops;

--
-- Name: job; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.job (
    id bigint NOT NULL,
    agent bigint,
    object_type text,
    object_name text,
    keyword text NOT NULL,
    parameters text,
    asset_url text NOT NULL,
    asset_type text NOT NULL,
    status bigint NOT NULL,
    start_time timestamp with time zone NOT NULL,
    entered_time timestamp with time zone NOT NULL,
    job_user bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    job_deliveryservice bigint
);


ALTER TABLE public.job OWNER TO traffic_ops;

--
-- Name: job_agent; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.job_agent (
    id bigint NOT NULL,
    name text,
    description text,
    active integer DEFAULT 0 NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.job_agent OWNER TO traffic_ops;

--
-- Name: job_agent_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.job_agent_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.job_agent_id_seq OWNER TO traffic_ops;

--
-- Name: job_agent_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.job_agent_id_seq OWNED BY public.job_agent.id;


--
-- Name: job_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.job_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.job_id_seq OWNER TO traffic_ops;

--
-- Name: job_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.job_id_seq OWNED BY public.job.id;


--
-- Name: job_status; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.job_status (
    id bigint NOT NULL,
    name text,
    description text,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.job_status OWNER TO traffic_ops;

--
-- Name: job_status_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.job_status_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.job_status_id_seq OWNER TO traffic_ops;

--
-- Name: job_status_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.job_status_id_seq OWNED BY public.job_status.id;


--
-- Name: last_deleted; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.last_deleted (
    table_name text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.last_deleted OWNER TO traffic_ops;

--
-- Name: lets_encrypt_account; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.lets_encrypt_account (
    email text NOT NULL,
    private_key text NOT NULL,
    uri text NOT NULL
);


ALTER TABLE public.lets_encrypt_account OWNER TO traffic_ops;

--
-- Name: log; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.log (
    id bigint NOT NULL,
    level text,
    message text NOT NULL,
    tm_user bigint NOT NULL,
    ticketnum text,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.log OWNER TO traffic_ops;

--
-- Name: log_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.log_id_seq OWNER TO traffic_ops;

--
-- Name: log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.log_id_seq OWNED BY public.log.id;


--
-- Name: origin; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.origin (
    id bigint NOT NULL,
    name text NOT NULL,
    fqdn text NOT NULL,
    protocol public.origin_protocol DEFAULT 'http'::public.origin_protocol NOT NULL,
    is_primary boolean DEFAULT false NOT NULL,
    port bigint,
    ip_address text,
    ip6_address text,
    deliveryservice bigint NOT NULL,
    coordinate bigint,
    profile bigint,
    cachegroup bigint,
    tenant bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.origin OWNER TO traffic_ops;

--
-- Name: origin_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.origin_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.origin_id_seq OWNER TO traffic_ops;

--
-- Name: origin_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.origin_id_seq OWNED BY public.origin.id;


--
-- Name: parameter; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.parameter (
    id bigint NOT NULL,
    name text NOT NULL,
    config_file text,
    value text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    secure boolean DEFAULT false NOT NULL
);


ALTER TABLE public.parameter OWNER TO traffic_ops;

--
-- Name: parameter_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.parameter_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.parameter_id_seq OWNER TO traffic_ops;

--
-- Name: parameter_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.parameter_id_seq OWNED BY public.parameter.id;


--
-- Name: phys_location; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.phys_location (
    id bigint NOT NULL,
    name text NOT NULL,
    short_name text NOT NULL,
    address text NOT NULL,
    city text NOT NULL,
    state text NOT NULL,
    zip text NOT NULL,
    poc text,
    phone text,
    email text,
    comments text,
    region bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.phys_location OWNER TO traffic_ops;

--
-- Name: phys_location_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.phys_location_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.phys_location_id_seq OWNER TO traffic_ops;

--
-- Name: phys_location_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.phys_location_id_seq OWNED BY public.phys_location.id;


--
-- Name: profile; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.profile (
    id bigint NOT NULL,
    name text NOT NULL,
    description text,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    type public.profile_type NOT NULL,
    cdn bigint NOT NULL,
    routing_disabled boolean DEFAULT false NOT NULL
);


ALTER TABLE public.profile OWNER TO traffic_ops;

--
-- Name: profile_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.profile_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.profile_id_seq OWNER TO traffic_ops;

--
-- Name: profile_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.profile_id_seq OWNED BY public.profile.id;


--
-- Name: profile_parameter; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.profile_parameter (
    profile bigint NOT NULL,
    parameter bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.profile_parameter OWNER TO traffic_ops;

--
-- Name: profile_type_values; Type: VIEW; Schema: public; Owner: traffic_ops
--

CREATE VIEW public.profile_type_values AS
 SELECT unnest(enum_range(NULL::public.profile_type)) AS value
  ORDER BY (unnest(enum_range(NULL::public.profile_type)));


ALTER TABLE public.profile_type_values OWNER TO traffic_ops;

--
-- Name: regex; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.regex (
    id bigint NOT NULL,
    pattern text DEFAULT ''::text NOT NULL,
    type bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.regex OWNER TO traffic_ops;

--
-- Name: regex_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.regex_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.regex_id_seq OWNER TO traffic_ops;

--
-- Name: regex_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.regex_id_seq OWNED BY public.regex.id;


--
-- Name: region; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.region (
    id bigint NOT NULL,
    name text NOT NULL,
    division bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.region OWNER TO traffic_ops;

--
-- Name: region_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.region_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.region_id_seq OWNER TO traffic_ops;

--
-- Name: region_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.region_id_seq OWNED BY public.region.id;


--
-- Name: role; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.role (
    id bigint NOT NULL,
    name text NOT NULL,
    description text,
    priv_level bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.role OWNER TO traffic_ops;

--
-- Name: role_capability; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.role_capability (
    role_id bigint NOT NULL,
    cap_name text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.role_capability OWNER TO traffic_ops;

--
-- Name: role_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.role_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.role_id_seq OWNER TO traffic_ops;

--
-- Name: role_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.role_id_seq OWNED BY public.role.id;


--
-- Name: server; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.server (
    id bigint NOT NULL,
    host_name text NOT NULL,
    domain_name text NOT NULL,
    tcp_port bigint,
    xmpp_id text,
    xmpp_passwd text,
    phys_location bigint NOT NULL,
    rack text,
    cachegroup bigint DEFAULT '0'::bigint NOT NULL,
    type bigint NOT NULL,
    status bigint NOT NULL,
    offline_reason text,
    upd_pending boolean DEFAULT false NOT NULL,
    profile bigint NOT NULL,
    cdn_id bigint NOT NULL,
    mgmt_ip_address text,
    mgmt_ip_netmask text,
    mgmt_ip_gateway text,
    ilo_ip_address text,
    ilo_ip_netmask text,
    ilo_ip_gateway text,
    ilo_username text,
    ilo_password text,
    router_host_name text,
    router_port_name text,
    guid text,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    https_port bigint,
    reval_pending boolean DEFAULT false NOT NULL,
    status_last_updated timestamp with time zone
);


ALTER TABLE public.server OWNER TO traffic_ops;

--
-- Name: server_capability; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.server_capability (
    name text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT name_empty CHECK ((length(name) > 0))
);


ALTER TABLE public.server_capability OWNER TO traffic_ops;

--
-- Name: server_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.server_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.server_id_seq OWNER TO traffic_ops;

--
-- Name: server_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.server_id_seq OWNED BY public.server.id;


--
-- Name: server_server_capability; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.server_server_capability (
    server_capability text NOT NULL,
    server bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.server_server_capability OWNER TO traffic_ops;

--
-- Name: servercheck; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.servercheck (
    id bigint NOT NULL,
    server bigint NOT NULL,
    aa bigint,
    ab bigint,
    ac bigint,
    ad bigint,
    ae bigint,
    af bigint,
    ag bigint,
    ah bigint,
    ai bigint,
    aj bigint,
    ak bigint,
    al bigint,
    am bigint,
    an bigint,
    ao bigint,
    ap bigint,
    aq bigint,
    ar bigint,
    bf bigint,
    at bigint,
    au bigint,
    av bigint,
    aw bigint,
    ax bigint,
    ay bigint,
    az bigint,
    ba bigint,
    bb bigint,
    bc bigint,
    bd bigint,
    be bigint,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.servercheck OWNER TO traffic_ops;

--
-- Name: servercheck_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.servercheck_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.servercheck_id_seq OWNER TO traffic_ops;

--
-- Name: servercheck_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.servercheck_id_seq OWNED BY public.servercheck.id;


--
-- Name: service_category; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.service_category (
    name text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT service_category_name_check CHECK ((name <> ''::text))
);


ALTER TABLE public.service_category OWNER TO traffic_ops;

--
-- Name: snapshot; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.snapshot (
    cdn text NOT NULL,
    crconfig json NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    monitoring json NOT NULL
);


ALTER TABLE public.snapshot OWNER TO traffic_ops;

--
-- Name: staticdnsentry; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.staticdnsentry (
    id bigint NOT NULL,
    host text NOT NULL,
    address text NOT NULL,
    type bigint NOT NULL,
    ttl bigint DEFAULT '3600'::bigint NOT NULL,
    deliveryservice bigint NOT NULL,
    cachegroup bigint,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.staticdnsentry OWNER TO traffic_ops;

--
-- Name: staticdnsentry_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.staticdnsentry_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.staticdnsentry_id_seq OWNER TO traffic_ops;

--
-- Name: staticdnsentry_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.staticdnsentry_id_seq OWNED BY public.staticdnsentry.id;


--
-- Name: stats_summary; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.stats_summary (
    id bigint NOT NULL,
    cdn_name text DEFAULT 'all'::text NOT NULL,
    deliveryservice_name text NOT NULL,
    stat_name text NOT NULL,
    stat_value double precision NOT NULL,
    summary_time timestamp with time zone DEFAULT now() NOT NULL,
    stat_date date
);


ALTER TABLE public.stats_summary OWNER TO traffic_ops;

--
-- Name: stats_summary_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.stats_summary_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.stats_summary_id_seq OWNER TO traffic_ops;

--
-- Name: stats_summary_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.stats_summary_id_seq OWNED BY public.stats_summary.id;


--
-- Name: status; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.status (
    id bigint NOT NULL,
    name text NOT NULL,
    description text,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.status OWNER TO traffic_ops;

--
-- Name: status_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.status_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.status_id_seq OWNER TO traffic_ops;

--
-- Name: status_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.status_id_seq OWNED BY public.status.id;


--
-- Name: steering_target; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.steering_target (
    deliveryservice bigint NOT NULL,
    target bigint NOT NULL,
    value bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    type bigint NOT NULL
);


ALTER TABLE public.steering_target OWNER TO traffic_ops;

--
-- Name: tenant; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.tenant (
    id bigint NOT NULL,
    name text NOT NULL,
    active boolean DEFAULT false NOT NULL,
    parent_id bigint DEFAULT 1,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT tenant_check CHECK ((id <> parent_id))
);


ALTER TABLE public.tenant OWNER TO traffic_ops;

--
-- Name: tenant_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.tenant_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.tenant_id_seq OWNER TO traffic_ops;

--
-- Name: tenant_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.tenant_id_seq OWNED BY public.tenant.id;


--
-- Name: tm_user; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.tm_user (
    id bigint NOT NULL,
    username text,
    public_ssh_key text,
    role bigint,
    uid bigint,
    gid bigint,
    local_passwd text,
    confirm_local_passwd text,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    company text,
    email text,
    full_name text,
    new_user boolean DEFAULT false NOT NULL,
    address_line1 text,
    address_line2 text,
    city text,
    state_or_province text,
    phone_number text,
    postal_code text,
    country text,
    token text,
    registration_sent timestamp with time zone,
    tenant_id bigint NOT NULL
);


ALTER TABLE public.tm_user OWNER TO traffic_ops;

--
-- Name: tm_user_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.tm_user_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.tm_user_id_seq OWNER TO traffic_ops;

--
-- Name: tm_user_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.tm_user_id_seq OWNED BY public.tm_user.id;


--
-- Name: to_extension; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.to_extension (
    id bigint NOT NULL,
    name text NOT NULL,
    version text NOT NULL,
    info_url text NOT NULL,
    script_file text NOT NULL,
    isactive boolean DEFAULT false NOT NULL,
    additional_config_json text,
    description text,
    servercheck_short_name text,
    servercheck_column_name text,
    type bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.to_extension OWNER TO traffic_ops;

--
-- Name: to_extension_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.to_extension_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.to_extension_id_seq OWNER TO traffic_ops;

--
-- Name: to_extension_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.to_extension_id_seq OWNED BY public.to_extension.id;


--
-- Name: topology; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.topology (
    name text NOT NULL,
    description text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.topology OWNER TO traffic_ops;

--
-- Name: topology_cachegroup; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.topology_cachegroup (
    id bigint NOT NULL,
    topology text NOT NULL,
    cachegroup text NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.topology_cachegroup OWNER TO traffic_ops;

--
-- Name: topology_cachegroup_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.topology_cachegroup_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.topology_cachegroup_id_seq OWNER TO traffic_ops;

--
-- Name: topology_cachegroup_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.topology_cachegroup_id_seq OWNED BY public.topology_cachegroup.id;


--
-- Name: topology_cachegroup_parents; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.topology_cachegroup_parents (
    child bigint NOT NULL,
    parent bigint NOT NULL,
    rank integer NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT topology_cachegroup_parents_rank_check CHECK (((rank = 1) OR (rank = 2)))
);


ALTER TABLE public.topology_cachegroup_parents OWNER TO traffic_ops;

--
-- Name: type; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.type (
    id bigint NOT NULL,
    name text NOT NULL,
    description text,
    use_in_table text,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.type OWNER TO traffic_ops;

--
-- Name: type_id_seq; Type: SEQUENCE; Schema: public; Owner: traffic_ops
--

CREATE SEQUENCE IF NOT EXISTS public.type_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.type_id_seq OWNER TO traffic_ops;

--
-- Name: type_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: traffic_ops
--

ALTER SEQUENCE public.type_id_seq OWNED BY public.type.id;


--
-- Name: user_role; Type: TABLE; Schema: public; Owner: traffic_ops
--

CREATE TABLE IF NOT EXISTS public.user_role (
    user_id bigint NOT NULL,
    role_id bigint NOT NULL,
    last_updated timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.user_role OWNER TO traffic_ops;

--
-- Name: api_capability id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.api_capability ALTER COLUMN id SET DEFAULT nextval('public.api_capability_id_seq'::regclass);


--
-- Name: asn id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.asn ALTER COLUMN id SET DEFAULT nextval('public.asn_id_seq'::regclass);


--
-- Name: cachegroup id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.cachegroup ALTER COLUMN id SET DEFAULT nextval('public.cachegroup_id_seq'::regclass);


--
-- Name: cdn id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.cdn ALTER COLUMN id SET DEFAULT nextval('public.cdn_id_seq'::regclass);


--
-- Name: coordinate id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.coordinate ALTER COLUMN id SET DEFAULT nextval('public.coordinate_id_seq'::regclass);


--
-- Name: deliveryservice id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.deliveryservice ALTER COLUMN id SET DEFAULT nextval('public.deliveryservice_id_seq'::regclass);


--
-- Name: deliveryservice_request id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.deliveryservice_request ALTER COLUMN id SET DEFAULT nextval('public.deliveryservice_request_id_seq'::regclass);


--
-- Name: deliveryservice_request_comment id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.deliveryservice_request_comment ALTER COLUMN id SET DEFAULT nextval('public.deliveryservice_request_comment_id_seq'::regclass);


--
-- Name: division id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.division ALTER COLUMN id SET DEFAULT nextval('public.division_id_seq'::regclass);


--
-- Name: federation id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.federation ALTER COLUMN id SET DEFAULT nextval('public.federation_id_seq'::regclass);


--
-- Name: federation_resolver id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.federation_resolver ALTER COLUMN id SET DEFAULT nextval('public.federation_resolver_id_seq'::regclass);


--
-- Name: goose_db_version id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.goose_db_version ALTER COLUMN id SET DEFAULT nextval('public.goose_db_version_id_seq'::regclass);


--
-- Name: hwinfo id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.hwinfo ALTER COLUMN id SET DEFAULT nextval('public.hwinfo_id_seq'::regclass);


--
-- Name: job id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.job ALTER COLUMN id SET DEFAULT nextval('public.job_id_seq'::regclass);


--
-- Name: job_agent id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.job_agent ALTER COLUMN id SET DEFAULT nextval('public.job_agent_id_seq'::regclass);


--
-- Name: job_status id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.job_status ALTER COLUMN id SET DEFAULT nextval('public.job_status_id_seq'::regclass);


--
-- Name: log id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.log ALTER COLUMN id SET DEFAULT nextval('public.log_id_seq'::regclass);


--
-- Name: origin id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.origin ALTER COLUMN id SET DEFAULT nextval('public.origin_id_seq'::regclass);


--
-- Name: parameter id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.parameter ALTER COLUMN id SET DEFAULT nextval('public.parameter_id_seq'::regclass);


--
-- Name: phys_location id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.phys_location ALTER COLUMN id SET DEFAULT nextval('public.phys_location_id_seq'::regclass);


--
-- Name: profile id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.profile ALTER COLUMN id SET DEFAULT nextval('public.profile_id_seq'::regclass);


--
-- Name: regex id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.regex ALTER COLUMN id SET DEFAULT nextval('public.regex_id_seq'::regclass);


--
-- Name: region id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.region ALTER COLUMN id SET DEFAULT nextval('public.region_id_seq'::regclass);


--
-- Name: role id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.role ALTER COLUMN id SET DEFAULT nextval('public.role_id_seq'::regclass);


--
-- Name: server id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.server ALTER COLUMN id SET DEFAULT nextval('public.server_id_seq'::regclass);


--
-- Name: servercheck id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.servercheck ALTER COLUMN id SET DEFAULT nextval('public.servercheck_id_seq'::regclass);


--
-- Name: staticdnsentry id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.staticdnsentry ALTER COLUMN id SET DEFAULT nextval('public.staticdnsentry_id_seq'::regclass);


--
-- Name: stats_summary id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.stats_summary ALTER COLUMN id SET DEFAULT nextval('public.stats_summary_id_seq'::regclass);


--
-- Name: status id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.status ALTER COLUMN id SET DEFAULT nextval('public.status_id_seq'::regclass);


--
-- Name: tenant id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.tenant ALTER COLUMN id SET DEFAULT nextval('public.tenant_id_seq'::regclass);


--
-- Name: tm_user id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.tm_user ALTER COLUMN id SET DEFAULT nextval('public.tm_user_id_seq'::regclass);


--
-- Name: to_extension id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.to_extension ALTER COLUMN id SET DEFAULT nextval('public.to_extension_id_seq'::regclass);


--
-- Name: topology_cachegroup id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.topology_cachegroup ALTER COLUMN id SET DEFAULT nextval('public.topology_cachegroup_id_seq'::regclass);


--
-- Name: type id; Type: DEFAULT; Schema: public; Owner: traffic_ops
--

ALTER TABLE ONLY public.type ALTER COLUMN id SET DEFAULT nextval('public.type_id_seq'::regclass);

DO $$
BEGIN
IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'api_capability_http_method_route_capability_key' AND table_name = 'api_capability') THEN
    --
    -- Name: api_capability api_capability_http_method_route_capability_key; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.api_capability
        ADD CONSTRAINT api_capability_http_method_route_capability_key UNIQUE (http_method, route, capability);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'api_capability_pkey' AND table_name = 'api_capability') THEN
    --
    -- Name: api_capability api_capability_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.api_capability
        ADD CONSTRAINT api_capability_pkey PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'cachegroup_fallbacks_primary_cg_backup_cg_key' AND table_name = 'cachegroup_fallbacks') THEN
    --
    -- Name: cachegroup_fallbacks cachegroup_fallbacks_primary_cg_backup_cg_key; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup_fallbacks
        ADD CONSTRAINT cachegroup_fallbacks_primary_cg_backup_cg_key UNIQUE (primary_cg, backup_cg);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'cachegroup_fallbacks_primary_cg_set_order_key' AND table_name = 'cachegroup_fallbacks') THEN
    --
    -- Name: cachegroup_fallbacks cachegroup_fallbacks_primary_cg_set_order_key; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup_fallbacks
        ADD CONSTRAINT cachegroup_fallbacks_primary_cg_set_order_key UNIQUE (primary_cg, set_order);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'cachegroup_localization_method_cachegroup_method_key' AND table_name = 'cachegroup_localization_method') THEN
    --
    -- Name: cachegroup_localization_method cachegroup_localization_method_cachegroup_method_key; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup_localization_method
        ADD CONSTRAINT cachegroup_localization_method_cachegroup_method_key UNIQUE (cachegroup, method);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'capability_pkey' AND table_name = 'capability') THEN
    --
    -- Name: capability capability_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.capability
        ADD CONSTRAINT capability_pkey PRIMARY KEY (name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'cdn_domain_name_unique' AND table_name = 'cdn') THEN
    --
    -- Name: cdn cdn_domain_name_unique; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cdn
        ADD CONSTRAINT cdn_domain_name_unique UNIQUE (domain_name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'coordinate_name_key' AND table_name = 'coordinate') THEN
    --
    -- Name: coordinate coordinate_name_key; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.coordinate
        ADD CONSTRAINT coordinate_name_key UNIQUE (name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'coordinate_pkey' AND table_name = 'coordinate') THEN
    --
    -- Name: coordinate coordinate_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.coordinate
        ADD CONSTRAINT coordinate_pkey PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'deliveryservice_consistent_hash_query_param_pkey' AND table_name = 'deliveryservice_consistent_hash_query_param') THEN
    --
    -- Name: deliveryservice_consistent_hash_query_param deliveryservice_consistent_hash_query_param_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_consistent_hash_query_param
        ADD CONSTRAINT deliveryservice_consistent_hash_query_param_pkey PRIMARY KEY (name, deliveryservice_id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'deliveryservice_request_comment_pkey' AND table_name = 'deliveryservice_request_comment') THEN
    --
    -- Name: deliveryservice_request_comment deliveryservice_request_comment_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_request_comment
        ADD CONSTRAINT deliveryservice_request_comment_pkey PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'deliveryservice_request_pkey' AND table_name = 'deliveryservice_request') THEN
    --
    -- Name: deliveryservice_request deliveryservice_request_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_request
        ADD CONSTRAINT deliveryservice_request_pkey PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'deliveryservices_required_capability_pkey' AND table_name = 'deliveryservices_required_capability') THEN
    --
    -- Name: deliveryservices_required_capability deliveryservices_required_capability_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservices_required_capability
        ADD CONSTRAINT deliveryservices_required_capability_pkey PRIMARY KEY (deliveryservice_id, required_capability);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'goose_db_version_pkey' AND table_name = 'goose_db_version') THEN
    --
    -- Name: goose_db_version goose_db_version_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.goose_db_version
        ADD CONSTRAINT goose_db_version_pkey PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89468_primary' AND table_name = 'asn') THEN
    --
    -- Name: asn idx_89468_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.asn
        ADD CONSTRAINT idx_89468_primary PRIMARY KEY (id, cachegroup);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89476_primary' AND table_name = 'cachegroup') THEN
    --
    -- Name: cachegroup idx_89476_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup
        ADD CONSTRAINT idx_89476_primary PRIMARY KEY (id, type);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89484_primary' AND table_name = 'cachegroup_parameter') THEN
    --
    -- Name: cachegroup_parameter idx_89484_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup_parameter
        ADD CONSTRAINT idx_89484_primary PRIMARY KEY (cachegroup, parameter);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89491_primary' AND table_name = 'cdn') THEN
    --
    -- Name: cdn idx_89491_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cdn
        ADD CONSTRAINT idx_89491_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89502_primary' AND table_name = 'deliveryservice') THEN
    --
    -- Name: deliveryservice idx_89502_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice
        ADD CONSTRAINT idx_89502_primary PRIMARY KEY (id, type);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89517_primary' AND table_name = 'deliveryservice_regex') THEN
    --
    -- Name: deliveryservice_regex idx_89517_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_regex
        ADD CONSTRAINT idx_89517_primary PRIMARY KEY (deliveryservice, regex);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89521_primary' AND table_name = 'deliveryservice_server') THEN
    --
    -- Name: deliveryservice_server idx_89521_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_server
        ADD CONSTRAINT idx_89521_primary PRIMARY KEY (deliveryservice, server);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89525_primary' AND table_name = 'deliveryservice_tmuser') THEN
    --
    -- Name: deliveryservice_tmuser idx_89525_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_tmuser
        ADD CONSTRAINT idx_89525_primary PRIMARY KEY (deliveryservice, tm_user_id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89531_primary' AND table_name = 'division') THEN
    --
    -- Name: division idx_89531_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.division
        ADD CONSTRAINT idx_89531_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89541_primary' AND table_name = 'federation') THEN
    --
    -- Name: federation idx_89541_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation
        ADD CONSTRAINT idx_89541_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89549_primary' AND table_name = 'federation_deliveryservice') THEN
    --
    -- Name: federation_deliveryservice idx_89549_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_deliveryservice
        ADD CONSTRAINT idx_89549_primary PRIMARY KEY (federation, deliveryservice);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89553_primary' AND table_name = 'federation_federation_resolver') THEN
    --
    -- Name: federation_federation_resolver idx_89553_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_federation_resolver
        ADD CONSTRAINT idx_89553_primary PRIMARY KEY (federation, federation_resolver);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89559_primary' AND table_name = 'federation_resolver') THEN
    --
    -- Name: federation_resolver idx_89559_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_resolver
        ADD CONSTRAINT idx_89559_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89567_primary' AND table_name = 'federation_tmuser') THEN
    --
    -- Name: federation_tmuser idx_89567_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_tmuser
        ADD CONSTRAINT idx_89567_primary PRIMARY KEY (federation, tm_user);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89583_primary' AND table_name = 'hwinfo') THEN
    --
    -- Name: hwinfo idx_89583_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.hwinfo
        ADD CONSTRAINT idx_89583_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89593_primary' AND table_name = 'job') THEN
    --
    -- Name: job idx_89593_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.job
        ADD CONSTRAINT idx_89593_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89603_primary' AND table_name = 'job_agent') THEN
    --
    -- Name: job_agent idx_89603_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.job_agent
        ADD CONSTRAINT idx_89603_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89624_primary' AND table_name = 'job_status') THEN
    --
    -- Name: job_status idx_89624_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.job_status
        ADD CONSTRAINT idx_89624_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89634_primary' AND table_name = 'log') THEN
    --
    -- Name: log idx_89634_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.log
        ADD CONSTRAINT idx_89634_primary PRIMARY KEY (id, tm_user);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89644_primary' AND table_name = 'parameter') THEN
    --
    -- Name: parameter idx_89644_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.parameter
        ADD CONSTRAINT idx_89644_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89655_primary' AND table_name = 'phys_location') THEN
    --
    -- Name: phys_location idx_89655_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.phys_location
        ADD CONSTRAINT idx_89655_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89665_primary' AND table_name = 'profile') THEN
    --
    -- Name: profile idx_89665_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.profile
        ADD CONSTRAINT idx_89665_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89673_primary' AND table_name = 'profile_parameter') THEN
    --
    -- Name: profile_parameter idx_89673_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.profile_parameter
        ADD CONSTRAINT idx_89673_primary PRIMARY KEY (profile, parameter);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89679_primary' AND table_name = 'regex') THEN
    --
    -- Name: regex idx_89679_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.regex
        ADD CONSTRAINT idx_89679_primary PRIMARY KEY (id, type);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89690_primary' AND table_name = 'region') THEN
    --
    -- Name: region idx_89690_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.region
        ADD CONSTRAINT idx_89690_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89700_primary' AND table_name = 'role') THEN
    --
    -- Name: role idx_89700_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.role
        ADD CONSTRAINT idx_89700_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89709_primary' AND table_name = 'server') THEN
    --
    -- Name: server idx_89709_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server
        ADD CONSTRAINT idx_89709_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89722_primary' AND table_name = 'servercheck') THEN
    --
    -- Name: servercheck idx_89722_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.servercheck
        ADD CONSTRAINT idx_89722_primary PRIMARY KEY (id, server);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89729_primary' AND table_name = 'staticdnsentry') THEN
    --
    -- Name: staticdnsentry idx_89729_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.staticdnsentry
        ADD CONSTRAINT idx_89729_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89740_primary' AND table_name = 'stats_summary') THEN
    --
    -- Name: stats_summary idx_89740_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.stats_summary
        ADD CONSTRAINT idx_89740_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89751_primary' AND table_name = 'status') THEN
    --
    -- Name: status idx_89751_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.status
        ADD CONSTRAINT idx_89751_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89759_primary' AND table_name = 'steering_target') THEN
    --
    -- Name: steering_target idx_89759_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.steering_target
        ADD CONSTRAINT idx_89759_primary PRIMARY KEY (deliveryservice, target);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89765_primary' AND table_name = 'tm_user') THEN
    --
    -- Name: tm_user idx_89765_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.tm_user
        ADD CONSTRAINT idx_89765_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89776_primary' AND table_name = 'to_extension') THEN
    --
    -- Name: to_extension idx_89776_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.to_extension
        ADD CONSTRAINT idx_89776_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'idx_89786_primary' AND table_name = 'type') THEN
    --
    -- Name: type idx_89786_primary; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.type
        ADD CONSTRAINT idx_89786_primary PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'interface_pkey' AND table_name = 'interface') THEN
    --
    -- Name: interface interface_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.interface
        ADD CONSTRAINT interface_pkey PRIMARY KEY (name, server);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'ip_address_pkey' AND table_name = 'ip_address') THEN
    --
    -- Name: ip_address ip_address_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.ip_address
        ADD CONSTRAINT ip_address_pkey PRIMARY KEY (address, interface, server);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'job_agent_name_unique' AND table_name = 'job_agent') THEN
    --
    -- Name: job_agent job_agent_name_unique; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.job_agent
        ADD CONSTRAINT job_agent_name_unique UNIQUE (name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'job_status_name_unique' AND table_name = 'job_status') THEN
    --
    -- Name: job_status job_status_name_unique; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.job_status
        ADD CONSTRAINT job_status_name_unique UNIQUE (name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'last_deleted_pkey' AND table_name = 'last_deleted') THEN
    --
    -- Name: last_deleted last_deleted_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.last_deleted
        ADD CONSTRAINT last_deleted_pkey PRIMARY KEY (table_name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'lets_encrypt_account_pkey' AND table_name = 'lets_encrypt_account') THEN
    --
    -- Name: lets_encrypt_account lets_encrypt_account_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.lets_encrypt_account
        ADD CONSTRAINT lets_encrypt_account_pkey PRIMARY KEY (email);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'origin_name_key' AND table_name = 'origin') THEN
    --
    -- Name: origin origin_name_key; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.origin
        ADD CONSTRAINT origin_name_key UNIQUE (name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'origin_pkey' AND table_name = 'origin') THEN
    --
    -- Name: origin origin_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.origin
        ADD CONSTRAINT origin_pkey PRIMARY KEY (id);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'role_capability_role_id_cap_name_key' AND table_name = 'role_capability') THEN
    --
    -- Name: role_capability role_capability_role_id_cap_name_key; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.role_capability
        ADD CONSTRAINT role_capability_role_id_cap_name_key UNIQUE (role_id, cap_name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'role_name_unique' AND table_name = 'role') THEN
    --
    -- Name: role role_name_unique; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.role
        ADD CONSTRAINT role_name_unique UNIQUE (name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'server_capability_pkey' AND table_name = 'server_capability') THEN
    --
    -- Name: server_capability server_capability_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server_capability
        ADD CONSTRAINT server_capability_pkey PRIMARY KEY (name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'server_server_capability_pkey' AND table_name = 'server_server_capability') THEN
    --
    -- Name: server_server_capability server_server_capability_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server_server_capability
        ADD CONSTRAINT server_server_capability_pkey PRIMARY KEY (server, server_capability);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'service_category_pkey' AND table_name = 'service_category') THEN
    --
    -- Name: service_category service_category_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.service_category
        ADD CONSTRAINT service_category_pkey PRIMARY KEY (name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'snapshot_pkey' AND table_name = 'snapshot') THEN
    --
    -- Name: snapshot snapshot_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.snapshot
        ADD CONSTRAINT snapshot_pkey PRIMARY KEY (cdn);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'status_name_unique' AND table_name = 'status') THEN
    --
    -- Name: status status_name_unique; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.status
        ADD CONSTRAINT status_name_unique UNIQUE (name);
END IF;

IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'tenant_name_key' AND table_name = 'tenant') THEN
    --
    -- Name: tenant tenant_name_key; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.tenant
        ADD CONSTRAINT tenant_name_key UNIQUE (name);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'tenant_pkey' AND table_name = 'tenant') THEN
    --
    -- Name: tenant tenant_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.tenant
        ADD CONSTRAINT tenant_pkey PRIMARY KEY (id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'topology_cachegroup_pkey' AND table_name = 'topology_cachegroup') THEN
    --
    -- Name: topology_cachegroup topology_cachegroup_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.topology_cachegroup
        ADD CONSTRAINT topology_cachegroup_pkey PRIMARY KEY (id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'topology_pkey' AND table_name = 'topology') THEN
    --
    -- Name: topology topology_pkey; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.topology
        ADD CONSTRAINT topology_pkey PRIMARY KEY (name);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'type_name_unique' AND table_name = 'type') THEN
    --
    -- Name: type type_name_unique; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.type
        ADD CONSTRAINT type_name_unique UNIQUE (name);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'unique_child_parent' AND table_name = 'topology_cachegroup_parents') THEN
    --
    -- Name: topology_cachegroup_parents unique_child_parent; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.topology_cachegroup_parents
        ADD CONSTRAINT unique_child_parent UNIQUE (child, parent);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'unique_child_rank' AND table_name = 'topology_cachegroup_parents') THEN
    --
    -- Name: topology_cachegroup_parents unique_child_rank; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.topology_cachegroup_parents
        ADD CONSTRAINT unique_child_rank UNIQUE (child, rank);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'unique_param' AND table_name = 'parameter') THEN
    --
    -- Name: parameter unique_param; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.parameter
        ADD CONSTRAINT unique_param UNIQUE (name, config_file, value);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.constraint_column_usage WHERE constraint_name = 'unique_topology_cachegroup' AND table_name = 'topology_cachegroup') THEN
    --
    -- Name: topology_cachegroup unique_topology_cachegroup; Type: CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.topology_cachegroup
        ADD CONSTRAINT unique_topology_cachegroup UNIQUE (topology, cachegroup);
END IF;
END $$;

DO $$ BEGIN
IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'api_capability' AND column_name = 'last_updated') THEN
    --
    -- Name: api_capability_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS api_capability_last_updated_idx ON public.api_capability USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'asn' AND column_name = 'last_updated') THEN
    --
    -- Name: asn_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS asn_last_updated_idx ON public.asn USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup' AND column_name = 'coordinate') THEN
    --
    -- Name: cachegroup_coordinate_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS cachegroup_coordinate_fkey ON public.cachegroup USING btree (coordinate);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup' AND column_name = 'last_updated') THEN
    --
    -- Name: cachegroup_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS cachegroup_last_updated_idx ON public.cachegroup USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup_localization_method' AND column_name = 'cachegroup') THEN
    --
    -- Name: cachegroup_localization_method_cachegroup_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS cachegroup_localization_method_cachegroup_fkey ON public.cachegroup_localization_method USING btree (cachegroup);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup_parameter' AND column_name = 'last_updated') THEN
    --
    -- Name: cachegroup_parameter_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS cachegroup_parameter_last_updated_idx ON public.cachegroup_parameter USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'capability' AND column_name = 'last_updated') THEN
    --
    -- Name: capability_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS capability_last_updated_idx ON public.capability USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cdn' AND column_name = 'last_updated') THEN
    --
    -- Name: cdn_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS cdn_last_updated_idx ON public.cdn USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'coordinate' AND column_name = 'last_updated') THEN
    --
    -- Name: coordinate_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS coordinate_last_updated_idx ON public.coordinate USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice' AND column_name = 'last_updated') THEN
    --
    -- Name: deliveryservice_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS deliveryservice_last_updated_idx ON public.deliveryservice USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice_regex' AND column_name = 'last_updated') THEN
    --
    -- Name: deliveryservice_regex_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS deliveryservice_regex_last_updated_idx ON public.deliveryservice_regex USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice_request_comment' AND column_name = 'last_updated') THEN
    --
    -- Name: deliveryservice_request_comment_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS deliveryservice_request_comment_last_updated_idx ON public.deliveryservice_request_comment USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice_request' AND column_name = 'last_updated') THEN
    --
    -- Name: deliveryservice_request_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS deliveryservice_request_last_updated_idx ON public.deliveryservice_request USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice_server' AND column_name = 'last_updated') THEN
    --
    -- Name: deliveryservice_server_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS deliveryservice_server_last_updated_idx ON public.deliveryservice_server USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice_tmuser' AND column_name = 'last_updated') THEN
    --
    -- Name: deliveryservice_tmuser_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS deliveryservice_tmuser_last_updated_idx ON public.deliveryservice_tmuser USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice' AND column_name = 'topology') THEN
    --
    -- Name: deliveryservice_topology_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS deliveryservice_topology_fkey ON public.deliveryservice USING btree (topology);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservices_required_capability' AND column_name = 'last_updated') THEN
    --
    -- Name: deliveryservices_required_capability_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS deliveryservices_required_capability_last_updated_idx ON public.deliveryservices_required_capability USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'division' AND column_name = 'last_updated') THEN
    --
    -- Name: division_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS division_last_updated_idx ON public.division USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_deliveryservice' AND column_name = 'last_updated') THEN
    --
    -- Name: federation_deliveryservice_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS federation_deliveryservice_last_updated_idx ON public.federation_deliveryservice USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_federation_resolver' AND column_name = 'last_updated') THEN
    --
    -- Name: federation_federation_resolver_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS federation_federation_resolver_last_updated_idx ON public.federation_federation_resolver USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation' AND column_name = 'last_updated') THEN
    --
    -- Name: federation_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS federation_last_updated_idx ON public.federation USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_resolver' AND column_name = 'last_updated') THEN
    --
    -- Name: federation_resolver_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS federation_resolver_last_updated_idx ON public.federation_resolver USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_tmuser' AND column_name = 'last_updated') THEN
    --
    -- Name: federation_tmuser_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS federation_tmuser_last_updated_idx ON public.federation_tmuser USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'hwinfo' AND column_name = 'last_updated') THEN
    --
    -- Name: hwinfo_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS hwinfo_last_updated_idx ON public.hwinfo USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'profile' AND column_name = 'cdn') THEN
    --
    -- Name: idx_181818_fk_cdn1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_181818_fk_cdn1 ON public.profile USING btree (cdn);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'asn' AND column_name = 'id') THEN
    --
    -- Name: idx_89468_cr_id_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89468_cr_id_unique ON public.asn USING btree (id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'asn' AND column_name = 'cachegroup') THEN
    --
    -- Name: idx_89468_fk_cran_cachegroup1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89468_fk_cran_cachegroup1 ON public.asn USING btree (cachegroup);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup' AND column_name = 'name') THEN
    --
    -- Name: idx_89476_cg_name_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89476_cg_name_unique ON public.cachegroup USING btree (name);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup' AND column_name = 'short_name') THEN
    --
    -- Name: idx_89476_cg_short_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89476_cg_short_unique ON public.cachegroup USING btree (short_name);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup' AND column_name = 'parent_cachegroup_id') THEN
    --
    -- Name: idx_89476_fk_cg_1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89476_fk_cg_1 ON public.cachegroup USING btree (parent_cachegroup_id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup' AND column_name = 'secondary_parent_cachegroup_id') THEN
    --
    -- Name: idx_89476_fk_cg_secondary; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89476_fk_cg_secondary ON public.cachegroup USING btree (secondary_parent_cachegroup_id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup' AND column_name = 'type') THEN
    --
    -- Name: idx_89476_fk_cg_type1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89476_fk_cg_type1 ON public.cachegroup USING btree (type);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup' AND column_name = 'id') THEN
    --
    -- Name: idx_89476_lo_id_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89476_lo_id_unique ON public.cachegroup USING btree (id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cachegroup_parameter' AND column_name = 'parameter') THEN
    --
    -- Name: idx_89484_fk_parameter; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89484_fk_parameter ON public.cachegroup_parameter USING btree (parameter);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'cdn' AND column_name = 'name') THEN
    --
    -- Name: idx_89491_cdn_cdn_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89491_cdn_cdn_unique ON public.cdn USING btree (name);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice' AND column_name = 'id') THEN
    --
    -- Name: idx_89502_ds_id_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89502_ds_id_unique ON public.deliveryservice USING btree (id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice' AND column_name = 'xml_id') THEN
    --
    -- Name: idx_89502_ds_name_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89502_ds_name_unique ON public.deliveryservice USING btree (xml_id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice' AND column_name = 'cdn_id') THEN
    --
    -- Name: idx_89502_fk_cdn1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89502_fk_cdn1 ON public.deliveryservice USING btree (cdn_id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice' AND column_name = 'profile') THEN
    --
    -- Name: idx_89502_fk_deliveryservice_profile1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89502_fk_deliveryservice_profile1 ON public.deliveryservice USING btree (profile);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice' AND column_name = 'type') THEN
    --
    -- Name: idx_89502_fk_deliveryservice_type1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89502_fk_deliveryservice_type1 ON public.deliveryservice USING btree (type);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice_regex' AND column_name = 'regex') THEN
    --
    -- Name: idx_89517_fk_ds_to_regex_regex1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89517_fk_ds_to_regex_regex1 ON public.deliveryservice_regex USING btree (regex);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice_server' AND column_name = 'server') THEN
    --
    -- Name: idx_89521_fk_ds_to_cs_contentserver1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89521_fk_ds_to_cs_contentserver1 ON public.deliveryservice_server USING btree (server);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice_tmuser' AND column_name = 'tm_user_id') THEN
    --
    -- Name: idx_89525_fk_tm_userid; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89525_fk_tm_userid ON public.deliveryservice_tmuser USING btree (tm_user_id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'division' AND column_name = 'name') THEN
    --
    -- Name: idx_89531_name_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89531_name_unique ON public.division USING btree (name);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_deliveryservice' AND column_name = 'deliveryservice') THEN
    --
    -- Name: idx_89549_fk_fed_to_ds1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89549_fk_fed_to_ds1 ON public.federation_deliveryservice USING btree (deliveryservice);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_federation_resolver' AND column_name = 'federation') THEN
    --
    -- Name: idx_89553_fk_federation_federation_resolver; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89553_fk_federation_federation_resolver ON public.federation_federation_resolver USING btree (federation);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_federation_resolver' AND column_name = 'federation_resolver') THEN
    --
    -- Name: idx_89553_fk_federation_resolver_to_fed1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89553_fk_federation_resolver_to_fed1 ON public.federation_federation_resolver USING btree (federation_resolver);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_resolver' AND column_name = 'ip_address') THEN
    --
    -- Name: idx_89559_federation_resolver_ip_address; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89559_federation_resolver_ip_address ON public.federation_resolver USING btree (ip_address);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_resolver' AND column_name = 'type') THEN
    --
    -- Name: idx_89559_fk_federation_mapping_type; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89559_fk_federation_mapping_type ON public.federation_resolver USING btree (type);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_tmuser' AND column_name = 'federation') THEN
    --
    -- Name: idx_89567_fk_federation_federation_resolver; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89567_fk_federation_federation_resolver ON public.federation_tmuser USING btree (federation);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_tmuser' AND column_name = 'role') THEN
    --
    -- Name: idx_89567_fk_federation_tmuser_role; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89567_fk_federation_tmuser_role ON public.federation_tmuser USING btree (role);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'federation_tmuser' AND column_name = 'tm_user') THEN
    --
    -- Name: idx_89567_fk_federation_tmuser_tmuser; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89567_fk_federation_tmuser_tmuser ON public.federation_tmuser USING btree (tm_user);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'hwinfo' AND column_name = 'serverid') THEN
    --
    -- Name: idx_89583_fk_hwinfo1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89583_fk_hwinfo1 ON public.hwinfo USING btree (serverid);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'hwinfo' AND column_name = 'serverid') AND EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'hwinfo' AND column_name = 'description') THEN
    --
    -- Name: idx_89583_serverid; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89583_serverid ON public.hwinfo USING btree (serverid, description);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'job' AND column_name = 'agent') THEN
    --
    -- Name: idx_89593_fk_job_agent_id1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89593_fk_job_agent_id1 ON public.job USING btree (agent);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'job' AND column_name = 'job_deliveryservice') THEN
    --
    -- Name: idx_89593_fk_job_deliveryservice1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89593_fk_job_deliveryservice1 ON public.job USING btree (job_deliveryservice);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'job' AND column_name = 'status') THEN
    --
    -- Name: idx_89593_fk_job_status_id1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89593_fk_job_status_id1 ON public.job USING btree (status);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'job' AND column_name = 'job_user') THEN
    --
    -- Name: idx_89593_fk_job_user_id1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89593_fk_job_user_id1 ON public.job USING btree (job_user);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'log' AND column_name = 'tm_user') THEN
    --
    -- Name: idx_89634_fk_log_1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89634_fk_log_1 ON public.log USING btree (tm_user);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'log' AND column_name = 'last_updated') THEN
    --
    -- Name: idx_89634_idx_last_updated; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89634_idx_last_updated ON public.log USING btree (last_updated);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'parameter' AND column_name = 'name') AND EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'parameter' AND column_name = 'value') THEN
    --
    -- Name: idx_89644_parameter_name_value_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89644_parameter_name_value_idx ON public.parameter USING btree (name, value);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'phys_location' AND column_name = 'region') THEN
    --
    -- Name: idx_89655_fk_phys_location_region_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89655_fk_phys_location_region_idx ON public.phys_location USING btree (region);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'phys_location' AND column_name = 'name') THEN
    --
    -- Name: idx_89655_name_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89655_name_unique ON public.phys_location USING btree (name);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'phys_location' AND column_name = 'short_name') THEN
    --
    -- Name: idx_89655_short_name_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89655_short_name_unique ON public.phys_location USING btree (short_name);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'profile' AND column_name = 'name') THEN
    --
    -- Name: idx_89665_name_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89665_name_unique ON public.profile USING btree (name);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'profile_parameter' AND column_name = 'parameter') THEN
    --
    -- Name: idx_89673_fk_atsprofile_atsparameters_atsparameters1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89673_fk_atsprofile_atsparameters_atsparameters1 ON public.profile_parameter USING btree (parameter);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'profile_parameter' AND column_name = 'profile') THEN
    --
    -- Name: idx_89673_fk_atsprofile_atsparameters_atsprofile1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89673_fk_atsprofile_atsparameters_atsprofile1 ON public.profile_parameter USING btree (profile);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'regex' AND column_name = 'type') THEN
    --
    -- Name: idx_89679_fk_regex_type1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89679_fk_regex_type1 ON public.regex USING btree (type);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'regex' AND column_name = 'id') THEN
    --
    -- Name: idx_89679_re_id_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89679_re_id_unique ON public.regex USING btree (id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'region' AND column_name = 'division') THEN
    --
    -- Name: idx_89690_fk_region_division1_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89690_fk_region_division1_idx ON public.region USING btree (division);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'region' AND column_name = 'name') THEN
    --
    -- Name: idx_89690_name_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89690_name_unique ON public.region USING btree (name);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'server' AND column_name = 'cdn_id') THEN
    --
    -- Name: idx_89709_fk_cdn2; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89709_fk_cdn2 ON public.server USING btree (cdn_id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'server' AND column_name = 'profile') THEN
    --
    -- Name: idx_89709_fk_contentserver_atsprofile1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89709_fk_contentserver_atsprofile1 ON public.server USING btree (profile);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'server' AND column_name = 'status') THEN
    --
    -- Name: idx_89709_fk_contentserver_contentserverstatus1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89709_fk_contentserver_contentserverstatus1 ON public.server USING btree (status);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'server' AND column_name = 'type') THEN
    --
    -- Name: idx_89709_fk_contentserver_contentservertype1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89709_fk_contentserver_contentservertype1 ON public.server USING btree (type);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'server' AND column_name = 'phys_location') THEN
    --
    -- Name: idx_89709_fk_contentserver_phys_location1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89709_fk_contentserver_phys_location1 ON public.server USING btree (phys_location);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'server' AND column_name = 'cachegroup') THEN
    --
    -- Name: idx_89709_fk_server_cachegroup1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89709_fk_server_cachegroup1 ON public.server USING btree (cachegroup);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'server' AND column_name = 'id') THEN
    --
    -- Name: idx_89709_se_id_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89709_se_id_unique ON public.server USING btree (id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'servercheck' AND column_name = 'server') THEN
    --
    -- Name: idx_89722_fk_serverstatus_server1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89722_fk_serverstatus_server1 ON public.servercheck USING btree (server);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'servercheck' AND column_name = 'server') THEN
    --
    -- Name: idx_89722_server; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89722_server ON public.servercheck USING btree (server);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'servercheck' AND column_name = 'id') THEN
    --
    -- Name: idx_89722_ses_id_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89722_ses_id_unique ON public.servercheck USING btree (id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'staticdnsentry' AND column_name = 'host')
    AND EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'staticdnsentry' AND column_name = 'address')
    AND EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'staticdnsentry' AND column_name = 'deliveryservice')
    AND EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'staticdnsentry' AND column_name = 'cachegroup') THEN
    --
    -- Name: idx_89729_combi_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89729_combi_unique ON public.staticdnsentry USING btree (host, address, deliveryservice, cachegroup);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'staticdnsentry' AND column_name = 'cachegroup') THEN
    --
    -- Name: idx_89729_fk_staticdnsentry_cachegroup1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89729_fk_staticdnsentry_cachegroup1 ON public.staticdnsentry USING btree (cachegroup);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'staticdnsentry' AND column_name = 'deliveryservice') THEN
    --
    -- Name: idx_89729_fk_staticdnsentry_ds; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89729_fk_staticdnsentry_ds ON public.staticdnsentry USING btree (deliveryservice);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'staticdnsentry' AND column_name = 'type') THEN
    --
    -- Name: idx_89729_fk_staticdnsentry_type; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89729_fk_staticdnsentry_type ON public.staticdnsentry USING btree (type);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'tm_user' AND column_name = 'role') THEN
    --
    -- Name: idx_89765_fk_user_1; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89765_fk_user_1 ON public.tm_user USING btree (role);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'tm_user' AND column_name = 'email') THEN
    --
    -- Name: idx_89765_tmuser_email_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89765_tmuser_email_unique ON public.tm_user USING btree (email);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'tm_user' AND column_name = 'username') THEN
    --
    -- Name: idx_89765_username_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89765_username_unique ON public.tm_user USING btree (username);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'to_extension' AND column_name = 'type') THEN
    --
    -- Name: idx_89776_fk_ext_type_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_89776_fk_ext_type_idx ON public.to_extension USING btree (type);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'to_extension' AND column_name = 'id') THEN
    --
    -- Name: idx_89776_id_unique; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS idx_89776_id_unique ON public.to_extension USING btree (id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'deliveryservice' AND column_name = 'tenant_id') THEN
    --
    -- Name: idx_k_deliveryservice_tenant_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_k_deliveryservice_tenant_idx ON public.deliveryservice USING btree (tenant_id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'tenant' AND column_name = 'parent_id') THEN
    --
    -- Name: idx_k_tenant_parent_tenant_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_k_tenant_parent_tenant_idx ON public.tenant USING btree (parent_id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'tm_user' AND column_name = 'tenant_id') THEN
    --
    -- Name: idx_k_tm_user_tenant_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS idx_k_tm_user_tenant_idx ON public.tm_user USING btree (tenant_id);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'job_agent' AND column_name = 'last_updated') THEN
    --
    -- Name: job_agent_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS job_agent_last_updated_idx ON public.job_agent USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'job' AND column_name = 'last_updated') THEN
    --
    -- Name: job_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS job_last_updated_idx ON public.job USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'job_status' AND column_name = 'last_updated') THEN
    --
    -- Name: job_status_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS job_status_last_updated_idx ON public.job_status USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'log' AND column_name = 'last_updated') THEN
    --
    -- Name: log_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS log_last_updated_idx ON public.log USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'origin' AND column_name = 'cachegroup') THEN
    --
    -- Name: origin_cachegroup_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS origin_cachegroup_fkey ON public.origin USING btree (cachegroup);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'origin' AND column_name = 'coordinate') THEN
    --
    -- Name: origin_coordinate_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS origin_coordinate_fkey ON public.origin USING btree (coordinate);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'origin' AND column_name = 'deliveryservice') THEN
    --
    -- Name: origin_deliveryservice_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS origin_deliveryservice_fkey ON public.origin USING btree (deliveryservice);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'origin' AND column_name = 'is_primary')
    AND EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'origin' AND column_name = 'deliveryservice') THEN
    --
    -- Name: origin_is_primary_deliveryservice_constraint; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE UNIQUE INDEX IF NOT EXISTS origin_is_primary_deliveryservice_constraint ON public.origin USING btree (is_primary, deliveryservice) WHERE is_primary;
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'origin' AND column_name = 'last_updated') THEN
    --
    -- Name: origin_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS origin_last_updated_idx ON public.origin USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'origin' AND column_name = 'profile') THEN
    --
    -- Name: origin_profile_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS origin_profile_fkey ON public.origin USING btree (profile);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'origin' AND column_name = 'tenant') THEN
    --
    -- Name: origin_tenant_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS origin_tenant_fkey ON public.origin USING btree (tenant);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'parameter' AND column_name = 'last_updated') THEN
    --
    -- Name: parameter_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS parameter_last_updated_idx ON public.parameter USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'profile' AND column_name = 'last_updated') THEN
    --
    -- Name: profile_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS profile_last_updated_idx ON public.profile USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'profile_parameter' AND column_name = 'last_updated') THEN
    --
    -- Name: profile_parameter_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS profile_parameter_last_updated_idx ON public.profile_parameter USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'phys_location' AND column_name = 'last_updated') THEN
    --
    -- Name: pys_location_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS pys_location_last_updated_idx ON public.phys_location USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'regex' AND column_name = 'last_updated') THEN
    --
    -- Name: regex_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS regex_last_updated_idx ON public.regex USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'region' AND column_name = 'last_updated') THEN
    --
    -- Name: region_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS region_last_updated_idx ON public.region USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'role_capability' AND column_name = 'last_updated') THEN
    --
    -- Name: role_capability_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS role_capability_last_updated_idx ON public.role_capability USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'role' AND column_name = 'last_updated') THEN
    --
    -- Name: role_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS role_last_updated_idx ON public.role USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'server_capability' AND column_name = 'last_updated') THEN
    --
    -- Name: server_capability_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS server_capability_last_updated_idx ON public.server_capability USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'server' AND column_name = 'last_updated') THEN
    --
    -- Name: server_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS server_last_updated_idx ON public.server USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'server_server_capability' AND column_name = 'last_updated') THEN
    --
    -- Name: server_server_capability_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS server_server_capability_last_updated_idx ON public.server_server_capability USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'servercheck' AND column_name = 'last_updated') THEN
    --
    -- Name: servercheck_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS servercheck_last_updated_idx ON public.servercheck USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'service_category' AND column_name = 'last_updated') THEN
    --
    -- Name: service_category_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS service_category_last_updated_idx ON public.service_category USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'snapshot' AND column_name = 'last_updated') THEN
    --
    -- Name: snapshot_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS snapshot_last_updated_idx ON public.snapshot USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'staticdnsentry' AND column_name = 'last_updated') THEN
    --
    -- Name: staticdnsentry_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS staticdnsentry_last_updated_idx ON public.staticdnsentry USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'status' AND column_name = 'last_updated') THEN
    --
    -- Name: status_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS status_last_updated_idx ON public.status USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'steering_target' AND column_name = 'last_updated') THEN
    --
    -- Name: steering_target_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS steering_target_last_updated_idx ON public.steering_target USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'tenant' AND column_name = 'last_updated') THEN
    --
    -- Name: tenant_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS tenant_last_updated_idx ON public.tenant USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'tm_user' AND column_name = 'last_updated') THEN
    --
    -- Name: tm_user_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS tm_user_last_updated_idx ON public.tm_user USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'to_extension' AND column_name = 'last_updated') THEN
    --
    -- Name: to_extension_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS to_extension_last_updated_idx ON public.to_extension USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'topology_cachegroup' AND column_name = 'cachegroup') THEN
    --
    -- Name: topology_cachegroup_cachegroup_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS topology_cachegroup_cachegroup_fkey ON public.topology_cachegroup USING btree (cachegroup);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'topology_cachegroup' AND column_name = 'last_updated') THEN
    --
    -- Name: topology_cachegroup_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS topology_cachegroup_last_updated_idx ON public.topology_cachegroup USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'topology_cachegroup_parents' AND column_name = 'child') THEN
    --
    -- Name: topology_cachegroup_parents_child_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS topology_cachegroup_parents_child_fkey ON public.topology_cachegroup_parents USING btree (child);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'topology_cachegroup_parents' AND column_name = 'last_updated') THEN
    --
    -- Name: topology_cachegroup_parents_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS topology_cachegroup_parents_last_updated_idx ON public.topology_cachegroup_parents USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'topology_cachegroup_parents' AND column_name = 'parent') THEN
    --
    -- Name: topology_cachegroup_parents_parents_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS topology_cachegroup_parents_parents_fkey ON public.topology_cachegroup_parents USING btree (parent);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'topology_cachegroup' AND column_name = 'topology') THEN
    --
    -- Name: topology_cachegroup_topology_fkey; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS topology_cachegroup_topology_fkey ON public.topology_cachegroup USING btree (topology);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'topology_cachegroup' AND column_name = 'last_updated') THEN
    --
    -- Name: topology_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS topology_last_updated_idx ON public.topology_cachegroup USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'type' AND column_name = 'last_updated') THEN
    --
    -- Name: type_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS type_last_updated_idx ON public.type USING btree (last_updated DESC NULLS LAST);
END IF;


IF EXISTS(SELECT FROM information_schema.columns WHERE table_name = 'user_role' AND column_name = 'last_updated') THEN
    --
    -- Name: user_role_last_updated_idx; Type: INDEX; Schema: public; Owner: traffic_ops
    --

    CREATE INDEX IF NOT EXISTS user_role_last_updated_idx ON public.user_role USING btree (last_updated DESC NULLS LAST);
END IF;
END $$;

DO $$
BEGIN
IF NOT EXISTS (SELECT FROM pg_catalog.pg_trigger WHERE tgname = 'before_create_ip_address_trigger' AND tgrelid = CAST('ip_address' AS REGCLASS)) THEN
    --
    -- Name: ip_address before_create_ip_address_trigger; Type: TRIGGER; Schema: public; Owner: traffic_ops
    --

    CREATE TRIGGER before_create_ip_address_trigger BEFORE INSERT ON public.ip_address FOR EACH ROW EXECUTE PROCEDURE public.before_ip_address_table();
END IF;

IF NOT EXISTS (SELECT FROM pg_catalog.pg_trigger WHERE tgname = 'before_create_server_trigger' AND tgrelid = CAST('server' AS REGCLASS)) THEN
    --
    -- Name: server before_create_server_trigger; Type: TRIGGER; Schema: public; Owner: traffic_ops
    --

    CREATE TRIGGER before_create_server_trigger BEFORE INSERT ON public.server FOR EACH ROW EXECUTE PROCEDURE public.before_server_table();
END IF;

IF NOT EXISTS (SELECT FROM pg_catalog.pg_trigger WHERE tgname = 'before_update_ip_address_trigger' AND tgrelid = CAST('ip_address' AS REGCLASS)) THEN
    --
    -- Name: ip_address before_update_ip_address_trigger; Type: TRIGGER; Schema: public; Owner: traffic_ops
    --

    CREATE TRIGGER before_update_ip_address_trigger BEFORE UPDATE ON public.ip_address FOR EACH ROW WHEN ((new.address <> old.address)) EXECUTE PROCEDURE public.before_ip_address_table();
END IF;

IF NOT EXISTS (SELECT FROM pg_catalog.pg_trigger WHERE tgname = 'before_update_server_trigger' AND tgrelid = CAST('server' AS REGCLASS)) THEN
    --
    -- Name: server before_update_server_trigger; Type: TRIGGER; Schema: public; Owner: traffic_ops
    --

    CREATE TRIGGER before_update_server_trigger BEFORE UPDATE ON public.server FOR EACH ROW WHEN ((new.profile <> old.profile)) EXECUTE PROCEDURE public.before_server_table();
END IF;
END $$;

DO $$
DECLARE
    table_names VARCHAR[] := CAST(ARRAY[
        'api_capability',
        'asn',
        'cachegroup',
        'cachegroup_fallbacks',
        'cachegroup_localization_method',
        'cachegroup_parameter',
        'capability',
        'cdn',
        'coordinate',
        'deliveryservice',
        'deliveryservice_regex',
        'deliveryservice_request',
        'deliveryservice_request_comment',
        'deliveryservice_server',
        'deliveryservice_tmuser',
        'deliveryservices_required_capability',
        'division',
        'federation',
        'federation_deliveryservice',
        'federation_federation_resolver',
        'federation_resolver',
        'federation_tmuser',
        'hwinfo',
        'job',
        'job_agent',
        'job_status',
        'log',
        'origin',
        'parameter',
        'phys_location',
        'profile',
        'profile_parameter',
        'regex',
        'region',
        'role',
        'role_capability',
        'server',
        'server_capability',
        'server_server_capability',
        'servercheck',
        'service_category',
        'snapshot',
        'staticdnsentry',
        'stats_summary',
        'status',
        'steering_target',
        'tenant',
        'tm_user',
        'to_extension',
        'topology',
        'topology_cachegroup',
        'topology_cachegroup_parents',
        'type',
        'user_role'
    ] AS VARCHAR[]);
    table_name TEXT;
    trigger_name TEXT := 'on_delete_current_timestamp';
    trigger_exists BOOLEAN;
BEGIN
    FOREACH table_name IN ARRAY table_names
    LOOP
        EXECUTE FORMAT('SELECT EXISTS (
            SELECT
            FROM pg_catalog.pg_trigger
            WHERE tgname = ''%s''
            AND tgrelid = CAST(''%s'' AS REGCLASS))
            ',
            QUOTE_IDENT(trigger_name),
            QUOTE_IDENT(table_name)) INTO trigger_exists;
        IF NOT trigger_exists
        THEN
            EXECUTE FORMAT('
                    CREATE TRIGGER %s
                    AFTER DELETE ON %s
                    FOR EACH ROW
                        EXECUTE PROCEDURE public.%s_last_updated(''%s'');
                ',
                QUOTE_IDENT(trigger_name),
                QUOTE_IDENT(table_name),
                QUOTE_IDENT(trigger_name),
                QUOTE_IDENT(table_name)
            );
        END IF;
    END LOOP;
END$$;

DO $$
DECLARE
    table_names VARCHAR[] := CAST(ARRAY[
        'api_capability',
        'asn',
        'cachegroup',
        'cachegroup_parameter',
        'capability',
        'cdn',
        'coordinate',
        'deliveryservice',
        'deliveryservice_regex',
        'deliveryservice_request',
        'deliveryservice_request_comment',
        'deliveryservice_server',
        'deliveryservice_tmuser',
        'division',
        'federation',
        'federation_deliveryservice',
        'federation_federation_resolver',
        'federation_resolver',
        'federation_tmuser',
        'hwinfo',
        'job',
        'job_agent',
        'job_status',
        'log',
        'origin',
        'parameter',
        'phys_location',
        'profile',
        'profile_parameter',
        'regex',
        'region',
        'role',
        'role_capability',
        'server',
        'servercheck',
        'snapshot',
        'staticdnsentry',
        'status',
        'steering_target',
        'tenant',
        'tm_user',
        'topology',
        'topology_cachegroup',
        'topology_cachegroup_parents',
        'type',
        'user_role'
    ] AS VARCHAR[]);
    table_name TEXT;
    trigger_name TEXT := 'on_update_current_timestamp';
    trigger_exists BOOLEAN;
BEGIN
    FOREACH table_name IN ARRAY table_names
    LOOP
        EXECUTE FORMAT('SELECT EXISTS (
            SELECT
            FROM pg_catalog.pg_trigger
            WHERE tgname = ''%s''
            AND tgrelid = CAST(''%s'' AS REGCLASS))
            ',
            QUOTE_IDENT(trigger_name),
            QUOTE_IDENT(table_name)) INTO trigger_exists;
        IF NOT trigger_exists
        THEN
            EXECUTE FORMAT('
                CREATE TRIGGER %s
                BEFORE UPDATE ON %s
                FOR EACH ROW
                    EXECUTE PROCEDURE %s_last_updated();
            ',
            QUOTE_IDENT(trigger_name),
            QUOTE_IDENT(table_name),
            QUOTE_IDENT(trigger_name)
        );
        END IF;
    END LOOP;
END$$;

DO $$
BEGIN
IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'cachegroup_coordinate_fkey' AND table_name = 'cachegroup') THEN
    --
    -- Name: cachegroup cachegroup_coordinate_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup
        ADD CONSTRAINT cachegroup_coordinate_fkey FOREIGN KEY (coordinate) REFERENCES public.coordinate(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'cachegroup_localization_method_cachegroup_fkey' AND table_name = 'cachegroup_localization_method') THEN
    --
    -- Name: cachegroup_localization_method cachegroup_localization_method_cachegroup_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup_localization_method
        ADD CONSTRAINT cachegroup_localization_method_cachegroup_fkey FOREIGN KEY (cachegroup) REFERENCES public.cachegroup(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'deliveryservice_service_category_fkey' AND table_name = 'deliveryservice') THEN
    --
    -- Name: deliveryservice deliveryservice_service_category_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice
        ADD CONSTRAINT deliveryservice_service_category_fkey FOREIGN KEY (service_category) REFERENCES public.service_category(name) ON UPDATE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'deliveryservice_topology_fkey' AND table_name = 'deliveryservice') THEN
    --
    -- Name: deliveryservice deliveryservice_topology_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice
        ADD CONSTRAINT deliveryservice_topology_fkey FOREIGN KEY (topology) REFERENCES public.topology(name) ON UPDATE CASCADE ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_assignee' AND table_name = 'deliveryservice_request') THEN
    --
    -- Name: deliveryservice_request fk_assignee; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_request
        ADD CONSTRAINT fk_assignee FOREIGN KEY (assignee_id) REFERENCES public.tm_user(id) ON DELETE SET NULL;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_atsprofile_atsparameters_atsparameters1' AND table_name = 'profile_parameter') THEN
    --
    -- Name: profile_parameter fk_atsprofile_atsparameters_atsparameters1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.profile_parameter
        ADD CONSTRAINT fk_atsprofile_atsparameters_atsparameters1 FOREIGN KEY (parameter) REFERENCES public.parameter(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_atsprofile_atsparameters_atsprofile1' AND table_name = 'profile_parameter') THEN
    --
    -- Name: profile_parameter fk_atsprofile_atsparameters_atsprofile1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.profile_parameter
        ADD CONSTRAINT fk_atsprofile_atsparameters_atsprofile1 FOREIGN KEY (profile) REFERENCES public.profile(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_author' AND table_name = 'deliveryservice_request') THEN
    --
    -- Name: deliveryservice_request fk_author; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_request
        ADD CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES public.tm_user(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_author' AND table_name = 'deliveryservice_request_comment') THEN
    --
    -- Name: deliveryservice_request_comment fk_author; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_request_comment
        ADD CONSTRAINT fk_author FOREIGN KEY (author_id) REFERENCES public.tm_user(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_backup_cg' AND table_name = 'cachegroup_fallbacks') THEN
    --
    -- Name: cachegroup_fallbacks fk_backup_cg; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup_fallbacks
        ADD CONSTRAINT fk_backup_cg FOREIGN KEY (backup_cg) REFERENCES public.cachegroup(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_cap_name' AND table_name = 'role_capability') THEN
    --
    -- Name: role_capability fk_cap_name; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.role_capability
        ADD CONSTRAINT fk_cap_name FOREIGN KEY (cap_name) REFERENCES public.capability(name) ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_capability' AND table_name = 'api_capability') THEN
    --
    -- Name: api_capability fk_capability; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.api_capability
        ADD CONSTRAINT fk_capability FOREIGN KEY (capability) REFERENCES public.capability(name) ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_cdn1' AND table_name = 'deliveryservice') THEN
    --
    -- Name: deliveryservice fk_cdn1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice
        ADD CONSTRAINT fk_cdn1 FOREIGN KEY (cdn_id) REFERENCES public.cdn(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_cdn1' AND table_name = 'profile') THEN
    --
    -- Name: profile fk_cdn1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.profile
        ADD CONSTRAINT fk_cdn1 FOREIGN KEY (cdn) REFERENCES public.cdn(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_cdn2' AND table_name = 'server') THEN
    --
    -- Name: server fk_cdn2; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server
        ADD CONSTRAINT fk_cdn2 FOREIGN KEY (cdn_id) REFERENCES public.cdn(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_cg_1' AND table_name = 'cachegroup') THEN
    --
    -- Name: cachegroup fk_cg_1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup
        ADD CONSTRAINT fk_cg_1 FOREIGN KEY (parent_cachegroup_id) REFERENCES public.cachegroup(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_cg_param_cachegroup1' AND table_name = 'cachegroup_parameter') THEN
    --
    -- Name: cachegroup_parameter fk_cg_param_cachegroup1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup_parameter
        ADD CONSTRAINT fk_cg_param_cachegroup1 FOREIGN KEY (cachegroup) REFERENCES public.cachegroup(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_cg_secondary' AND table_name = 'cachegroup') THEN
    --
    -- Name: cachegroup fk_cg_secondary; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup
        ADD CONSTRAINT fk_cg_secondary FOREIGN KEY (secondary_parent_cachegroup_id) REFERENCES public.cachegroup(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_cg_type1' AND table_name = 'cachegroup') THEN
    --
    -- Name: cachegroup fk_cg_type1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup
        ADD CONSTRAINT fk_cg_type1 FOREIGN KEY (type) REFERENCES public.type(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_contentserver_atsprofile1' AND table_name = 'server') THEN
    --
    -- Name: server fk_contentserver_atsprofile1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server
        ADD CONSTRAINT fk_contentserver_atsprofile1 FOREIGN KEY (profile) REFERENCES public.profile(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_contentserver_contentserverstatus1' AND table_name = 'server') THEN
    --
    -- Name: server fk_contentserver_contentserverstatus1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server
        ADD CONSTRAINT fk_contentserver_contentserverstatus1 FOREIGN KEY (status) REFERENCES public.status(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_contentserver_contentservertype1' AND table_name = 'server') THEN
    --
    -- Name: server fk_contentserver_contentservertype1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server
        ADD CONSTRAINT fk_contentserver_contentservertype1 FOREIGN KEY (type) REFERENCES public.type(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_contentserver_phys_location1' AND table_name = 'server') THEN
    --
    -- Name: server fk_contentserver_phys_location1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server
        ADD CONSTRAINT fk_contentserver_phys_location1 FOREIGN KEY (phys_location) REFERENCES public.phys_location(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_cran_cachegroup1' AND table_name = 'asn') THEN
    --
    -- Name: asn fk_cran_cachegroup1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.asn
        ADD CONSTRAINT fk_cran_cachegroup1 FOREIGN KEY (cachegroup) REFERENCES public.cachegroup(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_deliveryservice' AND table_name = 'deliveryservice_consistent_hash_query_param') THEN
    --
    -- Name: deliveryservice_consistent_hash_query_param fk_deliveryservice; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_consistent_hash_query_param
        ADD CONSTRAINT fk_deliveryservice FOREIGN KEY (deliveryservice_id) REFERENCES public.deliveryservice(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_deliveryservice_id' AND table_name = 'deliveryservices_required_capability') THEN
    --
    -- Name: deliveryservices_required_capability fk_deliveryservice_id; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservices_required_capability
        ADD CONSTRAINT fk_deliveryservice_id FOREIGN KEY (deliveryservice_id) REFERENCES public.deliveryservice(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_deliveryservice_profile1' AND table_name = 'deliveryservice') THEN
    --
    -- Name: deliveryservice fk_deliveryservice_profile1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice
        ADD CONSTRAINT fk_deliveryservice_profile1 FOREIGN KEY (profile) REFERENCES public.profile(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_deliveryservice_request' AND table_name = 'deliveryservice_request_comment') THEN
    --
    -- Name: deliveryservice_request_comment fk_deliveryservice_request; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_request_comment
        ADD CONSTRAINT fk_deliveryservice_request FOREIGN KEY (deliveryservice_request_id) REFERENCES public.deliveryservice_request(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_deliveryservice_type1' AND table_name = 'deliveryservice') THEN
    --
    -- Name: deliveryservice fk_deliveryservice_type1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice
        ADD CONSTRAINT fk_deliveryservice_type1 FOREIGN KEY (type) REFERENCES public.type(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_ds_to_cs_contentserver1' AND table_name = 'deliveryservice_server') THEN
    --
    -- Name: deliveryservice_server fk_ds_to_cs_contentserver1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_server
        ADD CONSTRAINT fk_ds_to_cs_contentserver1 FOREIGN KEY (server) REFERENCES public.server(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_ds_to_cs_deliveryservice1' AND table_name = 'deliveryservice_server') THEN
    --
    -- Name: deliveryservice_server fk_ds_to_cs_deliveryservice1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_server
        ADD CONSTRAINT fk_ds_to_cs_deliveryservice1 FOREIGN KEY (deliveryservice) REFERENCES public.deliveryservice(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_ds_to_regex_deliveryservice1' AND table_name = 'deliveryservice_regex') THEN
    --
    -- Name: deliveryservice_regex fk_ds_to_regex_deliveryservice1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_regex
        ADD CONSTRAINT fk_ds_to_regex_deliveryservice1 FOREIGN KEY (deliveryservice) REFERENCES public.deliveryservice(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_ds_to_regex_regex1' AND table_name = 'deliveryservice_regex') THEN
    --
    -- Name: deliveryservice_regex fk_ds_to_regex_regex1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_regex
        ADD CONSTRAINT fk_ds_to_regex_regex1 FOREIGN KEY (regex) REFERENCES public.regex(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_ext_type' AND table_name = 'to_extension') THEN
    --
    -- Name: to_extension fk_ext_type; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.to_extension
        ADD CONSTRAINT fk_ext_type FOREIGN KEY (type) REFERENCES public.type(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_federation_federation_resolver1' AND table_name = 'federation_federation_resolver') THEN
    --
    -- Name: federation_federation_resolver fk_federation_federation_resolver1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_federation_resolver
        ADD CONSTRAINT fk_federation_federation_resolver1 FOREIGN KEY (federation) REFERENCES public.federation(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_federation_mapping_type' AND table_name = 'federation_resolver') THEN
    --
    -- Name: federation_resolver fk_federation_mapping_type; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_resolver
        ADD CONSTRAINT fk_federation_mapping_type FOREIGN KEY (type) REFERENCES public.type(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_federation_resolver_to_fed1' AND table_name = 'federation_federation_resolver') THEN
    --
    -- Name: federation_federation_resolver fk_federation_resolver_to_fed1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_federation_resolver
        ADD CONSTRAINT fk_federation_resolver_to_fed1 FOREIGN KEY (federation_resolver) REFERENCES public.federation_resolver(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_federation_tmuser_federation' AND table_name = 'federation_tmuser') THEN
    --
    -- Name: federation_tmuser fk_federation_tmuser_federation; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_tmuser
        ADD CONSTRAINT fk_federation_tmuser_federation FOREIGN KEY (federation) REFERENCES public.federation(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_federation_tmuser_role' AND table_name = 'federation_tmuser') THEN
    --
    -- Name: federation_tmuser fk_federation_tmuser_role; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_tmuser
        ADD CONSTRAINT fk_federation_tmuser_role FOREIGN KEY (role) REFERENCES public.role(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_federation_tmuser_tmuser' AND table_name = 'federation_tmuser') THEN
    --
    -- Name: federation_tmuser fk_federation_tmuser_tmuser; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_tmuser
        ADD CONSTRAINT fk_federation_tmuser_tmuser FOREIGN KEY (tm_user) REFERENCES public.tm_user(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_federation_to_ds1' AND table_name = 'federation_deliveryservice') THEN
    --
    -- Name: federation_deliveryservice fk_federation_to_ds1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_deliveryservice
        ADD CONSTRAINT fk_federation_to_ds1 FOREIGN KEY (deliveryservice) REFERENCES public.deliveryservice(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_federation_to_fed1' AND table_name = 'federation_deliveryservice') THEN
    --
    -- Name: federation_deliveryservice fk_federation_to_fed1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.federation_deliveryservice
        ADD CONSTRAINT fk_federation_to_fed1 FOREIGN KEY (federation) REFERENCES public.federation(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_hwinfo1' AND table_name = 'hwinfo') THEN
    --
    -- Name: hwinfo fk_hwinfo1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.hwinfo
        ADD CONSTRAINT fk_hwinfo1 FOREIGN KEY (serverid) REFERENCES public.server(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_job_agent_id1' AND table_name = 'job') THEN
    --
    -- Name: job fk_job_agent_id1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.job
        ADD CONSTRAINT fk_job_agent_id1 FOREIGN KEY (agent) REFERENCES public.job_agent(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_job_deliveryservice1' AND table_name = 'job') THEN
    --
    -- Name: job fk_job_deliveryservice1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.job
        ADD CONSTRAINT fk_job_deliveryservice1 FOREIGN KEY (job_deliveryservice) REFERENCES public.deliveryservice(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_job_status_id1' AND table_name = 'job') THEN
    --
    -- Name: job fk_job_status_id1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.job
        ADD CONSTRAINT fk_job_status_id1 FOREIGN KEY (status) REFERENCES public.job_status(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_job_user_id1' AND table_name = 'job') THEN
    --
    -- Name: job fk_job_user_id1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.job
        ADD CONSTRAINT fk_job_user_id1 FOREIGN KEY (job_user) REFERENCES public.tm_user(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_last_edited_by' AND table_name = 'deliveryservice_request') THEN
    --
    -- Name: deliveryservice_request fk_last_edited_by; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_request
        ADD CONSTRAINT fk_last_edited_by FOREIGN KEY (last_edited_by_id) REFERENCES public.tm_user(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_log_1' AND table_name = 'log') THEN
    --
    -- Name: log fk_log_1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.log
        ADD CONSTRAINT fk_log_1 FOREIGN KEY (tm_user) REFERENCES public.tm_user(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_parameter' AND table_name = 'cachegroup_parameter') THEN
    --
    -- Name: cachegroup_parameter fk_parameter; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup_parameter
        ADD CONSTRAINT fk_parameter FOREIGN KEY (parameter) REFERENCES public.parameter(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_parentid' AND table_name = 'tenant') THEN
    --
    -- Name: tenant fk_parentid; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.tenant
        ADD CONSTRAINT fk_parentid FOREIGN KEY (parent_id) REFERENCES public.tenant(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_phys_location_region' AND table_name = 'phys_location') THEN
    --
    -- Name: phys_location fk_phys_location_region; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.phys_location
        ADD CONSTRAINT fk_phys_location_region FOREIGN KEY (region) REFERENCES public.region(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_primary_cg' AND table_name = 'cachegroup_fallbacks') THEN
    --
    -- Name: cachegroup_fallbacks fk_primary_cg; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.cachegroup_fallbacks
        ADD CONSTRAINT fk_primary_cg FOREIGN KEY (primary_cg) REFERENCES public.cachegroup(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_regex_type1' AND table_name = 'regex') THEN
    --
    -- Name: regex fk_regex_type1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.regex
        ADD CONSTRAINT fk_regex_type1 FOREIGN KEY (type) REFERENCES public.type(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_region_division1' AND table_name = 'region') THEN
    --
    -- Name: region fk_region_division1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.region
        ADD CONSTRAINT fk_region_division1 FOREIGN KEY (division) REFERENCES public.division(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_required_capability' AND table_name = 'deliveryservices_required_capability') THEN
    --
    -- Name: deliveryservices_required_capability fk_required_capability; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservices_required_capability
        ADD CONSTRAINT fk_required_capability FOREIGN KEY (required_capability) REFERENCES public.server_capability(name) ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_role_id' AND table_name = 'role_capability') THEN
    --
    -- Name: role_capability fk_role_id; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.role_capability
        ADD CONSTRAINT fk_role_id FOREIGN KEY (role_id) REFERENCES public.role(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_role_id' AND table_name = 'user_role') THEN
    --
    -- Name: user_role fk_role_id; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.user_role
        ADD CONSTRAINT fk_role_id FOREIGN KEY (role_id) REFERENCES public.role(id) ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_server' AND table_name = 'server_server_capability') THEN
    --
    -- Name: server_server_capability fk_server; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server_server_capability
        ADD CONSTRAINT fk_server FOREIGN KEY (server) REFERENCES public.server(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_server_cachegroup1' AND table_name = 'server') THEN
    --
    -- Name: server fk_server_cachegroup1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server
        ADD CONSTRAINT fk_server_cachegroup1 FOREIGN KEY (cachegroup) REFERENCES public.cachegroup(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_server_capability' AND table_name = 'server_server_capability') THEN
    --
    -- Name: server_server_capability fk_server_capability; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.server_server_capability
        ADD CONSTRAINT fk_server_capability FOREIGN KEY (server_capability) REFERENCES public.server_capability(name) ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_serverstatus_server1' AND table_name = 'servercheck') THEN
    --
    -- Name: servercheck fk_serverstatus_server1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.servercheck
        ADD CONSTRAINT fk_serverstatus_server1 FOREIGN KEY (server) REFERENCES public.server(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_staticdnsentry_cachegroup1' AND table_name = 'staticdnsentry') THEN
    --
    -- Name: staticdnsentry fk_staticdnsentry_cachegroup1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.staticdnsentry
        ADD CONSTRAINT fk_staticdnsentry_cachegroup1 FOREIGN KEY (cachegroup) REFERENCES public.cachegroup(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_staticdnsentry_ds' AND table_name = 'staticdnsentry') THEN
    --
    -- Name: staticdnsentry fk_staticdnsentry_ds; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.staticdnsentry
        ADD CONSTRAINT fk_staticdnsentry_ds FOREIGN KEY (deliveryservice) REFERENCES public.deliveryservice(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_staticdnsentry_type' AND table_name = 'staticdnsentry') THEN
    --
    -- Name: staticdnsentry fk_staticdnsentry_type; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.staticdnsentry
        ADD CONSTRAINT fk_staticdnsentry_type FOREIGN KEY (type) REFERENCES public.type(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_steering_target_delivery_service' AND table_name = 'steering_target') THEN
    --
    -- Name: steering_target fk_steering_target_delivery_service; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.steering_target
        ADD CONSTRAINT fk_steering_target_delivery_service FOREIGN KEY (deliveryservice) REFERENCES public.deliveryservice(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_steering_target_target' AND table_name = 'steering_target') THEN
    --
    -- Name: steering_target fk_steering_target_target; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.steering_target
        ADD CONSTRAINT fk_steering_target_target FOREIGN KEY (target) REFERENCES public.deliveryservice(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_tenantid' AND table_name = 'deliveryservice') THEN
    --
    -- Name: deliveryservice fk_tenantid; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice
        ADD CONSTRAINT fk_tenantid FOREIGN KEY (tenant_id) REFERENCES public.tenant(id) MATCH FULL;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_tenantid' AND table_name = 'tm_user') THEN
    --
    -- Name: tm_user fk_tenantid; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.tm_user
        ADD CONSTRAINT fk_tenantid FOREIGN KEY (tenant_id) REFERENCES public.tenant(id) MATCH FULL;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_tm_user_ds' AND table_name = 'deliveryservice_tmuser') THEN
    --
    -- Name: deliveryservice_tmuser fk_tm_user_ds; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_tmuser
        ADD CONSTRAINT fk_tm_user_ds FOREIGN KEY (deliveryservice) REFERENCES public.deliveryservice(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_tm_user_id' AND table_name = 'deliveryservice_tmuser')     THEN
    --
    -- Name: deliveryservice_tmuser fk_tm_user_id; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.deliveryservice_tmuser
        ADD CONSTRAINT fk_tm_user_id FOREIGN KEY (tm_user_id) REFERENCES public.tm_user(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_user_1' AND table_name = 'tm_user')     THEN
    --
    -- Name: tm_user fk_user_1; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.tm_user
        ADD CONSTRAINT fk_user_1 FOREIGN KEY (role) REFERENCES public.role(id) ON DELETE SET NULL;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'fk_user_id' AND table_name = 'user_role')     THEN
    --
    -- Name: user_role fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.user_role
        ADD CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES public.tm_user(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'interface_server_fkey' AND table_name = 'interface')     THEN
    --
    -- Name: interface interface_server_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.interface
        ADD CONSTRAINT interface_server_fkey FOREIGN KEY (server) REFERENCES public.server(id) ON UPDATE CASCADE ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'ip_address_interface_fkey' AND table_name = 'ip_address')     THEN
    --
    -- Name: ip_address ip_address_interface_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.ip_address
        ADD CONSTRAINT ip_address_interface_fkey FOREIGN KEY (interface, server) REFERENCES public.interface(name, server) ON UPDATE CASCADE ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'ip_address_server_fkey' AND table_name = 'ip_address')     THEN
    --
    -- Name: ip_address ip_address_server_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.ip_address
        ADD CONSTRAINT ip_address_server_fkey FOREIGN KEY (server) REFERENCES public.server(id) ON UPDATE CASCADE ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'origin_cachegroup_fkey' AND table_name = 'origin')     THEN
    --
    -- Name: origin origin_cachegroup_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.origin
        ADD CONSTRAINT origin_cachegroup_fkey FOREIGN KEY (cachegroup) REFERENCES public.cachegroup(id) ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'origin_coordinate_fkey' AND table_name = 'origin')     THEN
    --
    -- Name: origin origin_coordinate_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.origin
        ADD CONSTRAINT origin_coordinate_fkey FOREIGN KEY (coordinate) REFERENCES public.coordinate(id) ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'origin_deliveryservice_fkey' AND table_name = 'origin')     THEN
    --
    -- Name: origin origin_deliveryservice_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.origin
        ADD CONSTRAINT origin_deliveryservice_fkey FOREIGN KEY (deliveryservice) REFERENCES public.deliveryservice(id) ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'origin_profile_fkey' AND table_name = 'origin')     THEN
    --
    -- Name: origin origin_profile_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.origin
        ADD CONSTRAINT origin_profile_fkey FOREIGN KEY (profile) REFERENCES public.profile(id) ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'origin_tenant_fkey' AND table_name = 'origin')     THEN
    --
    -- Name: origin origin_tenant_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.origin
        ADD CONSTRAINT origin_tenant_fkey FOREIGN KEY (tenant) REFERENCES public.tenant(id) ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'snapshot_cdn_fkey' AND table_name = 'snapshot')     THEN
    --
    -- Name: snapshot snapshot_cdn_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.snapshot
        ADD CONSTRAINT snapshot_cdn_fkey FOREIGN KEY (cdn) REFERENCES public.cdn(name) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'steering_target_type_fkey' AND table_name = 'steering_target')     THEN
    --
    -- Name: steering_target steering_target_type_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.steering_target
        ADD CONSTRAINT steering_target_type_fkey FOREIGN KEY (type) REFERENCES public.type(id);
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'topology_cachegroup_cachegroup_fkey' AND table_name = 'topology_cachegroup')     THEN
    --
    -- Name: topology_cachegroup topology_cachegroup_cachegroup_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.topology_cachegroup
        ADD CONSTRAINT topology_cachegroup_cachegroup_fkey FOREIGN KEY (cachegroup) REFERENCES public.cachegroup(name) ON UPDATE CASCADE ON DELETE RESTRICT;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'topology_cachegroup_parents_child_fkey' AND table_name = 'topology_cachegroup_parents')     THEN
    --
    -- Name: topology_cachegroup_parents topology_cachegroup_parents_child_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.topology_cachegroup_parents
        ADD CONSTRAINT topology_cachegroup_parents_child_fkey FOREIGN KEY (child) REFERENCES public.topology_cachegroup(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'topology_cachegroup_parents_parent_fkey' AND table_name = 'topology_cachegroup_parents')     THEN
    --
    -- Name: topology_cachegroup_parents topology_cachegroup_parents_parent_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.topology_cachegroup_parents
        ADD CONSTRAINT topology_cachegroup_parents_parent_fkey FOREIGN KEY (parent) REFERENCES public.topology_cachegroup(id) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;


IF NOT EXISTS (SELECT FROM information_schema.table_constraints WHERE constraint_name = 'topology_cachegroup_topology_fkey' AND table_name = 'topology_cachegroup')     THEN
    --
    -- Name: topology_cachegroup topology_cachegroup_topology_fkey; Type: FK CONSTRAINT; Schema: public; Owner: traffic_ops
    --

    ALTER TABLE ONLY public.topology_cachegroup
        ADD CONSTRAINT topology_cachegroup_topology_fkey FOREIGN KEY (topology) REFERENCES public.topology(name) ON UPDATE CASCADE ON DELETE CASCADE;
END IF;
END $$;


--
-- Name: public; Type: ACL; Schema: -; Owner: traffic_ops
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM traffic_ops;
GRANT ALL ON SCHEMA public TO traffic_ops;
GRANT ALL ON SCHEMA public TO PUBLIC;

--
-- PostgreSQL database dump complete
--
