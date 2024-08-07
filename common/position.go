// Copyright © 2023 Meroxa, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"encoding/json"

	sdk "github.com/conduitio/conduit-connector-sdk"
)

type Position struct {
	ReceiptHandle string `json:"receipt_handle"`
	QueueName     string `json:"queue_name"`
}

func ParsePosition(sdkPosition sdk.Position) (Position, error) {
	var position Position
	err := json.Unmarshal(sdkPosition, &position)
	return position, err
}

func (p Position) ToSdkPosition() sdk.Position {
	bs, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return sdk.Position(bs)
}
