package main

import (
	"os"
	"regexp"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

// writeMethodsForbidden 是 OSS SDK 里**会改变云上资源**的方法。
//
// OSS SDK 的写方法很多（Bucket 生命周期、对象上传、策略、加密、CORS……），
// 一旦被误用就会真的建桶/删桶/改数据，而且不产生费用提示——比 ECS 那种
// 「建实例会报错」的风险更高。因此这里用源码扫描把它们全列出来挡住。
var writeMethodsForbidden = []string{
	// Bucket 生命周期
	"CreateBucket", "DeleteBucket", "DeleteBucketLogging", "DeleteBucketCORS",
	"DeleteBucketLifecycle", "DeleteBucketReferer", "DeleteBucketWebsite",
	"DeleteBucketEncryption", "DeleteBucketPolicy", "DeleteBucketTagging",
	"DeleteBucketVersioning", "DeleteBucketQoSInfo", "DeleteBucketReplication",
	"DeleteBucketInventory", "DeleteBucketCname", "DeleteBucketTransferAcceleration",
	"SetBucketACL", "SetBucketLifecycle", "SetBucketCORS", "SetBucketLogging",
	"SetBucketReferer", "SetBucketWebsite", "SetBucketEncryption", "SetBucketPolicy",
	"SetBucketTagging", "SetBucketVersioning", "SetBucketQoSInfo", "SetBucketReplication",
	"SetBucketInventory", "SetBucketCname", "SetBucketTransferAcceleration",
	"SetBucketStyle", "SetBucketProcess", "PutBucket",
	// Object 生命周期
	"PutObject", "AppendObject", "DeleteObject", "DeleteObjects", "CopyObject",
	"SetObjectACL", "SetObjectMeta", "SetObjectTagging", "DeleteObjectTagging",
	"CopyObjectTo", "CopyObjectFrom", "RestoreObject", "SignURL", "PutSymlink",
	// 分片上传
	"InitiateMultipartUpload", "UploadPart", "CompleteMultipartUpload",
	"AbortMultipartUpload", "UploadPartCopy",
	// 其它写操作
	"CreateLiveChannel", "DeleteLiveChannel", "PutLiveChannelStatus",
	"CreateBucketCname", "AddBucketCname", "UploadFile", "DoPostMeta",
	"DoPutObject", "DoDeleteObject", "DoGetBucketACL", "PutImage",
}

// sdkCallPattern 匹配对 OSS SDK 客户端/桶/包的调用。
//
// 接收者限定为代码里实际持有的那批变量与 oss 包——这样既不漏掉 SDK 调用，
// 又不会把 strings./fmt./os. 的标准库调用也卷进来。
var sdkCallPattern = regexp.MustCompile(`\b(oss|client|bucket|bucketClient|otherClient|badClient|missing)\.([A-Za-z]+)\(`)

// TestVerifyCommandHasNoWriteCall 是本命令最关键的安全守卫：源码里不得出现
// 任何 OSS 写方法调用。
//
// 用源码扫描而不是「人工核对调用清单」：OSS SDK 的写方法有几十个，靠 review
// 记住清单不可靠，而漏一个就可能真的删桶或改数据。
func TestVerifyCommandHasNoWriteCall(t *testing.T) {
	src, err := os.ReadFile("main.go")
	require.NoError(t, err)
	text := string(src)

	for _, method := range writeMethodsForbidden {
		require.NotContains(t, text, "."+method+"(",
			"命令不得调用 OSS 写方法 %q（会真的建桶/删桶/改数据）", method)
	}

	// 逐处扫描调用点：任何 SDK 方法调用都必须在只读白名单里。
	// 只读白名单：SDK 侧的调用只能是这几个。
	allowed := map[string]bool{
		"ListBuckets": true, "ListObjects": true,
		"Bucket": true, "New": true, "MaxKeys": true,
		"ServiceError": true,
	}
	matches := sdkCallPattern.FindAllStringSubmatch(text, -1)
	require.NotEmpty(t, matches, "扫描器零命中：调用写法已漂移，守卫失效")
	for _, m := range matches {
		receiver, method := m[1], m[2]
		if receiver == "oss" {
			// oss 包级引用：New/MaxKeys/ServiceError 都是构造或类型引用。
			require.True(t, allowed[method],
				"oss.%s 不在只读白名单内（oss 包级只能有构造与类型引用）", method)
			continue
		}
		require.True(t, allowed[method],
			"出现未在只读白名单内的 SDK 调用 %s.%s", receiver, method)
	}
}

// TestVerifyCommandOnlyUsesReadOnlyAPIs 锁住只读 API 白名单本身：只能用
// List/Get 系列。
func TestVerifyCommandOnlyUsesReadOnlyAPIs(t *testing.T) {
	src, err := os.ReadFile("main.go")
	require.NoError(t, err)
	text := string(src)

	for _, allowed := range []string{"ListBuckets(", "ListObjects("} {
		require.Contains(t, text, allowed, "只读验证应覆盖 %s", allowed)
	}
	// 不得出现写语义的动词开头的调用（与上一个测试互补：这里按动词维度兜底）。
	for _, verb := range []string{"Create", "Delete", "Put", "Set", "Upload", "Copy", "Abort"} {
		require.NotContains(t, text, "."+verb, "不应出现以 "+verb+" 开头的 OSS 调用")
	}
}

// TestClassifyOSSCodeRecognizesActionableBuckets 锁住错误分类：
// 服务未开通/凭据错/资源不存在/已存在/参数非法/限流要分得开。
//
// 误判方向两边都危险：把限流判成永久错误会让一次抖动变成「交付永久失败」；
// 把已存在判成不存在会让幂等判断失效。
func TestClassifyOSSCodeRecognizesActionableBuckets(t *testing.T) {
	cases := []struct {
		code    string
		status  int
		verdict string
	}{
		{"NoSuchBucket", 404, "not-found"},
		{"NoSuchKey", 404, "not-found"},
		{"InvalidAccessKeyId", 403, "bad-credential"},
		{"SignatureDoesNotMatch", 403, "bad-credential"},
		{"AccessDenied", 403, "access-denied"},
		{"BucketAlreadyExists", 409, "already-exists"},
		{"InvalidBucketName", 400, "invalid-parameter"},
		{"TooManyRequests", 503, "throttled"},
		{"ServiceUnavailable", 503, "throttled"},
		{"SomethingBrandNew", 400, "cloud-error"},
	}
	for _, c := range cases {
		require.Equal(t, c.verdict, classifyOSSCode(c.code, c.status),
			"错误码 %q 的分类", c.code)
	}
}

// TestClassifyOSSCodeDoesNotMarkUnknownAsPermanent 未识别的码不能被当成
// 「永久失败」——按可重试/待人工判定更安全。
func TestClassifyOSSCodeDoesNotMarkUnknownAsPermanent(t *testing.T) {
	verdict := classifyOSSCode("SomethingBrandNew", 400)
	require.NotEqual(t, "permanent", verdict)
	require.Equal(t, "cloud-error", verdict)
}

// TestAsServiceErrorExtractsFromChain 错误要从错误链里取出来（SDK 可能包装过）。
func TestAsServiceErrorExtractsFromChain(t *testing.T) {
	se := oss.ServiceError{
		Code:       "NoSuchBucket",
		StatusCode: 404,
		Message:    "The specified bucket does not exist.",
		RequestID:  "REQ-OSS-1",
	}
	var target oss.ServiceError
	require.True(t, asServiceError(se, &target))
	require.Equal(t, "NoSuchBucket", target.Code)
	require.Equal(t, 404, target.StatusCode)
	require.Equal(t, "REQ-OSS-1", target.RequestID)
}
