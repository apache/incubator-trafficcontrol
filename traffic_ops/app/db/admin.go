/*

Admin automates interactions with the Traffic Ops database.

Example:  admin --env=test reset

Purpose:  This script is used to manage database. The environments are defined in the dbconf.yml, as well as the database names.

NOTE:
Postgres Superuser: The 'postgres' superuser needs to be created to run ` + programName + ` and setup databases.
If the 'postgres' superuser has not been created or password has not been set then run the following commands accordingly.

Create the 'postgres' user as a super user (if not created):

	$ createuser postgres --superuser --createrole --createdb --login --pwprompt

Modify your ~/.pgpass file which allows for easy command line access by defaulting the user and password for the database
without prompts.

Postgres .pgpass file format:

	hostname:port:database:username:password

Example Contents:

	*:*:*:postgres:your-postgres-password
	*:*:*:traffic_ops:the-password-in-dbconf.yml

Save the following example into ~/.pgpass with the permissions of this file so only your user can read and write.

	$ chmod 0600 ~/.pgpass

*/
package main

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

// DBConfig holds configuration information for all the different environments.
type DBConfig struct {
	Development GooseConfig `yaml:"development"`
	Test        GooseConfig `yaml:"test"`
	Integration GooseConfig `yaml:"integration"`
	Production  GooseConfig `yaml:"production"`
}

// GooseConfig contains configuration information for a particular environment.
type GooseConfig struct {
	// Driver describes the type of database to which to connect. Only
	// 'postgres' is supported.
	Driver string `yaml:"driver"`
	// Open is a string containing enough information to connect to and
	// authenticate with the database.
	Open string `yaml:"open"`
}

func (conf DBConfig) getGooseConfig(env string) (GooseConfig, error) {
	switch env {
	case EnvDevelopment:
		return conf.Development, nil
	case EnvTest:
		return conf.Test, nil
	case EnvIntegration:
		return conf.Integration, nil
	case EnvProduction:
		return conf.Production, nil
	default:
		return GooseConfig{}, errors.New("invalid environment: " + env)
	}
}

// The possible environments that may be used.
const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvIntegration = "integration"
	EnvProduction  = "production"
)

// Keys in the goose config's "open" string value.
const (
	HostKey     = "host"
	PortKey     = "port"
	UserKey     = "user"
	PasswordKey = "password"
	DBNameKey   = "dbname"
)

// Available commands.
const (
	CmdCreateDB      = "createdb"
	CmdDropDB        = "dropdb"
	CmdCreateUser    = "create_user"
	CmdDropUser      = "drop_user"
	CmdShowUsers     = "show_users"
	CmdReset         = "reset"
	CmdUpgrade       = "upgrade"
	CmdMigrate       = "migrate"
	CmdDown          = "down"
	CmdRedo          = "redo"
	CmdStatus        = "status"
	CmdDBVersion     = "dbversion"
	CmdSeed          = "seed"
	CmdLoadSchema    = "load_schema"
	CmdReverseSchema = "reverse_schema"
	CmdPatch         = "patch"
)

// Goose commands that don't match the commands for this tool.
const (
	GooseUp = "up"
)

// Various filepaths - relative to {{Traffic Ops installation dir}}/app/.
const (
	DBConfigPath  = "dbconf.yml"
	DBSeedsPath   = "seeds.sql"
	DBSchemaPath  = "create_tables.sql"
	DBPatchesPath = "patches.sql"
)

// DBPath is the path to the database directory containing configuration and
// seeds/patches/creation scripts.
var DBPath = "db"

// Defaults for configurable values.
const (
	DefaultEnvironment = EnvDevelopment
	DefaultDBSuperUser = "postgres"
)

// Globals that are passed in via CLI flags and used in commands.
var (
	Environment string
	UseSqitch   bool
)

// Globals that are parsed out of DBConfigFile and used in commands.
var (
	DBName      string
	DBSuperUser = DefaultDBSuperUser
	DBUser      string
	DBPassword  string
	HostIP      string
	HostPort    string
)

