// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package metrics

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pmodel "github.com/prometheus/common/model"
)

// ThanosClient is a client for working with metrics from Thanos.
type ThanosClient struct {
	url string
}

// NewThanosClient returns a new Thanos client.
func NewThanosClient(url string) *ThanosClient {
	return &ThanosClient{url: url}
}

// GetInstallationUserMetrics returns a current snapshot of user metrics for
// all installations.
func (tc *ThanosClient) GetInstallationUserMetrics() (map[string]int64, error) {
	rawMetrics, err := queryInstallationMetrics(tc.url, "mattermost_db_active_users", time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "failed to query thanos")
	}

	return buildFinalInstallationUserCountMetrics(rawMetrics), nil
}

// DetermineIfInstallationHasNoNewPosts returns if an installation has any new
// posts since in the provided number of days.
func (tc *ThanosClient) DetermineIfInstallationHasNoNewPosts(installationID string, days int) (bool, error) {
	now := time.Now()
	startTime := now.AddDate(0, 0, -days)

	r := v1.Range{
		Start: startTime,
		End:   now,
		Step:  time.Duration(days) * 24 * time.Hour,
	}
	rawMetrics, err := queryRangeInstallationMetrics(tc.url, fmt.Sprintf("max(mattermost_post_total{installationId=\"%s\"})", installationID), r)
	if err != nil {
		return false, errors.Wrap(err, "failed to query thanos")
	}
	if len(rawMetrics) == 0 {
		return false, errors.New("no post metrics found for this installation")
	}

	return confirmNoNewPosts(rawMetrics), nil
}

func buildFinalInstallationUserCountMetrics(rawMetrics pmodel.Vector) map[string]int64 {
	installationMetrics := make(map[string]int64)

	for _, rawMetric := range rawMetrics {
		id, ok := rawMetric.Metric["installationId"]
		if !ok {
			continue
		}
		userCount := int64(rawMetric.Value)

		originalCount, ok := installationMetrics[string(id)]
		if !ok {
			installationMetrics[string(id)] = userCount
		} else {
			// Duplicate metric from other pods; use highest value seen.
			if userCount > originalCount {
				installationMetrics[string(id)] = userCount
			}
		}
	}

	return installationMetrics
}

func confirmNoNewPosts(rawMetrics pmodel.Matrix) bool {
	if len(rawMetrics) == 0 {
		return false
	}

	// Loop through each metric. Depending on the prometheus query, there could
	// be multiple results. For instance, we could have a max post count collected
	// from multiple pods.
	for _, rawMetric := range rawMetrics {
		// We expect two values for each metric. The first value will be the most
		// recent and the second will be the value from a previous point in time.
		if len(rawMetric.Values) != 2 {
			return false
		}
		if rawMetric.Values[0].Value != rawMetric.Values[1].Value {
			return false
		}
	}

	return true
}
