package API::Job;

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
use UI::Utils;
use Mojolicious::Validator;
use Mojolicious::Validator::Validation;
use Mojo::JSON;
use Time::Local;
use LWP;
use Email::Valid;
use MojoPlugins::Response;
use MojoPlugins::Job;
use Utils::Helper::ResponseHelper;
use Validate::Tiny ':all';
use UI::ConfigFiles;
use UI::Tools;
use Data::Dumper;

sub index {
	my $self    = shift;
	my $ds_id   = $self->param('dsId');
	my $user_id = $self->param('userId');

	my %criteria;
	if ( defined $ds_id ) {

		my $ds = $self->db->resultset('Deliveryservice')->find( { id => $ds_id } );
		if ( !defined($ds) ) {
			return $self->not_found();
		}

		my $tenant_utils = Utils::Tenant->new($self);
		my $tenants_data = $tenant_utils->create_tenants_data_from_db();

		if ( $tenant_utils->use_tenancy() ) {
			if ( !$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id) ) {
				return $self->forbidden("Forbidden. Delivery-service tenant is not available to the user.");
			}
		}

		$criteria{'job_deliveryservice'} = $ds_id;
	}
	if ( defined $user_id ) {
		$criteria{'job_user'} = $user_id;
	}

	my @data;
	my $jobs = $self->db->resultset("Job")->search( \%criteria, { prefetch => [ 'job_deliveryservice', 'job_user' ], order_by => 'me.start_time DESC' } );
	while ( my $job = $jobs->next ) {
		push(
			@data, {
				"id"              => $job->id,
				"assetUrl"        => $job->asset_url,
				"deliveryService" => $job->job_deliveryservice->xml_id,
				"keyword"         => $job->keyword,
				"parameters"      => $job->parameters,
				"startTime"       => $job->start_time,
				"createdBy"       => $job->job_user->username,
			}
		);
	}
	$self->success( \@data );
}

sub show {
	my $self = shift;
	my $id   = $self->param('id');

	my $jobs = $self->db->resultset("Job")->search( { 'me.id' => $id }, { prefetch => [ 'job_deliveryservice', 'job_user' ] } );
	my @data = ();
	while ( my $job = $jobs->next ) {
		push(
			@data, {
				"id"              => $job->id,
				"keyword"         => $job->keyword,
				"assetUrl"        => $job->asset_url,
				"parameters"      => $job->parameters,
				"startTime"       => $job->start_time,
				"deliveryService" => $job->job_deliveryservice->xml_id,
				"createdBy"       => $job->job_user->username,
			}
		);
	}
	$self->success( \@data );
}

sub get_current_user_jobs {
	my $self = shift;

	my $response = [];
	my $username = $self->current_user()->{username};
	my $ds_id    = $self->param('dsId');
	my $keyword  = $self->param('keyword') || 'PURGE';

	my $jobs;
	if ( defined($ds_id) ) {
		$jobs = $self->db->resultset('Job')->search(
			{ keyword  => $keyword, 'job_user.username'     => $username, 'job_deliveryservice.id' => $ds_id },
			{ prefetch => [         { 'job_deliveryservice' => undef } ], join                     => 'job_user' }
		);
		my $job_count = $jobs->count();
		if ( defined($jobs) && ( $job_count > 0 ) ) {
			my @data = $self->job_ds_data($jobs);
			my $rh   = new Utils::Helper::ResponseHelper();
			$response = $rh->camelcase_response_keys(@data);
		}
	}
	else {
		$jobs =
			$self->db->resultset('Job')
			->search( { keyword => $keyword, 'job_user.username' => $username }, { prefetch => [ { 'job_user' => undef } ], join => 'job_user' } );
		my $job_count = $jobs->count();
		if ( defined($jobs) && ( $job_count > 0 ) ) {
			my @data = $self->job_data($jobs);
			my $rh   = new Utils::Helper::ResponseHelper();
			$response = $rh->camelcase_response_keys(@data);
		}
	}

	return $self->success($response);
}

