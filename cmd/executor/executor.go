package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	proto "github.com/infinity-oj/api/protobuf-spec"
	"github.com/infinity-oj/executor/interval/consul"
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

		if err := ioutil.WriteFile("elf", res.Slots[0].Value, 0644); err != nil {
			log.Fatal(err)
		}
		if err := ioutil.WriteFile("in", res.Slots[1].Value, 0644); err != nil {
			log.Fatal(err)
		}
		cmd := exec.Command("./elf", "<", "in", ">", "out")

		//读取io.Writer类型的cmd.Stdout，再通过bytes.Buffer(缓冲byte类型的缓冲器)将byte类型转化为string类型(out.String():这是bytes类型提供的接口)
		var out bytes.Buffer
		cmd.Stdout = &out

		//Run执行c包含的命令，并阻塞直到完成。  这里stdout被取出，cmd.Wait()无法正确获取stdin,stdout,stderr，则阻塞在那了
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}

		data, err := ioutil.ReadFile("out")

		pushRes, err := pushJudgement(c, res.GetToken(), [][]byte{data})
		if err != nil {
			log.Fatal(err)
		}
		log.Println(pushRes.GetStatus())

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
