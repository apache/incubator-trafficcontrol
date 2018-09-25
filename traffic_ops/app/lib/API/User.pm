package API::User;
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
use Utils::Tenant;

use Mojo::Base 'Mojolicious::Controller';
use Utils::Helper;
use Mojolicious::Validator;
use Mojolicious::Validator::Validation;
use Data::Dumper;
use Test::More;
use Email::Valid;
use Utils::Helper::ResponseHelper;
use Validate::Tiny ':all';
use UI::ConfigFiles;
use UI::Tools;
use Data::GUID;
use POSIX qw(strftime);

sub login {
	my $self    = shift;
	my $options = shift;

	my $u     = $self->req->json->{u};
	my $p     = $self->req->json->{p};
	my $token = $options->{'token'};

	my $result = $self->authenticate( $u, $p, $options );
	if ($result) {
		return $self->success_message("Successfully logged in.");
	}
	elsif ( defined($token) ) {
		return $self->invalid_token;
	}
	else {
		return $self->invalid_username_or_password;
	}
}

sub token_login {
	my $self = shift;

	my $token = $self->req->json->{t};
	return $self->login( { token => $token } );
}

# Read
sub index {
	my $self = shift;
	my @data;
	my $username 	= $self->param('username');
	my $tenant_id	= $self->param('tenant');

	my $orderby = "username";
	$orderby = $self->param('orderby') if ( defined $self->param('orderby') );

	my %criteria;
	if ( defined $tenant_id ) {
		$criteria{'me.tenant_id'} = $tenant_id;
	}

	my $dbh;
	if ( defined $username ) {
		$dbh = $self->db->resultset("TmUser")->search( { username => $username }, { prefetch => [ 'role', 'tenant' ], order_by => 'me.' . $orderby } );
	}
	else {
		$dbh = $self->db->resultset("TmUser")->search( \%criteria, { prefetch => [ 'role', 'tenant' ], order_by => 'me.' . $orderby } );
	}

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();

	while ( my $row = $dbh->next ) {
		if (!$tenant_utils->is_user_resource_accessible($tenants_data, $row->tenant_id)) {
			next;
		}
		push(
			@data, {
				"addressLine1"     => $row->address_line1,
				"addressLine2"     => $row->address_line2,
				"city"             => $row->city,
				"company"          => $row->company,
				"country"          => $row->country,
				"email"            => $row->email,
				"fullName"         => $row->full_name,
				"gid"              => $row->gid,
				"id"               => $row->id,
				"lastUpdated"      => $row->last_updated,
				"newUser"          => \$row->new_user,
				"phoneNumber"      => $row->phone_number,
				"postalCode"       => $row->postal_code,
				"publicSshKey"     => $row->public_ssh_key,
				"registrationSent" => $row->registration_sent,
				"role"             => $row->role->id,
				"rolename"         => $row->role->name,
				"stateOrProvince"  => $row->state_or_province,
				"uid"              => $row->uid,
				"username"         => $row->username,
				"tenant"           => defined( $row->tenant ) ? $row->tenant->name : undef,
				"tenantId"         => $row->tenant_id
			}
		);
	}
	$self->success( \@data );
}

sub show {
	my $self = shift;
	my $id   = $self->param('id');

	my $rs_data = $self->db->resultset("TmUser")->search( { 'me.id' => $id }, { prefetch => [ 'role', 'tenant' ] } );
	my @data = ();

	my $tenant_utils = Utils::Tenant->new($self);
   	my $tenants_data = $tenant_utils->create_tenants_data_from_db();

	while ( my $row = $rs_data->next ) {
		if (!$tenant_utils->is_user_resource_accessible($tenants_data, $row->tenant_id)) {
			return $self->forbidden("Forbidden: User is not available for your tenant.");
		}
		push(
			@data, {
				"addressLine1"     => $row->address_line1,
				"addressLine2"     => $row->address_line2,
				"city"             => $row->city,
				"company"          => $row->company,
				"country"          => $row->country,
				"email"            => $row->email,
				"fullName"         => $row->full_name,
				"gid"              => $row->gid,
				"id"               => $row->id,
				"lastUpdated"      => $row->last_updated,
				"newUser"          => \$row->new_user,
				"phoneNumber"      => $row->phone_number,
				"postalCode"       => $row->postal_code,
				"publicSshKey"     => $row->public_ssh_key,
				"registrationSent" => $row->registration_sent,
				"role"             => $row->role->id,
				"rolename"         => $row->role->name,
				"stateOrProvince"  => $row->state_or_province,
				"uid"              => $row->uid,
				"username"         => $row->username,
				"tenant"           => defined( $row->tenant ) ? $row->tenant->name : undef,
				"tenantId"         => $row->tenant_id
			}
		);
	}
	$self->success( \@data );
}

