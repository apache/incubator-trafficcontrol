package Fixtures::Integration::Deliveryservice;

# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
# 
#   http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.


# Do not edit! Generated code.
# See https://github.com/Comcast/traffic_control/wiki/The%20Kabletown%20example

use Moose;
extends 'DBIx::Class::EasyFixture';
use namespace::autoclean;

my %definition_for = (
	## id => 1
	'0' => {
		new      => 'Deliveryservice',
		using => {
			xml_id                      => 'cdl-c2',
			regional_geo_blocking       => 0,
			check_path                  => '/crossdomain.xml',
			info_url                    => '',
			ccr_dns_ttl                 => '3600',
			dns_bypass_cname            => undef,
			global_max_mbps             => '0',
			http_bypass_fqdn            => '',
			last_updated                => '2015-12-10 15:44:37',
			long_desc                   => 'long_desc',
			long_desc_2                 => 'long_desc_2',
			miss_lat                    => '41.881944',
			protocol                    => '0',
			qstring_ignore              => '0',
			type                        => '16',
			dns_bypass_ip6              => undef,
			dscp                        => '40',
			mid_header_rewrite          => 'cond %{REMAP_PSEUDO_HOOK} __RETURN__ set-config proxy.config.http.parent_origin.dead_server_retry_enabled 1',
			active                      => '1',
			geo_limit                   => '0',
			miss_long                   => '-87.627778',
			origin_shield               => undef,
			regex_remap                 => undef,
			remap_text                  => undef,
			cacheurl                    => undef,
			display_name                => 'cdl-c2',
			ipv6_routing_enabled        => undef,
			long_desc_1                 => 'long_desc_1',
			profile                     => '8',
			ssl_key_version             => '0',
			global_max_tps              => '0',
			max_dns_answers             => '0',
			tr_response_headers         => undef,
			tenant_id                   => undef,
			cdn_id                      => '2',
			dns_bypass_ttl              => undef,
			initial_dispersion          => '1',
			org_server_fqdn             => 'http://cdl.origin.kabletown.net',
			range_request_handling      => '0',
			signed                      => '1',
			dns_bypass_ip               => '',
			edge_header_rewrite         => 'add-header X-Powered-By: KBLTN [L]',
			multi_site_origin           => undef,
			multi_site_origin_algorithm => undef,
			tr_request_headers          => undef,
		},
	},
	## id => 2
	'1' => {
		new      => 'Deliveryservice',
		using => {
			xml_id                      => 'games-c1',
			regional_geo_blocking       => 0,
			multi_site_origin           => undef,
			multi_site_origin_algorithm => undef,
			protocol                    => '0',
			dns_bypass_ttl              => undef,
			edge_header_rewrite         => 'cond %{SEND_RESPONSE_HDR_HOOK} __RETURN__ add-header X-CDN-Info "KableTown___CACHE_IPV4__" [L]',
			geo_limit                   => '0',
			long_desc                   => 'test-ds3 long_desc',
			miss_long                   => '-87.627778',
			profile                     => '5',
			qstring_ignore              => '0',
			active                      => '1',
			cacheurl                    => undef,
			dns_bypass_cname            => undef,
			dns_bypass_ip               => '',
			initial_dispersion          => '1',
			range_request_handling      => '0',
			type                        => '10',
			ipv6_routing_enabled        => undef,
			tr_response_headers         => undef,
			tenant_id                   => undef,
			cdn_id                      => '1',
			global_max_mbps             => '0',
			global_max_tps              => '0',
			long_desc_2                 => 'test-ds3 long_desc_2',
			origin_shield               => undef,
			long_desc_1                 => 'test-ds3 long_desc_1',
			ssl_key_version             => '0',
			tr_request_headers          => undef,
			check_path                  => '/crossdomain.xml',
			display_name                => 'games-c1',
			http_bypass_fqdn            => '',
			info_url                    => 'http://games.info.kabletown.net',
			org_server_fqdn             => 'http://games.origin.kabletown.net',
			ccr_dns_ttl                 => '3600',
			dns_bypass_ip6              => undef,
			last_updated                => '2015-12-10 15:44:37',
			max_dns_answers             => '0',
			miss_lat                    => '41.881944',
			dscp                        => '40',
			mid_header_rewrite          => undef,
			regex_remap                 => undef,
			remap_text                  => undef,
			signed                      => '0',
		},
	},
	## id => 3
	'2' => {
		new      => 'Deliveryservice',
		using => {
			xml_id                      => 'images-c1',
			regional_geo_blocking       => 0,
			ipv6_routing_enabled        => undef,
			mid_header_rewrite          => undef,
			signed                      => '0',
			type                        => '10',
			dns_bypass_cname            => undef,
			multi_site_origin           => undef,
			multi_site_origin_algorithm => undef,
			qstring_ignore              => '0',
			ccr_dns_ttl                 => '3600',
			dscp                        => '40',
			last_updated                => '2015-12-10 15:44:37',
			org_server_fqdn             => 'http://images.origin.kabletown.net',
			tr_response_headers         => undef,
			cacheurl                    => undef,
			check_path                  => '/crossdomain.xml',
			dns_bypass_ip               => '',
			long_desc_1                 => 'test-ds2 long_desc_1',
			info_url                    => 'http://images.info.kabletown.net',
			regex_remap                 => undef,
			active                      => '1',
			dns_bypass_ip6              => undef,
			dns_bypass_ttl              => undef,
			protocol                    => '0',
			range_request_handling      => '0',
			geo_limit                   => '0',
			global_max_mbps             => '0',
			long_desc_2                 => 'test-ds2 long_desc_2',
			origin_shield               => undef,
			max_dns_answers             => '0',
			miss_long                   => '-87.627778',
			profile                     => '5',
			display_name                => 'images-c1',
			edge_header_rewrite         => 'rm-header Cache-Control [L]',
			initial_dispersion          => '1',
			long_desc                   => 'test-ds2 long_desc',
			remap_text                  => undef,
			ssl_key_version             => '0',
			tr_request_headers          => undef,
			tenant_id                   => undef,
			cdn_id                      => '1',
			global_max_tps              => '0',
			http_bypass_fqdn            => '',
			miss_lat                    => '41.881944',
		},
	},
	## id => 4
	'3' => {
		new      => 'Deliveryservice',
		using => {
			xml_id                			=> 'movies-c1',
			regional_geo_blocking 			=> 0,
			remap_text            			=> undef,
			check_path            			=> '/crossdomain.xml',
			ipv6_routing_enabled  			=> '1',
			origin_shield         			=> undef,
			dns_bypass_ip         			=> '',
			dns_bypass_ip6        			=> undef,
			mid_header_rewrite	 				=>
				'cond %{REMAP_PSEUDO_HOOK} __RETURN__ set-config proxy.config.http.parent_origin.dead_server_retry_enabled 1__RETURN__ set-config proxy.config.http.parent_origin.simple_retry_enabled 1__RETURN__ set-config proxy.config.http.parent_origin.simple_retry_response_codes "400,404,412"__RETURN__ set-config proxy.config.http.parent_origin.dead_server_retry_response_codes "502,503" __RETURN__ set-config proxy.config.http.connect_attempts_timeout 2 __RETURN__ set-config proxy.config.http.connect_attempts_max_retries 2 __RETURN__ set-config proxy.config.http.connect_attempts_max_retries_dead_server 1__RETURN__ set-config proxy.config.http.transaction_active_timeout_in 5 [L]',
			dscp                        => '40',
			info_url                    => 'http://movies.info.kabletown.net',
			last_updated                => '2015-12-10 15:44:37',
			signed                      => '0',
			ccr_dns_ttl                 => '3600',
			tenant_id                   => undef,
			cdn_id                      => '1',
			display_name                => 'movies-c1',
			protocol                    => '0',
			cacheurl                    => undef,
			http_bypass_fqdn            => '',
			miss_lat                    => '41.881944',
			dns_bypass_ttl              => undef,
			geo_limit                   => '0',
			long_desc                   => 'test-ds1 long_desc',
			initial_dispersion          => '1',
			long_desc_1                 => 'test-ds1 long_desc_1',
			max_dns_answers             => '0',
			multi_site_origin           => '1',
			multi_site_origin_algorithm => '0',
			active                      => '1',
			edge_header_rewrite         => 'cond %{REMAP_PSEUDO_HOOK} __RETURN__ set-config proxy.config.http.transaction_active_timeout_out 5 [L]',
			global_max_mbps             => '0',
			ssl_key_version             => '0',
			org_server_fqdn             => 'http://movies.origin.kabletown.net',
			range_request_handling      => '0',
			regex_remap                 => undef,
			miss_long                   => '-87.627778',
			profile                     => '5',
			tr_response_headers         => undef,
			dns_bypass_cname            => undef,
			global_max_tps              => '0',
			long_desc_2                 => 'test-ds1 long_desc_2',
			qstring_ignore              => '0',
			tr_request_headers          => undef,
			type                        => '16',
		},
	},
	## id => 5
	'4' => {
		new      => 'Deliveryservice',
		using => {
			xml_id                      => 'tv-c1',
			regional_geo_blocking       => 0,
			miss_lat                    => '41.881944',
			remap_text                  => undef,
			signed                      => '0',
			tr_request_headers          => undef,
			long_desc_1                 => 'test-ds1 long_desc_1',
			long_desc                   => 'test-ds1 long_desc',
			miss_long                   => '-87.627778',
			ssl_key_version             => '0',
			type                        => '17',
			ccr_dns_ttl                 => '3600',
			dns_bypass_cname            => undef,
			regex_remap                 => undef,
			dns_bypass_ip6              => undef,
			dscp                        => '40',
			range_request_handling      => '0',
			display_name                => 'tv-c1',
			ipv6_routing_enabled        => undef,
			mid_header_rewrite          => undef,
			multi_site_origin           => undef,
			multi_site_origin_algorithm => undef,
			qstring_ignore              => '0',
			active                      => '1',
			cacheurl                    => undef,
			info_url                    => 'http://movies.info.kabletown.net',
			initial_dispersion          => '1',
			last_updated                => '2015-12-10 15:44:37',
			max_dns_answers             => '0',
			profile                     => '5',
			protocol                    => '0',
			check_path                  => '/crossdomain.xml',
			dns_bypass_ip               => '',
			tr_response_headers         => undef,
			org_server_fqdn             => 'http://movies.origin.kabletown.net',
			geo_limit                   => '0',
			long_desc_2                 => 'test-ds1 long_desc_2',
			edge_header_rewrite         => undef,
			global_max_mbps             => '0',
			global_max_tps              => '0',
			http_bypass_fqdn            => '',
			origin_shield               => undef,
			tenant_id                   => undef,
			cdn_id                      => '1',
			dns_bypass_ttl              => undef,
		},
	},
	## id => 6
	'5' => {
		new      => 'Deliveryservice',
		using => {
			xml_id                      => 'tv-local-c2',
			regional_geo_blocking       => 0,
			origin_shield               => undef,
			range_request_handling      => '0',
			display_name                => 'tv-local-c2',
			geo_limit                   => '0',
			global_max_tps              => '0',
			mid_header_rewrite          => undef,
			signed                      => '0',
			ssl_key_version             => '0',
			tenant_id                   => undef,
			cdn_id                      => '2',
			long_desc                   => 'test-ds3 long_desc',
			max_dns_answers             => '0',
			regex_remap                 => undef,
			active                      => '1',
			check_path                  => '/crossdomain.xml',
			dns_bypass_cname            => undef,
			dscp                        => '40',
			remap_text                  => undef,
			global_max_mbps             => '0',
			http_bypass_fqdn            => '',
			miss_long                   => '-87.627778',
			org_server_fqdn             => 'https://games.origin.kabletown.net',
			multi_site_origin           => undef,
			multi_site_origin_algorithm => undef,
			cacheurl                    => undef,
			dns_bypass_ip               => '',
			dns_bypass_ip6              => undef,
			miss_lat                    => '41.881944',
			info_url                    => 'http://games.info.kabletown.net',
			profile                     => '8',
			qstring_ignore              => '0',
			tr_response_headers         => undef,
			ipv6_routing_enabled        => undef,
			long_desc_2                 => 'test-ds3 long_desc_2',
			tr_request_headers          => undef,
			type                        => '17',
			ccr_dns_ttl                 => '3600',
			dns_bypass_ttl              => undef,
			edge_header_rewrite         => undef,
			initial_dispersion          => '1',
			last_updated                => '2015-12-10 15:44:37',
			long_desc_1                 => 'test-ds3 long_desc_1',
			protocol                    => '0',
		},
	},
	## id => 7
	'6' => {
		new      => 'Deliveryservice',
		using => {
			xml_id                      => 'tv-nat-c2',
			regional_geo_blocking       => 0,
			check_path                  => '/crossdomain.xml',
			dns_bypass_ip               => '',
			miss_long                   => '-87.627778',
			origin_shield               => undef,
			range_request_handling      => '0',
			remap_text                  => undef,
			tenant_id                   => undef,
			cdn_id                      => '2',
			long_desc_2                 => 'test long_desc_2',
			tr_request_headers          => undef,
			type                        => '18',
			info_url                    => 'http://tv.info.kabletown.net',
			max_dns_answers             => '0',
			profile                     => '8',
			ccr_dns_ttl                 => '3600',
			dns_bypass_cname            => undef,
			dscp                        => '40',
			global_max_tps              => '0',
			initial_dispersion          => '1',
			multi_site_origin           => undef,
			multi_site_origin_algorithm => undef,
			org_server_fqdn             => 'http://national-tv.origin.kabletown.net',
			signed                      => '0',
			display_name                => 'tv-nat-c2',
			dns_bypass_ip6              => undef,
			dns_bypass_ttl              => undef,
			long_desc                   => 'test long_desc',
			active                      => '0',
			cacheurl                    => undef,
			geo_limit                   => '0',
			mid_header_rewrite          => undef,
			tr_response_headers         => undef,
			edge_header_rewrite         => undef,
			last_updated                => '2015-12-10 15:44:37',
			regex_remap                 => undef,
			ssl_key_version             => '0',
			global_max_mbps             => '0',
			http_bypass_fqdn            => '',
			ipv6_routing_enabled        => undef,
			long_desc_1                 => 'test long_desc_1',
			miss_lat                    => '41.881944',
			protocol                    => '0',
			qstring_ignore              => '0',
		},
	},
	## id => 8
	'7' => {
		new      => 'Deliveryservice',
		using => {
			xml_id                      => 'tv-nocache-c2',
			regional_geo_blocking       => 0,
			check_path                  => '/crossdomain.xml',
			signed                      => '0',
			dns_bypass_cname            => undef,
			dscp                        => '40',
			geo_limit                   => '0',
			initial_dispersion          => '1',
			long_desc                   => 'test- long_desc',
			long_desc_2                 => 'test- long_desc_2',
			multi_site_origin           => undef,
			multi_site_origin_algorithm => undef,
			origin_shield               => undef,
			protocol                    => '0',
			ipv6_routing_enabled        => undef,
			long_desc_1                 => 'test- long_desc_1',
			tr_response_headers         => undef,
			active                      => '1',
			cacheurl                    => undef,
			ccr_dns_ttl                 => '3600',
			tenant_id                   => undef,
			cdn_id                      => '2',
			dns_bypass_ip               => '',
			dns_bypass_ttl              => undef,
			global_max_tps              => '0',
			max_dns_answers             => '0',
			regex_remap                 => undef,
			remap_text                  => undef,
			miss_long                   => '-87.627778',
			org_server_fqdn             => 'http://cc.origin.kabletown.net',
			display_name                => 'tv-nocache-c2',
			qstring_ignore              => '0',
			dns_bypass_ip6              => undef,
			http_bypass_fqdn            => '',
			last_updated                => '2015-12-10 15:44:37',
			mid_header_rewrite          => undef,
			miss_lat                    => '41.881944',
			profile                     => '8',
			ssl_key_version             => '0',
			type                        => '19',
			edge_header_rewrite         => undef,
			global_max_mbps             => '0',
			info_url                    => '',
			range_request_handling      => '0',
			tr_request_headers          => undef,
		},
	},
);

sub name {
	return "Deliveryservice";
}

sub get_definition {
	my ( $self, $name ) = @_;
	return $definition_for{$name};
}

sub all_fixture_names {
	# sort by db xml_id to guarantee insertion order
	return (sort { $definition_for{$a}{using}{xml_id} cmp $definition_for{$b}{using}{xml_id} } keys %definition_for);
}
__PACKAGE__->meta->make_immutable;
1;
