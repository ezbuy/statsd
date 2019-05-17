package statsd

import "testing"

func Test_shouldFire(t *testing.T) {
	type args struct {
		sampleRate float32
	}
	tests := []struct {
		name     string
		args     args
		errAllow float32
	}{
		{
			name: "sample-0.1",
			args: args{
				sampleRate: 0.1,
			},
			errAllow: 0.005,
		},
		{
			name: "sample-0.5",
			args: args{
				sampleRate: 0.5,
			},
			errAllow: 0.005,
		},
		{
			name: "sample-0.8",
			args: args{
				sampleRate: 0.8,
			},
			errAllow: 0.005,
		},
		{
			name: "sample-0.01",
			args: args{
				sampleRate: 0.01,
			},
			errAllow: 0.005,
		},
		{
			name: "sample-0.05",
			args: args{
				sampleRate: 0.05,
			},
			errAllow: 0.005,
		},
		{
			name: "sample-0.08",
			args: args{
				sampleRate: 0.08,
			},
			errAllow: 0.005,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var total, fireCnt float32
			for i := 0; i < 0xffffff; i++ {
				if shouldFire(tt.args.sampleRate) {
					fireCnt++
				}
				total++
			}

			if ret := fireCnt / total; ret-tt.args.sampleRate > tt.errAllow {
				t.Fatalf("[%s] ret: %.4f <=> want: %.4f",
					tt.name, ret, tt.args.sampleRate)
			}
		})
	}
}
