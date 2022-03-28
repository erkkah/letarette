// Copyright 2020 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/erkkah/letarette/pkg/client"
	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/pennant"
	"github.com/erkkah/letarette/pkg/protocol"
)

type testSet struct {
	Iterations int
	Spaces     []string
	Queries    []string
	Limit      int
	Offset     int
}

type testRequest struct {
	testSet
	Filter []string
}

type testResult struct {
	Start    time.Time
	End      time.Time
	Duration float32
	Status   protocol.SearchStatusCode
	Err      error
}

// NATSOptions holds common NATS connection params
type NATSOptions struct {
	NATSURL string `name:"n" default:"localhost"`
}

type runOptions struct {
	NATSOptions

	TestSet string `arg:"0"`
	Output  string `name:"o"`
	Limit   int    `name:"l"`
}

func main() {
	usage := `Letarette load generator

Usage:
    lrload agent [-n <natsURL>]
    lrload list [-n <natsURL>]
    lrload run [-n <natsURL>] [-o <file>] [-l <limit>] <testset.json>

Options:
    -n <natsURL> NATS server URL [default: localhost]
    -o <file>    Write raw CSV data to <file>
    -l <limit>   Limit the run to <limit> agents
`
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "agent":
		{
			var options NATSOptions
			pennant.MustParse(&options, args)
			err := startAgent(options.NATSURL)
			if err != nil {
				logger.Error.Printf("Failed to start load agent: %v", err)
				return
			}
			logger.Info.Printf("Agent waiting for load requests")
			select {}
		}
	case "list":
		{
			var options NATSOptions
			pennant.MustParse(&options, args)
			err := listAgents(options.NATSURL)
			if err != nil {
				logger.Error.Printf("Failed to list agents: %v", err)
			}
		}
	case "run":
		{
			var options runOptions
			pennant.MustParse(&options, args)
			testSet, err := loadTestSet(options.TestSet)
			if err != nil {
				logger.Error.Printf("Failed to load test set: %v", err)
				return
			}

			if err = runTestSet(options.NATSURL, testSet, options.Limit, options.Output); err != nil {
				logger.Error.Printf("Failed to run: %v", err)
			}
		}
	}

}

// NATSConnect connects to NATS :)
func NATSConnect(url string) (*nats.EncodedConn, error) {
	natsOptions := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Millisecond * 500),
	}

	nc, err := nats.Connect(url, natsOptions...)
	if err != nil {
		return nil, err
	}
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}

	return ec, nil
}

func listAgents(url string) error {
	ec, err := NATSConnect(url)
	if err != nil {
		return err
	}

	agents, err := getAgents(ec)
	if err != nil {
		return err
	}

	fmt.Printf("%v agents responding: %v\n", len(agents), agents)

	return nil
}

func startAgent(url string) error {
	agent, err := client.NewSearchAgent([]string{url}, client.WithTimeout(time.Second*10))
	if err != nil {
		return err
	}

	ec, err := NATSConnect(url)
	if err != nil {
		return err
	}

	clientID, _ := ec.Conn.GetClientID()
	stringID := fmt.Sprintf("%v", clientID)
	logger.Debug.Printf("ID: %v", clientID)

	_, err = ec.Subscribe("leta.load.ping", func(interface{}) {
		_ = ec.Publish("leta.load.pong", &stringID)
	})
	if err != nil {
		return err
	}

	_, err = ec.Subscribe("leta.load.request", func(set *testRequest) {
		found := false
		for _, id := range set.Filter {
			if id == stringID {
				found = true
				break
			}
		}
		if !found {
			return
		}
		logger.Info.Printf("Running load request")
		results := make([]testResult, set.Iterations)
		for i := 0; i < set.Iterations; i++ {
			q := set.Queries[rand.Intn(len(set.Queries))]
			start := time.Now()
			res, err := agent.Search(q, set.Spaces, set.Limit, set.Offset)
			results[i] = testResult{
				Start:    start,
				End:      time.Now(),
				Duration: res.Duration,
				Status:   res.Status,
				Err:      err,
			}
		}
		_ = ec.Publish("leta.load.response", &results)
	})
	if err != nil {
		return err
	}

	return nil
}

func getAgents(ec *nats.EncodedConn) ([]string, error) {
	agents := []string{}

	pingSub, err := ec.Subscribe("leta.load.pong", func(agent *string) {
		agents = append(agents, *agent)
	})
	if err != nil {
		return agents, err
	}

	_ = ec.Publish("leta.load.ping", nil)

	time.Sleep(time.Second * 2)
	_ = pingSub.Unsubscribe()

	return agents, nil
}

