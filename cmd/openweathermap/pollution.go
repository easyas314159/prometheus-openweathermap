package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/prometheus/client_golang/prometheus"
)

type airPollution struct {
	Coord coord `json:"coord"`
	List  []struct {
		Timestamp int64 `mapstructure:"dt"`
		Main      struct {
			AirQualityIndex float64 `mapstructure:"aqi"`
		} `mapstructure:"main"`
		Components map[string]float64 `json:"components"`
	} `json:"list"`
}

func (env *environment) collectPollution(s *station) collectorFunc {
	endpoint := env.BaseURL
	endpoint.Path = path.Join(endpoint.Path, "air_pollution")

	labels := prometheus.Labels{
		"station": s.Name,
	}

	metricAirPollution := env.Metrics.AirPollution.MustCurryWith(labels)
	metricAirQualityIndex := env.Metrics.AirQualityIndex.With(labels)

	return func(ctx context.Context) error {
		url := endpoint

		q := url.Query()
		q.Add("lat", fmt.Sprintf("%.6f", s.Latitude))
		q.Add("lon", fmt.Sprintf("%.6f", s.Longitude))
		url.RawQuery = q.Encode()

		log.Printf("Collecting pollution for station %s", s.Name)
		res, err := http.Get(url.String())
		if err != nil {
			return err
		}
		defer res.Body.Close()

		var data airPollution
		if err = json.NewDecoder(res.Body).Decode(&data); err != nil {
			return err
		}

		metricAirQualityIndex.Set(data.List[0].Main.AirQualityIndex)

		for key, value := range data.List[0].Components {
			metricAirPollution.With(prometheus.Labels{
				"component": key,
			}).Set(value)
		}

		return nil
	}
}