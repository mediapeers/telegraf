package librato_with_tags

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/stretchr/testify/require"
)

var (
	fakeURL   = "http://test.librato.com"
	fakeUser  = "telegraf@influxdb.com"
	fakeToken = "123456"
)

func fakeLibrato() *LibratoWithTags {
	l := NewLibratoWithTags(fakeURL)
	l.APIUser = fakeUser
	l.APIToken = fakeToken
	return l
}

func TestUriOverride(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
	defer ts.Close()

	l := NewLibratoWithTags(ts.URL)
	l.APIUser = "telegraf@influxdb.com"
	l.APIToken = "123456"
	err := l.Connect()
	require.NoError(t, err)
	err = l.Write([]telegraf.Metric{newHostMetric(int32(0), "name", "host")})
	require.NoError(t, err)
}

func TestBadStatusCode(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
	defer ts.Close()

	l := NewLibratoWithTags(ts.URL)
	l.APIUser = "telegraf@influxdb.com"
	l.APIToken = "123456"
	err := l.Connect()
	require.NoError(t, err)
	err = l.Write([]telegraf.Metric{newHostMetric(int32(0), "name", "host")})
	if err == nil {
		t.Errorf("error expected but none returned")
	} else {
		require.EqualError(
			t,
			fmt.Errorf("received bad status code, 503\n "), err.Error())
	}
}

func TestBuildMeasurements(t *testing.T) {

	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC).Unix()
	var gaugeTests = []struct {
		ptIn     telegraf.Metric
		outMeasurement *Measurement
		err      error
	}{
		{
			newHostMetric(0.0, "test1", "host1"),
			&Measurement{
				Name:        "test1",
				MeasureTime: mtime,
				Value:       0.0,
				Tags:   		 map[string]string{},
			},
			nil,
		},
		{
			newHostMetric(1.0, "test2", "host2"),
			&Measurement{
				Name:        "test2",
				MeasureTime: mtime,
				Value:       1.0,
				Tags:   		 map[string]string{},
			},
			nil,
		},
		{
			newHostMetric(10, "test3", "host3"),
			&Measurement{
				Name:        "test3",
				MeasureTime: mtime,
				Value:       10.0,
				Tags:   		 map[string]string{},
			},
			nil,
		},
		{
			newHostMetric(int32(112345), "test4", "host4"),
			&Measurement{
				Name:        "test4",
				MeasureTime: mtime,
				Value:       112345.0,
				Tags:   		 map[string]string{},
			},
			nil,
		},
		{
			newHostMetric(int64(112345), "test5", "host5"),
			&Measurement{
				Name:        "test5",
				MeasureTime: mtime,
				Value:       112345.0,
				Tags:   		 map[string]string{},
			},
			nil,
		},
		{
			newHostMetric(float32(11234.5), "test6", "host6"),
			&Measurement{
				Name:        "test6",
				MeasureTime: mtime,
				Value:       11234.5,
				Tags:   		 map[string]string{},
			},
			nil,
		},
		{
			newHostMetric("11234.5", "test7", "host7"),
			nil,
			nil,
		},
	}

	l := NewLibratoWithTags(fakeURL)
	for _, gt := range gaugeTests {
		gauges, err := l.buildMeasurements(gt.ptIn)
		if err != nil && gt.err == nil {
			t.Errorf("%s: unexpected error, %+v\n", gt.ptIn.Name(), err)
		}
		if gt.err != nil && err == nil {
			t.Errorf("%s: expected an error (%s) but none returned",
				gt.ptIn.Name(), gt.err.Error())
		}
		if len(gauges) != 0 && gt.outMeasurement == nil {
			t.Errorf("%s: unexpected gauge, %+v\n", gt.ptIn.Name(), gt.outMeasurement)
		}
		if len(gauges) == 0 {
			continue
		}
		if gt.err == nil && !reflect.DeepEqual(gauges[0], gt.outMeasurement) {
			t.Errorf("%s: \nexpected %+v\ngot %+v\n",
				gt.ptIn.Name(), gt.outMeasurement, gauges[0])
		}
	}
}

func newHostMetric(value interface{}, name, host string) telegraf.Metric {
	m, _ := metric.New(
		name,
		map[string]string{"host": host},
		map[string]interface{}{"value": value},
		time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
	)
	return m
}

func TestBuildMeasurementWithTags(t *testing.T) {
	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	pt1, _ := metric.New(
		"test1",
		map[string]string{"hostname": "192.168.0.1", "tag1": "value1"},
		map[string]interface{}{"value": 0.0},
		mtime,
	)
	pt2, _ := metric.New(
		"test2",
		map[string]string{"hostnam": "192.168.0.1", "tag1": "value1"},
		map[string]interface{}{"value": 1.0},
		mtime,
	)
	pt3, _ := metric.New(
		"test3",
		map[string]string{
			"hostname": "192.168.0.1",
			"tag2":     "value2",
			"tag1":     "value1"},
		map[string]interface{}{"value": 1.0},
		mtime,
	)
	pt4, _ := metric.New(
		"test4",
		map[string]string{
			"hostname": "192.168.0.1",
			"tag2":     "value2",
			"tag1":     "value1"},
		map[string]interface{}{"value": 1.0},
		mtime,
	)
	var measurementTests = []struct {
		ptIn     telegraf.Metric
		template string
		outMeasurement *Measurement
		err      error
	}{

		{
			pt1,
			"hostname",
			&Measurement{
				Name:        "test1",
				MeasureTime: mtime.Unix(),
				Value:       0.0,
        Tags:        map[string]string{"hostname": "192.168.0.1", "tag1": "value1"},
			},
			nil,
		},
		{
			pt2,
			"hostname",
			&Measurement{
				Name:        "test2",
				MeasureTime: mtime.Unix(),
				Value:       1.0,
			},
			fmt.Errorf("undeterminable Source type from Field, hostname"),
		},
		{
			pt3,
			"tags",
			&Measurement{
				Name:        "test3",
				MeasureTime: mtime.Unix(),
				Value:       1.0,
				Tags:   		 map[string]string{},
			},
			nil,
		},
		{
			pt4,
			"hostname.tag2",
			&Measurement{
				Name:        "test4",
				MeasureTime: mtime.Unix(),
				Value:       1.0,
				Tags:   		 map[string]string{
    			"hostname": "192.168.0.1",
		    	"tag2":     "value2",
    			"tag1":     "value"},
			},
			nil,
		},
	}

	l := NewLibratoWithTags(fakeURL)
	for _, gt := range measurementTests {
		l.Template = gt.template
		measurements, err := l.buildMeasurements(gt.ptIn)
		if err != nil && gt.err == nil {
			t.Errorf("%s: unexpected error, %+v\n", gt.ptIn.Name(), err)
		}
		if gt.err != nil && err == nil {
			t.Errorf(
				"%s: expected an error (%s) but none returned",
				gt.ptIn.Name(),
				gt.err.Error())
		}
		if len(measurements) == 0 {
			continue
		}
		if gt.err == nil && !reflect.DeepEqual(measurements[0], gt.outMeasurement) {
			t.Errorf(
				"%s: \nexpected %+v\ngot %+v\n",
				gt.ptIn.Name(),
				gt.outMeasurement, measurements[0])
		}
	}
}
