package main;
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
use Mojo::Base -strict;
use Test::More;
use Test::Mojo;
use DBI;
use strict;
use warnings;
no warnings 'once';
use warnings 'all';
use Test::TestHelper;

#no_transactions=>1 ==> keep fixtures after every execution, beware of duplicate data!
#no_transactions=>0 ==> delete fixtures after every execution

BEGIN { $ENV{MOJO_MODE} = "test" }

sub run_ut {
	my $t = shift;
	my $schema = shift;
	my $login_user = shift;
	my $login_password = shift;
	
	Test::TestHelper->unload_core_data($schema);
	Test::TestHelper->load_core_data($schema);

	my $tenant_id = $schema->resultset('TmUser')->find( { username => $login_user } )->get_column('tenant_id');
	my $tenant_name = defined ($tenant_id) ? $schema->resultset('Tenant')->find( { id => $tenant_id } )->get_column('name') : "null";

	
	ok $t->post_ok( '/login', => form => { u => $login_user, p => $login_password} )->status_is(302)
		->or( sub { diag $t->tx->res->content->asset->{content}; } ), 'Tenant $tenant_name: Should login?';
	
	$t->get_ok("/api/1.2/cdns")->status_is(200)->json_is( "/response/0/id", 100 )
	    ->json_is( "/response/0/name", "cdn1" )->or( sub { diag $t->tx->res->content->asset->{content}; } )
    	    ->json_is( "/response/0/tenantId", undef)
    	    ->json_is( "/response/2/name", "cdn-root" )
	    ->json_is( "/response/2/tenantId", 10**9 );
	
	$t->get_ok("/api/1.2/cdns/100")->status_is(200)->json_is( "/response/0/id", 100 )
	    ->json_is( "/response/0/name", "cdn1" )
	    ->json_is( "/response/0/tenantId", undef)
	    ->or( sub { diag $t->tx->res->content->asset->{content}; } );

	$t->get_ok("/api/1.2/cdns/300")->status_is(200)->json_is( "/response/0/id", 300 )
	    ->json_is( "/response/0/name", "cdn-root" )
	    ->json_is( "/response/0/tenantId", 10**9)
	    ->or( sub { diag $t->tx->res->content->asset->{content}; } );

	ok $t->post_ok('/api/1.2/cdns/100/queue_update' => {Accept => 'application/json'} => json => {
	            "action" => "queue" })
	        ->status_is(200)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	        ->json_is( "/response/cdnId" => 100 )
	        ->json_is( "/response/action" => "queue" )
	    , 'Tenant $tenant_name: Does the cdn details return?';

	$t->get_ok("/api/1.2/servers?cdnId=100")->status_is(200)
	    ->json_is( "/response/0/updPending", 1 )
	    ->json_is( "/response/1/updPending", 1 )
	    ->or( sub { diag $t->tx->res->content->asset->{content}; } );
	
	ok $t->post_ok('/api/1.2/cdns/100/queue_update' => {Accept => 'application/json'} => json => {
            "action" => "dequeue" })
	        ->status_is(200)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	        ->json_is( "/response/cdnId" => 100 )
	        ->json_is( "/response/action" => "dequeue" )
	    , 'Tenant $tenant_name: Does the cdn details return?';

	$t->get_ok("/api/1.2/servers?cdnId=100")->status_is(200)
    	->json_is( "/response/0/updPending", 0 )
	    ->json_is( "/response/1/updPending", 0 )
	    ->or( sub { diag $t->tx->res->content->asset->{content}; } );

	ok $t->post_ok('/api/1.2/cdns' => {Accept => 'application/json'} => json => {
        	"name" => "cdn_test", "dnssecEnabled" => "true", "tenantId" => $tenant_id })
	    ->status_is(200)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/response/name" => "cdn_test" )->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/response/tenantId", $tenant_id)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/alerts/0/level" => "success" )
	    ->json_is( "/alerts/0/text" => "cdn was created." )
	            , 'Tenant $tenant_name: Do the cdn queue update details return?';
	
	my $cdn_id = &get_cdn_id('cdn_test');

	#verify retrieved data in show
	$t->get_ok("/api/1.2/cdns/".$cdn_id)->status_is(200)
	    ->json_is( "/response/0/name", "cdn_test" )
	    ->json_is( "/response/0/tenantId", $tenant_id )
	    ->or( sub { diag $t->tx->res->content->asset->{content}; } );

	ok $t->put_ok('/api/1.2/cdns/' . $cdn_id  => {Accept => 'application/json'} => json => {
        	"name" => "cdn_test2", "dnssecEnabled" => "true", "tenantId" => $tenant_id
        	})
	    ->status_is(200)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/response/name" => "cdn_test2" )
	    ->json_is( "/response/tenantId", $tenant_id)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/alerts/0/level" => "success" )
        	    , 'Tenant $tenant_name: Does the cdn details return?';

	ok $t->delete_ok('/api/1.2/cdns/' . $cdn_id)->status_is(200)->or( sub { diag $t->tx->res->content->asset->{content}; } );

	ok $t->put_ok('/api/1.2/cdns/' . $cdn_id  => {Accept => 'application/json'} => json => {
        	"name" => "cdn_test3"
        	})
	    ->status_is(404)->or( sub { diag $t->tx->res->content->asset->{content}; } );

	#behvior when no tenant id set
	ok $t->post_ok('/api/1.2/cdns' => {Accept => 'application/json'} => json => {
        	"name" => "cdn_test", "dnssecEnabled" => "true"})
	    ->status_is(200)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/response/name" => "cdn_test" )
	    ->json_is( "/response/tenantId", $tenant_id)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/alerts/0/level" => "success" )
	    ->json_is( "/alerts/0/text" => "cdn was created." )
	            , 'Tenant $tenant_name: Do the cdn queue update details return?';
	
	$cdn_id = &get_cdn_id('cdn_test');

	ok $t->put_ok('/api/1.2/cdns/' . $cdn_id  => {Accept => 'application/json'} => json => {
        	"name" => "cdn_test2", "dnssecEnabled" => "true"
        	})
	    ->status_is(200)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/response/name" => "cdn_test2" )
	    ->json_is( "/response/tenantId", undef)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/alerts/0/level" => "success" )
        	    , 'Tenant $tenant_name: Does the cdn details return?';
        
        #putting tenancy back	    
       	ok $t->put_ok('/api/1.2/cdns/' . $cdn_id  => {Accept => 'application/json'} => json => {
        	"name" => "cdn_test2", "dnssecEnabled" => "true", "tenantId" => $tenant_id
        	})
	    ->status_is(200)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/response/name" => "cdn_test2" )
	    ->json_is( "/response/tenantId", $tenant_id)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/alerts/0/level" => "success" )
        	    , 'Tenant $tenant_name: Does the cdn details return?';

        #removing tenancy explictly
       	ok $t->put_ok('/api/1.2/cdns/' . $cdn_id  => {Accept => 'application/json'} => json => {
        	"name" => "cdn_test2", "dnssecEnabled" => "true", "tenantId" => undef
        	})
	    ->status_is(200)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/response/name" => "cdn_test2" )
	    ->json_is( "/response/tenantId", undef)->or( sub { diag $t->tx->res->content->asset->{content}; } )
	    ->json_is( "/alerts/0/level" => "success" )
        	    , 'Tenant $tenant_name: Does the cdn details return?';

	ok $t->delete_ok('/api/1.2/cdns/' . $cdn_id)->status_is(200)->or( sub { diag $t->tx->res->content->asset->{content}; } );

	ok $t->get_ok('/logout')->status_is(302)->or( sub { diag $t->tx->res->content->asset->{content}; } );
}

my $schema = Schema->connect_to_database;
my $dbh    = Schema->database_handle;
my $t      = Test::Mojo->new('TrafficOps');

run_ut($t, $schema, Test::TestHelper::ADMIN_USER,  Test::TestHelper::ADMIN_USER_PASSWORD);
run_ut($t, $schema, Test::TestHelper::ADMIN_ROOT_USER,  Test::TestHelper::ADMIN_ROOT_USER_PASSWORD);

$dbh->disconnect();
done_testing();

sub get_cdn_id {
    my $name = shift;
    my $q    = "select id from cdn where name = \'$name\'";
    my $get_svr = $dbh->prepare($q);
    $get_svr->execute();
    my $p = $get_svr->fetchall_arrayref( {} );
    $get_svr->finish();
    my $id = $p->[0]->{id};
    return $id;
}
