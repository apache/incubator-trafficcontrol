package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func checkErr(err error, txn *sql.Tx) {
	if err != nil {
		fmt.Println("Error 33:", err)
		fmt.Println("Attempting Roll back!")
		txn.Rollback()
		os.Exit(1)
	}
}

func doExec(stmt string, txn *sql.Tx) {
	fmt.Println("  " + stmt)
	_, err := txn.Exec(stmt)
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Attempting Roll back!")
		txn.Rollback()
		os.Exit(1)
	}
}

// Up is executed when this migration is applied
func Up_20170205101432(txn *sql.Tx) {
	fmt.Println("  Starting migration 20130106222315...")
	doExec("CREATE TYPE profile_type AS ENUM ("+
		"'ATS_PROFILE', 'TR_PROFILE', 'TM_PROFILE', 'TS_PROFILE', 'TP_PROFILE', 'INFLUXDB_PROFILE',"+
		"'RIAK_PROFILE', 'SPLUNK_PROFILE', 'DS_PROFILE', 'ORG_PROFILE', 'KAFKA_PROFILE', 'LOGSTASH_PROFILE',"+
		"'ES_PROFILE', 'UNK_PROFILE')", txn)

	doExec("CREATE OR REPLACE VIEW \"profile_type_values\" AS SELECT unnest(enum_range(NULL::profile_type )) AS value ORDER BY value", txn)

	doExec("ALTER TABLE public.profile ADD COLUMN type profile_type", txn)
	doExec("UPDATE public.profile SET type='UNK_PROFILE'", txn) // So we don't get any NULL, these should be checked.
	doExec("UPDATE public.profile SET type='TR_PROFILE' WHERE name like 'CCR_%' OR name like 'TR_%'", txn)
	doExec("UPDATE public.profile SET type='TM_PROFILE' WHERE name like 'RASCAL_%' OR name like 'TM_%'", txn)
	doExec("UPDATE public.profile SET type='TS_PROFILE' WHERE name like 'TRAFFIC_STATS%'", txn)
	doExec("UPDATE public.profile SET type='TP_PROFILE' WHERE name like 'TRAFFIC_PORTAL%'", txn)
	doExec("UPDATE public.profile SET type='INFLUXDB_PROFILE' WHERE name like 'INFLUXDB%'", txn)
	doExec("UPDATE public.profile SET type='RIAK_PROFILE' WHERE name like 'RIAK%'", txn)
	doExec("UPDATE public.profile SET type='SPLUNK_PROFILE' WHERE name like 'SPLUNK%'", txn)
	doExec("UPDATE public.profile SET type='ORG_PROFILE' WHERE name like '%ORG%' or name like 'MSO%' or name like '%ORIGIN%'", txn)
	doExec("UPDATE public.profile SET type='KAFKA_PROFILE' WHERE name like 'KAFKA%'", txn)
	doExec("UPDATE public.profile SET type='LOGSTASH_PROFILE' WHERE name like 'LOGSTASH_%'", txn)
	doExec("UPDATE public.profile SET type='ES_PROFILE' WHERE name like 'ELASTICSEARCH%'", txn)
	doExec("UPDATE public.profile SET type='ATS_PROFILE' WHERE name like 'EDGE%' or name like 'MID%'", txn)

	doExec("ALTER TABLE public.profile ALTER type SET NOT NULL", txn)

	doExec("ALTER TABLE public.cdn ADD COLUMN domain_name text", txn)

	doExec("UPDATE cdn SET domain_name=domainlist.value "+
		"FROM (SELECT distinct cdn_id,value FROM server,parameter WHERE type=(SELECT id FROM type WHERE name='EDGE') "+
		"AND parameter.id in (select parameter from profile_parameter WHERE profile_parameter.profile=server.profile) "+
		"AND parameter.name='domain_name' "+
		"AND config_file='CRConfig.json') AS domainlist "+
		"WHERE id = domainlist.cdn_id", txn)

	doExec("UPDATE public.cdn SET domain_name='-' WHERE name='ALL'", txn)

	doExec("ALTER TABLE public.cdn ALTER COLUMN domain_name SET NOT NULL", txn)

	doExec("ALTER TABLE public.profile ADD COLUMN cdn bigint", txn)

	doExec("ALTER TABLE public.profile "+
		"ADD CONSTRAINT fk_cdn1 FOREIGN KEY (cdn) "+
		"REFERENCES public.cdn (id) MATCH SIMPLE "+
		"ON UPDATE RESTRICT ON DELETE RESTRICT", txn)

	doExec("CREATE INDEX idx_181818_fk_cdn1 "+
		"ON public.profile "+
		"USING btree "+
		"(cdn)", txn)

	doExec("UPDATE profile set cdn=domainlist.cdn_id "+
		"FROM (SELECT distinct profile.id AS profile_id, value AS profile_domain_name, cdn.id cdn_id "+
		"FROM profile, parameter, cdn, profile_parameter "+
		"WHERE parameter.name='domain_name' "+
		"AND parameter.config_file='CRConfig.json' "+
		"AND parameter.value = cdn.domain_name "+
		"AND parameter.id in (select parameter from profile_parameter where profile=profile.id)) as domainlist "+
		"WHERE id = domainlist.profile_id", txn)

	doExec("ALTER TABLE deliveryservice ALTER profile DROP NOT NULL", txn)

	doExec("UPDATE deliveryservice SET profile=NULL", txn)

	type Profile struct {
		Id                 int64
		Name               string
		Desc               string
		Type               string
		Cdn                int64
		MidHeaderRewrite   string
		MultiSiteOriginAlg int64
		XMLId              string
	}
	// move data
	pmap := make(map[string]Profile)
	rows, err := txn.Query("select id,xml_id,mid_header_rewrite,multi_site_origin,multi_site_origin_algorithm,cdn_id from deliveryservice where multi_site_origin=true")
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Attempting Roll back!")
		txn.Rollback()
		os.Exit(1)
	}
	existingParam := make(map[string]int64)
	for rows.Next() {
		var (
			id                          int64
			xml_id                      string
			mid_header_rewrite          sql.NullString
			multi_site_origin           sql.NullBool
			multi_site_origin_algorithm sql.NullInt64
			cdn_id                      int64
		)
		err := rows.Scan(&id, &xml_id, &mid_header_rewrite, &multi_site_origin, &multi_site_origin_algorithm, &cdn_id)
		checkErr(err, txn)

		pName := "DS_" + xml_id
		pDesc := "Deliveryservice profile for " + xml_id
		pType := "DS_PROFILE"
		mhrString := mid_header_rewrite.String
		pmap[pName] = Profile{
			Id:                 -1,
			Name:               pName,
			Desc:               pDesc,
			Type:               pType,
			Cdn:                cdn_id,
			MidHeaderRewrite:   mhrString,
			MultiSiteOriginAlg: multi_site_origin_algorithm.Int64,
			XMLId:              xml_id,
		}
	}
	err = rows.Err()
	checkErr(err, txn)
	rows.Close()

	for _, prof := range pmap {
		fmt.Println("--\nINSERT INTO PROFILE (name, description, type, cdn) VALUES($1, $2, $3, $4) RETURNING id", prof.Name, prof.Desc, prof.Type, prof.Cdn)
		newRow := txn.QueryRow("INSERT INTO PROFILE (name, description, type, cdn) VALUES($1, $2, $3, $4) RETURNING id", prof.Name, prof.Desc, prof.Type, prof.Cdn)
		var newProfileId int64
		err := newRow.Scan(&newProfileId)
		checkErr(err, txn)

		remainingString := ""
		var regexpOne = regexp.MustCompile(`^\s*set-config\s+proxy.config.http.parent_origin.(\S+)\s+(.*)$`)
		if prof.MidHeaderRewrite != "" {
			remapParts := strings.Split(prof.MidHeaderRewrite, "__RETURN__")
			for _, line := range remapParts {
				fmt.Println(line)
				match := regexpOne.FindSubmatch([]byte(line))
				if len(match) != 0 {
					var newId int64
					var ok bool
					newId, ok = existingParam[string(match[1])+string(match[2])]
					fmt.Printf("%s -> %v %v\n", string(match[1])+string(match[2]), newId, ok)
					if !ok {
						fmt.Println("INSERT INTO PARAMETER (name, config_file, value) VALUES ($1, $2, $3) RETURNING id", "mso."+
							string(match[1]), "parent.config", string(match[2]))
						newRow := txn.QueryRow("INSERT INTO PARAMETER (name, config_file, value) VALUES ($1, $2, $3) RETURNING id", "mso."+
							string(match[1]), "parent.config", string(match[2]))
						err := newRow.Scan(&newId)
						checkErr(err, txn)
						existingParam[string(match[1])+string(match[2])] = newId
					} else {
						newId = existingParam[string(match[1])+string(match[2])]
					}
					fmt.Println("INSERT INTO PROFILE_PARAMETER (parameter, profile) VALUES ($1, $2)", newId, newProfileId)
					_, err = txn.Exec("INSERT INTO PROFILE_PARAMETER (parameter, profile) VALUES ($1, $2)", newId, newProfileId)
					checkErr(err, txn)
				} else {
					if !strings.HasSuffix(line, "__RETURN__") {
						remainingString = remainingString + line + "__RETURN__"
					}
				}
			}
		}
		fmt.Printf("MHRW was: %s, \nMHRW now: %s\n", prof.MidHeaderRewrite, remainingString)

		_, err = txn.Exec("UPDATE deliveryservice set mid_header_rewrite=$1 where xml_id=$2", remainingString, prof.XMLId)
		checkErr(err, txn)

		var newId int64
		var ok bool
		newId, ok = existingParam["mso.algorithm"+"parent.config"+string(prof.MultiSiteOriginAlg)]
		if !ok {
			fmt.Println("INSERT INTO PARAMETER (name, config_file, value) VALUES ($1, $2, $3) RETURNING id",
				"mso.algorithm", "parent.config", prof.MultiSiteOriginAlg)
			newRow = txn.QueryRow("INSERT INTO PARAMETER (name, config_file, value) VALUES ($1, $2, $3) RETURNING id",
				"mso.algorithm", "parent.config", prof.MultiSiteOriginAlg)
			err = newRow.Scan(&newId)
			checkErr(err, txn)
			existingParam["mso.algorithm"+"parent.config"+string(prof.MultiSiteOriginAlg)] = newId
		} else {
			newId = existingParam["mso.algorithm"+"parent.config"+string(prof.MultiSiteOriginAlg)]
		}

		fmt.Println("INSERT INTO PROFILE_PARAMETER (parameter, profile) VALUES ($1, $2)", newId, newProfileId)
		_, err = txn.Exec("INSERT INTO PROFILE_PARAMETER (parameter, profile) VALUES ($1, $2)", newId, newProfileId)
		checkErr(err, txn)
		_, err = txn.Exec("UPDATE deliveryservice SET profile=$1 WHERE xml_id=$2", newProfileId, prof.XMLId)
		checkErr(err, txn)
	}
}

// Down is executed when this migration is rolled back
func Down_20170205101432(txn *sql.Tx) {
	//
}
