// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/pkg/errors"
)

type sizeScaleValues struct {
	scaleDownUserCount int64
	scaleUpUserCount   int64

	scaleDownSize string
	scaleUpSize   string
}

const (
	cloud10users   = "cloud10users" // This size should never scale down.
	cloud100users  = "cloud100users"
	size1000users  = "1000users"
	size5000users  = "5000users"
	size10000users = "10000users"
	size25000users = "25000users" // This size should never scale up.

	// Sizes we don't ever scale
	miniSingleton = "miniSingleton"
	miniHA        = "miniHA"
)

var scaleDictionary = map[string]sizeScaleValues{
	cloud10users: {
		scaleDownUserCount: 0,
		scaleUpUserCount:   11,
		scaleDownSize:      cloud10users,
		scaleUpSize:        cloud100users,
	},
	cloud100users: {
		scaleDownUserCount: 10,
		scaleUpUserCount:   110,
		scaleDownSize:      cloud10users,
		scaleUpSize:        size1000users,
	},
	size1000users: {
		scaleDownUserCount: 90,
		scaleUpUserCount:   1100,
		scaleDownSize:      cloud100users,
		scaleUpSize:        size5000users,
	},
	size5000users: {
		scaleDownUserCount: 900,
		scaleUpUserCount:   5100,
		scaleDownSize:      size1000users,
		scaleUpSize:        size10000users,
	},
	size10000users: {
		scaleDownUserCount: 4900,
		scaleUpUserCount:   10100,
		scaleDownSize:      size5000users,
		scaleUpSize:        size25000users,
	},
	size25000users: {
		scaleDownUserCount: 9900,
		scaleUpUserCount:   9999999999999,
		scaleDownSize:      size10000users,
		scaleUpSize:        size25000users,
	},
	miniSingleton: {
		scaleDownUserCount: 0,
		scaleUpUserCount:   9999999999999,
		scaleDownSize:      miniSingleton,
		scaleUpSize:        miniSingleton,
	},
	miniHA: {
		scaleDownUserCount: 0,
		scaleUpUserCount:   9999999999999,
		scaleDownSize:      miniHA,
		scaleUpSize:        miniHA,
	},
}

func getScaleValues(size string) (*sizeScaleValues, error) {
	values, ok := scaleDictionary[size]
	if !ok {
		return nil, errors.Errorf("no scale values found for size %s", size)
	}

	return &values, nil
}

func getSuggestedScaleSize(currentSize string, currentUserCount int64) (string, error) {
	newSize := currentSize

	for {
		var recheck bool

		scaleValues, err := getScaleValues(newSize)
		if err != nil {
			return "", err
		}

		if currentUserCount < scaleValues.scaleDownUserCount {
			newSize = scaleValues.scaleDownSize
			recheck = true
		}
		if currentUserCount > scaleValues.scaleUpUserCount {
			newSize = scaleValues.scaleUpSize
			recheck = true
		}

		if !recheck {
			break
		}
	}

	return newSize, nil
}