sub update {
	my $self    = shift;
	my $user_id = $self->param('id');
	my $params  = $self->req->json;

	my $user = $self->db->resultset('TmUser')->find( { id => $user_id } );
	if ( !defined($user) ) {
		return $self->not_found();
	}

	my $tenant_utils = Utils::Tenant->new($self);

	#setting tenant_id to undef if tenant is not set.
	my $tenant_id = exists( $params->{tenantId} ) ? $params->{tenantId} : undef;
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_user_resource_accessible($tenants_data, $user->tenant_id)) {
		#no access to resource tenant
		return $self->forbidden("Forbidden: User is not available for your tenant.");
	}
	if ($tenant_utils->use_tenancy() and !defined($tenant_id) and defined($user->tenant_id)) {
		return $self->alert("Invalid tenant. Cannot clear the user tenancy.");
	}
	if (!$tenant_utils->is_user_resource_accessible($tenants_data, $tenant_id)) {
		#no access to target tenancy
		return $self->alert("Invalid tenant. This tenant is not available to you for assignment.");
	}

	my ( $is_valid, $result ) = $self->is_valid( $params, $user_id );

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $values = {
		address_line1     => $params->{addressLine1},
		address_line2     => $params->{addressLine2},
		city              => $params->{city},
		company           => $params->{company},
		country           => $params->{country},
		email             => $params->{email},
		full_name         => $params->{fullName},
		phone_number      => $params->{phoneNumber},
		postal_code       => $params->{postalCode},
		public_ssh_key    => $params->{publicSshKey},
		role              => $params->{role},
		state_or_province => $params->{stateOrProvince},
		username          => $params->{username},
		tenant_id         => $tenant_id
	};

	if ( defined( $params->{localPasswd} ) && $params->{localPasswd} ne '' ) {
		$values->{"local_passwd"} = Utils::Helper::hash_pass($params->{localPasswd});
	}
	if ( defined( $params->{confirmLocalPasswd} ) && $params->{confirmLocalPasswd} ne '' ) {
		$values->{"confirm_local_passwd"} = Utils::Helper::hash_pass( $params->{confirmLocalPasswd} );
	}

	my $rs = $user->update($values);
	if ($rs) {
		my $response;
		$response->{addressLine1}     = $rs->address_line1;
		$response->{addressLine2}     = $rs->address_line2;
		$response->{city}             = $rs->city;
		$response->{company}          = $rs->company;
		$response->{country}          = $rs->country;
		$response->{email}            = $rs->email;
		$response->{fullName}         = $rs->full_name;
		$response->{gid}              = $rs->gid;
		$response->{id}               = $rs->id;
		$response->{lastUpdated}      = $rs->last_updated;
		$response->{newUser}          = \$rs->new_user;
		$response->{phoneNumber}      = $rs->phone_number;
		$response->{postalCode}       = $rs->postal_code;
		$response->{publicSshKey}     = $rs->public_ssh_key;
		$response->{registrationSent} = $rs->registration_sent;
		$response->{role}             = $rs->role->id;
		$response->{roleName}         = $rs->role->name;
		$response->{stateOrProvince}  = $rs->state_or_province;
		$response->{uid}              = $rs->uid;
		$response->{username}         = $rs->username;
		$response->{tenantId}         = $rs->tenant_id;
		$response->{tenant}           = defined( $rs->tenant ) ? $rs->tenant->name : undef;

		&log( $self, "Updated User with username '" . $rs->username . "' for id: " . $rs->id, "APICHANGE" );

		return $self->success( $response, "User update was successful." );
	}
	else {
		return $self->alert("User update failed.");
	}

}

