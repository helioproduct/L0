package nats

import (
	"L0/models"
	"encoding/json"
	"fmt"
	"github.com/nats-io/stan.go"
)

func MakeOrderHandler(orderModel models.OrderModel) func(*stan.Msg) {
	return func(m *stan.Msg) {
		var order models.Order
		if err := json.Unmarshal(m.Data, &order); err != nil {
			fmt.Printf("info: nats order handler can't unmarshal: %v\n", err)
			return
		}
		if err := orderModel.Insert(order); err != nil {
			fmt.Printf("info: can't insert order: %v | %v\n", order.Uid, err)
			return
		}
	}
}
