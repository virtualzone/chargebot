package goshared

type PasswordProtectedRequest struct {
	Password string `json:"password"`
}

type AccessTokenRequest struct {
	PasswordProtectedRequest
	AccessToken string `json:"access_token" validate:"required"`
}

type SetChargeLimitRequest struct {
	AccessTokenRequest
	ChargeLimit int `json:"limit" validate:"min:0,max:100,required"`
}

type SetChargeAmpsRequest struct {
	AccessTokenRequest
	Amps int `json:"limit" validate:"min:0,max:16,required"`
}

type TeslaAPIVehicleEntity struct {
	VehicleID   int    `json:"vehicle_id"`
	VIN         string `json:"vin"`
	DisplayName string `json:"display_name"`
}

type TeslaAPITokenReponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

type TeslaAPIChargeState struct {
	BatteryLevel         int    `json:"battery_level"`
	ChargeAmps           int    `json:"charge_amps"`
	ChargeLimitSoC       int    `json:"charge_limit_soc"`
	ChargingState        string `json:"charging_state"`
	Timestamp            int    `json:"timestamp"`
	ConnectedChargeCable string `json:"conn_charge_cable"`
	ChargePortLatch      string `json:"charge_port_latch"`
	ChargePortDoorOpen   bool   `json:"charge_port_door_open"`
}

type TeslaAPIVehicleData struct {
	VIN         string              `json:"vin"`
	ChargeState TeslaAPIChargeState `json:"charge_state"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type PersistedTelemetryState struct {
	VIN         string `json:"vehicle_vin"`
	PluggedIn   bool   `json:"pluggedIn"`
	Charging    bool   `json:"charging"`
	SoC         int    `json:"soc"`
	Amps        int    `json:"amps"`
	ChargeLimit int    `json:"chargeLimit"`
	IsHome      bool   `json:"is_home"`
	UTC         int64  `json:"ts"`
}
