<!--
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
-->
# Enhanced physlocations 

## Problem Description

This blue print proposes two semi major alterations to the physical location model with ATC

* Multiple addresses per physical location
* Multiple POC per physical location

This improvement allows for TC to remain the source of truth for metadata about server locations. Specifically this enhanced functionality will allow storage/retrieval of information required for CDN hardware mailing, location specific contact management. The below solution is extensible without unnecessary complexity enabling our users to define useful server location metadata to fit their needs.

## Proposed Change
<!--
*How* will this be implemented (at a high level)?
-->

This functionality involves adjusting the representation of physical locations by altering the following components.

* Traffic Ops Database representation of physical locations
* TO Rest API physical location response value
* TP physical location data insertion views

### Traffic Portal Impact
<!--
*How* will this impact Traffic Portal?
What new UI changes will be required?
Will entirely new pages/views be necessary?
Will a new field be added to an existing form?
How will the user interact with the new UI changes?
-->

The change should only expand the number of fields required to input `N` new physical location and points of contact. New visuals could be added for displaying this additional information but I think it would clutter the table view. 

### Traffic Ops Impact
<!--
*How* will this impact Traffic Ops (at a high level)?
-->

The primarily changes take place within Traffic ops. There will be an API response change for the physical location endpoints, a database table addition and modification.

#### REST API Impact
<!--
*How* will this impact the Traffic Ops REST API?

What new endpoints will be required?
How will existing endpoints be changed?
What will the requests and responses look like?
What fields are required or optional?
What are the defaults for optional fields?
What are the validation constraints?
-->

This modification will update the `GET` response and `POST` request fields for the `/phys_locations` endpoint. It should be a breaking change modifying the names of existing fields and adding new fields.

The `GET` response will be modified as follows.

```
{ "response": [
    {
        "comments": "",
        "id": 2,
        "lastUpdated": "2018-12-05 17:50:58+00",
        "name": "CDN_in_a_Box",
        "regionId": 1,
        "region": "Washington, D.C",
        "shortName": "ciab",
        "addresses": [ {
        		"address1": "1600 Pennsylvania Avenue NW",
		 		"address2": "",
		 		"address3": "",
		 		"address4": "",
        		"locality": "Washington",
        		"administrativeArea": "DC",
        		"postalCode": "20500",
        		"longitude": -77.03650834403494,
        		"latitude": 38.897846677690815,
        		"type": "primary"
        }],
        "contacts": [ {
        		"contact": "Joe Schmo",
        		"method": "phone",
        		"value": "800-555-1234",
        		"comments": "Security Guard",
			"priority": 1,
			"purpose": "controls access to physical location"
        }]
    }
]}
```

The `POST` request will look very similar. 

```
{
    "comments": "",
    "name": "CDN_in_a_Box",
    "regionId": 1,
    "region": "Washington, D.C",
    "shortName": "ciab",
    "addresses": [ {
    		"address1": "1600 Pennsylvania Avenue NW",
	 		"address2": "",
	 		"address3": "",
	 		"address4": "",
    		"locality": "Washington",
    		"administrativeArea": "DC",
    		"postalCode": "20500",
    		"longitude": -77.03650834403494,
    		"latitude": 38.897846677690815,
    		"type": "primary"
    }],
    "contacts": [ {
    		"contact": "Joe Schmo",
     		"method": "phone",
    		"value": "800-555-1234",
    		"comments": "Security Guard",
		"priority": 1,
		"purpose": "controls access to physical location"
    }]
}

```

For the POST request body it will be important to validate the longitude and latitude as sensible values.

Some of the address lines can be optional.

The type and method fields could probably be enumeration types. But I am not sure we understand/want to enforce all of the possible use cases so it may be important to limit the input text length. 

It may be desirable to update the address and contacts without re-POSTing all of the above data. I am not sure there is enough necessity to create this endpoint.

#### Client Impact
<!--
*How* will this impact Traffic Ops REST API clients (Go, Python, Java)?

If new endpoints are required, will corresponding client methods be added?
-->

I am unaware of the client use of phys_location but they will most likely need to be updated to support the new formats. 

#### Data Model / Database Impact
<!--
*How* will this impact the Traffic Ops data model?
*How* will this impact the Traffic Ops database schema?

What changes to the lib/go-tc structs will be required?
What new tables and columns will be required?
How will existing tables and columns be changed?
What are the column data types and modifiers?
What are the FK references and constraints?
-->

Firstly the phys_location table will be altered. Simply dropping many of to be duplicated columns.


```
                        Table "traffic_ops.phys_location"
      Column      |           Type           | Collation | Nullable | Default
------------------+--------------------------+-----------+----------+---------
id                | bigint                   |           | NOT NULL | 
name              | text                     |           | NOT NULL |
short_name        | text                     |           | NOT NULL | 
region            | bigint                   |           | NOT NULL | 
last_updated      | timestamp with time zone |           | NOT NULL | now()

Indexes:    
	idx_89655_primary PRIMARY KEY (id)
	idx_89655_fk_phys_location_region_idx ON phys_location USING btree (region)
	idx_89655_name_unique ON phys_location USING btree (name);
	idx_89655_short_name_unique ON phys_location USING btree (short_name);
Foreign-key constraints:
	fk_phys_location_region FOREIGN KEY (region) REFERENCES region(id);
	
```

