package API::Iso;
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
#
#

# JvD Note: you always want to put Utils as the first use. Sh*t don't work if it's after the Mojo lines.
use UI::Utils;

use UI::Utils;
use Mojo::Base 'Mojolicious::Controller';
use Mojolicious::Plugins;
use File::Find;
use File::Basename;
use File::Path qw(make_path);
use Data::Dumper;
use Common::ReturnCodes qw(SUCCESS ERROR);
use Mojolicious::Plugin::Config;
use Data::Validate::IP qw(is_ipv4 is_ipv6);
use Validate::Tiny ':all';
use Net::Domain qw(hostfqdn);

my $filebasedir             = "/var/www/files";
my $ksfiles_parm_name       = "kickstart.files.location";
my $ksfiles_configfile_name = "mkisofs";

# This is the directory we put the configuration files in for kickstart &
# scripts to process:
my $install_cfg = "ks_scripts";

sub osversions {
	my $self = shift;

	my $osversionsdir;
	my $ksdir = $self->db->resultset('Parameter')->search( { -and => [ name => $ksfiles_parm_name, config_file => $ksfiles_configfile_name ] } )
		->get_column('value')->single();

	if ( defined $ksdir && $ksdir ne "" ) {
		$osversionsdir = $ksdir;
	}
	else {
		$osversionsdir = $filebasedir;
	}

	my %osversions;

	{
		open( CFG, "<$osversionsdir/osversions.cfg" ) || die("$osversionsdir/osversions.cfg:$!");
		local $/;
		eval <CFG>;
		close CFG;
	}

	$self->success( \%osversions );

}

sub generate {
	my $self   = shift;
	my $params = $self->req->json;

	my ( $is_valid, $result ) = $self->is_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $response = $self->generate_iso($params);

	if ( $params->{stream} eq 'yes' ) {
		$self->res->headers->content_type("application/download");
		$self->res->headers->content_disposition("attachment; filename=\"$response->{isoName}\"");

		return $self->render( data => $response->{iso} );
	} else {
		return $self->success( $response, "Generate ISO was successful." );
	}

}