sub create {
	my $self   = shift;
	my $params = $self->req->json;

	#setting tenant_id to the user's tenant if tenant is not set.
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();

	my $tenant_id = $params->{tenantId};
	if (!defined($tenant_id) and $tenant_utils->use_tenancy()){
		return $self->alert("Invalid tenant. Must set tenant for new user.");
	}

	if (!$tenant_utils->is_user_resource_accessible($tenants_data, $tenant_id)) {
		return $self->alert("Invalid tenant. This tenant is not available to you for assignment.");
	}

	my ( $is_valid, $result ) = $self->is_valid( $params, 0 );

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	if ( !defined( $params->{localPasswd} ) ) {
		return $self->alert("localPasswd is required.");
	}

	if ( !defined( $params->{confirmLocalPasswd} ) ) {
		return $self->alert("confirmLocalPasswd is required.");
	}


	my $values = {
		address_line1        => $params->{addressLine1},
		address_line2        => $params->{addressLine2},
		city                 => $params->{city},
		company              => $params->{company},
		country              => $params->{country},
		email                => $params->{email},
		full_name            => $params->{fullName},
		phone_number         => $params->{phoneNumber},
		postal_code          => $params->{postalCode},
		public_ssh_key       => $params->{publicSshKey},
		role                 => $params->{role},
		state_or_province    => $params->{stateOrProvince},
		username             => $params->{username},
		local_passwd         => Utils::Helper::hash_pass( $params->{localPasswd} ),
		confirm_local_passwd => Utils::Helper::hash_pass( $params->{confirmLocalPasswd} ),
		tenant_id            => $tenant_id,
	};

	my $insert = $self->db->resultset('TmUser')->create($values);
	my $rs     = $insert->insert();

	if ($rs) {
		my $response;
		$response->{addressLine1}     = $rs->address_line1;
		$response->{addressLine2}     = $rs->address_line2;
		$response->{city}             = $rs->city;
		$response->{company}          = $rs->company;
		$response->{country}          = $rs->country;
		$response->{email}            = $rs->email;
		$response->{fullName}         = $rs->full_name;
		$response->{gid}              = $rs->gid;
		$response->{id}               = $rs->id;
		$response->{lastUpdated}      = $rs->last_updated;
		$response->{newUser}          = \$rs->new_user;
		$response->{phoneNumber}      = $rs->phone_number;
		$response->{postalCode}       = $rs->postal_code;
		$response->{publicSshKey}     = $rs->public_ssh_key;
		$response->{registrationSent} = $rs->registration_sent;
		$response->{role}             = $rs->role->id;
		$response->{roleName}         = $rs->role->name;
		$response->{stateOrProvince}  = $rs->state_or_province;
		$response->{uid}              = $rs->uid;
		$response->{username}         = $rs->username;
		$response->{tenantId}         = $rs->tenant_id;
		$response->{tenant}           = defined( $rs->tenant ) ? $rs->tenant->name : undef;

		&log( $self, "Adding User with username '" . $rs->username . "' for id: " . $rs->id, "APICHANGE" );

		return $self->success( $response, "User creation was successful." );
	}
	else {
		return $self->alert("User creation failed.");
	}
}

# Reset the User Profile password
sub reset_password {
	my $self     = shift;
	my $email_to = $self->req->json->{email};
	my $dbh      = $self->db->resultset('TmUser')->find( { email => $email_to } );
	if ( defined($dbh) ) {

		my $email_notice = 'Successfully sent reset password to: ' . $email_to;
		$self->app->log->info($email_notice);

		my $token = Data::GUID->new;
		if ( $self->send_password_reset_email( $email_to, $token ) ) {
			$self->update_user_token( $email_to, $token );
		}

		return $self->success_message( "Successfully sent password reset to email '" . $email_to . "'" );
	}
	else {
		return $self->alert( { "Email not found " => "'" . $email_to . "'" } );
	}

}

