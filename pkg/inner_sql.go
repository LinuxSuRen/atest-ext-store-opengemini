/*
Copyright 2025 API Testing Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pkg

import (
	"fmt"
	"strings"
)

type InnerSQL interface {
	ToNativeSQL(query string) string
}

const (
	InnerSelectTable_      = "@selectTable_"
	InnerSelectTableLimit_ = "@selectTableLImit100_"
	InnerDescribeTable_    = "@describeTable_"
	InnerShowDatabases     = "@showDatabases"
	InnerShowTables        = "@showTables"
	InnerCurrentDB         = "@currentDB"
)

func GetInnerSQL(dialect string) InnerSQL {
	switch dialect {
	case DialectorPostgres:
		return &postgresDialect{}
	case Dialectoropengemini:
		return &opengeminindraDialect{}
	default:
		return &opengeminindraDialect{}
	}
}

type mysqlDialect struct{}

func (m *mysqlDialect) ToNativeSQL(query string) (sql string) {
	if strings.HasPrefix(query, InnerSelectTable_) {
		sql = "SELECT * FROM " + strings.ReplaceAll(query, InnerSelectTable_, "")
	} else if strings.HasPrefix(query, InnerSelectTableLimit_) {
		sql = "SELECT * FROM " + strings.ReplaceAll(query, InnerSelectTableLimit_, "") + " LIMIT 100"
	} else if query == InnerShowDatabases {
		sql = "SHOW DATABASES"
	} else if query == InnerShowTables {
		sql = "SHOW TABLES"
	} else if query == InnerCurrentDB {
		sql = "SELECT DATABASE() as name"
	} else {
		sql = query
	}
	return
}

type postgresDialect struct{}

func (p *postgresDialect) ToNativeSQL(query string) (sql string) {
	if strings.HasPrefix(query, InnerSelectTable_) {
		sql = `SELECT * FROM "` + strings.ReplaceAll(query, InnerSelectTable_, "") + `"`
	} else if strings.HasPrefix(query, InnerSelectTableLimit_) {
		sql = `SELECT * FROM "` + strings.ReplaceAll(query, InnerSelectTableLimit_, "") + `" LIMIT 100`
	} else if query == InnerShowDatabases {
		sql = "SELECT table_catalog as name FROM information_schema.tables"
	} else if query == InnerShowTables {
		sql = `SELECT table_name FROM information_schema.tables WHERE table_catalog = '%s' and table_schema != 'pg_catalog' and table_schema != 'information_schema'`
	} else if query == InnerCurrentDB {
		sql = "SELECT current_database() as name"
	} else {
		sql = query
	}
	return
}

type opengeminindraDialect struct{}

func (p *opengeminindraDialect) ToNativeSQL(query string) (sql string) {
	if strings.HasPrefix(query, InnerSelectTable_) {
		sql = `SELECT * FROM ` + strings.ReplaceAll(query, InnerSelectTable_, "")
	} else if strings.HasPrefix(query, InnerSelectTableLimit_) {
		sql = `SELECT * FROM ` + strings.ReplaceAll(query, InnerSelectTableLimit_, "") + ` LIMIT 100`
	} else if query == InnerShowDatabases {
		sql = "DESCRIBE KEYSPACES"
	} else if query == InnerShowTables {
		sql = "SELECT * FROM system_schema.tables WHERE keyspace_name = '%s'"
	} else if query == InnerCurrentDB {
		sql = "SELECT CURRENT_STORAGE_GROUP"
	} else if strings.HasPrefix(query, InnerDescribeTable_) {
		keyspaceAndtableName := strings.ReplaceAll(query, InnerDescribeTable_, "")
		items := strings.Split(keyspaceAndtableName, ":")
		fmt.Println(query, items)
		if len(items) != 2 {
			sql = query
			return
		}
		sql = fmt.Sprintf("SELECT * FROM system_schema.columns WHERE keyspace_name = '%s' AND table_name = '%s'",
			items[0], items[1])
	} else {
		sql = query
	}
	return
}
