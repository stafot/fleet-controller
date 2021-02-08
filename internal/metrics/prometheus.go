// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pmodel "github.com/prometheus/common/model"
)

func queryInstallationMetrics(url, queryValue string, queryTime time.Time) (pmodel.Vector, error) {
	client, err := api.NewClient(api.Config{Address: url})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create prometheus client")
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, warnings, err := v1api.Query(ctx, queryValue, queryTime)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	if len(warnings) > 0 {
		return nil, errors.Errorf("encounted warnings obtaining metrics: %s", strings.Join(warnings, ", "))
	}

	return result.(pmodel.Vector), nil
}

func queryRangeInstallationMetrics(url, queryValue string, queryRange v1.Range) (pmodel.Matrix, error) {
	client, err := api.NewClient(api.Config{Address: url})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create prometheus client")
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	result, warnings, err := v1api.QueryRange(ctx, queryValue, queryRange)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	if len(warnings) > 0 {
		return nil, errors.Errorf("encounted warnings obtaining metrics: %s", strings.Join(warnings, ", "))
	}

	return result.(pmodel.Matrix), nil
}
