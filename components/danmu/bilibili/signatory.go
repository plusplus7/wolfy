package bilibili

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const remoteSignTimeout = 10 * time.Second

var remoteSignHTTPClient = &http.Client{Timeout: remoteSignTimeout}

type ISignatory interface {
	Sign(reqJson string) (*CommonHeader, error)
}

type LocalSignatory struct {
	accessKeyId     string
	accessKeySecret string
}

func NewLocalSignatory(accessKeyId, accessKeySecret string) *LocalSignatory {
	return &LocalSignatory{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
	}
}

func (s *LocalSignatory) Sign(reqJson string) (*CommonHeader, error) {
	header := &CommonHeader{
		ContentType:       JsonType,
		ContentAcceptType: JsonType,
		Timestamp:         strconv.FormatInt(time.Now().Unix(), 10),
		SignatureMethod:   HmacSha256,
		SignatureVersion:  BiliVersion,
		Authorization:     "",
		Nonce:             strconv.FormatInt(time.Now().UnixNano(), 10),
		AccessKeyId:       s.accessKeyId,
		ContentMD5:        Md5(reqJson),
	}
	header.Authorization = HmacSHA256(s.accessKeySecret, header.ToSortedString())
	return header, nil
}

// Md5 md5加密
func Md5(str string) (md5str string) {
	data := []byte(str)
	has := md5.Sum(data)
	md5str = fmt.Sprintf("%x", has)
	return md5str
}

// HmacSHA256 HMAC-SHA256算法
func HmacSHA256(key string, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

type RemoteSignatory struct {
	remoteServerAddr string
	anchorCode       string
}

func NewRemoteSignatory(remoteServerAddr string, anchorCode string) *RemoteSignatory {
	return &RemoteSignatory{remoteServerAddr: remoteServerAddr, anchorCode: anchorCode}
}

func (s *RemoteSignatory) Sign(reqJson string) (*CommonHeader, error) {

	marshal, err := json.Marshal(RemoteSignRequest{
		ReqJson:    reqJson,
		AnchorCode: s.anchorCode,
	})
	if err != nil {
		return nil, fmt.Errorf("sign remote req json marshal err: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), remoteSignTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.remoteServerAddr+"/sign", bytes.NewBuffer(marshal))
	if err != nil {
		return nil, fmt.Errorf("sign remote req create err: %v", err)
	}
	req.Header.Set(ContentTypeHeader, JsonType)
	req.Header.Set(AcceptHeader, JsonType)

	do, err := remoteSignHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sign remote req do err: %v", err)
	}
	defer do.Body.Close()
	if do.StatusCode < http.StatusOK || do.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("sign remote returned %s", do.Status)
	}
	respBody, err := io.ReadAll(do.Body)
	if err != nil {
		return nil, fmt.Errorf("sign remote resp read err: %v", err)
	}

	var resp RemoteSignResponse
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return nil, fmt.Errorf("sign remote resp unmarshal err: %v", err)
	}
	return &resp.Header, nil
}
