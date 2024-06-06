package main

import (
	"L0/config"
	"L0/models"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/stan.go"
)

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// randomElement returns a random element from a given slice of strings
func randomElement(slice []string) string {
	return slice[rand.Intn(len(slice))]
}

// generateUID generates a random UID in the format "b563feb7b2b84b6"
func generateUID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

var (
	names   = []string{"Nikolay Popov", "Ivan Ivanov", "Sergey Petrov", "Dmitry Sidorov", "Alexander Smirnov"}
	cities  = []string{"Kiryat Mozkin", "Moscow", "New York", "Los Angeles", "London"}
	regions = []string{"Kraiot", "Moscow Region", "California", "New York", "Greater London"}
	brands  = []string{"Vivienne Sabo", "Maybelline", "L'Oreal", "Max Factor", "Revlon"}
)

// setRealyUnique modifies the order to ensure unique values and randomizes some data
func setRealyUnique(o models.Order) models.Order {
	o.Uid = generateUID()
	o.Delivery.Name = randomElement(names)
	o.Delivery.Phone = "+972" + strconv.Itoa(rand.Intn(1000000000))
	o.Delivery.City = randomElement(cities)
	o.Delivery.Address = "Address_" + randomString(10)
	o.Delivery.Region = randomElement(regions)
	o.Delivery.Email = randomString(5) + "@example.com"
	o.Payment.Transaction = o.Uid
	o.Payment.Provider = "Provider_" + randomString(4)
	o.Payment.Bank = "Bank_" + randomString(6)

	itemsCount := rand.Intn(5) + 1 // Randomize number of items between 1 and 5
	item := o.Items[0]
	newItems := make([]models.Item, itemsCount)
	for i := 0; i < itemsCount; i++ {
		newItems[i] = item
		newItems[i].ChrtId = rand.Intn(1000000)
		newItems[i].TrackNumber = randomString(10)
		newItems[i].Price = rand.Intn(1000)
		newItems[i].Rid = randomString(15)
		newItems[i].Name = "Item_" + randomString(6)
		newItems[i].TotalPrice = newItems[i].Price * (100 - newItems[i].Sale) / 100
		newItems[i].NmId = rand.Intn(1000000)
		newItems[i].Brand = randomElement(brands)
		newItems[i].Status = rand.Intn(1000)
	}
	o.Items = newItems

	return o
}

// fatalError prints the error message and exits the program
func fatalError(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

// msg contains the template order message
var msg = []byte(`{
    "order_uid": "",
    "track_number": "WBILMTRACK",
    "entry": "WBIL",
    "delivery": {
        "name": "Nikolay Popov",
        "phone": "+9720000000",
        "zip": "2639809",
        "city": "Kiryat Mozkin",
        "address": "Ploshad Mira 15",
        "region": "Kraiot",
        "email": "email@gmail.com"
    },
    "payment": {
        "transaction": "b563feb7b2b84b6",
        "request_id": "",
        "currency": "USD",
        "provider": "wbpay",
        "amount": 1817,
        "payment_dt": 1637907727,
        "bank": "alpha",
        "delivery_cost": 1500,
        "goods_total": 317,
        "custom_fee": 0
    },
    "items": [
        {
            "chrt_id": 9934930,
            "track_number": "WBILMTRACK",
            "price": 453,
            "rid": "ab4219087a764ae0b",
            "name": "Mascaras",
            "sale": 30,
            "size": "0",
            "total_price": 317,
            "nm_id": 2389212,
            "brand": "Vivienne Sabo",
            "status": 202
        }
    ],
    "locale": "en",
    "internal_signature": "",
    "customer_id": "",
    "delivery_service": "meest",
    "shardkey": "9",
    "sm_id": 99,
    "date_created": "2021-11-26T06:22:19Z",
    "oof_shard": "1"
}`)

// breakData randomly truncates the data
func breakData(data []byte) []byte {
	return data[:rand.Intn(len(data))]
}

// breakOrder randomly removes a field from the order
func breakOrder(data []byte) []byte {
	r := rand.Intn(4)
	p := make(map[string]interface{})
	json.Unmarshal(data, &p)
	switch r {
	case 0:
		delete(p, "delivery")
	case 1:
		delete(p, "payment")
	case 2:
		delete(p, "items")
	case 3:
		delete(p, "order_uid")
	}
	data, _ = json.Marshal(p)
	return data
}

func main() {
	cfg, err := config.GetConfig()
	cfg.NATS.ClientID = "-publisher"
	if err != nil {
		fatalError(err)
	}
	sc, err := stan.Connect(cfg.NATS.ClusterID, cfg.NATS.ClientID)
	if err != nil {
		fatalError(err)
	}

	rand.Seed(time.Now().UnixNano())
	ordersAmount := rand.Intn(20) + 10 // Randomize number of messages between 10 and 30
	breakStructure := 0.15             // Probability to break order structure
	truncateData := 0.2                // Probability to truncate data

	var defo models.Order
	_ = json.Unmarshal(msg, &defo)

	for i := 0; i < ordersAmount; i++ {
		var data []byte
		o := setRealyUnique(defo)
		r := rand.Float64()
		data, _ = json.Marshal(o)

		if r < breakStructure {
			data = breakOrder(data)
		} else if r < breakStructure+truncateData {
			data = breakData(data)
		}

		sc.Publish("orders", data)
	}

	defer sc.Close()
}