sub generate_iso {
	my $self = shift;
	my $params = shift;

	my $osversion_dir  	= $params->{osversionDir};
	my $hostname       	= $params->{hostName};
	my $domain_name    	= $params->{domainName};
	my $rootpass       	= $params->{rootPass};
	my $dhcp           	= $params->{dhcp};
	my $ipaddr         	= $params->{ipAddress};
	my $netmask        	= $params->{ipNetmask};
	my $gateway        	= $params->{ipGateway};
	my $ip6_address    	= $params->{ip6Address};
	my $ip6_gateway    	= $params->{ip6Gateway};
	my $interface_name 	= $params->{interfaceName};
	my $interface_mtu  	= $params->{interfaceMtu};
	my $ondisk         	= $params->{disk};
	my $mgmt_ip_address = $params->{mgmtIpAddress};
	my $mgmt_ip_netmask = $params->{mgmtIpNetmask};
	my $mgmt_ip_gateway = $params->{mgmtIpGateway};
	my $mgmt_interface 	= $params->{mgmtInterface};
	my $stream          = $params->{stream};

	#The API has hostname and domainName, the UI does not
	my $fqdn = $hostname;
	if (defined($domain_name)) {
		$fqdn .= "." . $domain_name
	}

	# This sets up the "strength" of the hash. So far $1 works (md5). It will produce a sha256 ($5), but it's untested.
	# PROTIP: Do not put the $ in.
	my $digest = "1";

	# Read /etc/resolv.conf and get the nameservers. This is (supposedly) a hack
	# until we get a reasonable UI set up.
	my ( $nameservers, $line, $nsip ) = "";
	open( RESOLV, '< /etc/resolv.conf' ) || die("/etc/resolv.conf: $!");
	while ( $line = <RESOLV> ) {
		if ( $line =~ /^nameserver / ) {
			$nsip = ( split( " ", $line ) )[1];
			$nameservers = "$nsip $nameservers";
		}
	}

	$nameservers =~ s/ /,/g;
	$nameservers =~ s/,$//;

	my $dir;
	my $ksdir = $self->db->resultset('Parameter')->search( { -and => [ name => $ksfiles_parm_name, config_file => $ksfiles_configfile_name ] } )
		->get_column('value')->single();

	if ( defined $ksdir && $ksdir ne "" ) {
		$dir = $ksdir . "/" . $osversion_dir;
	}
	else {
		$dir = $filebasedir . "/" . $osversion_dir;
	}

	my $cfg_dir = "$dir/$install_cfg";
	$self->app->log->info( "cfg_dir: " . $cfg_dir );

	# This constructs the network.cfg file that gets written in the $install_cfg directory
	# in network.cfg
	# This is what we need to create:
	# IP='192.168.0.2'
	# IPV6='...'
	# NETMASK='255.255.255.0'
	# GATEWAY='192.168.0.1'
	# NAMESERVER='8.8.8.8,8.8.4.4'
	# HOSTNAME='mid-cache-01.mycdn.mydomain.net'
	# MTU='1500'
	# BOND_DEVICE='em0'
	# BONDOPTS='mode=802.3ad,lacp_rate=fast,xmit_hash_policy=layer3+4'
	my $network_string = "IPADDR=\"$ipaddr\"\nNETMASK=\"$netmask\"\nGATEWAY=\"$gateway\"\nDEVICE=\"$interface_name\"\nMTU=\"$interface_mtu\"\nNAMESERVER=\"$nameservers\"\nHOSTNAME=\"$fqdn\"\nNETWORKING_IPV6=\"yes\"\nIPV6ADDR=\"$ip6_address\"\nIPV6_DEFAULTGW=\"$ip6_gateway\"\nDHCP=\"$dhcp\"";
	if ($interface_name =~ m/^bond\d+/) {
		$network_string = "IPADDR=\"$ipaddr\"\nNETMASK=\"$netmask\"\nGATEWAY=\"$gateway\"\nBOND_DEVICE=\"$interface_name\"\nMTU=\"$interface_mtu\"\nNAMESERVER=\"$nameservers\"\nHOSTNAME=\"$fqdn\"\nNETWORKING_IPV6=\"yes\"\nIPV6ADDR=\"$ip6_address\"\nIPV6_DEFAULTGW=\"$ip6_gateway\"\nBONDING_OPTS=\"miimon=100 mode=4 lacp_rate=fast xmit_hash_policy=layer3+4\"\nDHCP=\"$dhcp\"";
	}
	# Write out the networking config:
	open( NF, "> $cfg_dir/network.cfg" ) or die "$cfg_dir/network.cfg: $!";
	print NF $network_string;
	close NF;

	#generate and write management network config file if mgmt_IPAddress is defined
	my $mgmt_network_string = "IPADDR=\"$mgmt_ip_address\"\nNETMASK=\"$mgmt_ip_netmask\"\nGATEWAY=\"$mgmt_ip_gateway\"\nDEVICE=$mgmt_interface";
	$mgmt_ip_address =~ s/\/.*//g;
	if (is_ipv6($mgmt_ip_address)) {
		$mgmt_network_string = "IPV6ADDR=\"$mgmt_ip_address\"\nNETMASK=\"$mgmt_ip_netmask\"\nGATEWAY=\"$mgmt_ip_gateway\"\nDEVICE=$mgmt_interface";
	}

	open( NF, "> $cfg_dir/mgmt_network.cfg" ) or die "$cfg_dir/mgmt_network.cfg: $!";
	print NF $mgmt_network_string;
	close NF;

	my $root_pass_string;
	my @chars = ( "A" .. "Z", "a" .. "z", 0 .. 9 ) or die 'the @chars thing didn\'t work';
	my $salt;
	$salt .= $chars[ rand @chars ] for 1 .. 8;
	my $kripted_pw = crypt( "$rootpass", "\$$digest\$$salt\$" ) . "\n";
	$root_pass_string = "rootpw --iscrypted $kripted_pw";

	open( PWF, "> $cfg_dir/password.cfg" ) or die "$cfg_dir/password.cfg: $!";
	print PWF "$root_pass_string";
	close PWF;

	open( DSK, "> $cfg_dir/disk.cfg" ) or die "$cfg_dir/disk.cfg: $!";
	print DSK "boot_drives=\"$ondisk\"";
	close DSK;

	my $iso_dir       = "iso";
	my $config        = $self->app->config;
	my $iso_root_path = $config->{'geniso'}{'iso_root_path'};

	my $iso_dir_path = join( "/", $iso_root_path, $iso_dir );
	make_path($iso_dir_path);

	my $iso_file_name = "$fqdn-$osversion_dir.iso";
	my $iso_file_path = join( "/", $iso_dir_path, $iso_file_name );

	my $cmd =
		"mkisofs -o $iso_file_path -joliet-long -input-charset utf-8 -b isolinux/isolinux.bin -c isolinux/boot.cat -no-emul-boot -boot-load-size 4 -boot-info-table -R -J -v -T $dir";

	if ( $stream eq 'yes' ) {
		$cmd = "mkisofs -joliet-long -input-charset utf-8 -b isolinux/isolinux.bin -c isolinux/boot.cat -no-emul-boot -boot-load-size 4 -boot-info-table -R -J -v -T $dir";
	}

	my $type = "default";
	my $custom_cmd = sprintf( "%s/%s", $dir, "generate" );

	if ( -f $custom_cmd && -x $custom_cmd ) {
		$cmd = sprintf( "%s %s", $custom_cmd, $iso_file_path );
		$type = "custom";
	}
	elsif ( -f $custom_cmd && !-x $custom_cmd ) {
		$self->app->log->warn("$custom_cmd exists but is not executable; using $type ISO generation command");
	}

	$self->app->log->info( "Using $type ISO generation command: " . $cmd );

	# This just writes the string we're going to use to generate the ISO. You
	# won't need it unless you're debugging stuff, but it doesn't really hurt.
	open( STUF, "> $cfg_dir/state.out" ) or die "$cfg_dir/state.out: $!";
	print STUF "Dir== $dir\n";
	print STUF "$cmd\n";
	close STUF;

	my $response = {};

	if ( $stream ne 'yes' ) {
		$self->app->log->info("Writing ISO: " . $iso_file_path);
		my $output = `$cmd 2>&1` || die("Error executing $cmd:");
		$self->app->log->info($output);

		if ( $fqdn eq "" ) {
			&log($self, "ISO created [ " . $osversion_dir . " ] for unspecified hostname", "APICHANGE");
		} else {
			&log($self, "ISO created [ " . $osversion_dir . " ] for " . $fqdn, "APICHANGE");
		}

		# parse out http / https from to.base_url config; use local fqdn for download link
		my @protocol = split( '://', $config->{'to'}{'base_url'} );
		my $iso_url = join( '/', $protocol[0] . ':/', lc hostfqdn(), $iso_dir, $iso_file_name );

		$response = {
			isoName => $iso_file_name,
			isoURL  => $iso_url,
		};
	} else {
		my $data = `$cmd`;
		if ( $type eq 'custom' ) {
			my $ok = open my $fh, "<$iso_file_path";
			if (! $ok ) {
				$self->internal_server_error( { Error => "Error reading $iso_file_path" } );
				return;
			}

			# slurp it in..
			undef $/;
			$data = <$fh>;

			close $fh;
			unlink $iso_file_path;
		}

		if ( $fqdn eq "" ) {
			&log($self, "ISO created [ " . $osversion_dir . " ] for unspecified hostname", "APICHANGE");
		} else {
			&log($self, "ISO created [ " . $osversion_dir . " ] for " . $fqdn, "APICHANGE");
		}

		$response = {
			iso => $data,
			isoName => $iso_file_name,
		};
	}
	return $response;
}

