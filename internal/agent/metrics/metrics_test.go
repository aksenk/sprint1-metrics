package metrics

import (
	"github.com/aksenk/go-yandex-metrics/internal/converter"
	"github.com/aksenk/go-yandex-metrics/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func Test_generateCustomMetrics(t *testing.T) {
	type want struct {
		Name string
		Type string
		//Delta int64
		Value any
	}
	tests := []struct {
		name  string
		want1 want
		want2 want
	}{
		{
			name: "test custom metrics",
			want1: want{
				Name:  "PollCount",
				Type:  "counter",
				Value: 1,
			},
			want2: want{
				Name:  "RandomValue",
				Type:  "gauge",
				Value: 1.123,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pollMetric, randMetric models.Metric
			var counter int64
			want1, err := models.NewMetric(tt.want1.Name, tt.want1.Type, tt.want1.Value)
			require.NoError(t, err)
			//if tt.want1.Type == "counter" {
			//	want1 = models.Metric{
			//		ID:    tt.want1.Name,
			//		MType: tt.want1.Type,
			//		Delta: &tt.want1.Delta,
			//		Value: nil,
			//	}
			//} else {
			//	want1 = models.Metric{
			//		ID:    tt.want1.Name,
			//		MType: tt.want1.Type,
			//		Delta: nil,
			//		Value: &tt.want1.Value,
			//	}
			//}

			want2, err := models.NewMetric(tt.want2.Name, tt.want2.Type, tt.want2.Value)
			require.NoError(t, err)
			//if tt.want2.Type == "counter" {
			//	want2 = models.Metric{
			//		ID:    tt.want2.Name,
			//		MType: tt.want2.Type,
			//		Delta: &tt.want2.Delta,
			//		Value: nil,
			//	}
			//} else {
			//	want2 = models.Metric{
			//		ID:    tt.want2.Name,
			//		MType: tt.want2.Type,
			//		Delta: nil,
			//		Value: &tt.want2.Value,
			//	}
			//}

			generateCustomMetrics(&pollMetric, &randMetric, &counter)
			if !reflect.DeepEqual(want1, pollMetric) {
				t.Error("Metrics are not equals")
			}
			assert.Equal(t, want2.ID, randMetric.ID)
			assert.Equal(t, want2.MType, randMetric.MType)
			oldRandValue := randMetric.Value
			requiredNewValue := *pollMetric.Delta + 1
			generateCustomMetrics(&pollMetric, &randMetric, &counter)
			assert.Equal(t, requiredNewValue, *pollMetric.Delta, "Value of the PollCount metric "+
				"should be incremented to 1")
			assert.NotEqualf(t, oldRandValue, randMetric.Value, "Value of the RandomValue metric "+
				"should be a random values")
		})
	}
}

func Test_getSystemMetrics(t *testing.T) {
	metrics := getSystemMetrics()
	assert.Contains(t, metrics, "Alloc", "The system handlers is not contains 'Alloc' metric")
}

