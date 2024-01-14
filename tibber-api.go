package main

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

type TibberPrice struct {
	Total    float32   `json:"total"`
	StartsAt time.Time `json:"startsAt"`
}

type TibberPriceInfo struct {
	Current  TibberPrice   `json:"current"`
	Today    []TibberPrice `json:"today"`
	Tomorrow []TibberPrice `json:"tomorrow"`
}

type TibberSubscription struct {
	PriceInfo TibberPriceInfo `json:"priceInfo"`
}

type TibberHomes struct {
	Subscription TibberSubscription `json:"currentSubscription"`
}

type TibberViewer struct {
	Homes []TibberHomes `json:"homes"`
}

type TibberData struct {
	Viewer TibberViewer `json:"viewer"`
}

type TibberResponse struct {
	Data TibberData `json:"data"`
}

func TibberAPIGetPrices(token string) (*TibberPriceInfo, error) {
	target := "https://api.tibber.com/v1-beta/gql"
	data := `{ "query": "{viewer {homes {currentSubscription {priceInfo {current {total startsAt} today {total startsAt} tomorrow {total startsAt} } }}}}" }`
	r, _ := http.NewRequest("POST", target, strings.NewReader(data))

	resp, err := RetryHTTPJSONRequest(r, token)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var m TibberResponse
	if err := UnmarshalValidateBody(resp.Body, &m); err != nil {
		return nil, err
	}
	if len(m.Data.Viewer.Homes) == 0 {
		return nil, errors.New("no homes found")
	}
	return &m.Data.Viewer.Homes[0].Subscription.PriceInfo, nil
}
