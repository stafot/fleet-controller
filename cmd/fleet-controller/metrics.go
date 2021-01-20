// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pmodel "github.com/prometheus/common/model"
)

func queryInstallationMetrics(url, queryValue string) (pmodel.Vector, error) {
	client, err := api.NewClient(api.Config{Address: url})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create prometheus client")
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, warnings, err := v1api.Query(ctx, queryValue, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	if len(warnings) > 0 {
		return nil, errors.Errorf("encounted warnings obtaining metrics: %s", strings.Join(warnings, ", "))
	}

	return result.(pmodel.Vector), nil
}
