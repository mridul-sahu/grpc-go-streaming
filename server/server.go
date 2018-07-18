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
	"github.com/golang/protobuf/proto"
	"math"
	"time"
	"io"
	"fmt"
	"sync"
)

const DefaultFeaturesFile = "default-data/route_guide_db.json"

type server struct {
	knownFeatures []*rp.Feature

	routeNotes map[string][]*rp.RouteNote
	mu         sync.Mutex // protects routeNotes
}

func inRange(point *rp.Point, rect *rp.Rectangle) bool {

	left := math.Min(float64(rect.Lo.Longitude), float64(rect.Hi.Longitude))
	right := math.Max(float64(rect.Lo.Longitude), float64(rect.Hi.Longitude))
	top := math.Max(float64(rect.Lo.Latitude), float64(rect.Hi.Latitude))
	bottom := math.Min(float64(rect.Lo.Latitude), float64(rect.Hi.Latitude))

	if float64(point.Longitude) >= left &&
		float64(point.Longitude) <= right &&
		float64(point.Latitude) >= bottom &&
		float64(point.Latitude) <= top {
			return true
	}

	return false
}

func toRadians(num float64) float64 {
	return num * math.Pi / float64(180)
}

// calcDistance calculates the distance between two points using the "haversine" formula.
// The formula is based on http://mathforum.org/library/drmath/view/51879.html.
func calcDistance(p1 *rp.Point, p2 *rp.Point) int32 {
	const CordFactor float64 = 1e7
	const R float64 = float64(6371000) // earth radius in metres
	lat1 := toRadians(float64(p1.Latitude) / CordFactor)
	lat2 := toRadians(float64(p2.Latitude) / CordFactor)
	lng1 := toRadians(float64(p1.Longitude) / CordFactor)
	lng2 := toRadians(float64(p2.Longitude) / CordFactor)
	dlat := lat2 - lat1
	dlng := lng2 - lng1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := R * c
	return int32(distance)
}


// GetFeature returns the feature at the given point.
func (s *server) GetFeature(ctx context.Context, point *rp.Point) (*rp.Feature, error) {

	for _, feature := range s.knownFeatures {
		if proto.Equal(feature.Location, point) {
			return feature, nil
		}
	}

	// No feature was found, return an unnamed feature
	return &rp.Feature{Location: point}, nil
}

// ListFeatures lists all features contained within the given bounding Rectangle.
func (s *server) ListFeatures(rect *rp.Rectangle, stream rp.RouteGuide_ListFeaturesServer) error {

	for _, feature := range s.knownFeatures {
		if inRange(feature.Location, rect) {
			if err := stream.Send(feature); err != nil {
				log.Printf("Stream send falied: %v", err)
				return err
			}
		}
	}

	// Everything went fine
	return nil
}

// RecordRoute records a route composed of a sequence of points.
//
// It gets a stream of points, and responds with statistics about the "trip":
// number of points,  number of known features visited, total distance traveled, and
// total time spent.
func (s *server) RecordRoute(stream rp.RouteGuide_RecordRouteServer) error {

	var pointCount, featureCount, distance int32
	var lastPoint *rp.Point
	startTime := time.Now()

	for {
		point, err := stream.Recv()

		if err == io.EOF {
			endTime := time.Now()
			return stream.SendAndClose(&rp.RouteSummary{
				PointCount: pointCount,
				FeatureCount: featureCount,
				Distance: distance,
				ElapsedTime: int32(endTime.Sub(startTime).Seconds()),
			})
		} else if err != nil {
			log.Printf("Stream Recieve failed: %v", err)
			return err
		}

		pointCount++

		for _, feature := range s.knownFeatures {
			if proto.Equal(feature.Location, point) {
				featureCount++
			}
		}

		if lastPoint != nil {
			distance += calcDistance(lastPoint, point)
		}
		lastPoint = point
	}
}


// RouteChat receives a stream of message/location pairs, and responds with a stream of all
// previous messages at each of those locations.
func (s *server) RouteChat(stream rp.RouteGuide_RouteChatServer) error {

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		//Serialize location
		key := fmt.Sprintf("%d %d", in.Location.Latitude, in.Location.Longitude)

		s.mu.Lock()
		s.routeNotes[key] = append(s.routeNotes[key], in)

		// Note: this copy prevents blocking other clients while serving this one.
		// We don't need to do a deep copy, because elements in the slice are
		// insert-only and never modified.
		rn := make([]*rp.RouteNote, len(s.routeNotes[key]))
		copy(rn, s.routeNotes[key])

		s.mu.Unlock()

		for _, note := range rn {
			if err := stream.Send(note); err != nil {
				return err
			}
		}
	}
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