func parseDBConfig() error {
	configPath := strings.Join([]string{DBPath, DBConfigPath}, "/")
	confBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading DB conf '%s': %v", configPath, err)
	}

	dbConfig := DBConfig{}
	err = yaml.Unmarshal(confBytes, &dbConfig)
	if err != nil {
		return errors.New("unmarshalling DB conf yaml: " + err.Error())
	}

	gooseCfg, err := dbConfig.getGooseConfig(Environment)
	if err != nil {
		return errors.New("getting goose config: " + err.Error())
	}

	// parse the 'open' string into a map
	open := make(map[string]string)
	pairs := strings.Split(gooseCfg.Open, " ")
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		kv := strings.Split(pair, "=")
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			continue
		}
		open[kv[0]] = kv[1]
	}

	ok := false
	HostIP, ok = open[HostKey]
	if !ok {
		return errors.New("unable to get '" + HostKey + "' for environment '" + Environment + "'")
	}
	HostPort, ok = open[PortKey]
	if !ok {
		return errors.New("unable to get '" + PortKey + "' for environment '" + Environment + "'")
	}
	DBUser, ok = open[UserKey]
	if !ok {
		return errors.New("unable to get '" + UserKey + "' for environment '" + Environment + "'")
	}
	DBPassword, ok = open[PasswordKey]
	if !ok {
		return errors.New("unable to get '" + PasswordKey + "' for environment '" + Environment + "'")
	}
	DBName, ok = open[DBNameKey]
	if !ok {
		return errors.New("unable to get '" + DBNameKey + "' for environment '" + Environment + "'")
	}

	return nil
}

func targetURI() string {
	return fmt.Sprintf("db:pg://%s:%s@%s:%s/%s", DBSuperUser, DBPassword, HostIP, HostPort, DBName)
}

func createDB() {
	dbExistsCmd := exec.Command("psql", "-h", HostIP, "-U", DBSuperUser, "-p", HostPort, "-tAc", "SELECT 1 FROM pg_database WHERE datname='"+DBName+"'")
	out, err := dbExistsCmd.Output()
	if err != nil {
		die("unable to check if DB already exists: " + err.Error())
	}
	if len(out) > 0 {
		fmt.Println("Database " + DBName + " already exists")
		return
	}
	createDBCmd := exec.Command("createdb", "-h", HostIP, "-p", HostPort, "-U", DBSuperUser, "-e", "--owner", DBUser, DBName)
	out, err = createDBCmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create db '%s': %v\n", DBName, err)
		os.Exit(1)
	}
}

func dropDB() {
	fmt.Println("Dropping database: " + DBName)
	cmd := exec.Command("dropdb", "-h", HostIP, "-p", HostPort, "-U", DBSuperUser, "-e", "--if-exists", DBName)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't drop db '%s': %v\n", DBName, err)
		os.Exit(1)
	}
}

func createUser() {
	fmt.Println("Creating user: " + DBUser)
	userExistsCmd := exec.Command("psql", "-h", HostIP, "-U", DBSuperUser, "-p", HostPort, "-tAc", "SELECT 1 FROM pg_roles WHERE rolname='"+DBUser+"'")
	out, err := userExistsCmd.Output()
	if err != nil {
		die("unable to check if user already exists: " + err.Error())
	}
	if len(out) > 0 {
		fmt.Println("User " + DBUser + " already exists")
		return
	}
	createUserCmd := exec.Command("psql", "-h", HostIP, "-p", HostPort, "-U", DBSuperUser, "-etAc", "CREATE USER "+DBUser+" WITH LOGIN ENCRYPTED PASSWORD '"+DBPassword+"'")
	out, err = createUserCmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create user '%s': %v\n", DBUser, err)
		os.Exit(1)
	}
}

func dropUser() {
	cmd := exec.Command("dropuser", "-h", HostIP, "-p", HostPort, "-U", DBSuperUser, "-i", "-e", DBUser)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't drop user '%s': %v\n", DBUser, err)
		os.Exit(1)
	}
}

