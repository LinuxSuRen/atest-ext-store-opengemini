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
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/openGemini/opengemini-client-go/opengemini"

	"github.com/linuxsuren/api-testing/pkg/server"
)

func (s *dbserver) Query(ctx context.Context, query *server.DataQuery) (result *server.DataQueryResult, err error) {
	var db opengemini.Client
	var dbQuery DataQuery
	if dbQuery, err = s.getClientWithDatabase(ctx, query.Key); err != nil {
		return
	}

	db = dbQuery.GetClient()

	result = &server.DataQueryResult{
		Data:  []*server.Pair{},
		Items: make([]*server.Pairs, 0),
		Meta:  &server.DataMeta{},
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		// query database and tables
		if result.Meta.Databases, err = dbQuery.GetDatabases(ctx); err != nil {
			log.Printf("failed to query databases: %v\n", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if result.Meta.CurrentDatabase = query.Key; query.Key == "" {
			if result.Meta.CurrentDatabase, err = dbQuery.GetCurrentDatabase(); err != nil {
				log.Printf("failed to query current database: %v\n", err)
			}
		}

		if result.Meta.Tables, err = dbQuery.GetTables(ctx, result.Meta.CurrentDatabase); err != nil {
			log.Printf("failed to query tables: %v\n", err)
		}
	}()

	defer wg.Wait()
	// query data
	if query.Sql == "" {
		return
	}

	query.Sql = dbQuery.GetInnerSQL().ToNativeSQL(query.Sql)

	wg.Add(1)
	go func() {
		defer wg.Done()

		result.Meta.Labels = dbQuery.GetLabels(ctx, query.Sql)
		result.Meta.Labels = append(result.Meta.Labels, &server.Pair{
			Key:   "_native_sql",
			Value: query.Sql,
		})
	}()

	var dataResult *server.DataQueryResult
	now := time.Now()
	if dataResult, err = sqlQuery(ctx, query.Key, query.Sql, db); err == nil {
		result.Items = dataResult.Items
		result.Meta.Duration = time.Since(now).String()
	}
	return
}

func sqlQuery(_ context.Context, database, sql string, client opengemini.Client) (result *server.DataQueryResult, err error) {
	fmt.Println("query sql", sql)
	q := opengemini.Query{
		Database: database,
		Command:  sql,
	}

	var res *opengemini.QueryResult
	if res, err = client.Query(q); err != nil {
		return
	}
	if res.Error != "" {
		err = fmt.Errorf("%s", res.Error)
		return
	}

	result = &server.DataQueryResult{
		Data:  []*server.Pair{},
		Items: make([]*server.Pairs, 0),
		Meta:  &server.DataMeta{},
	}

	for _, rt := range res.Results {
		for _, rs := range rt.Series {
			if len(rs.Columns) == 0 {
				continue
			}

			for _, col := range rs.Columns {
				for _, v := range rs.Values {
					data := make([]*server.Pair, 0)
					data = append(data, &server.Pair{
						Key:   col,
						Value: fmt.Sprintf("%v", v),
					})
					fmt.Println(col, v, "=====")
					result.Items = append(result.Items, &server.Pairs{
						Data: data,
					})
				}
			}
		}
	}
	return
}

const queryDatabaseSql = "show databases"

type DataQuery interface {
	GetDatabases(context.Context) (databases []string, err error)
	GetTables(ctx context.Context, currentDatabase string) (tables []string, err error)
	GetCurrentDatabase() (string, error)
	GetLabels(context.Context, string) []*server.Pair
	GetClient() opengemini.Client
	GetInnerSQL() InnerSQL
}

type commonDataQuery struct {
	client   opengemini.Client
	innerSQL InnerSQL
}

var _ DataQuery = &commonDataQuery{}

func NewCommonDataQuery(innerSQL InnerSQL, client opengemini.Client) DataQuery {
	return &commonDataQuery{
		innerSQL: innerSQL,
		client:   client,
	}
}

func (q *commonDataQuery) GetDatabases(ctx context.Context) (databases []string, err error) {
	databases, err = q.client.ShowDatabases()
	sort.Strings(databases)
	return
}

func (q *commonDataQuery) GetTables(ctx context.Context, currentDatabase string) (tables []string, err error) {
	tables, err = q.client.ShowMeasurements(opengemini.NewMeasurementBuilder().Database(currentDatabase).Show())
	fmt.Println(tables, err, currentDatabase)
	return
}

func (q *commonDataQuery) GetCurrentDatabase() (current string, err error) {
	var data *server.DataQueryResult
	if data, err = sqlQuery(context.Background(), "", q.GetInnerSQL().ToNativeSQL(InnerCurrentDB), q.client); err == nil && len(data.Items) > 0 && len(data.Items[0].Data) > 0 {
		current = data.Items[0].Data[0].Value
	}
	return
}

func (q *commonDataQuery) GetLabels(ctx context.Context, sql string) (metadata []*server.Pair) {
	metadata = make([]*server.Pair, 0)
	return
}

func (q *commonDataQuery) GetClient() opengemini.Client {
	return q.client
}

func (q *commonDataQuery) GetInnerSQL() InnerSQL {
	return q.innerSQL
}
