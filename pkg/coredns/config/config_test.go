package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		config  Config
		want    string
		wantErr bool
	}{
		{
			config: Config{},
			want: `
.:53 {
	k8s_crds
}
`,
		},
		{
			config: Config{
				Forward: []string{"8.8.8.8", "8.8.4.4"},
				Metrics: true,
			},
			want: `
.:53 {
	k8s_crds
	forward . 8.8.8.8 8.8.4.4 
	prometheus
}
`,
		},
		{
			config: Config{
				Forward: []string{"8.8.8.8", "8.8.4.4"},
				Metrics: true,
				Errors:  true,
				Log:     true,
			},
			want: `
.:53 {
	k8s_crds
	forward . 8.8.8.8 8.8.4.4 
	log
	errors
	prometheus
}
`,
		},
	}
	for _, tt := range tests {
		got, err := tt.config.Render()
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, tt.want, got)
	}
}
