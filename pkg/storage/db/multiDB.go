package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"math/big"
	"sync"

	"k8s.io/klog/v2"
)

type MultiDB struct {
	sync.Mutex
	PrimaryDB      *sql.DB
	ReadReplicaDBs []*sql.DB
}

func NewMultiDB(primaryDB *sql.DB, readReplicaDBs []*sql.DB) *MultiDB {
	return &MultiDB{
		PrimaryDB:      primaryDB,
		ReadReplicaDBs: readReplicaDBs,
	}
}

func (m *MultiDB) ReadQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	m.Lock()
	replicas := append([]*sql.DB(nil), m.ReadReplicaDBs...)
	m.Unlock()

	// crypto-secure shuffle
	shuffleReplicas(replicas)

	// try each in turn
	for _, readReplicaDB := range replicas {
		rows, err := readReplicaDB.Query(query, args...)
		if err != nil {
			klog.ErrorS(err, "failed to query read replica, retrying next")
			continue
		}
		return rows, nil
	}

	// fallback to primary
	klog.Info("no read replicas available, querying primary db")
	return m.PrimaryDB.Query(query, args...)
}

func (m *MultiDB) ReadQueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	m.Lock()
	replicas := append([]*sql.DB(nil), m.ReadReplicaDBs...)
	m.Unlock()

	shuffleReplicas(replicas)

	for _, readReplicaDB := range replicas {
		row := readReplicaDB.QueryRow(query, args...)
		if err := row.Err(); err != nil {
			klog.ErrorS(err, "failed to query read replica, retrying next")
			continue
		}
		return row
	}

	klog.Info("no read replicas available, querying primary db")
	return m.PrimaryDB.QueryRow(query, args...)
}

// shuffleReplicas performs an in-place Fisher–Yates shuffle using crypto/rand.
func shuffleReplicas(replicas []*sql.DB) {
	n := len(replicas)
	for i := n - 1; i > 0; i-- {
		// generate a secure random index j ∈ [0, i]
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			// If crypto/rand fails for some reason, just skip shuffling that iteration
			continue
		}
		j := int(jBig.Int64())
		replicas[i], replicas[j] = replicas[j], replicas[i]
	}
}
