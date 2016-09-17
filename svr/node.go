package main

import (
	"log"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/jusongchen/gRPC-db-sync/replica"
	"github.com/pkg/errors"
)

type nodeChgResult struct {
	res *pb.NodeChgResponse
	err error
}

type nodeQryResult struct {
	res *pb.NodeQryResponse
	err error
}

// func (c *Client) NodeChange(ctx context.Context, req *pb.NodeChgRequest, opts ...grpc.CallOption) (*pb.NodeChgResponse, error) {
func (c *Client) NodeChange(ctx context.Context, req *pb.NodeChgRequest, opts ...grpc.CallOption) error {

	ch := make(chan *nodeChgResult, len(c.peers))

	for _, b := range c.peers {
		go func(peer pb.SyncUpClient) {
			res, err := peer.NodeChange(ctx, req)
			ch <- &nodeChgResult{res, err}
		}(b.RpcClient)
	}

	// chgResp := pb.NodeChgResponse{}

	var err error
	for i := range c.peers {
		// res []*pb.NodeChgResponse
		r := <-ch
		if r.err != nil {
			log.Printf("From client%s:%s", c.peers[i].Addr, r.err.Error())
			err = r.err
		}
	}
	return err
	// return &pb.NodeChgResponse{Success:true}, nil
}

func (c *Client) Ping(ctx context.Context, req *pb.PingRequest, opts ...grpc.CallOption) (*pb.PingResponse, error) {
	return nil, nil
}

func (s *Server) NodeChange(ctx context.Context, req *pb.NodeChgRequest) (*pb.NodeChgResponse, error) {
	// fmt.Printf("\nServer get Node change request:%#v\n", req)

	switch req.Operation {
	case pb.NodeChgRequest_JOIN:
		//notify all clients to add Node
		s.c.NodeChange(ctx, &pb.NodeChgRequest{Operation: pb.NodeChgRequest_ADD, NodeAddr: req.NodeAddr})

		//add this new node as peer
		if err := s.c.addPeer(req.NodeAddr); err != nil {
			return &pb.NodeChgResponse{Fail: true}, errors.Wrap(err, "Server:NodeChgJoin:addPeer fail")
		}
		return &pb.NodeChgResponse{Fail: false}, nil

	case pb.NodeChgRequest_ADD:
		//notify all clients to add Node

		// log.Printf("get NodeChgRequest:AddNode:%s", req.NodeAddr)
		//add this new node as peer
		if err := s.c.addPeer(req.NodeAddr); err != nil {
			return &pb.NodeChgResponse{Fail: true}, errors.Wrapf(err, "Server:NodeChgAdd:addPeer %s fail", req.NodeAddr)
		}
		return &pb.NodeChgResponse{Fail: false}, nil

	default:
		log.Fatal("Server NodeChange:unknown Operation %v", req.Operation)
	}

	return &pb.NodeChgResponse{}, nil
}

func (c *Client) NodeQuery(ctx context.Context, req *pb.NodeQryRequest, opts ...grpc.CallOption) (*pb.NodeQryResponse, error) {
	//[TODO] simplified version, only query the first peer
	res, err := c.peers[0].RpcClient.NodeQuery(ctx, req)
	if err != nil {
		return &pb.NodeQryResponse{}, err
	}
	return &pb.NodeQryResponse{NodeAddr: res.NodeAddr}, nil
}
func (s *Server) NodeQuery(ctx context.Context, req *pb.NodeQryRequest) (*pb.NodeQryResponse, error) {
	// fmt.Printf("\nServer get Node query request:%#v\n", req)

	//return server's own address as well
	addrs := []string{s.c.ownAddr}
	for _, p := range s.c.peers {
		addrs = append(addrs, p.Addr)
	}

	return &pb.NodeQryResponse{NodeAddr: addrs}, nil
}

func (s *Server) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {

	return nil, nil
}
