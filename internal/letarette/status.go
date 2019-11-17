package letarette

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/erkkah/letarette/pkg/logger"
	"github.com/erkkah/letarette/pkg/protocol"
)

// StatusMonitor communicates worker status with the cluster
type StatusMonitor interface {
	Close()
}

// StartStatusMonitor creates a new StatusMonitor, listening to status broadcasts
// and broadcasting our status.
func StartStatusMonitor(nc *nats.Conn, db Database, cfg Config) (StatusMonitor, error) {
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return &monitor{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	self := &monitor{
		ctx:            ctx,
		close:          cancel,
		cfg:            cfg,
		conn:           ec,
		db:             db.(*database),
		updates:        make(chan protocol.IndexStatus),
		workerStatus:   map[string]protocol.IndexStatus{},
		workerPingtime: map[string]time.Time{},
	}

	indexID, err := self.db.getIndexID()
	if err != nil {
		return nil, err
	}
	self.workerStatus[indexID] = protocol.IndexStatus{
		IndexID:        indexID,
		ShardgroupSize: cfg.ShardgroupSize,
		Shardgroup:     cfg.ShardgroupIndex,
		Status:         protocol.IndexStatusStartingUp,
	}

	sub, err := ec.Subscribe(cfg.Nats.Topic+".status", func(sub, reply string, status *protocol.IndexStatus) {
		if status.IndexID != indexID {
			self.updates <- *status
		}
	})
	if err != nil {
		return nil, err
	}

	go func() {
		checkpoint := time.After(time.Millisecond * 100)
		for {
			select {
			case status := <-self.updates:
				self.workerStatus[status.IndexID] = status
				self.workerPingtime[status.IndexID] = time.Now()
			case <-self.ctx.Done():
				sub.Unsubscribe()
				return
			case <-checkpoint:
				self.checkpoint()
				checkpoint = time.After(time.Second * 2)
			}
		}
	}()

	return self, nil
}

type monitor struct {
	cfg            Config
	conn           *nats.EncodedConn
	db             *database
	close          context.CancelFunc
	ctx            context.Context
	updates        chan protocol.IndexStatus
	statusCode     protocol.IndexStatusCode
	workerStatus   map[string]protocol.IndexStatus
	workerPingtime map[string]time.Time
}

func (m *monitor) Close() {
	m.close()
}

func (m *monitor) checkpoint() {
	indexID, err := m.db.getIndexID()
	if err != nil {
		logger.Error.Printf("Failed to read index ID: %w", err)
		return
	}

	workersPerShard := map[uint16][]string{}
	staleTime := time.Now().Add(-1 * time.Minute)
	var numWorkers int
	newStatus := protocol.IndexStatusInSync

	for _, v := range m.workerStatus {
		if v.ShardgroupSize != m.cfg.ShardgroupSize {
			logger.Error.Printf(
				"Shard group size mismatch: worker@%v(%v) != local(%v)",
				v.IndexID, v.ShardgroupSize, m.cfg.ShardgroupSize,
			)
			newStatus = protocol.IndexStatusIncompleteShardgroup
		}
		if indexID == v.IndexID || m.workerPingtime[v.IndexID].After(staleTime) {
			workers := workersPerShard[v.Shardgroup]
			workers = append(workers, v.IndexID)
			workersPerShard[v.Shardgroup] = workers
			numWorkers++
		}
	}

	var workerIndex uint16
	var missingWorkers []string
	for workerIndex = 0; workerIndex < m.cfg.ShardgroupSize; workerIndex++ {
		workers := workersPerShard[workerIndex]
		if len(workers) < 1 {
			missingWorkers = append(missingWorkers, fmt.Sprintf("%v", workerIndex))
		}
	}
	if len(missingWorkers) > 0 {
		logger.Error.Printf("No active workers for shards %s!", strings.Join(missingWorkers, ","))
		newStatus = protocol.IndexStatusIncompleteShardgroup
	}

	if newStatus == protocol.IndexStatusInSync {
		for _, space := range m.cfg.Index.Spaces {
			list, err := m.db.getInterestList(m.ctx, space)
			if err != nil {
				logger.Error.Printf("Failed to get interest list: %w", err)
			}
			if len(list) > 0 {
				newStatus = protocol.IndexStatusSyncing
				break
			}
		}
	}
	m.statusCode = newStatus

	status := m.workerStatus[indexID]

	docCount, err := m.db.getDocumentCount(m.ctx)
	if err != nil {
		logger.Error.Printf("Failed to get document count: %w", err)
	}
	status.DocCount = docCount
	var lastUpdate time.Time
	for _, space := range m.cfg.Index.Spaces {
		update, err := m.db.getLastUpdateTime(m.ctx, space)
		if err != nil {
			logger.Error.Printf("Failed to get last update time: %w", err)
		}
		if update.After(lastUpdate) {
			lastUpdate = update
		}
	}

	status.LastUpdate = lastUpdate
	status.Status = m.statusCode

	m.workerStatus[indexID] = status
	m.conn.Publish(m.cfg.Nats.Topic+".status", &status)
}
