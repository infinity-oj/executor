package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	proto "github.com/infinity-oj/api/protobuf-spec"
	"github.com/infinity-oj/executor/internal/consul"
	"google.golang.org/grpc"
)

const (
	TARGET = "consul://10.0.0.233:8500/Judgements"
)

func main() {
	consul.Init()
	// Set up a connection to the server.
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	conn, err := grpc.DialContext(ctx, TARGET, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := proto.NewJudgementsClient(conn)

	// Contact the server and print out its response.
	for {

		res, err := pullJudgement(c, "executor/elf")
		if err != nil {
			log.Fatal(err)
		}
		if res.GetToken() == "" {
			log.Printf("nothing...")
			time.Sleep(time.Second * 5)
			continue
		}

		log.Printf("Get judgement, token: %s", res.GetToken())

		if err := ioutil.WriteFile("elf", res.Slots[0].Value, 0755); err != nil {
			log.Fatal(err)
		}

		cmd := exec.Command("./elf")

		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}

		stdout, err := os.OpenFile("stdout", os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			log.Fatalln(err)
		}
		stderr, err := os.OpenFile("stderr", os.O_CREATE|os.O_WRONLY, 0777)
		if err != nil {
			log.Fatalln(err)
		}

		cmd.Stdout = stdout
		cmd.Stderr = stderr

		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		_, err = stdin.Write(res.Slots[1].Value)
		if err != nil {
			log.Fatal(err)
		}

		cmd.Wait()
		stdin.Close()
		stdout.Close()
		stderr.Close()

		stdOut, err := ioutil.ReadFile("stdout")
		stdErr, err := ioutil.ReadFile("stderr")


		pushRes, err := pushJudgement(c, res.GetToken(), [][]byte{stdOut})
		fmt.Println(stdOut)
		fmt.Println(stdErr)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(pushRes.GetStatus())

		os.Remove("stdin")
		os.Remove("stdout")
		os.Remove("stderr")

		time.Sleep(time.Second)
	}
}

func pullJudgement(client proto.JudgementsClient, tp string) (*proto.PullJudgementResponse, error) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	req := &proto.PullJudgementRequest{
		Type: tp,
	}

	res, err := client.PullJudgement(ctx, req)
	if err != nil {
		log.Fatalf("could not get judgement: %v", err)
		return nil, err
	}

	return res, nil
}

func pushJudgement(client proto.JudgementsClient, token string, data [][]byte) (*proto.PushJudgementResponse, error) {

	var slots []*proto.Slot
	for k, v := range data {
		slot := &proto.Slot{
			Id:    uint32(k),
			Value: v,
		}
		slots = append(slots, slot)
	}

	req := &proto.PushJudgementRequest{
		Token: token,
		Slots: slots,
	}

	res, err := client.PushJudgement(context.TODO(), req)
	if err != nil {
		return nil, err
	}
	return res, err
}
