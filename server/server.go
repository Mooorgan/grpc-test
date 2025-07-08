package main

import (
	pb "github.com/github.com/grpc-test/proto/todo/v1"
)

type server struct {
	d db

	pb.UnimplementedTodoServiceServer
}
