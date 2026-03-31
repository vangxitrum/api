package ip_helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
)

var logger *slog.Logger = slog.Default()

type Ip2LocationHelper struct {
	apiKey string
}

func NewIp2LocationHelper(apiKey string, options ...Option) IpHelper {
	h := &Ip2LocationHelper{
		apiKey: apiKey,
	}

	for _, option := range options {
		option(h)
	}

	return h
}

type IpInfoResponse struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region_name"`
	City        string  `json:"city_name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

func (i *Ip2LocationHelper) GetIpInfo(ip string) (*IpInfo, error) {
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 3 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 3 * time.Second,
	}
	client := http.Client{
		Transport: transport,
		Timeout:   3 * time.Second,
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.ip2location.io/?key=%s&ip=%s", i.apiKey, ip),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ipInfoResponse IpInfoResponse
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&ipInfoResponse); err != nil {
		return nil, err
	}

	rs := &IpInfo{
		IP:          ipInfoResponse.IP,
		CountryCode: ipInfoResponse.CountryCode,
		Region:      ipInfoResponse.Region,
		Latitude:    ipInfoResponse.Latitude,
		Longitude:   ipInfoResponse.Longitude,
		Continent:   countryToContinentMapping[ipInfoResponse.CountryCode],
		City:        ipInfoResponse.City,
	}

	logger.Debug(
		"ip info",
		slog.Any("info", rs),
		slog.Any("ip", ip),
		slog.Any("Body", string(data)),
	)

	return rs, nil
}