func showUsers() {
	cmd := exec.Command("psql", "-h", HostIP, "-p", HostPort, "-U", DBSuperUser, "-ec", `\du`)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't show users: %v\n", err)
	}
}

func reset() {
	createUser()
	dropDB()
	createDB()
	loadSchema()
	migrate()
}

func upgrade() {
	if UseSqitch {
		sqitch("deploy")
	} else {
		goose(GooseUp)
	}
	seed()
	patch()
}

func migrate() {
	if UseSqitch {
		sqitch("deploy")
	} else {
		goose(GooseUp)
	}
}

func down() {
	if UseSqitch {
		sqitch("revert")
	} else {
		goose(CmdDown)
	}
}

func redo() {
	goose(CmdRedo)
}

func status() {
	if UseSqitch {
		sqitch("status")
	} else {
		goose(CmdStatus)
	}
}

func dbVersion() {
	goose(CmdDBVersion)
}

func seed() {
	fmt.Println("Seeding database w/ required data.")
	seedsPath := strings.Join([]string{DBPath, DBSeedsPath}, "/")
	seedsBytes, err := ioutil.ReadFile(seedsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read '%s': %v\n", seedsPath, err)
		os.Exit(1)
	}
	cmd := exec.Command("psql", "-h", HostIP, "-p", HostPort, "-d", DBName, "-U", DBUser, "-e", "-v", "ON_ERROR_STOP=1")
	cmd.Stdin = bytes.NewBuffer(seedsBytes)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+DBPassword)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't patch database w/ required data: %v\n", err)
		os.Exit(1)
	}
}

func loadSchema() {
	fmt.Println("Creating database tables.")
	schemaPath := strings.Join([]string{DBPath, DBSchemaPath}, "/")
	schemaBytes, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read '%s': %v\n", schemaPath, err)
		os.Exit(1)
	}
	cmd := exec.Command("psql", "-h", HostIP, "-p", HostPort, "-d", DBName, "-U", DBUser, "-e", "-v", "ON_ERROR_STOP=1")
	cmd.Stdin = bytes.NewBuffer(schemaBytes)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+DBPassword)
	out, err := cmd.CombinedOutput()
	fmt.Printf("%s", out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't create database tables: %v\n", err)
		os.Exit(1)
	}
}

func reverseSchema() {
	fmt.Fprintf(os.Stderr, "WARNING: the '%s' command will be removed with Traffic Ops Perl because it will no longer be necessary\n", CmdReverseSchema)
	reversePath := strings.Join([]string{DBPath, "reverse_schema.pl"}, "/")
	cmd := exec.Command(reversePath)
	cmd.Env = append(os.Environ(), "MOJO_MODE="+Environment)
	out, err := cmd.CombinedOutput()
	fmt.Printf("%s", out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't run `%s`: %v\n", reversePath, err)
	}
}

func patch() {
	fmt.Println("Patching database with required data fixes.")
	patchesPath := strings.Join([]string{DBPath, DBPatchesPath}, "/")
	patchesBytes, err := ioutil.ReadFile(patchesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read '%s': %v\n", patchesPath, err)
	}
	cmd := exec.Command("psql", "-h", HostIP, "-p", HostPort, "-d", DBName, "-U", DBUser, "-e", "-v", "ON_ERROR_STOP=1")
	cmd.Stdin = bytes.NewBuffer(patchesBytes)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+DBPassword)
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't patch database w/ required data: %v\n", err)
		os.Exit(1)
	}
}

func verify() {
	if !UseSqitch {
		die("This command can only be invoked with --use-sqitch")
	}
	sqitch("verify")
}

func goose(arg string) {
	fmt.Println("Running goose " + arg + "...")
	cmd := exec.Command("goose", "--env="+Environment, arg)
	out, err := cmd.CombinedOutput()
	fmt.Printf("%s", out)
	if err != nil {
		die("Can't run goose: " + err.Error())
	}
}

