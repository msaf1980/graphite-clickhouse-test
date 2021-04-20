package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"github.com/lomik/graphite-clickhouse/pkg/dry"
	"github.com/msaf1980/go-yamladdons"
	gchreader "github.com/msaf1980/graphite-clickhouse-test/graphite-clickhouse"
)

type Aggregation int

const (
	AggrAvg Aggregation = iota
	AggrMin
	AggrMax
	AggrSum
)

var aggrStrings []string = []string{"avg", "min", "max", "sum"}

func (a *Aggregation) Set(value string) error {
	switch value {
	case "avg":
		*a = AggrAvg
	case "min":
		*a = AggrMin
	case "max":
		*a = AggrMax
	case "sum":
		*a = AggrSum
	default:
		return fmt.Errorf("invalid aggregation %s", value)
	}
	return nil
}

func (a *Aggregation) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}

	if err := a.Set(value); err != nil {
		return fmt.Errorf("failed to parse '%s' to Aggregation: %v", value, err)
	}

	return nil
}

func (a *Aggregation) String() string {
	return aggrStrings[*a]
}

type InputSchema struct {
	Cch     string               `yaml:"carbon_clickhouse"`
	Metrics []string             `yaml:"metrics"`
	From    yamladdons.YDuration `yaml:"from"`
	Until   yamladdons.YDuration `yaml:"until"`
	Step    yamladdons.YDuration `yaml:"step"`
}

type TargetResult struct {
	Step        yamladdons.YDuration `yaml:"step"`
	Aggregation Aggregation          `yaml:"aggregation"`
	Points      []float64            `yaml:"-"`
}

type QueryResult map[string]*TargetResult

type TestSchema struct {
	Gch           string               `yaml:"graphite_clickhouse"`
	Formats       []string             `yaml:"formats"`
	From          yamladdons.YDuration `yaml:"from"`
	Until         yamladdons.YDuration `yaml:"until"`
	MaxDataPoints uint64               `yaml:"max_data_points"`
	Targets       []string             `yaml:"targets"`
	Result        QueryResult          `yaml:"result"`
	//Step    YDuration `yaml:"step"`
}

type Paths struct {
	Docker             string `yaml:"docker"`
	ClickhouseDocker   string `yaml:"clickhouse_docker"`
	Carbonapi          string `yaml:"carbonapi"`
	CarbonClickhouse   string `yaml:"carbon_clickhouse"`
	GraphiteClickhouse string `yaml:"graphite_clickhouse"`
}

type TestConfig struct {
	Dir     string       `yaml:"-"`
	Version string       `yaml:"version"`
	Paths   Paths        `yaml:"paths"`
	Ch      []Clickhouse `yaml:"clickhouse"`
	Input   InputSchema  `yaml:"input"`
	Tests   []TestSchema `yaml:"tests"`
}

func roundTimestap(from int64, step int64) int64 {
	result := from - (from % step)
	if result < from {
		return result + step
	}
	return result
}

func truncateTimestap(from int64, step int64) int64 {
	return from - (from % step)
}

func rollupFlush(a float64, count int, aggr Aggregation, points *[]float64, n *int) {
	if count > 0 {
		if len(*points) <= *n {
			*points = append(*points, math.NaN())
		}
		if math.IsNaN(a) {
			(*points)[*n] = a
		} else {
			switch aggr {
			case AggrAvg:
				(*points)[*n] = a / float64(count)
			default:
				(*points)[*n] = a
			}
		}
		*n++
	}
}

