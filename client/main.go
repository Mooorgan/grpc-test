package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/github.com/grpc-test/proto/todo/v1"
)

// addTask calls the AddTask unary endpoint with a AddTaskRequest
// generated from description and dueDate.
func addTask(c pb.TodoServiceClient, description string, dueDate time.Time) uint64 {
	req := &pb.AddTaskRequest{
		Description: description,
		DueDate:     timestamppb.New(dueDate),
	}
	res, err := c.AddTask(context.Background(), req)
	if err != nil {
		panic(err)
	}

	fmt.Printf("added task: %d\n", res.Id)
	return res.Id
}

// printTasks calls the ListTasks server streaming endpoint
// and displays the Tasks on stdout.
func printTasks(c pb.TodoServiceClient) {
	req := &pb.ListTasksRequest{}
	stream, err := c.ListTasks(context.Background(), req)

	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	for {
		res, err := stream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("unexpected error: %v", err)
		}

		fmt.Println(res.Task.String(), "overdue: ", res.Overdue)
	}
}

// updateTasks calls the UpdateTasks client streaming endpoint
// with the provided reqs.
func updateTasks(c pb.TodoServiceClient, reqs ...*pb.UpdateTasksRequest) {
	stream, err := c.UpdateTasks(context.Background())

	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	for _, req := range reqs {
		if err := stream.Send(req); err != nil {
			log.Fatalf("unexpected error: %v", err)
		}

		if req.Task != nil {
			fmt.Printf("updated task with id: %d\n", req.Task.Id)
		}
	}

	if _, err = stream.CloseAndRecv(); err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
}

// deleteTasks calls the DeleteTasks bidi streaming endpoint
// with the provided reqs.
func deleteTasks(c pb.TodoServiceClient, reqs ...*pb.DeleteTasksRequest) {
	stream, err := c.DeleteTasks(context.Background())

	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	waitc := make(chan struct{})

	go func() {
		for {
			_, err := stream.Recv()

			if err == io.EOF {
				close(waitc)
				break
			}
			if err != nil {
				log.Fatalf("error while receiving: %v\n", err)
			}

			log.Println("deleted tasks")
		}
	}()

	for _, req := range reqs {
		if err := stream.Send(req); err != nil {
			return
		}
	}
	if err := stream.CloseSend(); err != nil {
		return
	}

	<-waitc
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		log.Fatalln("usage: client [IP_ADDR]")
	}

	addr := args[0]
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.Dial(addr, opts...)

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			log.Fatalf("unexpected error: %v", err)
		}
	}(conn)

	c := pb.NewTodoServiceClient(conn)

	fmt.Println("--------ADD--------")
	dueDate := time.Now().Add(5 * time.Second)
	id1 := addTask(c, "This is a task", dueDate)
	id2 := addTask(c, "This is another task", dueDate)
	id3 := addTask(c, "And yet another task", dueDate)
	fmt.Println("-------------------")

	fmt.Println("--------LIST-------")
	printTasks(c)
	fmt.Println("-------------------")

	fmt.Println("-------UPDATE------")
	updateTasks(c, []*pb.UpdateTasksRequest{
		{Task: &pb.Task{Id: id1, Description: "A better name for the task"}},
		{Task: &pb.Task{Id: id2, DueDate: timestamppb.New(dueDate.Add(5 * time.Hour))}},
		{Task: &pb.Task{Id: id3, Done: true}},
	}...)
	printTasks(c)
	fmt.Println("-------------------")

	fmt.Println("-------DELETE------")
	deleteTasks(c, []*pb.DeleteTasksRequest{
		{Id: id1},
		{Id: id2},
		{Id: id3},
	}...)

	printTasks(c)
	fmt.Println("-------------------")
}