sub register_user {
	my $self    = shift;
	my $params  = $self->req->json;

	my ( $is_valid, $result ) = $self->is_registration_valid($params);

	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $email_to 	= $params->{email};
	my $token    	= Data::GUID->new;
	my $role 		= $params->{role};
	my $tenant 		= $params->{tenantId};

	my $now = strftime( "%Y-%m-%d %H:%M:%S", gmtime() );
	my $existing_user = $self->db->resultset('TmUser')->find( { email => $email_to } );

	if (!defined($existing_user)) {
		my $new_user = $self->db->resultset('TmUser')->create(
			{
				email				=> $email_to,
				role				=> $role,
				tenant_id			=> $tenant,
				username			=> $token,
				token				=> $token,
				new_user			=> 1,
				registration_sent	=>  $now,
			}
		);
		$new_user->insert();
	} elsif ( defined($existing_user) && $existing_user->new_user() ) {
		$existing_user->token($token);
		$existing_user->role($role);
		$existing_user->tenant_id($tenant);
		$existing_user->registration_sent($now);
		$existing_user->update();
	} else {
		return $self->alert("User already exists and has completed registration.");
	}

	#send the registration email with a link the user can follow to finish the registration
	$self->send_registration_email( $email_to, $token );

	my $role_name 	= $self->db->resultset("Role")->search( { id => $role } )->get_column('name')->single();
	my $tenant_name = $self->db->resultset("Tenant")->search( { id => $tenant } )->get_column('name')->single();

	my $msg = "Sent user registration to $email_to with the following permissions [ role: $role_name | tenant: $tenant_name ]";
	&log( $self, $msg, "APICHANGE" );

	return $self->success_message($msg);
}


sub get_available_deliveryservices_not_assigned_to_user {
	my $self = shift;
	my @data;
	my $id = $self->param('id');
	my %takendsids;

	my $user = $self->db->resultset('TmUser')->find( { id => $id } );
	if ( !defined($user) ) {
		return $self->not_found();
	}
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_user_resource_accessible($tenants_data, $user->tenant_id)) {
		#no access to resource tenant
		return $self->forbidden();
	}

	my $rs_takendsids = undef;
	$rs_takendsids = $self->db->resultset("DeliveryserviceTmuser")->search( { 'tm_user_id' => $id } );

	while ( my $row = $rs_takendsids->next ) {
		$takendsids{ $row->deliveryservice->id } = undef;
	}

	my $rs_links = $self->db->resultset("Deliveryservice")->search( undef, { order_by => "xml_id" } );
	while ( my $row = $rs_links->next ) {
        if (!$tenant_utils->is_ds_resource_accessible($tenants_data, $row->tenant_id)) {
            #the current user cannot access this DS
            next;
        }
		if (!$tenant_utils->is_ds_resource_accessible_to_tenant($tenants_data, $row->tenant_id, $user->tenant_id)) {
			#the user under inspection cannot access this DS
			next;
		}
		if ( !exists( $takendsids{ $row->id } ) ) {
			push( @data, {
				"id" 			=> $row->id,
				"xmlId" 		=> $row->xml_id,
				"displayName" 	=> $row->display_name,
			} );
		}
	}

	$self->success( \@data );
}

sub assign_deliveryservices {
	my $self				= shift;
	my $params				= $self->req->json;
	my $user_id				= $params->{userId};
	my $delivery_services	= $params->{deliveryServices};
	my $replace				= $params->{replace};
	my $count				= 0;

	if ( ref($delivery_services) ne 'ARRAY' ) {
		return $self->alert("Delivery services must be an array");
	}

	my $user = $self->db->resultset('TmUser')->find( { id => $user_id } );
	if ( !defined($user) ) {
		return $self->not_found();
	}
	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_user_resource_accessible($tenants_data, $user->tenant_id)) {
		#no access to resource tenant
		return $self->alert("Invalid user. This user is not available to you for assignment.");
	}

	if ( $replace ) {
		# start fresh and delete existing user/deliveryservice associations
		# We are not checking DS tenancy on deletion - we manage the user here - we remove permissions to touch a DS
		my $delete = $self->db->resultset('DeliveryserviceTmuser')->search( { tm_user_id => $user_id } );
		$delete->delete();
	}

	my @values = ( [ qw( deliveryservice tm_user_id ) ]); # column names are required for 'populate' function

	foreach my $ds_id (@{ $delivery_services }) {
		push(@values, [ $ds_id, $user_id ]);
		$count++;
	}

	$self->db->resultset("DeliveryserviceTmuser")->populate(\@values);

	&log( $self, $count . " delivery services were assigned to " . $user->username, "APICHANGE" );

	my $response = $params;
	return $self->success($response, "Delivery service assignments complete.");
}

