// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package metrics

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
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

// GetInstallationNewPostCount returns the number of new posts an installation
// in the given number of days.
func (tc *ThanosClient) GetInstallationNewPostCount(installationID string, days int) (float64, error) {
	query := fmt.Sprintf("sum(increase(mattermost_post_total{installationId=\"%s\"}[%dd]))", installationID, days)
	rawMetrics, err := queryInstallationMetrics(tc.url, query, time.Now())
	if err != nil {
		return 0, errors.Wrap(err, "failed to query thanos")
	}

	if len(rawMetrics) != 1 {
		return 0, errors.Errorf("expected 1 metric result, but received %d", len(rawMetrics))
	}

	return float64(rawMetrics[0].Value), nil
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

// TODO: possibly deprecate this.
// Used with original assumption that the mattermost_post_total was actually the
// total number of posts in Mattermost.
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
