// TDatahe svr command runs a syncUp Server that promote changes and handle changes to/from multiple peers.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"google.golang.org/grpc"

	pb "github.com/jusongchen/gRPC-db-sync/replica"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

var (
	// peers = flag.String("peers", "localhost:36061,localhost:36062", "comma-separated peer addresses")
	port   = flag.Int("p", 36060, "the port to bind to.")
	joinTo = flag.String("join", "",
		`       A address to use when a new node is joining an existing cluster. 
		For the first node in a cluster, -join should NOT be specified.`)
	dbVendor = flag.String("vendor", "Oracle", "Database vendor, either ora or pg.")
	dbUser   = flag.String("dbUser", "gRPC", "The user name used to connect to the DB server")
	dbPasswd = flag.String("dbPassword", "gRPC", "password of the DB user")
	dbServer = flag.String("dbServer", "//localhost/chen", "The database server")
)

type Peer struct {
	Addr string
	// lastAliveTime time.Time
	RpcClient  pb.SyncUpClient
	in         chan *pb.Record
	lastRpcErr error
}

type Client struct {
	ownAddr string

	//set to true after connected to any peers
	connected bool
	peers     []Peer
}

type Server struct {
	c        *Client
	db       *sql.DB
	dbVendor string
}

func main() {
	flag.Parse()

	var db *sql.DB
	db, err := OpenDB()
	if err != nil {
		log.Fatal(err)
		return
	}

	client := Client{ownAddr: getOwnAddr(*port)}
	//launch the client go rountine
	go func() {
		if err := client.connToPeers(*joinTo); err != nil {
			log.Fatal(err)
		}
		go func() {
			if err := client.MonitorAndPromoteChg(db); err != nil {
				log.Fatal(err)
			}
		}()
	}()

	// go handleSignals()

	//launch the server goroutine
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port)) // RPC port
	if err != nil {
		log.Fatalf("\nfailed to listen: %v", err)
	}

	g := grpc.NewServer()

	s := &Server{c: &client, db: db, dbVendor: *dbVendor}

	pb.RegisterSyncUpServer(g, s)
	g.Serve(lis)

}

func handleSignals() {
	var wg sync.WaitGroup
	wg.Add(1)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, os.Kill)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGTRAP)
	// signal.Notify(signalCh, os.Interrupt)
	go func() {
		<-signalCh

		fmt.Printf("\n processing signal")
		wg.Done()
	}()
	wg.Wait()
}

func getOwnAddr(portNumber int) string {
	hostname, err := os.Hostname()
	// hostname = "localhost"
	if err != nil {
		log.Fatal(errors.Wrap(err, "os.Hostname fail"))
	}
	return fmt.Sprintf("%s:%d", hostname, portNumber)
}

func (c *Client) connToPeers(AddrToJoin string) error {
	//first node
	if AddrToJoin == "" {
		return nil
	}
	//add first peer
	if err := c.addPeer(AddrToJoin); err != nil {
		return errors.Wrapf(err, "JoinCluster addPeer %s fail", AddrToJoin)
	}

	ctx := context.Background()

	//making a Node Join request
	req := &pb.NodeChgRequest{Operation: pb.NodeChgRequest_JOIN, NodeAddr: c.ownAddr}

	err := c.NodeChange(ctx, req)
	if err != nil {
		err = errors.Wrap(err, "rpc NodeChange fail")
		return err
	}

	return c.QueryAndAddPeers()

}

func (c *Client) QueryAndAddPeers() error {

	ctx := context.Background()

	//making a Node Query Request to get a list of nodes from the peer this node joined to
	req := &pb.NodeQryRequest{NodeAddr: c.ownAddr}
	res, err := c.NodeQuery(ctx, req)
	if err != nil {
		errors.Wrap(err, "gRPC NodeQuery fail")
		return err
	}

	//for each addresses, add them as peer
	for _, addr := range res.NodeAddr {
		if addr == c.ownAddr {
			continue
		}
		err_ := c.addPeer(addr)
		if err_ != nil {
			err = err_
		}

	}
	return err
}

func (c *Client) addPeer(clientAddr string) error {
	//check if it is this server, skip
	if clientAddr == c.ownAddr {
		return nil
	}
	//check if client has already registered
	for _, v := range c.peers {
		if v.Addr == clientAddr {
			return nil
		}
	}

	conn, err := grpc.Dial(clientAddr, grpc.WithInsecure())
	if err != nil {
		return errors.Wrapf(err, "grpc.Dail to %s fail", clientAddr)
	}
	// log.Printf("Connected to peer:%s\n", clientAddr)
	// c.inCluster = true

	peer := Peer{Addr: clientAddr, RpcClient: pb.NewSyncUpClient(conn), in: make(chan *pb.Record)}

	c.peers = append(c.peers, peer)
	c.connected = true
	log.Printf("Server %s: peer update:", c.ownAddr)
	for i := range c.peers {
		log.Printf("Connect to peer %s", c.peers[i].Addr)
	}
	return nil
}
