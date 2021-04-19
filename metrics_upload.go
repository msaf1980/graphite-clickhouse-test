package main

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
)

type Point struct {
	Name      string
	Value     float64
	Timestamp int64
}

func MetricUpload(address string, in *InputSchema, tryConnect int) ([]Point, error) {
	from := now + in.From.Duration().Milliseconds()/(1000*60)*60
	until := now + in.Until.Duration().Milliseconds()/(1000*60)*60

	step := in.Step.Duration().Milliseconds() / 1000

	var conn net.Conn
	var err error
	for i := 0; i < tryConnect; i++ {
		conn, err = net.Dial("tcp", address)
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		return nil, err
	}
	w := bufio.NewWriter(conn)

	defer conn.Close()

	count := (until - from) / step
	points := make([]Point, count*int64(len(in.Metrics)))
	var n int
	var j int
	for i := from; i < until; i += step {
		for _, m := range in.Metrics {
			points[j].Name = m
			points[j].Value = float64(n)
			points[j].Timestamp = i
			metric := fmt.Sprintf("%s %.2f %d\n", points[j].Name, points[j].Value, points[j].Timestamp)
			_, err = w.WriteString(metric)
			if err != nil {
				return nil, err
			}
			j++
		}
		n++
	}

	//fmt.Printf("%d\n", n)
	//fmt.Printf("%d\n", l)

	err = w.Flush()
	if err == nil {
		logger.Info("upload",
			zap.Int("metrics", n), zap.Int("names", len(in.Metrics)),
			zap.Int64("from", from), zap.Int64("until", until), zap.Int64("step", step))
		return points, nil
	}
	return nil, err
}
