package main

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_rollup(t *testing.T) {
	tests := []struct {
		name   string
		points []Point
		from   int64
		until  int64
		step   int64
		aggr   Aggregation
		want   []float64
	}{
		{
			"test",
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
			120, 180, 60,
			AggrAvg,
			[]float64{25.5, 122.0},
		},
		{
			"test2",
			[]Point{
				{"test2", 62.0, 2},
				{"test", 61.0, 1},
				{"test", 10.0, 61},
				{"test", 11.0, 71},
				{"test2", 12.0, 72},
				{"test", 22.0, 82},
				{"test2", 23.0, 83},
				{"test", 59.0, 119},
				{"test2", 59.0, 119},
				{"test2", 121.0, 121},
				{"test", 122.0, 122},
				{"test2", 2.0, 180},
				{"test2", 181.0, 181},
				{"test", 182.0, 182},
			},
			120, 180, 60,
			AggrSum,
			[]float64{94.0, 123.0},
		},
	}
	for n, tt := range tests {
		t.Run("#"+strconv.Itoa(n)+"_"+tt.aggr.String()+"_"+tt.name, func(t *testing.T) {
			if got, gotFrom, gotUntil := rollup(tt.name, tt.points, tt.from, tt.until, tt.step, tt.aggr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rollup() = points[] %v, want %v", got, tt.want)
			} else {
				assert.Equal(t, tt.from, gotFrom, "rollup() = from ")
				assert.Equal(t, tt.until, gotUntil, "rollup() = until ")
			}
		})
	}
}