# Read the current user profile and produce the result
sub current {
	my $self = shift;
	my @data;
	my $current_username = $self->current_user()->{username};
	if ( &is_ldap($self) ) {
		my $role = $self->db->resultset('Role')->search( { name => "read-only" } )->single;

		push(
			@data, {
				"id"              => "0",
				"username"        => $current_username,
				"tenantId"        => undef,
				"tenant"          => undef,
				"publicSshKey"    => "",
				"role"            => $role->id,
				"roleName"        => $role->name,
				"uid"             => "0",
				"gid"             => "0",
				"company"         => "",
				"email"           => "",
				"fullName"        => "",
				"newUser"         => \0,
				"localUser"       => \0,
				"addressLine1"    => "",
				"addressLine2"    => "",
				"city"            => "",
				"stateOrProvince" => "",
				"phoneNumber"     => "",
				"postalCode"      => "",
				"country"         => "",
			}
		);

		return $self->success(@data);
	}
	else {
		my $dbh = $self->db->resultset('TmUser')->search( { username => $current_username }, { prefetch => [ 'role', 'tenant' ] } );
		while ( my $row = $dbh->next ) {
			push(
				@data, {
					"id"              => $row->id,
					"username"        => $row->username,
					"publicSshKey"    => $row->public_ssh_key,
					"role"            => $row->role->id,
					"roleName"        => $row->role->name,
					"uid"             => $row->uid,
					"gid"             => $row->gid,
					"company"         => $row->company,
					"email"           => $row->email,
					"fullName"        => $row->full_name,
					"newUser"         => \$row->new_user,
					"localUser"       => \1,
					"addressLine1"    => $row->address_line1,
					"addressLine2"    => $row->address_line2,
					"city"            => $row->city,
					"stateOrProvince" => $row->state_or_province,
					"phoneNumber"     => $row->phone_number,
					"postalCode"      => $row->postal_code,
					"country"         => $row->country,
					"tenant"          => defined( $row->tenant ) ? $row->tenant->name : undef,
					"tenantId"        => $row->tenant_id,
				}
			);
		}
		return $self->success(@data);
	}
}