sub is_valid {
	my $self   = shift;
	my $params = shift;
	my $mgmtIpAddress = $params->{mgmtIpAddress};

	my $rules = {
		fields => [qw/osversionDir hostName domainName rootPass dhcp ipAddress ipNetmask ipGateway ip6Address ip6Gateway interfaceName interfaceMtu disk mgmtInterface mgmtIpGateway mgmtIpAddress mgmtIpNetmask/],

		# Validation checks to perform
		checks => [
			osversionDir    => [ is_required("is required") ],
			hostName     => [ is_required("is required") ],
			domainName   => [ is_required("is required") ],
			rootPass     => [ is_required("is required") ],
			dhcp         => [ is_required("is required") ],
			interfaceMtu => [ is_required("is required") ],
			disk => [ is_required("is required") ],
			mgmtInterface => [ is_required_if((defined($mgmtIpAddress) && $mgmtIpAddress ne ""), "- Management interface is required when Management IP is provided") ],
			mgmtIpGateway => [ is_required_if((defined($mgmtIpAddress) && $mgmtIpAddress ne ""), "- Management gateway is required when Management IP is provided") ],
			ipAddress    => is_required_if(
				sub {
					my $params = shift;
					return $params->{dhcp} eq 'no';
				},
				"is required if DHCP is no"
			),
			ipNetmask => is_required_if(
				sub {
					my $params = shift;
					return $params->{dhcp} eq 'no';
				},
				"is required if DHCP is no"
			),
			ipGateway => is_required_if(
				sub {
					my $params = shift;
					return $params->{dhcp} eq 'no';
				},
				"is required if DHCP is no"
			),
		]
	};

	# Validate the input against the rules
	my $result = validate( $params, $rules );

	if ( $result->{success} ) {
		return ( 1, $result->{data} );
	}
	else {
		return ( 0, $result->{error} );
	}
}

