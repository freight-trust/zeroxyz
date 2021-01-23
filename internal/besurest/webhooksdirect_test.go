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

package maidenlanedrest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/freight-trust/zeroxyz/internal/maidenlanedeth"
	"github.com/freight-trust/zeroxyz/internal/maidenlanedmessages"
	"github.com/freight-trust/zeroxyz/internal/maidenlanedtx"

	"github.com/stretchr/testify/assert"
)

type mockProcessor struct {
	capturedCtx *msgContext
}

func (p *mockProcessor) ResolveAddress(from string) (string, error) { return "", nil }
func (p *mockProcessor) OnMessage(ctx maidenlanedtx.TxnContext) {
	p.capturedCtx = ctx.(*msgContext)
}
func (p *mockProcessor) Init(maidenlanedeth.RPCClient) {}

func newTestWebhooksDirect(maxMsgs int) (*webhooksDirect, *memoryReceipts, *mockProcessor) {
	rsc := &ReceiptStoreConf{}
	r := newMemoryReceipts(rsc)
	rs := newReceiptStore(rsc, r, nil)
	conf := &WebhooksDirectConf{
		MaxInFlight: maxMsgs,
	}
	p := &mockProcessor{}
	wd := newWebhooksDirect(conf, p, rs)
	wd.processor = p
	return wd, r, p
}

func newTestWebhooksDirectServer(maxMsgs int) (*webhooksDirect, *httptest.Server, *memoryReceipts, *mockProcessor) {
	wd, r, p := newTestWebhooksDirect(maxMsgs)
	router := &httprouter.Router{}
	wh := newWebhooks(wd, nil)
	wh.addRoutes(router)
	ts := httptest.NewServer(router)
	return wd, ts, r, p
}

func newTestMsg() maidenlanedmessages.SendTransaction {
	return maidenlanedmessages.SendTransaction{
		TransactionCommon: maidenlanedmessages.TransactionCommon{
			RequestCommon: maidenlanedmessages.RequestCommon{
				Headers: maidenlanedmessages.RequestHeaders{
					CommonHeaders: maidenlanedmessages.CommonHeaders{
						MsgType: maidenlanedmessages.MsgTypeSendTransaction,
					},
				},
			},
			From:       "0xd912641Eb51a311A1C6BD32c1ED200C2a5abD7FE",
			Gas:        json.Number("12345"),
			Parameters: []interface{}{10},
		},
		To:         "0x112dd80dd5c598d16b557a6b70f0ca92adc09d41",
		MethodName: "set",
	}
}

func TestWebhooksDirectSubmitSendTransaction(t *testing.T) {
	assert := assert.New(t)

	_, ts, _, p := newTestWebhooksDirectServer(1)
	defer ts.Close()

	msg := newTestMsg()
	msgBytes, err := json.Marshal(&msg)
	assert.NoError(err)
	url := fmt.Sprintf("%s/hook", ts.URL)
	resp, err := http.Post(url, "application/json", bytes.NewReader(msgBytes))

	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	replyBytes, _ := ioutil.ReadAll(resp.Body)
	t.Logf("Received reply: %s", string(replyBytes))
	reply := maidenlanedmessages.AsyncSentMsg{}
	json.Unmarshal(replyBytes, &reply)
	assert.True(reply.Sent)
	assert.NotEmpty(reply.Request)
	assert.Equal(p.capturedCtx.msgID, reply.Request)

	headers := p.capturedCtx.Headers()
	assert.Equal(reply.Request, headers.ID)

	reconstructed := &maidenlanedmessages.SendTransaction{}
	err = p.capturedCtx.Unmarshal(&reconstructed)
	assert.NoError(err)
	assert.Equal("0xd912641Eb51a311A1C6BD32c1ED200C2a5abD7FE", reconstructed.From)
}

func TestWebhooksDirectMsgLimit(t *testing.T) {
	assert := assert.New(t)

	_, ts, r, p := newTestWebhooksDirectServer(1)
	defer ts.Close()

	msg := newTestMsg()
	msgBytes, _ := json.Marshal(&msg)
	url := fmt.Sprintf("%s/hook", ts.URL)

	resp, err := http.Post(url, "application/json", bytes.NewReader(msgBytes))
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)
	replyBytes, _ := ioutil.ReadAll(resp.Body)
	t.Logf("Received reply: %s", string(replyBytes))
	reply1 := maidenlanedmessages.AsyncSentMsg{}
	json.Unmarshal(replyBytes, &reply1)
	msgID1 := reply1.Request

	resp, err = http.Post(url, "application/json", bytes.NewReader(msgBytes))
	assert.NoError(err)
	assert.Equal(429, resp.StatusCode)
	replyBytes, _ = ioutil.ReadAll(resp.Body)
	t.Logf("Received reply: %s", string(replyBytes))
	reply2 := hookErrMsg{}
	json.Unmarshal(replyBytes, &reply2)
	assert.Equal(false, reply2.Sent)

	p.capturedCtx.SendErrorReply(500, fmt.Errorf("pop"))
	receipt1, _ := r.GetReceipt(msgID1)
	assert.NotNil(receipt1)
	t.Logf("Receipt: %+v", receipt1)
	assert.NotNil((*receipt1)["requestPayload"])

	resp, err = http.Post(url, "application/json", bytes.NewReader(msgBytes))
	assert.NoError(err)
	assert.Equal(200, resp.StatusCode)

}

func TestWebhooksDirectSendWebhooksMsgBadHeaders(t *testing.T) {
	assert := assert.New(t)
	wd, _, _ := newTestWebhooksDirect(1)
	msgMap := make(map[string]interface{})
	msgMap["headers"] = false
	_, statusCode, err := wd.sendWebhookMsg(context.Background(), "", "", msgMap, false)
	assert.Equal(400, statusCode)
	assert.EqualError(err, "Failed to process headers in message")
}

func TestWebhooksDirectUnmarshalBadMsg(t *testing.T) {
	assert := assert.New(t)
	msg := make(map[string]interface{})
	ctx := &msgContext{msg: msg}
	msg["bad"] = map[bool]string{}
	err := ctx.Unmarshal(nil)
	assert.EqualError(err, "json: unsupported type: map[bool]string")
}