# Creates a purge job based upon the Deliveryservice (ds_id) instead
# of the ds_xml_id like the UI does.
sub create_current_user_job {
	my $self = shift;

	my $ds_id      = $self->req->json->{dsId};
	my $regex      = $self->req->json->{regex};
	my $ttl        = $self->req->json->{ttl};
	my $start_time = $self->req->json->{startTime};

	my ( $is_valid, $result ) = $self->is_valid( { dsId => $ds_id, regex => $regex, startTime => $start_time, ttl => $ttl } );
	if ( !$is_valid ) {
		return $self->alert($result);
	}

	my $ds = $self->db->resultset('Deliveryservice')->find( { id => $ds_id } );
	if ( !defined($ds) ) {
		return $self->not_found();
	}
	my $org_server_fqdn = UI::DeliveryService::compute_org_server_fqdn($self, $ds_id);
	if ( !defined($org_server_fqdn) ) {
		return $self->alert("cannot invalidate content: requested delivery service has no origin");
	}

	my $tenant_utils = Utils::Tenant->new($self);
	my $tenants_data = $tenant_utils->create_tenants_data_from_db();

	if ( $tenant_utils->use_tenancy() ) {
		if ( !$tenant_utils->is_ds_resource_accessible($tenants_data, $ds->tenant_id) ) {
			return $self->forbidden("Forbidden. Delivery-service tenant is not available to the user.");
		}
	}

	# Just pass "true" in the urgent key to make it urgent.
	my $urgent = $self->req->json->{urgent};

	my $new_id = $self->create_new_job( $ds_id, $regex, $start_time, $ttl, 'PURGE', $urgent );
	if ($new_id) {
		my $saved_job = $self->db->resultset("Job")->find( { id => $new_id } );
		my $asset_url = $saved_job->asset_url;
		my $msg = "Invalidate content request submitted for " . $ds->xml_id() . " [ $asset_url - " . $saved_job->parameters . " ]";
		&log( $self, $msg, "APICHANGE" );
		return $self->success_message( $msg );
	}
	else {
		return $self->alert( { "Error creating invalidate content request" . $ds_id } );
	}
}

sub is_valid {
	my $self = shift;
	my $job  = shift;

	my $rules = {
		fields => [qw/dsId regex startTime ttl/],

		# Checks to perform on all fields
		checks => [

			# All of these are required
			[qw/regex startTime ttl dsId/] => is_required("is required"),

			ttl => sub {
				my $value  = shift;
				my $params = shift;
				if ( defined( $params->{'ttl'} ) ) {
					return $self->is_ttl_in_range($value);
				}
			},

			startTime => sub {
				my $value  = shift;
				my $params = shift;
				if ( defined( $params->{'startTime'} ) ) {
					return $self->is_valid_date_format($value);
				}
			},

			startTime => sub {
				my $value  = shift;
				my $params = shift;
				if ( defined( $params->{'startTime'} ) ) {
					return $self->is_more_than_two_days($value);
				}
			},

		]
	};

	# Validate the input against the rules
	my $result = validate( $job, $rules );

	if ( $result->{success} ) {

		#print "success: " . dump( $result->{data} );
		return ( 1, $result->{data} );
	}
	else {

		#print "failed " . Dumper( $result->{error} );
		return ( 0, $result->{error} );
	}

}

sub is_valid_date_format {
	my $self  = shift;
	my $value = shift;
	if ( !defined $value or $value eq '' ) {
		return undef;
	}

	if (
		( $value ne '' )
		&& ( $value !~
			qr/^((((19|[2-9]\d)\d{2})[\/\.-](0[13578]|1[02])[\/\.-](0[1-9]|[12]\d|3[01])\s(0[0-9]|1[0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9]))|(((19|[2-9]\d)\d{2})[\/\.-](0[13456789]|1[012])[\/\  .-](0[1-9]|[12]\d|30)\s(0[0-9]|1[0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9]))|(((19|[2-9]\d)\d{2})[\/\.-](02)[\/\.-](0[1-9]|1\d|2[0-8])\s(0[0-9]|1[0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9]))|(((1[  6-9]|[2-9]\d)(0[48]|[2468][048]|[13579][26])|((16|[2468][048]|[3579][26])00))[\/\.-](02)[\/\.-](29)\s(0[0-9]|1[0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])))$/
		)
		)
	{
		return "has an invalidate date format, should be in the form of YYYY-MM-DD HH:MM:SS";
	}

	return undef;
}

sub is_ttl_in_range {
	my $self      = shift;
	my $value     = shift;
	my $min_hours = 1;
	my $max_days =
		$self->db->resultset('Parameter')->search( { name => "maxRevalDurationDays" }, { config_file => "regex_revalidate.config" } )->get_column('value')
		->first;
	my $max_hours = $max_days * 24;

	if ( !defined $value or $value eq '' ) {
		return undef;
	}

	if ( ( $value ne '' ) && ( $value < $min_hours || $value > $max_hours ) ) {
		return "should be between " . $min_hours . " and " . $max_hours;
	}

	return undef;
}

sub is_more_than_two_days {
	my $self  = shift;
	my $value = shift;
	if ( !defined $value or $value eq '' ) {
		return undef;
	}

	my $dh               = new Utils::Helper::DateHelper();
	my $start_time_epoch = $dh->date_to_epoch($value);
	my $date_range       = abs( $start_time_epoch - time() );
	if ( ( $value ne '' ) && ( $date_range > 172800 ) ) {
		return "needs to be within two days from now.";
	}

	return undef;
}

1;
