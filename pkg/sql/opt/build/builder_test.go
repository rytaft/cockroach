// Copyright 2018 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package build

// This file is home to TestBuilder, which is similar to the logic tests, except it
// is used for optimizer builder-specific testcases.
//
// Each testfile contains testcases of the form
//   <command>
//   <SQL statement or expression>
//   ----
//   <expected results>
//
// The supported commands are:
//
//  - exec-raw
//
//    Runs the given SQL expression. It does not produce any output.
//
//  - build
//
//    Builds a memo structure from a SQL query and outputs a representation
//    of the memo structure.
//

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/internal/client"
	"github.com/cockroachdb/cockroach/pkg/sql/parser"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sqlbase"
	"github.com/cockroachdb/cockroach/pkg/testutils/serverutils"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"

	"github.com/cockroachdb/cockroach/pkg/sql/opt/testutils"
	"github.com/cockroachdb/cockroach/pkg/sql/opt/xform"
	"github.com/cockroachdb/cockroach/pkg/sql/optbase"
	_ "github.com/cockroachdb/cockroach/pkg/sql/sem/builtins"
)

var (
	testDataGlob = flag.String("d", "testdata/[^.]*", "test data glob")
)

// testCatalog implements the sqlbase.Catalog interface.
type testCatalog struct {
	kvDB *client.DB
}

// FindTable implements the sqlbase.Catalog interface.
func (c testCatalog) FindTable(ctx context.Context, name *tree.TableName) (optbase.Table, error) {
	return sqlbase.GetTableDescriptor(c.kvDB, string(name.DatabaseName), string(name.TableName)), nil
}

func TestBuilder(t *testing.T) {
	defer leaktest.AfterTest(t)()

	paths, err := filepath.Glob(*testDataGlob)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no testfiles found matching: %s", *testDataGlob)
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			ctx := context.Background()
			s, sqlDB, kvDB := serverutils.StartServer(t, base.TestServerArgs{})
			defer s.Stopper().Stop(ctx)
			catalog := testCatalog{kvDB: kvDB}

			testutils.RunDataDrivenTest(t, path, func(d *testutils.TestData) string {
				switch d.Cmd {
				case "exec-raw":
					_, err := sqlDB.Exec(d.Input)
					if err != nil {
						d.Fatalf(t, "%v", err)
					}
					return ""

				case "build":
					stmt, err := parser.ParseOne(d.Input)
					if err != nil {
						d.Fatalf(t, "%v", err)
					}

					f := xform.NewFactory(catalog, 0 /* maxSteps */)
					b := NewBuilder(ctx, f, stmt)
					_, _, err = b.Build()
					if err != nil {
						return fmt.Sprintf("error: %v\n", err)
					}
					return f.MemoString()

				default:
					d.Fatalf(t, "unsupported command: %s", d.Cmd)
					return ""
				}
			})
		})
	}
}
