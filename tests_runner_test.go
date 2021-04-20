package main

import (
	"reflect"
	"strconv"
	"testing"

	gchreader "github.com/msaf1980/graphite-clickhouse-test/graphite-clickhouse"
	"github.com/stretchr/testify/assert"
)

func Test_rollup(t *testing.T) {
	type result struct {
		from   int64
		until  int64
		step   int64
		aggr   Aggregation
		points []float64
	}
	tests := []struct {
		name   string
		points []Point
		from   int64
		until  int64
		want   map[string]result
	}{
		{
			"test#1",
			[]Point{
				{"test.metric", 8609.00, 1618884922},
				{"test.metric", 8610.00, 1618884932},
				{"test.metric", 8611.00, 1618884942},
				{"test.metric", 8612.00, 1618884952},
				{"test.metric", 8613.00, 1618884962},
				{"test.metric", 8614.00, 1618884972},
				{"test.metric", 8615.00, 1618884982},
				{"test.metric", 8616.00, 1618884992},
				{"test.metric", 8617.00, 1618885002},
				{"test.metric", 8618.00, 1618885012},
				{"test.metric", 8619.00, 1618885022},
				{"test.metric", 8620.00, 1618885032},
				{"test.metric", 8621.00, 1618885042},
				{"test.metric", 8622.00, 1618885052},
				{"test.metric", 8623.00, 1618885062},
				{"test.metric", 8624.00, 1618885072},
				{"test.metric", 8625.00, 1618885082},
				{"test.metric", 8626.00, 1618885092},
				{"test.metric", 8627.00, 1618885102},
				{"test.metric", 8628.00, 1618885112},
				{"test.metric", 8629.00, 1618885122},
				{"test.metric", 8630.00, 1618885132},
				{"test.metric", 8631.00, 1618885142},
				{"test.metric", 8632.00, 1618885152},
				{"test.metric", 8633.00, 1618885162},
				{"test.metric", 8634.00, 1618885172},
				{"test.metric", 8635.00, 1618885182},
				{"test.metric", 8636.00, 1618885192},
				{"test.metric", 8637.00, 1618885202},
				{"test.metric", 8638.00, 1618885212},
				{"test.metric", 8639.00, 1618885222},
				{"test.metric2.sum", 8609.00, 1618884922},
				{"test.metric2.sum", 8610.00, 1618884932},
				{"test.metric2.sum", 8611.00, 1618884942},
				{"test.metric2.sum", 8612.00, 1618884952},
				{"test.metric2.sum", 8613.00, 1618884962},
				{"test.metric2.sum", 8614.00, 1618884972},
				{"test.metric2.sum", 8615.00, 1618884982},
				{"test.metric2.sum", 8616.00, 1618884992},
				{"test.metric2.sum", 8617.00, 1618885002},
				{"test.metric2.sum", 8618.00, 1618885012},
				{"test.metric2.sum", 8619.00, 1618885022},
				{"test.metric2.sum", 8620.00, 1618885032},
				{"test.metric2.sum", 8621.00, 1618885042},
				{"test.metric2.sum", 8622.00, 1618885052},
				{"test.metric2.sum", 8623.00, 1618885062},
				{"test.metric2.sum", 8624.00, 1618885072},
				{"test.metric2.sum", 8625.00, 1618885082},
				{"test.metric2.sum", 8626.00, 1618885092},
				{"test.metric2.sum", 8627.00, 1618885102},
				{"test.metric2.sum", 8628.00, 1618885112},
				{"test.metric2.sum", 8629.00, 1618885122},
				{"test.metric2.sum", 8630.00, 1618885132},
				{"test.metric2.sum", 8631.00, 1618885142},
				{"test.metric2.sum", 8632.00, 1618885152},
				{"test.metric2.sum", 8633.00, 1618885162},
				{"test.metric2.sum", 8634.00, 1618885172},
				{"test.metric2.sum", 8635.00, 1618885182},
				{"test.metric2.sum", 8636.00, 1618885192},
				{"test.metric2.sum", 8637.00, 1618885202},
				{"test.metric2.sum", 8638.00, 1618885212},
				{"test.metric2.sum", 8639.00, 1618885222},
			},
			1618884992, 1618885112,
			map[string]result{
				"test.metric": {
					1618885020, 1618885140, 60,
					AggrAvg,
					[]float64{8621.5, 8627.5},
				},
				"test.metric2.sum": {
					1618885020, 1618885140, 60,
					AggrSum,
					[]float64{51729, 51765},
				},
			},
		},
		{
			"test#2",
			[]Point{
				{"test2", 62.0, 2},
				{"test", 61.0, 1},
				{"test2", 60.0, 60},
				{"test", 60.0, 60},
				{"test", 10.0, 61},
				{"test2", 10.0, 61},
				{"test", 11.0, 71},
				{"test2", 12.0, 72},
				{"test", 22.0, 82},
				{"test2", 23.0, 83},
				{"test", 59.0, 119},
				{"test2", 59.0, 119},
				{"test2", 121.0, 121},
				{"test", 122.0, 122},
				{"test2", 1.0, 180},
				{"test2", 181.0, 181},
				{"test", 182.0, 182},
			},
			59, 121,
			map[string]result{
				"test": {
					60, 180, 60,
					AggrAvg,
					[]float64{32.4, 122.0},
				},
				"test2": {
					60, 180, 60,
					AggrSum,
					[]float64{164.0, 121.0},
				},
			},
		},
	}
	for n, tt := range tests {
		for name, want := range tt.want {
			t.Run("#"+strconv.Itoa(n)+"_"+want.aggr.String()+"_"+tt.name, func(t *testing.T) {
				got, gotFrom, gotUntil := rollup(name, tt.points, tt.from, tt.until, want.step, want.aggr)
				got, gotFrom, gotUntil = gchreader.TruncatePoints(got, gotFrom, gotUntil, want.step, want.from, want.until)
				if !reflect.DeepEqual(got, want.points) {
					if want.from == gotFrom {
						t.Errorf("rollup(%s) = points[] %v, want %v", name, got, want.points)
					} else {
						t.Errorf("rollup(%s) = points[] %v (from %d), want %v (from %d)",
							name, got, gotFrom, want.points, want.from)
					}
				} else {
					assert.Equal(t, want.from, gotFrom, "rollup(%s) = from ", name)
					assert.Equal(t, want.until, gotUntil, "rollup(%s) = until ", name)
				}
			})
		}
	}
}
