package gorethink

import (
	"sync"

	"github.com/hailocab/go-hostpool"
	p "gopkg.in/dancannon/gorethink.v1/ql2"
)

// Node represents a database server in the cluster
type Node struct {
	ID      string
	Host    Host
	aliases []Host

	cluster *Cluster
	pool    *Pool
	hpr     hostpool.HostPoolResponse

	mu     sync.RWMutex
	closed bool
}

func newNode(id string, aliases []Host, cluster *Cluster, pool *Pool) *Node {
	node := &Node{
		ID:      id,
		Host:    aliases[0],
		aliases: aliases,
		cluster: cluster,
		pool:    pool,
	}

	return node
}

// Closed returns true if the node is closed
func (n *Node) Closed() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.closed
}

// Close closes the session
func (n *Node) Close(optArgs ...CloseOpts) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return nil
	}

	if len(optArgs) >= 1 {
		if optArgs[0].NoReplyWait {
			n.NoReplyWait()
		}
	}

	if n.pool != nil {
		n.pool.Close()
	}
	n.pool = nil
	n.closed = true

	return nil
}

// SetMaxIdleConns sets the maximum number of connections in the idle
// connection pool.
func (n *Node) SetMaxIdleConns(idleConns int) {
	n.pool.SetMaxIdleConns(idleConns)
}

// SetMaxOpenConns sets the maximum number of open connections to the database.
func (n *Node) SetMaxOpenConns(openConns int) {
	n.pool.SetMaxOpenConns(openConns)
}

// NoReplyWait ensures that previous queries with the noreply flag have been
// processed by the server. Note that this guarantee only applies to queries
// run on the given connection
func (n *Node) NoReplyWait() error {
	return n.pool.Exec(Query{
		Type: p.Query_NOREPLY_WAIT,
	})
}

// Query executes a ReQL query using this nodes connection pool.
func (n *Node) Query(q Query) (cursor *Cursor, err error) {
	if n.Closed() {
		return nil, ErrInvalidNode
	}

	cursor, err = n.pool.Query(q)
	if err != nil {
		return cursor, err
	}

	return cursor, err
}

// Exec executes a ReQL query using this nodes connection pool.
func (n *Node) Exec(q Query) (err error) {
	if n.Closed() {
		return ErrInvalidNode
	}

	err = n.pool.Exec(q)
	if err != nil {
		return err
	}

	return err
}

// Server returns the server name and server UUID being used by a connection.
func (n *Node) Server() (ServerResponse, error) {
	var response ServerResponse

	if n.Closed() {
		return response, ErrInvalidNode
	}

	return n.pool.Server()
}

type nodeStatus struct {
	ID      string `gorethink:"id"`
	Name    string `gorethink:"name"`
	Status  string `gorethink:"status"`
	Network struct {
		Hostname           string `gorethink:"hostname"`
		ClusterPort        int64  `gorethink:"cluster_port"`
		ReqlPort           int64  `gorethink:"reql_port"`
		CanonicalAddresses []struct {
			Host string `gorethink:"host"`
			Port int64  `gorethink:"port"`
		} `gorethink:"canonical_addresses"`
	} `gorethink:"network"`
}
