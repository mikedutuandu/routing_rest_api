package model

import (
	"net/url"

	//"regexp"
	"time"
)

type Order struct {
	ID         uint      `gorm:"primary_key"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	PickupDate string    `json:"pickup_date"`

	OrderID string `json:"internal_tn"`
	//AdditionalData                string
	AgentApplicationIDID string `json:"agent_application_id_id"` //add
	//AgentIDID                     int
	CodAmtToCollect   string `json:"cod_amt_to_collect"`
	ConsigneeAddress  string `json:"consignee_address"`
	ConsigneeCity     string `json:"consignee_city"`
	ConsigneeCountry  string `json:"consignee_country"`
	ConsigneeEmail    string `json:"consignee_email"`
	ConsigneeName     string `json:"consignee_name"`
	ConsigneeNumber   string `json:"consignee_number"`
	ConsigneePostal   string `json:"consignee_postal"`
	ConsigneeProvince string `json:"consignee_province"`
	ConsigneeState    string `json:"consignee_state"`
	//CreatedOn                     time.Time
	DeliveryNote string `json:"delivery_note"`
	//HawbNo                        string
	//Incoterm                      string
	//InvoiceCreated                bool
	//IsProcessing                  bool
	//ModelLogLinkID                int
	OrderHeight         string `json:"order_height"`    //add
	OrderLabelURL       string `json:"order_label_url"` //add
	OrderLength         string `json:"order_length"`    //add
	OrderWeight         string `json:"order_weight"`    //add
	OrderWidth          string `json:"order_width"`     //add
	PaymentType         string `json:"payment_type"`    //add
	PickupAddress       string `json:"pickup_address"`
	PickupCity          string `json:"pickup_city"`
	PickupContactName   string `json:"pickup_contact_name"`
	PickupContactNumber string `json:"pickup_contact_number"`
	PickupCountry       string `json:"pickup_country"`
	PickupPostal        string `json:"pickup_postal"`
	PickupProvince      string `json:"pickup_province"`
	PickupState         string `json:"pickup_state"`
	//PrintDefaultLabel             bool
	//PrintURL                      string
	//PrivateTrackerStatusCode      string
	//PrivateTrackerUpdatedOn       time.Time
	//ServiceIDID                   int
	ShipperOrderID string `json:"shipper_order_id"` //add
	//ShipperSubAccountID           string
	//ShipperSubOrderID             string
	//StatusCodeStoreID             string
	//SubmitExternalServiceDatetime time.Time
	//SubmitFirstmileDatetime       time.Time
	//SubmitWarehouseDatetime       string
	//TrackerMainText               string
	TrackerStatusCode string `json:"tracker_status_code"` //add
	//TrackerUpdatedOn              time.Time
	TrackingNo string `json:"external_tn"`
	//UpdatedOn                     time.Time
	UploadBatchNo string `json:"upload_batch_no"`
	//WarehouseAddress              string
	FirstMileDriverId   int    `json:"first_mile_driver_id,omitempty"`
	LastMileDriverId    int    `json:"last_mile_driver_id,omitempty"`
	FirstMileDriverName string `json:"first_mile_driver_name,omitempty"`
	LastMileDriverName  string `json:"last_mile_driver_name,omitempty"`
	Status              string `json:"status"`

	PickupNote     string    `json:"pickup_note"`
	StatusNote     string    `json:"status_note"`
	RescheduleDate time.Time `json:"reschedule_date"`

	FirstMilePods []*FirstMilePodOrder `gorm:"foreignkey:OrderId",json:"first_mile_pods"`
	LastMilePods  []*LastMilePodOrder  `gorm:"foreignkey:OrderId",json:"first_mile_pods"`

	PickupId int `json:"pickup_id"`

	DeliveryDate               string  `json:"delivery_date"`
	CashOnDelivery             float32 `json:"cash_on_delivery,omitempty"`
	CashOnDeliveryCurrency     string  `json:"cash_on_delivery_currency"`
	CustomerSignatureFirstMile string  `json:"customer_signature_first_mile"`
	CustomerSignatureLastMile  string  `json:"customer_signature_last_mile"`
	Duplicated                 bool    `json:"duplicated"`

	PickupStart string `json:"pickup_start"`
	PickupEnd   string `json:"pickup_end"`

	DeliveryStart string `json:"delivery_start"`
	DeliveryEnd   string `json:"delivery_end"`

	PickupLat     float64 `json:"pickup_lat"`
	PickupLng     float64 `json:"pickup_lng"`
	DeliveryLat   float64 `json:"delivery_lat"`
	DeliveryLng   float64 `json:"delivery_lng"`
	AddLatLng     string  `json:"add_lat_lng"`
	SolutionOrder int32   `json:"solution_order"`
	ArrivalTime   string  `json:"arrival_time"`
	FinishTime    string  `json:"finish_time"`
	Duration      int32   `json:"duration"`

	PickupRegion    string  `json:"pickup_region"`
	DeliveryRegion    string  `json:"delivery_region"`
	AddRegion     string  `json:"add_region"`

	ImportId          int       `json:"import_id"`
}

func (o *Order) IsValid() (errs url.Values) {
	// check if the name empty
	if o.OrderID == "" {
		errs.Add("order_id", "The order_id is required!")
	}
	if o.OrderID == "" {
		errs.Add("order_id", "The order_id is required!")
	}

	return errs
}

func (o *Order) AfterFind() (err error) {
	o.PickupDate = o.PickupDate[0:10]
	o.DeliveryDate = o.DeliveryDate[0:10]
	return
}
