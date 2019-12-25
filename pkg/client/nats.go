// Copyright 2019 Erik Agsjö
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

package client

import (
	"time"

	"github.com/nats-io/nats.go"
)

func connect(URLs string, opts state) (*nats.EncodedConn, error) {
	natsOptions := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Millisecond * 500),
	}

	if len(opts.rootCAs) > 0 {
		natsOptions = append(natsOptions, nats.RootCAs(opts.rootCAs...))
	}

	if opts.seedFile != "" {
		option, err := nats.NkeyOptionFromSeed(opts.seedFile)
		if err != nil {
			return nil, err
		}
		natsOptions = append(natsOptions, option)
	}

	nc, err := nats.Connect(URLs, natsOptions...)
	if err != nil {
		return nil, err
	}
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}

	return ec, nil
}
