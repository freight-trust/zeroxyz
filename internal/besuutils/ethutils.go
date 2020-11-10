// Copyright 2018, 2019 SEE CONTRIBUTORS

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package turbokeeperdutils

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/freight-trust/zeroxyz/internal/turbokeeperderrors"
)

// StrToAddress is a helper to parse eth addresses with useful errors
func StrToAddress(desc string, strAddr string) (addr common.Address, err error) {
	if strAddr == "" {
		err = turbokeeperderrors.Errorf(turbokeeperderrors.HelperStrToAddressRequiredField, desc)
		return
	}
	if !strings.HasPrefix(strAddr, "0x") {
		strAddr = "0x" + strAddr
	}
	if !common.IsHexAddress(strAddr) {
		err = turbokeeperderrors.Errorf(turbokeeperderrors.HelperStrToAddressBadAddress, desc)
		return
	}
	addr = common.HexToAddress(strAddr)
	return
}
