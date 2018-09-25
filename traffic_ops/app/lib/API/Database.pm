package API::Database;
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

use Mojo::Base 'Mojolicious::Controller';
use Data::Dumper;
use UI::Utils;
use IO::Compress::Gzip qw(gzip $GzipError);

sub dbdump {
	my $self = shift;
	my $filename = $self->get_filename();

	my ($db_name, $host, $port) = $Schema::dsn =~ /:database=([^;]*);host=([^;]+);port=(\d+)/;
	my $db_user = $Schema::user;
	my $db_pass = $Schema::pass;

	my $ok = open my $fh, '-|', "PGPASSWORD=\"$db_pass\" pg_dump -b -Fc --no-owner -h $host -p $port -U $db_user -d $db_name";
	if (! $ok ) {
		$self->internal_server_error( { Error => "Error dumping database" } );	
		return;
	}

	# slurp it in..
	undef $/;
	my $data = <$fh>;


	$self->res->headers->content_type("application/download");
	$self->res->headers->content_disposition( "attachment; filename=\"" . $filename . "\"" );
	$self->render( data => $data );
	close $fh;
}

sub get_filename {
	my ( $sec, $min, $hour, $day, $month, $year ) = (localtime)[ 0, 1, 2, 3, 4, 5 ];
    $month = sprintf '%02d', $month + 1;
    $day   = sprintf '%02d', $day;
    $hour  = sprintf '%02d', $hour;
    $min   = sprintf '%02d', $min;
    $sec   = sprintf '%02d', $sec;
    $year += 1900;
    my $host = `hostname`;
    chomp($host);

    my $extension = ".pg_dump";
    my $filename = "to-backup-" . $host . "-" . $year . $month . $day . $hour . $min . $sec . $extension;
    return $filename;
}

1;
