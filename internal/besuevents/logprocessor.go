// Copyright 2019 SEE CONTRIBUTORS

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package maidenlanedevents

import (
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/freight-trust/zeroxyz/internal/maidenlanederrors"
	"github.com/freight-trust/zeroxyz/internal/maidenlanedeth"
	log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/freight-trust/zeroxyz/internal/maidenlanedbind"
)

type logEntry struct {
	Address          maidenlanedbind.Address   `json:"address"`
	BlockNumber      maidenlanedbind.HexBigInt `json:"blockNumber"`
	TransactionIndex maidenlanedbind.HexUint   `json:"transactionIndex"`
	TransactionHash  maidenlanedbind.Hash      `json:"transactionHash"`
	Data             string            `json:"data"`
	Topics           []*maidenlanedbind.Hash   `json:"topics"`
	Timestamp        uint64            `json:"timestamp,omitempty"`
}

type eventData struct {
	Address          string                 `json:"address"`
	BlockNumber      string                 `json:"blockNumber"`
	TransactionIndex string                 `json:"transactionIndex"`
	TransactionHash  string                 `json:"transactionHash"`
	Data             map[string]interface{} `json:"data"`
	SubID            string                 `json:"subId"`
	Signature        string                 `json:"signature"`
	LogIndex         string                 `json:"logIndex"`
	Timestamp        string                 `json:"timestamp,omitempty"`
	// Used for callback handling
	batchComplete func(*eventData)
}

type logProcessor struct {
	subID    string
	event    *maidenlanedbind.ABIEvent
	stream   *eventStream
	blockHWM big.Int
	hwnSync  sync.Mutex
}

func newLogProcessor(subID string, event *maidenlanedbind.ABIEvent, stream *eventStream) *logProcessor {
	return &logProcessor{
		subID:  subID,
		event:  event,
		stream: stream,
	}
}

func (lp *logProcessor) batchComplete(newestEvent *eventData) {
	lp.hwnSync.Lock()
	i := new(big.Int)
	i.SetString(newestEvent.BlockNumber, 10)
	i.Add(i, big.NewInt(1)) // restart from the next block
	if i.Cmp(&lp.blockHWM) > 0 {
		lp.blockHWM.Set(i)
	}
	lp.hwnSync.Unlock()
	log.Debugf("%s: HWM: %s", lp.subID, lp.blockHWM.String())
}

func (lp *logProcessor) getBlockHWM() big.Int {
	lp.hwnSync.Lock()
	v := lp.blockHWM
	lp.hwnSync.Unlock()
	return v
}

func (lp *logProcessor) initBlockHWM(intVal *big.Int) {
	lp.hwnSync.Lock()
	lp.blockHWM = *intVal
	lp.hwnSync.Unlock()
}

func (lp *logProcessor) processLogEntry(subInfo string, entry *logEntry, idx int) (err error) {

	var data []byte
	if strings.HasPrefix(entry.Data, "0x") {
		data, err = maidenlanedbind.HexDecode(entry.Data)
		if err != nil {
			return maidenlanederrors.Errorf(maidenlanederrors.EventStreamsLogDecode, subInfo, err)
		}
	}

	result := &eventData{
		Address:          entry.Address.String(),
		BlockNumber:      entry.BlockNumber.ToInt().String(),
		TransactionIndex: entry.TransactionIndex.String(),
		TransactionHash:  entry.TransactionHash.String(),
		Signature:        maidenlanedbind.ABIEventSignature(lp.event),
		Data:             make(map[string]interface{}),
		SubID:            lp.subID,
		LogIndex:         strconv.Itoa(idx),
		batchComplete:    lp.batchComplete,
	}
	if lp.stream.spec.Timestamps {
		result.Timestamp = strconv.FormatUint(entry.Timestamp, 10)
	}
	topicIdx := 0
	if !lp.event.Anonymous {
		topicIdx++ // first index is the hash of the event description
	}

	// We need split out the indexed args that we parse out of the topic, from the data args
	var dataArgs maidenlanedbind.ABIArguments
	dataArgs = make([]maidenlanedbind.ABIArgument, 0, len(lp.event.Inputs))
	for idx, input := range lp.event.Inputs {
		var val interface{}
		if input.Indexed {
			if topicIdx >= len(entry.Topics) {
				return maidenlanederrors.Errorf(maidenlanederrors.EventStreamsLogDecodeInsufficientTopics, subInfo, idx, maidenlanedbind.ABIEventSignature(lp.event))
			}
			topic := entry.Topics[topicIdx]
			topicIdx++
			if topic != nil {
				val = topicToValue(topic, &input)
			} else {
				val = nil
			}
			result.Data[input.Name] = val
		} else {
			dataArgs = append(dataArgs, input)
		}
	}

	// Retrieve the data args from the RLP and merge the results
	if len(dataArgs) > 0 {
		dataMap := maidenlanedeth.ProcessRLPBytes(dataArgs, data)
		for k, v := range dataMap {
			result.Data[k] = v
		}
	}

	// Ok, now we have the full event in a friendly map output. Pass it down to the event processor
	log.Infof("%s: Dispatching event. Address=%s BlockNumber=%s TxIndex=%s", subInfo, result.Address, result.BlockNumber, result.TransactionIndex)
	lp.stream.handleEvent(result)
	return nil
}

func topicToValue(topic *maidenlanedbind.Hash, input *maidenlanedbind.ABIArgument) interface{} {
	switch input.Type.T {
	case maidenlanedbind.IntTy, maidenlanedbind.UintTy, maidenlanedbind.BoolTy:
		h := maidenlanedbind.HexBigInt{}
		h.UnmarshalText([]byte(topic.Hex()))
		bI, _ := math.ParseBig256(topic.Hex())
		if input.Type.T == maidenlanedbind.IntTy {
			// It will be a two's complement number, so needs to be interpretted
			bI = math.S256(bI)
			return bI.String()
		} else if input.Type.T == maidenlanedbind.BoolTy {
			return (bI.Uint64() != 0)
		}
		return bI.String()
	case maidenlanedbind.AddressTy:
		topicBytes := topic.Bytes()
		addrBytes := topicBytes[len(topicBytes)-20:]
		return maidenlanedbind.BytesToAddress(addrBytes)
	default:
		// For all other types it is just a hash of the output for indexing, so we can only
		// logically return it as a hex string. The Solidity developer has to include
		// the same data a second type non-indexed to get the real value.
		return topic.String()
	}
}
