package internal

import "testing"

func TestIsCI(t *testing.T) {
	allEnvVars := append([]string{}, ciTruthyEnvVars...)
	allEnvVars = append(allEnvVars, ciPresenceEnvVars...)

	testCases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{
			name: "ci_true",
			env: map[string]string{
				"CI": "true",
			},
			want: true,
		},
		{
			name: "ci_false",
			env: map[string]string{
				"CI": "false",
			},
			want: false,
		},
		{
			name: "ci_numeric",
			env: map[string]string{
				"CI": "1",
			},
			want: true,
		},
		{
			name: "continuous_integration_truthy",
			env: map[string]string{
				"CI":                     "0",
				"CONTINUOUS_INTEGRATION": "yes",
			},
			want: true,
		},
		{
			name: "github_actions",
			env: map[string]string{
				"CI":             "",
				"GITHUB_ACTIONS": "true",
			},
			want: true,
		},
		{
			name: "jenkins_url",
			env: map[string]string{
				"CI":          "0",
				"JENKINS_URL": "http://jenkins.example",
			},
			want: true,
		},
		{
			name: "no_indicators",
			env:  map[string]string{},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range allEnvVars {
				t.Setenv(key, "")
			}
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			if got := IsCI(); got != tc.want {
				t.Fatalf("IsCI() = %v, want %v", got, tc.want)
			}
		})
	}
}
