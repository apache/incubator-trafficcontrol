package Fixtures::Server;
#
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
use Moose;
extends 'DBIx::Class::EasyFixture';
use namespace::autoclean;

my %definition_for = (
	server_edge1 => {
		new   => 'Server',
		using => {
			id               => 100,
			host_name        => 'atlanta-edge-01',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-edge-01\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.1',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.1',
			ip6_address      => '2345:1234:12:8::2/64',
			ip6_gateway      => '2345:1234:12:8::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 1,
			status           => 2,
			profile          => 100,
			cdn_id           => 100,
			cachegroup       => 300,
			phys_location    => 100,
		},
	},
	server_mid1 => {
		new   => 'Server',
		using => {
			id               => 200,
			host_name        => 'atlanta-mid-01',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-mid-01\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.2',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.2',
			ip6_address      => '2345:1234:12:9::2/64',
			ip6_gateway      => '2345:1234:12:9::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 2,
			status           => 2,
			profile          => 200,
			cdn_id           => 100,
			cachegroup       => 100,
			phys_location    => 100,
		},
	},
	rascal_server => {
		new   => 'Server',
		using => {
			id               => 300,
			host_name        => 'rascal01',
			domain_name      => 'kabletown.net',
			tcp_port         => 81,
			xmpp_id          => 'rascal\@kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.4',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.4',
			ip6_address      => '2345:1234:12:b::2/64',
			ip6_gateway      => '2345:1234:12:b::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 4,
			status           => 2,
			profile          => 300,
			cdn_id           => 200,
			cachegroup       => 100,
			phys_location    => 100,
		},
	},
	riak_server1 => {
		new   => 'Server',
		using => {
			id               => 400,
			host_name        => 'riak01',
			domain_name      => 'kabletown.net',
			tcp_port         => 8088,
			xmpp_id          => '',
			xmpp_passwd      => '',
			interface_name   => 'eth1',
			ip_address       => '127.0.0.5',
			ip_netmask       => '255.255.252.0',
			ip_gateway       => '127.0.0.5',
			ip6_address      => '',
			ip6_gateway      => '',
			interface_mtu    => 1500,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 31,
			status           => 2,
			profile          => 500,
			cdn_id           => 100,
			cachegroup       => 100,
			phys_location    => 100,
		},
	},
	rascal_server2 => {
		new   => 'Server',
		using => {
			id               => 500,
			host_name        => 'rascal02',
			domain_name      => 'kabletown.net',
			tcp_port         => 81,
			xmpp_id          => 'rascal\@kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.6',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.6',
			ip6_address      => '2345:1234:12:c::2/64',
			ip6_gateway      => '2345:1234:12:c::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.05',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 4,
			status           => 2,
			profile          => 300,
			cdn_id           => 200,
			cachegroup       => 100,
			phys_location    => 100,
		},
	},
	server_edge2 => {
		new   => 'Server',
		using => {
			id               => 600,
			host_name        => 'atlanta-edge-02',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-edge-02\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.7',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.7',
			ip6_address      => '2345:1234:12:d::2/64',
			ip6_gateway      => '2345:1234:12:d::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 1,
			status           => 2,
			profile          => 100,
			cdn_id           => 100,
			cachegroup       => 300,
			phys_location    => 100,
		},
	},
	server_mid2 => {
		new   => 'Server',
		using => {
			id               => 700,
			host_name        => 'atlanta-mid-02',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-mid-02\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.8',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.8',
			ip6_address      => '2345:1234:12:e::2/64',
			ip6_gateway      => '2345:1234:12:e::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 2,
			status           => 2,
			profile          => 200,
			cdn_id           => 200,
			cachegroup       => 200,
			phys_location    => 200,
		},
	},
	riak_server2 => {
		new   => 'Server',
		using => {
			id               => 800,
			host_name        => 'riak02',
			domain_name      => 'kabletown.net',
			tcp_port         => 8088,
			xmpp_id          => '',
			xmpp_passwd      => '',
			interface_name   => 'eth1',
			ip_address       => '127.0.0.9',
			ip_netmask       => '255.255.252.0',
			ip_gateway       => '127.0.0.9',
			ip6_address      => '2345:1234:12:f::2/64',
			ip6_gateway      => '2345:1234:12:f::1',
			interface_mtu    => 1500,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 31,
			status           => 2,
			profile          => 500,
			cdn_id           => 100,
			cachegroup       => 100,
			phys_location    => 200,
		},
	},
	influxdb_server1 => {
		new   => 'Server',
		using => {
			id               => 900,
			host_name        => 'influxdb01',
			domain_name      => 'kabletown.net',
			tcp_port         => 8086,
			xmpp_id          => '',
			xmpp_passwd      => '',
			interface_name   => 'eth1',
			ip_address       => '127.0.0.10',
			ip_netmask       => '255.255.252.0',
			ip_gateway       => '127.0.0.10',
			ip6_address      => '127.0.0.10',
			ip6_gateway      => '127.0.0.10',
			interface_mtu    => 1500,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 32,
			status           => 2,
			profile          => 500,
			cdn_id           => 100,
			cachegroup       => 100,
			phys_location    => 300,
		},
	},
	influxdb_server2 => {
		new   => 'Server',
		using => {
			id               => 1000,
			host_name        => 'influxdb02',
			domain_name      => 'kabletown.net',
			tcp_port         => 8086,
			xmpp_id          => '',
			xmpp_passwd      => '',
			interface_name   => 'eth1',
			ip_address       => '127.0.0.11',
			ip_netmask       => '255.255.252.0',
			ip_gateway       => '127.0.0.11',
			ip6_address      => '127.0.0.11',
			ip6_gateway      => '127.0.0.11',
			interface_mtu    => 1500,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 32,
			status           => 2,
			profile          => 500,
			cdn_id           => 100,
			cachegroup       => 100,
			phys_location    => 300,
		},
	},
	server_router => {
		new   => 'Server',
		using => {
			id               => 1100,
			host_name        => 'atlanta-router-01',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-router-01\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.12',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.1',
			ip6_address      => '2345:1234:12:8::10/64',
			ip6_gateway      => '2345:1234:12:8::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 4,
			status           => 2,
			profile          => 100,
			cdn_id           => 100,
			cachegroup       => 300,
			phys_location    => 100,
		},
	},
	server_edge_reported => {
		new   => 'Server',
		using => {
			id               => 1200,
			host_name        => 'atlanta-edge-03',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-edge-03\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.13',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.1',
			ip6_address      => '2345:1234:12:2::2/64',
			ip6_gateway      => '2345:1234:12:8::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 1,
			status           => 3,
			profile          => 100,
			cdn_id           => 100,
			cachegroup       => 300,
			phys_location    => 100,
		},
	},
	server_edge13 => {
		new   => 'Server',
		using => {
			id               => 1300,
			host_name        => 'atlanta-edge-14',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-edge-14\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.14',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.1',
			ip6_address      => '2345:1234:12:8::14/64',
			ip6_gateway      => '2345:1234:12:8::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 1,
			status           => 2,
			profile          => 100,
			cdn_id           => 100,
			cachegroup       => 900,
			phys_location    => 100,
		},
	},
	server_edge14 => {
		new   => 'Server',
		using => {
			id               => 1400,
			host_name        => 'atlanta-edge-15',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-edge-15\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.15',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.7',
			ip6_address      => '2345:1234:12:d::15/64',
			ip6_gateway      => '2345:1234:12:d::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 1,
			status           => 2,
			profile          => 100,
			cdn_id           => 100,
			cachegroup       => 900,
			phys_location    => 100,
		},
	},
	server_mid15 => {
		new   => 'Server',
		using => {
			id               => 1500,
			host_name        => 'atlanta-mid-16',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-mid-16\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.16',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.7',
			ip6_address      => '2345:1234:12:d::16/64',
			ip6_gateway      => '2345:1234:12:d::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 2,
			status           => 2,
			profile          => 100,
			cdn_id           => 100,
			cachegroup       => 800,
			phys_location    => 100,
		},
	},
	server_org1 => {
		new   => 'Server',
		using => {
			id               => 1600,
			host_name        => 'atlanta-org-1',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-org-1\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.17',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.17',
			ip6_address      => '2345:1234:12:d::17/64',
			ip6_gateway      => '2345:1234:12:d::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 3,
			status           => 2,
			profile          => 100,
			cdn_id           => 100,
			cachegroup       => 800,
			phys_location    => 100,
		},
	},
	server_org2 => {
		new   => 'Server',
		using => {
			id               => 1700,
			host_name        => 'atlanta-org-2',
			domain_name      => 'ga.atlanta.kabletown.net',
			tcp_port         => 80,
			xmpp_id          => 'atlanta-org-1\@ocdn.kabletown.net',
			xmpp_passwd      => 'X',
			interface_name   => 'bond0',
			ip_address       => '127.0.0.18',
			ip_netmask       => '255.255.255.252',
			ip_gateway       => '127.0.0.18',
			ip6_address      => '2345:1234:12:d::18/64',
			ip6_gateway      => '2345:1234:12:d::1',
			interface_mtu    => 9000,
			rack             => 'RR 119.02',
			mgmt_ip_address  => '',
			mgmt_ip_netmask  => '',
			mgmt_ip_gateway  => '',
			ilo_ip_address   => '',
			ilo_ip_netmask   => '',
			ilo_ip_gateway   => '',
			ilo_username     => '',
			ilo_password     => '',
			router_host_name => '',
			router_port_name => '',
			type             => 3,
			status           => 2,
			profile          => 900,
			cdn_id           => 200,
			cachegroup       => 800,
			phys_location    => 100,
		},
	},
);

sub get_definition {
	my ( $self, $name ) = @_;
	return $definition_for{$name};
}

sub all_fixture_names {
	# sort by db xml_id to guarantee insertion order
	return (sort { $definition_for{$a}{using}{id} cmp $definition_for{$b}{using}{id} } keys %definition_for);
}

__PACKAGE__->meta->make_immutable;

1;
