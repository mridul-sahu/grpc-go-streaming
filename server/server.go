package main

import (
	"net"
	"log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"golang.org/x/net/context"
	"io/ioutil"
	"encoding/json"

	rp "github.com/mridul-sahu/grpc-go-streaming/proto"
)

const DefaultFeaturesFile = "default-data/route_guide_db.json"

type server struct {
	knownFeatures []*rp.Feature

	routeNotes map[string][]*rp.RouteNote
}


// GetFeature returns the feature at the given point.
func (s *server) GetFeature(ctx context.Context, point *rp.Point) (*rp.Feature, error) {

	return nil, nil
}

// ListFeatures lists all features contained within the given bounding Rectangle.
func (s *server) ListFeatures(rect *rp.Rectangle, stream rp.RouteGuide_ListFeaturesServer) error {

	return nil
}

// RecordRoute records a route composited of a sequence of points.
//
// It gets a stream of points, and responds with statistics about the "trip":
// number of points,  number of known features visited, total distance traveled, and
// total time spent.
func (s *server) RecordRoute(stream rp.RouteGuide_RecordRouteServer) error {

	return nil
}


// RouteChat receives a stream of message/location pairs, and responds with a stream of all
// previous messages at each of those locations.
func (s *server) RouteChat(stream rp.RouteGuide_RouteChatServer) error {

	return nil
}

func (s *server) loadFeatures(filePath string) {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to load default features: %v", err)
	}
	if err := json.Unmarshal(file, &s.knownFeatures); err != nil {
		log.Fatalf("Failed to load default features: %v", err)
	}
}


func newServer() *server {
	s := &server{routeNotes: make(map[string][]*rp.RouteNote)}
	s.loadFeatures(DefaultFeaturesFile)
	return s
}


func main() {

	lis, err := net.Listen("tcp", ":3000")

	if err != nil {
		log.Printf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()

	rp.RegisterRouteGuideServer(s, newServer())
	reflection.Register(s) //Just in case we decide to something interesting with this.

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v ", err)
	}
}