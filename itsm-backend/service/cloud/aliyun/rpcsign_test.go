package aliyun

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// 官方签名示例：https://help.aliyun.com/zh/sdk/product-overview/rpc-mechanism
func TestBuildRPCSignatureMatchesOfficialVector(t *testing.T) {
	params := map[string]string{
		"AccessKeyId":      "testid",
		"Action":           "DescribeRegions",
		"Format":           "XML",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   "3ee8c1b8-83d3-44af-a94f-4e0ad82fd6cf",
		"SignatureVersion": "1.0",
		"Timestamp":        "2016-02-23T12:46:24Z",
		"Version":          "2014-05-26",
	}

	require.Equal(t, "OLeaidS1JvxuMvnyHOwuJ+uX5qY=", BuildRPCSignature("GET", params, "testsecret"))
}

func TestBuildRPCSignatureIgnoresMapOrder(t *testing.T) {
	first := map[string]string{"Action": "DescribeVulList", "Type": "cve", "StatusList": "1"}
	second := map[string]string{"StatusList": "1", "Type": "cve", "Action": "DescribeVulList"}

	require.Equal(t, BuildRPCSignature("POST", first, "secret"), BuildRPCSignature("POST", second, "secret"))
}

func TestBuildRPCSignatureEncodesReservedCharacters(t *testing.T) {
	plain := BuildRPCSignature("POST", map[string]string{"Dimensions": "a b"}, "secret")
	encoded := BuildRPCSignature("POST", map[string]string{"Dimensions": "a+b"}, "secret")

	require.NotEqual(t, plain, encoded, "空格与加号必须编码成不同的值")
}

func TestParseRPCErrorKeepsProviderCode(t *testing.T) {
	err := parseRPCError(403, []byte(`{"Code":"Forbidden.RAM","Message":"User not authorized","RequestId":"req-1"}`))

	rpcErr, ok := err.(*RPCError)
	require.True(t, ok)
	require.Equal(t, 403, rpcErr.StatusCode)
	require.Equal(t, "Forbidden.RAM", rpcErr.Code)
	require.Equal(t, "req-1", rpcErr.RequestID)
}
