package letarette

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/erkkah/immutable"

	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
	"github.com/nats-io/nats.go"
)

// The CloningService listens to clone requests over NATS and
// responds with a URL to a clone stream.
type CloningService struct {
	close context.CancelFunc
	conn  *nats.EncodedConn
	db    *database

	server       http.Server
	cloneHost    string
	clonePort    uint16
	cloneStreams immutable.Map

	completed chan string
}

type cloneStream struct {
	started     time.Time
	targetGroup string
}

// StartCloner returns a running cloning service, listening to NATS requests
// and providing shard clones over HTTPS.
func StartCloner(nc *nats.Conn, db Database, cfg Config) (*CloningService, error) {
	logger.Info.Printf("Cloner starting")

	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return nil, err
	}

	cloneHost := cfg.CloningHost
	if cloneHost == "" {
		cloneHost, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}

	self := &CloningService{
		conn:         ec,
		db:           db.(*database),
		cloneHost:    cloneHost,
		clonePort:    cfg.CloningPort,
		cloneStreams: immutable.Map{},
		completed:    make(chan string, 10),
	}

	self.server = http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.CloningPort),
		Handler: self,
	}

	type cloneReq struct {
		reply            string
		targetShardGroup string
	}

	requests := make(chan cloneReq, 10)

	go func() {
		housekeeping := time.After(time.Minute)

		for {
			select {
			case req, more := <-requests:
				if !more {
					return
				}
				stream := self.createCloneStream(req.targetShardGroup)
				err = ec.Publish(req.reply, stream)
				if err != nil {
					logger.Error.Printf("Failed to publish clone response: %v", err)
				}
			case name := <-self.completed:
				self.cloneStreams = self.cloneStreams.Delete(name)
			case <-housekeeping:
				timeout := time.Minute * 15
				limit := time.Now().Add(-timeout)
				self.cloneStreams.Range(func(k, v interface{}) bool {
					stream := v.(cloneStream)
					if stream.started.Before(limit) {
						self.cloneStreams = self.cloneStreams.Delete(k)
					}
					return true
				})
				housekeeping = time.After(time.Minute)
			}
		}
	}()

	subscription, err := ec.QueueSubscribe(
		cfg.Nats.Topic+".clone", cfg.Shardgroup,
		func(sub, reply string, req *protocol.CloneRequest) {
			requests <- cloneReq{
				reply, req.TargetShardgroup,
			}
		})

	if err != nil {
		return nil, err
	}

	go func() {
		_ = self.server.ListenAndServe()
		close(requests)
		_ = subscription.Unsubscribe()
	}()

	return self, nil
}

// Close stops the cloning service
func (cs *CloningService) Close() error {
	err := cs.server.Close()
	return err
}

func textResponse(w http.ResponseWriter, code int, message string) error {
	w.WriteHeader(code)
	_, err := w.Write([]byte(message))
	return err
}

func errorResponse(w http.ResponseWriter, statusCode int, err error) error {
	return textResponse(w, statusCode, fmt.Sprintf("Error: %v", err))
}

func (cs *CloningService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	uri := req.RequestURI
	parsed, err := url.Parse(uri)
	if err != nil {
		_ = errorResponse(w, http.StatusBadRequest, err)
		return
	}

	streamName := parsed.EscapedPath()
	if value, ok := cs.cloneStreams.Get(streamName); ok {
		ctx := context.Background()
		stream := value.(*cloneStream)
		shardCloner, err := StartShardClone(ctx, cs.db, stream.targetGroup, w)
		if err != nil {
			_ = errorResponse(w, http.StatusInternalServerError, err)
		}
		for {
			keepGoing, err := shardCloner.Step(ctx)
			if err != nil {
				logger.Error.Printf("Failed to run cloner step, stopping: %v", err)
				return
			}
			if !keepGoing {
				break
			}
		}
		cs.completed <- streamName
	} else {
		_ = textResponse(w, http.StatusNotFound, "")
	}
}

func (cs *CloningService) createCloneStream(shardGroup string) protocol.CloneStream {
	streamName := randomStreamName()
	cs.cloneStreams = cs.cloneStreams.Set(streamName, cloneStream{
		started:     time.Now(),
		targetGroup: shardGroup,
	})
	streamURL := fmt.Sprintf("https://%s:%d/%s", cs.cloneHost, cs.clonePort, streamName)
	return protocol.CloneStream{
		URL: streamURL,
	}
}

func randomStreamName() string {
	streamNameBytes := make([]byte, 48)
	rand.Read(streamNameBytes)
	raw := bytes.NewBuffer(streamNameBytes)

	var encoded bytes.Buffer
	encoder := base64.NewEncoder(base64.URLEncoding, &encoded)
	_, _ = io.Copy(encoder, raw)

	return encoded.String()
}
