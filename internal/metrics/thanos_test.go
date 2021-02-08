package metrics

import (
	"testing"

	pmodel "github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

func TestConfirmNoNewPosts(t *testing.T) {
	testCases := []struct {
		Description string
		Metrics     pmodel.Matrix
		Expected    bool
	}{
		{
			"empty",
			[]*pmodel.SampleStream{},
			false,
		},
		{
			"one value",
			[]*pmodel.SampleStream{{
				Values: []pmodel.SamplePair{{Value: 0}},
			}},
			false,
		},
		{
			"two values, different",
			[]*pmodel.SampleStream{{
				Values: []pmodel.SamplePair{
					{Value: 2},
					{Value: 10},
				},
			}},
			false,
		},
		{
			"two values, same",
			[]*pmodel.SampleStream{{
				Values: []pmodel.SamplePair{
					{Value: 2},
					{Value: 2},
				},
			}},
			true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			require.Equal(t, testCase.Expected, confirmNoNewPosts(testCase.Metrics))
		})
	}
}
