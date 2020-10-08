package validator

import "net/url"

func OptimizeDeliveryValidator(data map[string]float64)  (err url.Values)  {
	errs :=  url.Values{}
	if data["lat"] == 0 {
		errs.Add("lat", "The lat is required!")
	}
	if data["lng"] == 0 {
		errs.Add("lng", "The lng is required!")
	}

	return errs
}