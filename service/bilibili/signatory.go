package bilibili

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

type ISignatory interface {
	Sign(reqJson string) *CommonHeader
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

func (s *LocalSignatory) Sign(reqJson string) *CommonHeader {
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
	return header
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