func runTestSet(url string, set testSet, limit int, output string) error {
	ec, err := NATSConnect(url)
	if err != nil {
		return err
	}

	agents, err := getAgents(ec)
	if err != nil {
		return err
	}
	numAgents := len(agents)

	if limit < 0 || numAgents < 1 {
		return fmt.Errorf("no agents available")
	}

	if limit != 0 && numAgents > limit {
		numAgents = limit
		agents = agents[:numAgents]
	}

	rand.Seed(time.Now().Unix())

	var wg sync.WaitGroup
	wg.Add(numAgents + 1)

	resultChannel := make(chan []testResult, 10)
	results := make([]testResult, 0, numAgents)
	go func() {
		for result := range resultChannel {
			results = append(results, result...)
			logger.Debug.Printf("Adding result")
			if len(results) == numAgents*set.Iterations {
				logger.Debug.Printf("All done")
				wg.Done()
				break
			}
		}
	}()
	responseSub, err := ec.Subscribe("leta.load.response", func(result *[]testResult) {
		logger.Debug.Printf("Got response with %v results", len(*result))
		resultChannel <- *result
		wg.Done()
	})
	if err != nil {
		return err
	}
	_ = responseSub.AutoUnsubscribe(numAgents)

	start := time.Now()
	_ = ec.Publish("leta.load.request", &testRequest{set, agents})

	logger.Debug.Printf("Waiting...")
	wg.Wait()
	end := time.Now()

	logger.Debug.Printf("Reporting...")
	report(results, numAgents, end.Sub(start), output)
	return nil
}

func report(results []testResult, clients int, total time.Duration, output string) {
	if output != "" {
		output, err := os.Create(output)
		if err != nil {
			logger.Error.Printf("Failed to create output file: %v", err)
			return
		}
		defer output.Close()
		for _, res := range results {
			var status = res.Status.String()
			if res.Err != nil {
				status = fmt.Sprintf("%v", res.Err)
			}
			realDuration := res.End.Sub(res.Start)
			fmt.Fprintf(output, "%v,%v,%q\n", realDuration.Seconds(), res.Duration, status)
		}
	}

	var durationMean float32
	var totalMean float64
	var successful = 0

	for _, res := range results {
		durationMean += res.Duration
		totalMean += res.End.Sub(res.Start).Seconds()
		if res.Err == nil {
			successful++
		}
	}
	durationMean /= float32(len(results))
	totalMean /= float64(len(results))

	sort.Slice(results, func(i, j int) bool {
		return results[i].Duration < results[j].Duration
	})

	durationMedian := results[len(results)/2].Duration
	duration90 := results[int(float32(len(results))*0.9)].Duration
	duration95 := results[int(float32(len(results))*0.95)].Duration
	duration99 := results[int(float32(len(results))*0.99)].Duration

	sort.Slice(results, func(i, j int) bool {
		totalA := results[i].End.Sub(results[i].Start).Seconds()
		totalB := results[j].End.Sub(results[j].Start).Seconds()
		return totalA < totalB
	})

	totalMedian := results[len(results)/2].Duration
	total90 := results[int(float32(len(results))*0.9)].Duration
	total95 := results[int(float32(len(results))*0.95)].Duration
	total99 := results[int(float32(len(results))*0.99)].Duration

	fmt.Printf("Testset run on %v concurrent agents in %.2fs\n", clients, total.Seconds())
	fmt.Printf("\nSuccess ratio: %.4f%%\n", 100*float32(successful)/float32(len(results)))

	fmt.Printf("\nQuery processing times:\n")
	fmt.Printf("Mean:\t%v\nMedian:\t%v\n", durationMean, durationMedian)
	fmt.Printf("90%%:\t%v\n", duration90)
	fmt.Printf("95%%:\t%v\n", duration95)
	fmt.Printf("99%%:\t%v\n", duration99)

	fmt.Printf("\nTotal roundtrip times:\n")
	fmt.Printf("Mean:\t%v\nMedian:\t%v\n", float32(totalMean), totalMedian)
	fmt.Printf("90%%:\t%v\n", total90)
	fmt.Printf("95%%:\t%v\n", total95)
	fmt.Printf("99%%:\t%v\n", total99)

}

func loadTestSet(path string) (testSet, error) {
	file, err := os.Open(path)
	if err != nil {
		return testSet{}, err
	}

	decoder := json.NewDecoder(file)
	var loaded testSet
	err = decoder.Decode(&loaded)
	return loaded, err
}
