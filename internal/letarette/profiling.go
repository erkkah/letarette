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

package letarette

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/erkkah/letarette/pkg/logger"
	// Pull in pprof HTTP handler
	// _ "net/http/pprof"
)

// Profiler wraps native profiling tools
type Profiler struct {
	cpuWriter *os.File
	memWriter *os.File

	blockProfile *pprof.Profile
	blockWriter  *os.File

	mutexProfile *pprof.Profile
	mutexWriter  *os.File
}

// StartProfiler starts a profiler if setup in the config
func StartProfiler(cfg Config) (*Profiler, error) {
	profiler := &Profiler{}

	if cfg.Profile.HTTP != 0 {
		runtime.SetBlockProfileRate(100)
		runtime.SetMutexProfileFraction(5)
		go func() {
			logger.Info.Printf("Starting profiler HTTP listener on port %d", cfg.Profile.HTTP)
			log.Println(http.ListenAndServe(fmt.Sprintf("localhost:%d", cfg.Profile.HTTP), nil))
		}()
	}

	if cfg.Profile.CPU != "" {
		logger.Info.Printf("Starting CPU profiler writer to %s", cfg.Profile.CPU)
		f, err := os.Create(cfg.Profile.CPU)
		if err != nil {
			return nil, fmt.Errorf("failed to create CPU profile: %v", err)
		}
		profiler.cpuWriter = f
		if err = pprof.StartCPUProfile(f); err != nil {
			return nil, fmt.Errorf("failed to start CPU profile: %v", err)
		}
	}

	if cfg.Profile.Mem != "" {
		logger.Info.Printf("Starting memory profiler writer to %s", cfg.Profile.Mem)
		f, err := os.Create(cfg.Profile.Mem)
		if err != nil {
			log.Fatalf("failed to create memory profile: %v", err)
		}
		profiler.memWriter = f
	}

	if cfg.Profile.Block != "" {
		logger.Info.Printf("Starting block profiler writer to %s", cfg.Profile.Block)
		runtime.SetBlockProfileRate(1)
		f, err := os.Create(cfg.Profile.Block)
		if err != nil {
			log.Fatalf("Failed to create block profile: %v", err)
		}
		p := pprof.Lookup("block")

		profiler.blockProfile = p
		profiler.blockWriter = f
	}

	if cfg.Profile.Mutex != "" {
		logger.Info.Printf("Starting mutex profiler writer to %s", cfg.Profile.Mutex)
		runtime.SetMutexProfileFraction(1000)
		f, err := os.Create(cfg.Profile.Mutex)
		if err != nil {
			log.Fatalf("Failed to create mutex profile: %v", err)
		}
		p := pprof.Lookup("mutex")

		profiler.mutexProfile = p
		profiler.mutexWriter = f
	}

	return profiler, nil
}

// Close finishes profiling
func (p *Profiler) Close() error {
	if p.cpuWriter != nil {
		pprof.StopCPUProfile()
		p.cpuWriter.Close()
	}

	if p.memWriter != nil {
		runtime.GC()
		if err := pprof.WriteHeapProfile(p.memWriter); err != nil {
			return fmt.Errorf("could not write memory profile: %v", err)
		}
		p.memWriter.Close()
	}

	if p.blockProfile != nil {
		_ = p.blockProfile.WriteTo(p.blockWriter, 1)
	}

	if p.mutexProfile != nil {
		_ = p.mutexProfile.WriteTo(p.mutexWriter, 1)
	}
	return nil
}
