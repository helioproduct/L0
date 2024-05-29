package main

import (
	"L0/api"
	"L0/config"
	"L0/models"
	"L0/nats"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/nats-io/stan.go"
)

func fatalError(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		fatalError(err)
	}

	orderModel, err := models.MakeCachedOrderModel(cfg.DB)
	if err != nil {
		fatalError(err)
	}
	defer fmt.Println("close db")
	defer orderModel.Close()

	sc, err := stan.Connect(cfg.NATS.ClusterID, cfg.NATS.ClientID, stan.NatsURL(cfg.NATS.URL))
	if err != nil {
		fatalError(err)
	}
	sub, err := sc.Subscribe("orders", nats.MakeOrderHandler(orderModel))
	if err != nil {
		fatalError(err)
	}
	defer fmt.Println("close nats conn")
	defer sc.Close()
	defer sub.Unsubscribe()

	fmt.Println("serving HTTP")
	server := &http.Server{
		Addr:    cfg.HTTP.Addr,
		Handler: api.MakeHandler(orderModel),
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}
}
