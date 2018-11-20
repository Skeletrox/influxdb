package reads_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/influxdata/platform/models"
	"github.com/influxdata/platform/pkg/testing/gen"
	"github.com/influxdata/platform/storage/reads"
	"github.com/influxdata/platform/storage/reads/datatypes"
)

func TestGroupGroupResultSetSorting(t *testing.T) {
	tests := []struct {
		name  string
		cur   reads.SeriesCursor
		group datatypes.ReadRequest_Group
		keys  []string
		exp   string
	}{
		{
			name: "group by tag1 in all series",
			cur: &sliceSeriesCursor{
				rows: newSeriesRows(
					"cpu,tag0=val00,tag1=val10",
					"cpu,tag0=val00,tag1=val11",
					"cpu,tag0=val00,tag1=val12",
					"cpu,tag0=val01,tag1=val10",
					"cpu,tag0=val01,tag1=val11",
					"cpu,tag0=val01,tag1=val12",
				)},
			group: datatypes.GroupBy,
			keys:  []string{"tag1"},
			exp: `group:
  tag key      : _m,tag0,tag1
  partition key: val10
    series: _m=cpu,tag0=val00,tag1=val10
    series: _m=cpu,tag0=val01,tag1=val10
group:
  tag key      : _m,tag0,tag1
  partition key: val11
    series: _m=cpu,tag0=val00,tag1=val11
    series: _m=cpu,tag0=val01,tag1=val11
group:
  tag key      : _m,tag0,tag1
  partition key: val12
    series: _m=cpu,tag0=val00,tag1=val12
    series: _m=cpu,tag0=val01,tag1=val12
`,
		},
		{
			name: "group by tag1 in partial series",
			cur: &sliceSeriesCursor{
				rows: newSeriesRows(
					"aaa,tag0=val00",
					"aaa,tag0=val01",
					"cpu,tag0=val00,tag1=val10",
					"cpu,tag0=val00,tag1=val11",
					"cpu,tag0=val00,tag1=val12",
					"cpu,tag0=val01,tag1=val10",
					"cpu,tag0=val01,tag1=val11",
					"cpu,tag0=val01,tag1=val12",
				)},
			group: datatypes.GroupBy,
			keys:  []string{"tag1"},
			exp: `group:
  tag key      : _m,tag0,tag1
  partition key: val10
    series: _m=cpu,tag0=val00,tag1=val10
    series: _m=cpu,tag0=val01,tag1=val10
group:
  tag key      : _m,tag0,tag1
  partition key: val11
    series: _m=cpu,tag0=val01,tag1=val11
    series: _m=cpu,tag0=val00,tag1=val11
group:
  tag key      : _m,tag0,tag1
  partition key: val12
    series: _m=cpu,tag0=val01,tag1=val12
    series: _m=cpu,tag0=val00,tag1=val12
group:
  tag key      : _m,tag0
  partition key: <nil>
    series: _m=aaa,tag0=val00
    series: _m=aaa,tag0=val01
`,
		},
		{
			name: "group by tag2,tag1 with partial series",
			cur: &sliceSeriesCursor{
				rows: newSeriesRows(
					"aaa,tag0=val00",
					"aaa,tag0=val01",
					"cpu,tag0=val00,tag1=val10",
					"cpu,tag0=val00,tag1=val11",
					"cpu,tag0=val00,tag1=val12",
					"mem,tag1=val10,tag2=val20",
					"mem,tag1=val11,tag2=val20",
					"mem,tag1=val11,tag2=val21",
				)},
			group: datatypes.GroupBy,
			keys:  []string{"tag2", "tag1"},
			exp: `group:
  tag key      : _m,tag1,tag2
  partition key: val20,val10
    series: _m=mem,tag1=val10,tag2=val20
group:
  tag key      : _m,tag1,tag2
  partition key: val20,val11
    series: _m=mem,tag1=val11,tag2=val20
group:
  tag key      : _m,tag1,tag2
  partition key: val21,val11
    series: _m=mem,tag1=val11,tag2=val21
group:
  tag key      : _m,tag0,tag1
  partition key: <nil>,val10
    series: _m=cpu,tag0=val00,tag1=val10
group:
  tag key      : _m,tag0,tag1
  partition key: <nil>,val11
    series: _m=cpu,tag0=val00,tag1=val11
group:
  tag key      : _m,tag0,tag1
  partition key: <nil>,val12
    series: _m=cpu,tag0=val00,tag1=val12
group:
  tag key      : _m,tag0
  partition key: <nil>,<nil>
    series: _m=aaa,tag0=val00
    series: _m=aaa,tag0=val01
`,
		},
		{
			name: "group by tag0,tag2 with partial series",
			cur: &sliceSeriesCursor{
				rows: newSeriesRows(
					"aaa,tag0=val00",
					"aaa,tag0=val01",
					"cpu,tag0=val00,tag1=val10",
					"cpu,tag0=val00,tag1=val11",
					"cpu,tag0=val00,tag1=val12",
					"mem,tag1=val10,tag2=val20",
					"mem,tag1=val11,tag2=val20",
					"mem,tag1=val11,tag2=val21",
				)},
			group: datatypes.GroupBy,
			keys:  []string{"tag0", "tag2"},
			exp: `group:
  tag key      : _m,tag0,tag1
  partition key: val00,<nil>
    series: _m=aaa,tag0=val00
    series: _m=cpu,tag0=val00,tag1=val10
    series: _m=cpu,tag0=val00,tag1=val11
    series: _m=cpu,tag0=val00,tag1=val12
group:
  tag key      : _m,tag0
  partition key: val01,<nil>
    series: _m=aaa,tag0=val01
group:
  tag key      : _m,tag1,tag2
  partition key: <nil>,val20
    series: _m=mem,tag1=val10,tag2=val20
    series: _m=mem,tag1=val11,tag2=val20
group:
  tag key      : _m,tag1,tag2
  partition key: <nil>,val21
    series: _m=mem,tag1=val11,tag2=val21
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			newCursor := func() (reads.SeriesCursor, error) {
				return tt.cur, nil
			}

			var hints datatypes.HintFlags
			hints.SetHintSchemaAllTime()
			rs := reads.NewGroupResultSet(context.Background(), &datatypes.ReadRequest{Group: tt.group, GroupKeys: tt.keys, Hints: hints}, newCursor)

			sb := new(strings.Builder)
			GroupResultSetToString(sb, rs, SkipNilCursor())

			if got := sb.String(); !cmp.Equal(got, tt.exp) {
				t.Errorf("unexpected value; -got/+exp\n%s", cmp.Diff(strings.Split(got, "\n"), strings.Split(tt.exp, "\n")))
			}
		})
	}
}

func TestNewGroupResultSet_Sorting(t *testing.T) {
	tests := []struct {
		name string
		keys []string
		opts []reads.GroupOption
		exp  string
	}{
		{
			name: "nil hi",
			keys: []string{"tag0", "tag2"},
			exp: `group:
  tag key      : _m,tag0,tag1
  partition key: val00,<nil>
    series: _m=aaa,tag0=val00
    series: _m=cpu,tag0=val00,tag1=val10
    series: _m=cpu,tag0=val00,tag1=val11
    series: _m=cpu,tag0=val00,tag1=val12
group:
  tag key      : _m,tag0
  partition key: val01,<nil>
    series: _m=aaa,tag0=val01
group:
  tag key      : _m,tag1,tag2
  partition key: <nil>,val20
    series: _m=mem,tag1=val10,tag2=val20
    series: _m=mem,tag1=val11,tag2=val20
group:
  tag key      : _m,tag1,tag2
  partition key: <nil>,val21
    series: _m=mem,tag1=val11,tag2=val21
`,
		},
		{
			name: "nil lo",
			keys: []string{"tag0", "tag2"},
			opts: []reads.GroupOption{reads.GroupOptionNilSortLo()},
			exp: `group:
  tag key      : _m,tag1,tag2
  partition key: <nil>,val20
    series: _m=mem,tag1=val11,tag2=val20
    series: _m=mem,tag1=val10,tag2=val20
group:
  tag key      : _m,tag1,tag2
  partition key: <nil>,val21
    series: _m=mem,tag1=val11,tag2=val21
group:
  tag key      : _m,tag0,tag1
  partition key: val00,<nil>
    series: _m=cpu,tag0=val00,tag1=val10
    series: _m=cpu,tag0=val00,tag1=val11
    series: _m=cpu,tag0=val00,tag1=val12
    series: _m=aaa,tag0=val00
group:
  tag key      : _m,tag0
  partition key: val01,<nil>
    series: _m=aaa,tag0=val01
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newCursor := func() (reads.SeriesCursor, error) {
				return &sliceSeriesCursor{
					rows: newSeriesRows(
						"aaa,tag0=val00",
						"aaa,tag0=val01",
						"cpu,tag0=val00,tag1=val10",
						"cpu,tag0=val00,tag1=val11",
						"cpu,tag0=val00,tag1=val12",
						"mem,tag1=val10,tag2=val20",
						"mem,tag1=val11,tag2=val20",
						"mem,tag1=val11,tag2=val21",
					)}, nil
			}

			var hints datatypes.HintFlags
			hints.SetHintSchemaAllTime()
			rs := reads.NewGroupResultSet(context.Background(), &datatypes.ReadRequest{Group: datatypes.GroupBy, GroupKeys: tt.keys, Hints: hints}, newCursor, tt.opts...)

			sb := new(strings.Builder)
			GroupResultSetToString(sb, rs, SkipNilCursor())

			if got := sb.String(); !cmp.Equal(got, tt.exp) {
				t.Errorf("unexpected value; -got/+exp\n%s", cmp.Diff(strings.Split(got, "\n"), strings.Split(tt.exp, "\n")))
			}
		})
	}
}

type sliceSeriesCursor struct {
	rows []reads.SeriesRow
	i    int
}

func newSeriesRows(keys ...string) []reads.SeriesRow {
	rows := make([]reads.SeriesRow, len(keys))
	for i := range keys {
		rows[i].Name, rows[i].SeriesTags = models.ParseKeyBytes([]byte(keys[i]))
		rows[i].Tags = rows[i].SeriesTags.Clone()
		rows[i].Tags.Set([]byte("_m"), rows[i].Name)
	}
	return rows
}

func (s *sliceSeriesCursor) Close()     {}
func (s *sliceSeriesCursor) Err() error { return nil }

func (s *sliceSeriesCursor) Next() *reads.SeriesRow {
	if s.i < len(s.rows) {
		s.i++
		return &s.rows[s.i-1]
	}
	return nil
}
