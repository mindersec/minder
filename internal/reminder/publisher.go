// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/mindersec/minder/pkg/eventer"
)

func (r *reminder) getMessagePublisher(ctx context.Context) (message.Publisher, error) {
	pub, err := eventer.New(ctx, nil, &r.cfg.EventConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	return pub, nil
}