Specifically the following columns have been dropped.

* address
* city    
* state 
* zip 
* poc 
* phone 
* email
* comments

The first new table stores sets of addresses. This adds 3 extra address lines and changes the terminology for (state -> adminstrativeArea, city -> locality, zip -> postalCode), this is more friendly to international addresses. The other notable change is the addition of the type column. This is to be used for differentiating address. All locations in the phy_location table should have at least one full address of type `primary`, others could include a hardware mailing address or a contact address

```
                        Table "traffic_ops.location_addresses"
      Column      |           Type           | Collation | Nullable | Default
------------------+--------------------------+-----------+----------+---------
id                | bigint                   |           | NOT NULL | 
addressLine1      | text					 |           | NOT NULL | 
addressLine2      | text					 |           | NOT NULL | 
addressLine3      | text					 |           | NOT NULL | 
addressLine4      | text					 |           | NOT NULL | 
country           | text					 |           | NOT NULL | 
administrativeArea| text					 |           | NOT NULL | 
longitude         | bigfloat                 |           | NOT NULL | 
latitude          | bigfloat                 |           | NOT NULL | 
locality          | text                     |           | NOT NULL |
postalCode        | text                     |           | NOT NULL |
type              | text                     |           | NOT NULL | 
comments          | text                     |           |          |  
location          | bigint                   |           | NOT NULL | 
last_updated      | timestamp with time zone |           | NOT NULL | now()

Indexes:  
		idx_89560_primary PRIMARY KEY (id)
Foreign-key constraints:
	fk_address_location FOREIGN KEY (location) REFERENCES phys_location(id);
```

The second new table stores sets of contacts.  A contact stores a contact identifier ie (persons name), a method which describes how you are contacting them ie (email, cell, work phone, distribution list), and a value the actual way of communicating ie (the phone number or the email address)

```
                        Table "traffic_ops.location_contacts"
      Column      |           Type           | Collation | Nullable | Default
------------------+--------------------------+-----------+----------+---------
id                | bigint                   |           | NOT NULL | 
location          | bigint                   |           | NOT NULL | 
contact           | text                     |           | NOT NULL | 
method            | text                     |           | NOT NULL | 
value             | text                     |           | NOT NULL | 
comments          | text                     |           |          |   
priority          | int                      |           | NOT NULL | 
purpose           | text                     |           | NOT NULL | 
last_updated      | timestamp with time zone |           | NOT NULL | now()


Indexes:  
		idx_89560_primary PRIMARY KEY (id)
Foreign-key constraints:
	fk_contact_location FOREIGN KEY (location) REFERENCES phys_location(id);
```


### Documentation Impact
<!--
*How* will this impact the documentation?

What new documentation will be required?
What existing documentation will need to be updated?
-->

Minor update to the phys_location documentation

### Testing Impact
<!--
*How* will this impact testing?

What is the high-level test plan?
How should this be tested?
Can this be tested within the existing test frameworks?
How should the existing frameworks be enhanced in order to test this properly?
-->

TO unit tests will be added with the addition fields in the API.

### Performance Impact
<!--
*How* will this impact performance?

Are the changes expected to improve performance in any way?
Is there anything particularly CPU, network, or storage-intensive to be aware of?
What are the known bottlenecks to be aware of that may need to be addressed?
-->

My suspicion is this should not have a performance impact but I have no hard evidence at this time

### Security Impact
<!--
*How* will this impact overall security?

Are there any security risks to be aware of?
What privilege level is required for these changes?
Do these changes increase the attack surface (e.g. new untrusted input)?
How will untrusted input be validated?
If these changes are used maliciously or improperly, what could go wrong?
Will these changes adhere to multi-tenancy?
Will data be protected in transit (e.g. via HTTPS or TLS)?
Will these changes require sensitive data that should be encrypted at rest?
Will these changes require handling of any secrets?
Will new SQL queries properly use parameter binding?
-->

There will be new SQL queries which will have to be evaluated for security concerns

### Upgrade Impact
<!--
*How* will this impact the upgrade of an existing system?

Will a database migration be required?
Do the various components need to be upgraded in a specific order?
Will this affect the ability to rollback an upgrade?
Are there any special steps to be followed before an upgrade can be done?
Are there any special steps to be followed during the upgrade?
Are there any special steps to be followed after the upgrade is complete?
-->

This will require a database migration. The design of this solution was intended to not impact other tables. We have to re-write the phys_location table. My recommendation would be to duplicate the location and POC data into their new locations first, temporarily duplicating data. Then drop those fields from the phys_location field once we have verified the new data is being correctly represented.  

### Developer Impact
<!--
*How* will this impact other developers?

Will it make it easier to set up a development environment?
Will it make the code easier to maintain?
What do other developers need to know about these changes?
Are the changes straightforward, or will new developer instructions be necessary?
-->

I would argue the complexity of this solution is negligible 


## Alternatives
<!--
What are some of the alternative solutions for this problem?
What are the pros and cons of each approach?
What design trade-offs were made and why?
-->

The minimal use case for this solution would be simply extending the phys_location table to contain a more rich set of mailing address fields and POC fields.

Pro's to Alternative: 

* Simple to implement, minimal code change

Con's to Alternative:

* Less extensible, limits our ability to represent arbitrary geo-location data about our servers. 