# Update
sub update_current {
	my $self = shift;

	my $user = $self->req->json->{user};
	if ( &is_ldap($self) ) {
		return $self->alert( "Profile cannot be updated because '" . $user->{username} . "' is logged in as LDAP." );
	}

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();
	if (!$tenant_utils->is_user_resource_accessible($tenants_data, $user->{"tenantId"})) {
		return $self->alert("Invalid tenant. This tenant is not available to you for assignment.");
	}

	my $db_user;

	# Prevent these from getting updated
	# Do not modify the localPasswd if it comes across as blank.
	my $local_passwd = $user->{"localPasswd"};
	if ( defined($local_passwd) && ( $local_passwd eq '' ) ) {
		delete( $user->{"localPasswd"} );
	}

	# Do not modify the confirmLocalPasswd if it comes across as blank.
	my $confirm_local_passwd = $user->{"confirmLocalPasswd"};
	if ( defined($confirm_local_passwd) && ( $confirm_local_passwd eq '' ) ) {
		delete( $user->{"confirmLocalPasswd"} );
	}

	my $current_user_id = $self->db->resultset('TmUser')->search( { username => $self->current_user()->{username} } )->get_column('id')->single;

	my ( $is_valid, $result ) = $self->is_valid( $user, $current_user_id );

	if ($is_valid) {
		my $username = $self->current_user()->{username};
		my $dbh = $self->db->resultset('TmUser')->find( { username => $username } );

		# Updating a user implies it is no longer new
		$db_user->{"new_user"} = 0;

		# These if "defined" checks allow for partial user updates, otherwise the entire
		# user would need to be passed through.
		if ( defined($local_passwd) && $local_passwd ne '' ) {
			$db_user->{"local_passwd"} = Utils::Helper::hash_pass( $local_passwd );
		}
		if ( defined($confirm_local_passwd) && $confirm_local_passwd ne '' ) {
			$db_user->{"confirm_local_passwd"} = Utils::Helper::hash_pass( $confirm_local_passwd );
		}
		if ( defined( $user->{"id"} ) ) {
			$db_user->{"id"} = $user->{"id"};
		}
		if ( defined( $user->{"username"} ) ) {
			$db_user->{"username"} = $user->{"username"};
		}
		if ( defined( $user->{"tenantId"} ) ) {
			$db_user->{"tenant_id"} = $user->{"tenantId"};
		}
		if ( defined( $user->{"public_ssh_key"} ) ) {
			$db_user->{"public_ssh_key"} = $user->{"public_ssh_key"};
		}
		if ( defined( $user->{"role"} ) ) {
			$db_user->{"role"} = $user->{"role"};
		}
		if ( defined( $user->{"uid"} ) ) {
			$db_user->{"uid"} = $user->{"uid"};
		}
		if ( defined( $user->{"gid"} ) ) {
			$db_user->{"gid"} = $user->{"gid"};
		}
		if ( defined( $user->{"company"} ) ) {
			$db_user->{"company"} = $user->{"company"};
		}
		if ( defined( $user->{"email"} ) ) {
			$db_user->{"email"} = $user->{"email"};
		}
		if ( defined( $user->{"fullName"} ) ) {
			$db_user->{"full_name"} = $user->{"fullName"};
		}
		if ( defined( $user->{"newUser"} ) ) {
			$db_user->{"new_user"} = $user->{"newUser"};
		}
		if ( defined( $user->{"addressLine1"} ) ) {
			$db_user->{"address_line1"} = $user->{"addressLine1"};
		}
		if ( defined( $user->{"addressline2"} ) ) {
			$db_user->{"address_line2"} = $user->{"addressLine2"};
		}
		if ( defined( $user->{"city"} ) ) {
			$db_user->{"city"} = $user->{"city"};
		}
		if ( defined( $user->{"stateOrProvince"} ) ) {
			$db_user->{"state_or_province"} = $user->{"stateOrProvince"};
		}
		if ( defined( $user->{"phoneNumber"} ) ) {
			$db_user->{"phone_number"} = $user->{"phoneNumber"};
		}
		if ( defined( $user->{"postalCode"} ) ) {
			$db_user->{"postal_code"} = $user->{"postalCode"};
		}
		if ( defined( $user->{"country"} ) ) {
			$db_user->{"country"} = $user->{"country"};
		}
		# token is intended for new user registrations and on current user update, it should be cleared from the db
		$db_user->{"token"} = undef;
		# new_user flag is intended to identify new user registrations and on current user update, registration is complete
		$db_user->{"new_user"} = 0;
		$dbh->update($db_user);
		return $self->success_message("User profile was successfully updated");
	}
	else {
		return $self->alert($result);
	}
}

sub is_valid {
	my $self        = shift;
	my $user_params = shift;
	my $user_id     = shift;

	my $rules = {
		fields => [
			qw/fullName username public_ssh_key email role uid gid localPasswd confirmLocalPasswd company newUser addressLine1 addressLine2 city stateOrProvince phoneNumber postalCode country/
		],

		# Checks to perform on all fields
		checks => [

			fullName	=> [ is_required("is required") ],
			username	=> [ is_required("is required") ],
			email		=> [ is_required("is required") ],
			role		=> [ is_required("is required"), sub { is_valid_role($self, @_) } ],

			# pass2 must be equal to pass
			localPasswd => sub {
				my $value  = shift;
				my $params = shift;
				if ( defined( $params->{'localPasswd'} ) ) {
					return $self->is_good_password( $value, $params );
				}
			},

			# email must be unique
			email => sub {
				my $value  = shift;
				my $params = shift;
				if ( defined( $params->{'email'} ) ) {
					return $self->is_email_taken( $value, $user_id );
				}
			},

			# custom sub validates an email address
			email => sub {
				my ($value) = @_;
				Email::Valid->address($value) ? undef : 'email is not a valid format';
			},

			# username must be unique
			username => sub {
				my $value  = shift;
				my $params = shift;
				if ( defined( $params->{'username'} ) ) {
					return $self->is_username_taken( $value, $user_id );
				}
			},

		]
	};

	# Validate the input against the rules
	my $result = validate( $user_params, $rules );

	if ( $result->{success} ) {

		#print "success: " . dump( $result->{data} );
		return ( 1, $result->{data} );
	}
	else {
		#print "failed " . Dumper( $result->{error} );
		return ( 0, $result->{error} );
	}

}

