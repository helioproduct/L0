package models

import "database/sql"
import "fmt"
import "L0/config"
import "errors"
import _ "github.com/lib/pq"
import "sort"

type CachedOrderModel struct {
	db    *sql.DB
	cache map[string]*Order
}

type OrderModel interface {
	Insert(Order) error
	GetByUid(string) (*Order, error)
	ListOfUids() []string
	Close()
}

const (
	qSelectOrders = `SELECT o.order_uid, o.track_number, o.entry, o.locale, 
		o.internal_signature, o.customer_id, o.delivery_service, 
		o.shardkey, o.sm_id, o.date_created, o.oof_shard,
		
		d.id, d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
		
		p.id, p.transaction, p.request_id, p.currency, p.provider, p.amount, 
		p.payment_dt, p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders as o INNER JOIN 
		deliveries as d ON d.order_uid = o.order_uid INNER JOIN 
		payments as p ON p.order_uid = o.order_uid`

	qInsertOrder = `INSERT INTO orders(
			order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	qInsertPayment = `INSERT INTO payments(
				order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	qInsertItem = `INSERT INTO items(
					order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	qSelectOrder = `SELECT o.order_uid, o.track_number, o.entry, o.locale, 
					o.internal_signature, o.customer_id, o.delivery_service, 
					o.shardkey, o.sm_id, o.date_created, o.oof_shard,
					
					d.id, d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
					
					p.id, p.transaction, p.request_id, p.currency, p.provider, p.amount, 
					p.payment_dt, p.bank, p.delivery_cost, p.goods_total, p.custom_fee
					FROM orders as o INNER JOIN 
					deliveries as d ON d.order_uid = o.order_uid INNER JOIN 
					payments as p ON p.order_uid = o.order_uid
					WHERE o.order_uid = $1`

	qSelectItems = `SELECT id, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
	FROM items WHERE order_uid = $1`
)

func MakeCachedOrderModel(cfg config.DBConfig) (OrderModel, error) {
	connStr := fmt.Sprintf("sslmode=disable host=%s port=%s user=%s password=%s dbname=%s", cfg.Host, cfg.Port, cfg.User, cfg.Pass, cfg.DBname)
	fmt.Println(connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	model := CachedOrderModel{
		db:    db,
		cache: make(map[string]*Order),
	}
	if err := model.restoreCacheFromDB(); err != nil {
		return nil, err
	}
	return &model, nil
}

func (c *CachedOrderModel) restoreCacheFromDB() error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}

	if err := c.scanOrders(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := c.scanItems(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (c *CachedOrderModel) scanOrders(tx *sql.Tx) error {
	rows, err := tx.Query(qSelectOrders)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		order := &Order{Items: make([]Item, 0)}
		if err := rows.Scan(
			&order.Uid, &order.TrackNumber, &order.Entry, &order.Locale,
			&order.InternalSignature, &order.CustomerId, &order.DeliveryService,
			&order.Shardkey, &order.SmId, &order.DateCreated, &order.OofShard,
			&order.Delivery.dbId, &order.Delivery.Name, &order.Delivery.Phone,
			&order.Delivery.Zip, &order.Delivery.City, &order.Delivery.Address,
			&order.Delivery.Region, &order.Delivery.Email, &order.Payment.dbId,
			&order.Payment.Transaction, &order.Payment.RequestId, &order.Payment.Currency,
			&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt,
			&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal,
			&order.Payment.CustomFee); err != nil {
			return err
		}
		c.cache[order.Uid] = order
	}

	return nil
}

func (c *CachedOrderModel) scanItems(tx *sql.Tx) error {
	qSelectItems := `SELECT order_uid, id, chrt_id, track_number, price, 
					rid, name, sale, size, total_price, nm_id, brand, status
					FROM items`
	rows, err := tx.Query(qSelectItems)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item Item
		var orderUid string
		if err := rows.Scan(
			&orderUid, &item.dbId, &item.ChrtId, &item.TrackNumber, &item.Price,
			&item.Rid, &item.Name, &item.Sale, &item.Size, &item.TotalPrice,
			&item.NmId, &item.Brand, &item.Status); err != nil {
			return err
		}
		order := c.cache[orderUid]
		order.Items = append(order.Items, item)
	}

	return nil
}

func (c *CachedOrderModel) Insert(order Order) error {
	if _, ok := c.cache[order.Uid]; ok {
		return errors.New("such an element already exists")
	}

	tx, err := c.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(qInsertOrder,
		order.Uid,
		order.TrackNumber,
		order.Entry,
		order.Locale,
		order.InternalSignature,
		order.CustomerId,
		order.DeliveryService,
		order.Shardkey,
		order.SmId,
		order.DateCreated,
		order.OofShard,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	qInsertDelivery := `INSERT INTO deliveries(
		order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = tx.Exec(qInsertDelivery,
		order.Uid,
		order.Delivery.Name,
		order.Delivery.Phone,
		order.Delivery.Zip,
		order.Delivery.City,
		order.Delivery.Address,
		order.Delivery.Region,
		order.Delivery.Email)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(qInsertPayment,
		order.Uid,
		order.Payment.Transaction,
		order.Payment.RequestId,
		order.Payment.Currency,
		order.Payment.Provider,
		order.Payment.Amount,
		order.Payment.PaymentDt,
		order.Payment.Bank,
		order.Payment.DeliveryCost,
		order.Payment.GoodsTotal,
		order.Payment.CustomFee)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, item := range order.Items {
		_, err = tx.Exec(qInsertItem,
			order.Uid,
			item.ChrtId,
			item.TrackNumber,
			item.Price,
			item.Rid,
			item.Name,
			item.Sale,
			item.Size,
			item.TotalPrice,
			item.NmId,
			item.Brand,
			item.Status)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	c.cache[order.Uid] = &order

	return nil
}

func (c *CachedOrderModel) GetByUid(uid string) (*Order, error) {
	if order, ok := c.cache[uid]; ok {
		return order, nil
	}

	var order Order
	tx, err := c.db.Begin()
	if err != nil {
		return nil, err
	}

	err = tx.QueryRow(qSelectOrder, uid).Scan(
		&order.Uid,
		&order.TrackNumber,
		&order.Entry,
		&order.Locale,
		&order.InternalSignature,
		&order.CustomerId,
		&order.DeliveryService,
		&order.Shardkey,
		&order.SmId,
		&order.DateCreated,
		&order.OofShard,
		&order.Delivery.dbId,
		&order.Delivery.Name,
		&order.Delivery.Phone,
		&order.Delivery.Zip,
		&order.Delivery.City,
		&order.Delivery.Address,
		&order.Delivery.Region,
		&order.Delivery.Email,
		&order.Payment.dbId,
		&order.Payment.Transaction,
		&order.Payment.RequestId,
		&order.Payment.Currency,
		&order.Payment.Provider,
		&order.Payment.Amount,
		&order.Payment.PaymentDt,
		&order.Payment.Bank,
		&order.Payment.DeliveryCost,
		&order.Payment.GoodsTotal,
		&order.Payment.CustomFee)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	rows, err := tx.Query(qSelectItems, order.Uid)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	defer rows.Close()
	order.Items = make([]Item, 0)
	for rows.Next() {
		var item Item
		err := rows.Scan(
			&item.dbId,
			&item.ChrtId,
			&item.TrackNumber,
			&item.Price,
			&item.Rid,
			&item.Name,
			&item.Sale,
			&item.Size,
			&item.TotalPrice,
			&item.NmId,
			&item.Brand,
			&item.Status,
		)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		order.Items = append(order.Items, item)
	}
	tx.Commit()

	c.cache[order.Uid] = &order
	return &order, nil
}

func (c *CachedOrderModel) ListOfUids() []string {
	orderUids := make([]string, 0, len(c.cache))
	for k := range c.cache {
		orderUids = append(orderUids, k)
	}
	sort.Strings(orderUids)
	return orderUids
}

func (c *CachedOrderModel) Close() {
	if c != nil {
		c.db.Close()
	}
}
