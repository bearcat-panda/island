package main

import (
	"errors"
	"island/crawler_distributed/consul"
	"island/crawler_distributed/gredis"
	pb "island/crawler_distributed/proto"

	"log"

	"island/crawler/config"
	"island/crawler/engine"
	"island/crawler/scheduler"
	"island/crawler/xcar/parser"
	itemsaver "island/crawler_distributed/persist/client"
	"island/crawler_distributed/rpcsupport"
	worker "island/crawler_distributed/worker/client"
)

func main() {
	var (
		itemChan chan pb.Item
		err error
	)
	gredis.Setup()
	// 这里从服务发现中 发现地址
	if len(consul.Find("item")) > 0{
		itemChan, err = itemsaver.ItemSaver(consul.Find("item")[0])
		if err != nil {
			panic(err)
		}
	}

	pool, err := createClientPool(consul.Find("car"))
	if err != nil {
		panic(err)
	}

	processor := worker.CreateProcessor(pool)
	e := engine.ConcurrentEngine{
		Scheduler:        &scheduler.QueuedScheduler{},
		WorkerCount:      2,
		ItemChan:         itemChan,
		RequestProcessor: processor,
	}

	e.Run(engine.Request{
		Url: "http://newcar.xcar.com.cn/car/",
		Parser: engine.NewFuncParser(
			parser.ParseCarList,
			config.ParseCarList),
	})
}

func createClientPool(
	hosts []string) (chan pb.ReptilesClient, error) {
	var clients []pb.ReptilesClient
	for _, h := range hosts {
		client, err := rpcsupport.NewClient(h)
		if err == nil {
			clients = append(clients, client)
			log.Printf("Connected to %s", h)
		} else {
			log.Printf(
				"Error connecting to %s: %v",
				h, err)
		}
	}

	if len(clients) == 0 {
		return nil, errors.New(
			"no connections available")
	}
	out := make(chan pb.ReptilesClient)
	go func() {
		for {
			for _, client := range clients {
				out <- client
			}
		}
	}()
	return out, nil
}