sub is_registration_valid {
	my $self   = shift;
	my $params = shift;

	my $rules = {
		fields => [
			qw/email role tenantId/
		],

		# Validation checks to perform
		checks => [
			email		=> [ is_required("is required"), sub { is_valid_email($self, @_) } ],
			role		=> [ is_required("is required"), sub { is_valid_role($self, @_) } ],
			tenantId	=> [ is_required("is required"), sub { is_valid_tenant($self, @_) } ],
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

sub is_valid_email {
	my $self    = shift;
	my ( $value, $params ) = @_;

	return Email::Valid->address($value) ? undef : 'is not valid';
}

sub is_valid_role {
	my $self    = shift;
	my ( $value, $params ) = @_;

	my $role_priv_level = $self->db->resultset("Role")->search( { id => $value } )->get_column('priv_level')->single();
	if ( !defined($role_priv_level) ) {
		return "not found";
	}

	my $my_role = $self->db->resultset('TmUser')->search( { username => $self->current_user()->{username} } )->get_column('role')->single();
	my $my_role_priv_level = $self->db->resultset("Role")->search( { id => $my_role } )->get_column('priv_level')->single();

	if ( $role_priv_level > $my_role_priv_level ) {
		return "cannot exceed current user's privilege level ($my_role_priv_level)";
	}

	return undef;
}

sub is_valid_tenant {
	my $self    = shift;
	my ( $value, $params ) = @_;

	my $tenant = $self->db->resultset("Tenant")->search( { id => $value } )->single();
	if ( !defined($tenant) ) {
		return "not found";
	}

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db(undef);

	if (!$tenant_utils->is_tenant_resource_accessible($tenants_data, $value)) {
		return "not available to current user.";
	}

	return undef;
}

sub is_username_taken {
	my $self     = shift;
	my $username = shift;
	my $user_id  = shift;

	my $user_with_username = $self->db->resultset('TmUser')->search( { username => $username } )->single;
	if ( defined($user_with_username) ) {
		my %condition = ( -and => [ { username => $username }, { id => { '!=' => $user_id } } ] );
		my $count = $self->db->resultset('TmUser')->search( \%condition )->count();

		if ( $count > 0 ) {
			return "is already taken";
		}
	}

	return undef;
}

sub is_email_taken {
	my $self    = shift;
	my $email   = shift;
	my $user_id = shift;

	my $user_with_email = $self->db->resultset('TmUser')->search( { email => $email } )->single;
	if ( defined($user_with_email) ) {
		my %condition = ( -and => [ { email => $email }, { id => { '!=' => $user_id } } ] );
		my $count = $self->db->resultset('TmUser')->search( \%condition )->count();

		if ( $count > 0 ) {
			return "is already taken";
		}
	}

	return undef;
}

sub is_good_password {
	my $self   = shift;
	my $value  = shift;
	my $params = shift;
	if ( !defined $value or $value eq '' ) {
		return undef;
	}

	if ( $value ne $params->{'confirmLocalPasswd'} ) {
		return "Your 'New Password' must match the 'Confirm New Password'.";
	}

	if ( $value eq $params->{'username'} ) {
		return "Your password cannot be the same as your username.";
	}

	if ( length($value) < 8 ) {
		return "Password must be greater than 7 chars.";
	}

	if ( defined( $self->app->{invalid_passwords}->{$value} ) ) {
		return "Password is too common.";
	}

	# At this point we're happy with the password
	return undef;
}

1;
