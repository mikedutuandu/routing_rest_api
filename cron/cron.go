package cron

import (
	"janio-backend/db"
	"janio-backend/services"

	"github.com/jasonlvhit/gocron"
)

func DeletePickupNoOrders() {
	sql1 := `delete from pickups where pickups.id in (select pickups.id
			from pickups left join orders o on pickups.id = o.pickup_id
			group by o.pickup_id
			having count(o.order_id) = 0)`
	db.DbManager().Exec(sql1)

	sql2 := `delete from pickups where pickups.id in (select pickups.id
		from pickups left join orders o on pickups.id = o.pickup_id
		where o.status = "ORDER_INFO_RECEIVED"
		group by o.pickup_id
		having count(o.order_id) = 0)`
	db.DbManager().Exec(sql2)
}

func UpdateFailImport() {
	sql3 := `UPDATE imports
		SET status ="FAIL"
		WHERE  created_at < DATE_SUB( NOW(), INTERVAL 10 minute ) AND status = "PENDING";
		`
	db.DbManager().Exec(sql3)
}

func Init1() {
	x := gocron.NewScheduler()
	x.Every(60).Minutes().Do(DeletePickupNoOrders)
	x.Every(30).Seconds().Do(services.ReschedulePickups)
	x.Every(10).Seconds().Do(UpdateFailImport)
	//x.Every(1).Day().At("16:52").Do(services.ReschedulePickups)
	<-x.Start()
}
