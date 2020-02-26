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

package spinner

import (
	"io"
	"strings"
	"time"
)

// Spinner is a tiny spinner implementation
type Spinner struct {
	spinnerChars []rune
	pos          int
	writer       io.Writer
	endWith      chan string
	stopped      bool
}

// New creates a spinner for the given destination.
// The spinner starts non-spinning, there is no output until Start() is called.
func New(writer io.Writer) *Spinner {
	return &Spinner{
		spinnerChars: []rune{'▖', '▘', '▝', '▗'},
		writer:       writer,
		endWith:      make(chan string),
	}
}

// Start makes the spinner go round.
// Providing a prompt will print it before the spinner.
func (s *Spinner) Start(prompt ...string) {
	go func() {
		prefix := strings.Join(prompt, " ") + "  "
		s.writer.Write([]byte(prefix))
		for {
			select {
			case <-time.After(time.Millisecond * 300):
				s.spin()
			case msg := <-s.endWith:
				s.stop(msg, len(prefix))
				close(s.endWith)
				return
			}
		}
	}()
}

// Stop stops the spinner and optionally prints an ending
// message.
func (s *Spinner) Stop(message ...string) {
	if !s.stopped {
		s.endWith <- (strings.Join(message, " "))
	}
	s.stopped = true
}

func (s *Spinner) spin() {
	s.pos++
	s.pos %= len(s.spinnerChars)
	char := s.spinnerChars[s.pos]
	s.writer.Write([]byte("\b\b" + string(char) + " "))
}

func (s *Spinner) stop(message string, prefixLen int) {
	backup := strings.Repeat("\b", prefixLen)
	cleanup := strings.Repeat(" ", prefixLen)
	s.writer.Write([]byte(backup + cleanup + backup + message))
}