func Test_convertToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{
			name:    "test uint32",
			value:   uint32(32),
			wantErr: false,
		},
		{
			name:    "test uint64",
			value:   uint64(32),
			wantErr: false,
		},
		{
			name:    "test float64",
			value:   float64(32),
			wantErr: false,
		},
		{
			name:    "test string",
			value:   "kek",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := converter.AnyToFloat64(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_getRequiredSystemMetrics(t *testing.T) {
	type args struct {
		systemMetrics   map[string]interface{}
		requiredMetrics []string
	}
	type want struct {
		Name  string
		Type  string
		Delta int64
		Value float64
	}
	tests := []struct {
		name    string
		args    args
		want    []want
		wantErr bool
	}{
		{
			name: "successful test",
			args: args{
				systemMetrics: map[string]interface{}{
					"Metric1": uint32(123),
					"Metric2": float64(321),
					"Metric3": uint64(11),
				},
				requiredMetrics: []string{"Metric1", "Metric3"},
			},
			wantErr: false,
			want: []want{
				{
					Name:  "Metric1",
					Type:  "gauge",
					Value: 123,
				},
				{
					Name:  "Metric3",
					Type:  "gauge",
					Value: 11,
				},
			},
		},
		{
			name: "unsuccessful test",
			args: args{
				systemMetrics: map[string]interface{}{
					"Metric3": uint64(0),
				},
				requiredMetrics: []string{"Metric1", "Metric3"},
			},
			wantErr: true,
			want: []want{
				{
					Name:  "Metric1",
					Type:  "gauge",
					Value: float64(123),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultMetrics := getRequiredSystemMetrics(tt.args.systemMetrics, tt.args.requiredMetrics)
			// reflect.DeepEqual работает некорректно когда мы работаем со слайсом карт, поэтому проверяем сами
			// если длина не совпадает - сразу выдаём ошибку
			// далее перебираем объекты из первого слайса и проверяем есть ли они во втором слайсе
			if len(resultMetrics) != len(tt.want) {
				require.Equal(t, tt.want, resultMetrics)
			}
			var wantedMetrics []models.Metric
			for _, v := range tt.want {
				m, err := models.NewMetric(v.Name, v.Type, v.Value)
				require.NoError(t, err)
				wantedMetrics = append(wantedMetrics, m)
			}
			isEq := false
			for _, rm := range resultMetrics {
				for _, vm := range wantedMetrics {
					if reflect.DeepEqual(rm, vm) {
						isEq = true
						break
					}
				}
			}
			if !tt.wantErr {
				assert.Truef(t, isEq, "Incorrect result handlers map\n"+
					"The maps must be equal\nWant: %+v\nGot: %+v",
					tt.want, resultMetrics)
			} else {
				assert.Falsef(t, isEq, "Incorrect result handlers map\n"+
					"The maps don't have to be equal\nWant: %+v\nGot: %+v",
					tt.want, resultMetrics)
			}
		})
	}
}

//func TestGetMetrics(t *testing.T) {
//	type args struct {
//		c chan []models.Metric
//		s time.Duration
//		r []string
//	}
//	type want struct {
//		Name  string
//		Type  string
//		Delta int64
//		Value float64
//	}
//	tests := []struct {
//		name       string
//		args       args
//		checkAfter time.Duration
//		wantErr    bool
//		want       want
//	}{
//		{
//			name: "successful test with custom metric",
//			args: args{
//				s: time.Duration(time.Millisecond * 500),
//				r: []string{},
//				c: make(chan []models.Metric, 1),
//			},
//			checkAfter: time.Duration(time.Millisecond * 750),
//			wantErr:    false,
//			want: want{
//				Name:  "PollCount",
//				Type:  "counter",
//				Delta: 2,
//			},
//		},
//		//{
//		//	name: "successful test with system metric",
//		//	args: args{
//		//		s: time.Duration(time.Millisecond * 500),
//		//		r: []string{"LastGC"},
//		//		c: make(chan []models.Metric, 1),
//		//	},
//		//	checkAfter: time.Duration(time.Millisecond * 750),
//		//	wantErr:    false,
//		//	want: models.Metric{
//		//		ID:  "LastGC",
//		//		MType:  "gauge",
//		//		Value: 0,
//		//	},
//		//},
//		//{
//		//	name: "unsuccessful test",
//		//	args: args{
//		//		s: time.Duration(time.Millisecond * 500),
//		//		r: []string{"LastGC"},
//		//		c: make(chan []models.Metric, 1),
//		//	},
//		//	checkAfter: time.Duration(time.Millisecond * 750),
//		//	wantErr:    true,
//		//	want: models.Metric{
//		//		Name:  "Kek",
//		//		Type:  "gauge",
//		//		Value: float64(0),
//		//	},
//		//},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			go GetMetrics(tt.args.c, tt.args.s, tt.args.r)
//			time.Sleep(tt.checkAfter)
//			var data []models.Metric
//			select {
//			case data = <-tt.args.c:
//				//t.Logf("received %+v", data)
//			default:
//				//t.Log("empty")
//			}
//
//			if !tt.wantErr {
//				assert.Contains(t, data, tt.want)
//				//assert.Equal(t, tt.want, data)
//			} else {
//				assert.NotContains(t, data, tt.want)
//			}
//		})
//	}
//}
