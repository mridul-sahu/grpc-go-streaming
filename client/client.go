package main

import (
	rp "github.com/mridul-sahu/grpc-go-streaming/proto"
	"google.golang.org/grpc"
	"log"
	"time"
	"golang.org/x/net/context"
	"io"
	"math/rand"
)

func randomPoint(r *rand.Rand) *rp.Point {
	lat := (r.Int31n(180) - 90) * 1e7
	long := (r.Int31n(360) - 180) * 1e7
	return &rp.Point{Latitude: lat, Longitude: long}
}

// printFeature gets the feature for the given point.
func printFeature(client rp.RouteGuideClient, point *rp.Point) {
	log.Printf("Getting feature for point %v", point)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	feature, err := client.GetFeature(ctx, point)
	if err != nil {
		log.Fatalf("%v.GetFeatures failed: %v: ", client, err)
	}
	log.Println(feature)
}

// printFeatures lists all the features within the given bounding Rectangle.
func printFeatures(client rp.RouteGuideClient, rect *rp.Rectangle) {
	log.Printf("Looking for features within %v", rect)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.ListFeatures(ctx, rect)
	if err != nil {
		log.Fatalf("%v.ListFeatures failed: %v", client, err)
	}
	for {
		feature, err := stream.Recv()

		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("%v.ListFeatures failed while recieving from stream: %v", client, err)
		}

		log.Println(feature)
	}
}

// runRecordRoute sends a sequence of points to server and expects to get a RouteSummary from server.
func runRecordRoute(client rp.RouteGuideClient) {
	// Create a random number of random points
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pointCount := int(r.Int31n(100)) + 2 // Traverse 2 to 101 points
	var points []*rp.Point

	// Want at least one featured point
	points = append(points,  &rp.Point{Latitude: 409146138, Longitude: -746188906})
	// Rest can be random
	for i := 0; i < pointCount-1; i++ {
		points = append(points, randomPoint(r))
	}
	log.Printf("Traversing %d points.", pointCount)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.RecordRoute(ctx)
	if err != nil {
		log.Fatalf("%v.RecordRoute failed: %v", client, err)
	}

	for _, point := range points {
		if err := stream.Send(point); err != nil {
			log.Fatalf("%v.RecordRouteSend(%v): %v", stream, point, err)
		}
		//Waiting for a while to elapse some times
		time.Sleep(100 * time.Millisecond)
	}

	reply, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.RecordRouteCloseAndRecv failed: %v", stream, err)
	}

	log.Printf("Route summary: %v", reply)
}

// runRouteChat receives a sequence of route notes, while sending notes for various locations.
func runRouteChat(client rp.RouteGuideClient) {
	// I will make this funny.
	notes := []*rp.RouteNote{
		{Location: &rp.Point{Latitude: 0, Longitude: 1}, Message: "I was here!"},
		{Location: &rp.Point{Latitude: 0, Longitude: 2}, Message: "I was here too."},
		{Location: &rp.Point{Latitude: 0, Longitude: 3}, Message: "Here is the exit!"},
		// Because we want replies :-)
		{Location: &rp.Point{Latitude: 0, Longitude: 1}, Message: "Ummm..."},
		{Location: &rp.Point{Latitude: 0, Longitude: 2}, Message: "Is this..."},
		{Location: &rp.Point{Latitude: 0, Longitude: 3}, Message: "I am stuck in a loop, aren't I?"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.RouteChat(ctx)
	if err != nil {
		log.Fatalf("%v.RouteChat failed: %v", client, err)
	}

	waitc := make(chan struct{})

	//Because go is awesome!
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				close(waitc)
				return

			} else if err != nil {
				log.Fatalf("%v.RouteChatReceive Failed: %v", stream, err)
			}
			log.Printf("%s at point: %v", in.Message, in.Location)
		}
	}()

	for _, note := range notes {
		if err := stream.Send(note); err != nil {
			log.Fatalf("%v.RouteChaSend Failed: %v", stream, err)
		}
	}

	stream.CloseSend()
	<-waitc
}

func main() {
	//Because I know it's only going to be here!
	conn, err := grpc.Dial("localhost:3000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := rp.NewRouteGuideClient(conn)

	// Looking for a valid feature
	printFeature(client, &rp.Point{Latitude: 409146138, Longitude: -746188906})

	// Feature missing.
	printFeature(client, &rp.Point{Latitude: 0, Longitude: 0})

	// Looking for features between 40, -75 and 42, -73, i.e all Default Features
	printFeatures(client, &rp.Rectangle{
		Lo: &rp.Point{Latitude: 400000000, Longitude: -750000000},
		Hi: &rp.Point{Latitude: 420000000, Longitude: -730000000},
	})

	// RecordRoute
	runRecordRoute(client)

	// RouteChat
	runRouteChat(client)
}