func sqitch(arg string) {
	fmt.Printf("Running: sqitch %s\n", arg)
	cmd := exec.Command("sqitch", arg, targetURI())
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't run sqitch: %v", err)
		os.Exit(1)
	}
}

func die(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

// Command represents a command that may be invoked. The name of the command is
// given by its key in Commands.
type Command struct {
	// A function that runs the command's logic.
	Function func()
	// A human-readable string that describes what the command does.
	Help string
}

// Commands contains all of the available commands that may be run.
var Commands = map[string]Command{
	"createdb": {
		Function: createDB,
		Help:     "Execute db 'createdb' the database for the current environment.",
	},
	"create_user": {
		Function: createUser,
		Help:     "Execute 'create_user' the user for the current environment (traffic_ops).",
	},
	"dropdb": {
		Function: dropDB,
		Help:     "Execute db 'dropdb' on the database for the current environment.",
	},
	"down": {
		Function: down,
		Help:     "Roll back a single migration from the current version.",
	},
	"drop_user": {
		Function: dropUser,
		Help:     "Execute 'drop_user' the user for the current environment (traffic_ops).",
	},
	"patch": {
		Function: patch,
		Help:     "Execute sql from db/patches.sql for loading post-migration data patches.",
	},
	"redo": {
		Function: redo,
		Help:     "Roll back the most recently applied migration, then run it again.",
	},
	"reset": {
		Function: reset,
		Help:     "Execute db 'dropdb', 'createdb', load_schema, migrate on the database for the current environment.",
	},
	"reverse_schema": {
		Function: reverseSchema,
		Help:     "Reverse engineer the lib/Schema/Result files from the environment database.",
	},
	"seed": {
		Function: seed,
		Help:     "Execute sql from db/seeds.sql for loading static data.",
	},
	"show_users": {
		Function: showUsers,
		Help:     "Execute sql to show all of the user for the current environment.",
	},
	"status": {
		Function: status,
		Help:     "Print the status of all migrations.",
	},
	"upgrade": {
		Function: upgrade,
		Help:     "Execute migrate, seed, and patches on the database for the current environment.",
	},
	"migrate": {
		Function: migrate,
		Help:     "Execute migrate (without seeds or patches) on the database for the current environment.",
	},
	"verify": {
		Function: verify,
		Help:     "Run database verification tests",
	},
}

func usage() {
	fmt.Printf("Usage: %s [--env ENVIRONMENT] [--use-sqitch] COMMAND\n", os.Args[0])
	fmt.Printf("       %s --help\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Println("  COMMAND")
	fmt.Println("        An operation to perform on the database. One of the following:")
	for cmdName, cmd := range Commands {
		fmt.Printf("          %s\n            %s\n", cmdName, cmd.Help)
	}
}

func main() {
	flag.CommandLine.SetOutput(os.Stdout)
	flag.StringVar(&Environment, "env", DefaultEnvironment, "The environment to use (defined in "+DBConfigPath+")")
	flag.StringVar(&DBPath, "db-path", "db", "Path to the database directory (containing create_tables.sql)")
	flag.BoolVar(&UseSqitch, "use-sqitch", false, "Use Sqitch instead of goose (applies only Sqitch migrations!)")
	help := flag.Bool("help", false, "Print usage information and exit")
	flag.Usage = usage
	flag.Parse()
	if help != nil && *help {
		flag.Usage()
		return
	}
	if flag.NArg() != 1 || flag.Arg(0) == "" {
		flag.Usage()
		os.Exit(1)
	}
	if Environment == "" {
		flag.Usage()
		os.Exit(1)
	}
	if err := parseDBConfig(); err != nil {
		die(err.Error())
	}

	userCmd := flag.Arg(0)
	if cmd, ok := Commands[userCmd]; ok {
		cmd.Function()
	} else {
		flag.Usage()
		die("invalid command: " + userCmd)
	}
}