func rollup(name string, points []Point, from, until, step int64, aggr Aggregation) ([]float64, int64, int64) {
	if step == 0 {
		return nil, 0, 0
	}
	from = dry.CeilToMultiplier(from, step) - step
	until = dry.FloorToMultiplier(until, step) + 2*step - 1
	count := (until-from)/step + 2
	rPoints := make([]float64, count)
	i := 0
	n := 0
	a := math.NaN()
	var startFrom int64 = -1
	var endUntil int64
	var end int64
	for _, p := range points {
		if name != p.Name {
			continue
		}
		if p.Timestamp < from {
			continue
		}
		if p.Timestamp > until {
			break
		}
		if startFrom == -1 {
			// k := j - 64
			// if k < 0 {
			// 	k = 0
			// }
			// for i := k; i < j; i++ {
			// 	if name != points[i].Name {
			// 		continue
			// 	}
			// 	fmt.Printf("-- {\"%s\", %.2f, %d},\n", points[i].Name, points[i].Value, points[i].Timestamp)
			// }
			startFrom = dry.FloorToMultiplier(p.Timestamp, step)
			end = startFrom + step
		}
		if p.Timestamp >= end && n > 0 {
			rollupFlush(a, n, aggr, &rPoints, &i)
			//fmt.Printf("%s [%d] %f\n", name, end, rPoints[i-1])
			endUntil = end
			end += step
			n = 0
			a = math.NaN()
		}
		//fmt.Printf("{\"%s\", %.2f, %d},\n", p.Name, p.Value, p.Timestamp)
		n++
		if math.IsNaN(p.Value) {
			continue
		}
		if math.IsNaN(a) {
			a = p.Value
		} else {
			switch aggr {
			case AggrMin:
				if a > p.Value {
					a = p.Value
				}
			case AggrMax:
				if a < p.Value {
					a = p.Value
				}
			default:
				a += p.Value
			}
		}
	}
	rollupFlush(a, n, aggr, &rPoints, &i)
	// if n > 0 {
	// 	fmt.Printf("%s [%d] %f\n", name, end, rPoints[i-1])
	// }
	endUntil = end
	return rPoints[0:i], startFrom, endUntil
}

func convertQueryResult(t *TestSchema, now int64, points []Point) gchreader.QueryResult {
	result := make(gchreader.QueryResult)

	for name, tResult := range t.Result {
		step := tResult.Step.Duration().Microseconds() / 1000000
		from := dry.CeilToMultiplier(now+t.From.Duration().Microseconds()/1000000, step)
		//until := dry.FloorToMultiplier(now+t.Until.Duration().Microseconds()/1000000, step) + step
		until := dry.CeilToMultiplier(now+t.Until.Duration().Microseconds()/1000000, step) + step
		var pp []float64
		pp, from, until = rollup(name, points, from, until, step, tResult.Aggregation)
		r := &gchreader.TargetResult{
			From:   from,
			Until:  until,
			Step:   step,
			Points: pp,
		}
		result[name] = r
	}
	return result
}

func loadTest(testDir string) (*TestConfig, error) {
	d, err := ioutil.ReadFile(path.Join(testDir, "test.yml"))
	if err != nil {
		return nil, err
	}

	cfg := &TestConfig{
		Dir: testDir,
		Paths: Paths{
			Docker:             "docker",
			ClickhouseDocker:   "yandex/clickhouse-server",
			Carbonapi:          "./carbonapi",
			CarbonClickhouse:   "./carbon_clickhouse",
			GraphiteClickhouse: "./graphite_clickhouse",
		},
	}

	err = yaml.Unmarshal(d, cfg)
	if err != nil {
		return cfg, err
	}

	//fmt.Printf("%+v\n", cfg)

	if len(cfg.Input.Metrics) == 0 {
		return cfg, fmt.Errorf("input metric not set")
	}
	if cfg.Input.From.Duration() >= cfg.Input.Until.Duration() {
		return cfg, fmt.Errorf("input from/until is incorrect")
	}
	return cfg, nil
}

