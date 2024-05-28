package main

import (
	// "L0/api"
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
	// configs
	fmt.Println("read configs")
	cfg, err := config.GetConfig()
	if err != nil {
		fatalError(err)
	}

	// cache and db
	fmt.Println("make order model")
	orderModel, err := models.MakeCachedOrderModel(cfg.DB)
	if err != nil {
		fatalError(err)
	}
	defer fmt.Println("close db")
	defer orderModel.Close()

	// nats
	fmt.Println("init nats subsribe")
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

	// http
	fmt.Println("http serve")
	server := &http.Server{
		Addr:    cfg.HTTP.Addr,
		Handler: api.MakeHandler(orderModel),
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()

	// system interrupts
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	// ждем прерываний
	<-stop
	// Завершаем работу http сервера
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}
}
