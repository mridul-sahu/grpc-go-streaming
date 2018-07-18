# grpc-go-streaming

## Description

The route guide server and client demonstrate how to use grpc go libraries to
perform unary, client streaming, server streaming and full duplex RPCs.

Please refer to [gRPC Basics: Go](https://grpc.io/docs/tutorials/basic/go.html) for more information.

See the definition of the route guide service in proto/route_guide.proto.

## Run the sample code
To compile and run the server, assuming you are in the root of the
folder, i.e., grpc-go-streaming/, simply:
 
```sh
$ go run server/server.go
```
 
Likewise, to run the client:
 
```sh
 $ go run client/client.go
```
**Note: There are no command line flags like the original example**