func runTest(cfg *TestConfig) bool {
	succesTest := true

	for _, db := range cfg.Ch {
		if err, out := db.Start(cfg.Paths.Docker, cfg.Paths.ClickhouseDocker); err != nil {
			logger.Error("Failed to start",
				zap.String("dir", db.Dir),
				zap.String("version", db.Version),
				zap.Error(err),
				zap.String("out", out),
			)
			if breakOnError {
				logger.Fatal("break")
			}
			succesTest = false
			continue
		} else {
			time.Sleep(time.Second)
			if cCh, err := CarbonClickhouseStart(cfg.Paths.CarbonClickhouse, cfg.Input.Cch, cfg.Dir, db.address,
				filepath.Join(storeDir, "carbon-clickhouse.conf")); err == nil {
				logger.Info("Start",
					zap.String("carbon-clickhouse", cfg.Input.Cch),
					zap.String("command", cfg.Paths.CarbonClickhouse+" -config "+cCh.configFile),
				)

				time.Sleep(time.Second)
				metricsUploaded := true
				var points []Point
				if points, err = MetricUpload(cCh.Address(), &cfg.Input, 100); err != nil {
					logger.Error("Test failed", zap.String("dir", cfg.Dir), zap.Error(err))
					succesTest = false
					metricsUploaded = false
				}

				if metricsUploaded {
					// wait for upload metrics
					time.Sleep(20 * time.Second)

					for _, t := range cfg.Tests {
						if gCh, err := GraphiteClickhouseStart(cfg.Paths.GraphiteClickhouse, t.Gch, cfg.Dir, db.address,
							filepath.Join(storeDir, "graphite-clickhouse.conf")); err == nil {
							logger.Info("Start",
								zap.String("graphite-clickhouse", t.Gch),
								zap.String("command", cfg.Paths.GraphiteClickhouse+" -config "+gCh.configFile),
							)

							time.Sleep(2 * time.Second)
							param := gchreader.RequestParam{
								From:          now + t.From.Duration().Milliseconds()/(1000*60)*60,
								Until:         now + t.Until.Duration().Milliseconds()/(1000*60)*60,
								MaxDataPoints: t.MaxDataPoints,
								Targets:       t.Targets,
							}
							verifyResults := convertQueryResult(&t, now, points)
							for _, format := range t.Formats {
								var result gchreader.QueryResult
								var qInfo gchreader.QueryInfo
								switch format {
								case "pickle":
									qInfo, result, err = gchreader.PickleQuery(gCh.Address(), &param)
								default:
									succesTest = false
									logger.Error("Unsupported graphite-clickhouse response format",
										zap.String("format", format),
										zap.String("graphite-clickhouse", t.Gch),
										zap.String("command", cfg.Paths.GraphiteClickhouse+" -config "+gCh.configFile),
										zap.Error(err),
									)
									continue
								}
								if err == nil {
									if mismatch, diff := gchreader.VerifyQueryResults(result, verifyResults, param.From, param.Until); mismatch {
										succesTest = false
										logger.Error("query",
											zap.String("graphite-clickhouse", t.Gch),
											zap.String("command", cfg.Paths.GraphiteClickhouse+" -config "+gCh.configFile),
											zap.Any("param", param),
											zap.Any("query", qInfo),
											zap.Strings("diff", diff),
										)
										if breakOnError {
											logger.Fatal("break")
										}
									} else {
										logger.Info("query",
											zap.String("graphite-clickhouse", t.Gch),
											zap.String("status", "done"),
											zap.Any("query", qInfo),
										)
									}
								} else {
									logger.Error("query",
										zap.String("graphite-clickhouse", t.Gch),
										zap.String("command", cfg.Paths.GraphiteClickhouse+" -config "+gCh.configFile),
										zap.Any("param", param),
										zap.Any("query", qInfo),
										zap.Error(err),
									)
									if breakOnError {
										logger.Fatal("break")
									}
									succesTest = false
								}
							}

							if err := gCh.Stop(); err != nil {
								logger.Error("Failed to stop",
									zap.String("graphite-clickhouse", t.Gch),
									zap.String("command", cfg.Paths.GraphiteClickhouse+" -config "+gCh.configFile),
									zap.Error(err),
								)
								if breakOnError {
									logger.Fatal("break")
								}
								succesTest = false
							}
						} else {
							logger.Error("Failed to start",
								zap.String("graphite-clickhouse", t.Gch),
								zap.Error(err),
							)
							if breakOnError {
								logger.Fatal("break")
							}
							succesTest = false
						}
					}
				}

				if err := cCh.Stop(); err != nil {
					logger.Error("Failed to stop",
						zap.String("carbon-clickhouse", cfg.Input.Cch),
						zap.String("config", cCh.configFile),
						zap.Error(err),
					)
					if breakOnError {
						logger.Fatal("break")
					}
					succesTest = false
				}
			} else {
				logger.Error("Failed to start",
					zap.String("carbon-clickhouse", cfg.Input.Cch),
					zap.Error(err),
				)
				if breakOnError {
					logger.Fatal("break")
				}
				succesTest = false
			}
		}

		if exit {
			logger.Error("Leave clickhouse",
				zap.String("dir", db.Dir), zap.String("version", db.Version),
				zap.String("address", db.Address()),
			)
			os.Exit(0)
		} else {
			if err, out := db.Stop(true); err != nil {
				logger.Fatal("Failed to stop",
					zap.String("dir", db.Dir),
					zap.String("version", db.Version),
					zap.String("container", db.Container()),
					zap.Error(err),
					zap.String("out", out),
				)
				if breakOnError {
					logger.Fatal("break")
				}
				succesTest = false
			}
		}
	}
	if succesTest {
		logger.Info("SUCESS", zap.String("config", cfg.Dir))
	} else {
		logger.Error("FAILED", zap.String("config", cfg.Dir))
	}
	return succesTest
}
