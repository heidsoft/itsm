//
// OSS 是服务请求交付链路的第二资源类型候选（能力矩阵把「第二资源类型」列为
// 晋升条件之一）。本命令只做只读调用，不创建/修改/删除任何 Bucket 与对象，
// 不产生费用：
//
//   - ListBuckets：服务级列举，验证 OSS 签名（Authorization: OSS AK:sig）、
//     凭据有效性与 oss:ListBuckets 权限；
//   - ListObjects：**按 Bucket 自己的 location 路由 endpoint** 列举对象，
//     验证桶级读权限与数据面签名；
//   - 负向用例：不存在的 Bucket / 错地域 / 错凭据，观察错误码形态，
//     供后续错误分类设计参考。
//
// 一个必须记住的契约（2026-11-03 真实账号实测）：**对象级操作必须按 Bucket
// 所在地域访问**。用账号级地域访问别的地域的 Bucket，会得到
// 403 AccessDenied + "must be addressed using the specified endpoint"，
// 而且响应体里带着正确的 endpoint——错误码本身不提示要换地域，只能自己解析。
// 服务级的 ListBuckets 则与地域无关（任意地域 endpoint 都能列出全部 Bucket）。
//
// 凭据只经 env://ITSM_ALIYUN 从进程环境读取，AK/SK 不进命令行、不进输出。
//
// 用法：
//
//	export ITSM_ALIYUN_ACCESS_KEY_ID=... ITSM_ALIYUN_ACCESS_KEY_SECRET=...
//	go run ./cmd/aliyun-oss-api-verify -region cn-hangzhou -json-out /tmp/oss-verify.json
//
// 退出码：0 = 只读面完成验证（云侧业务拒绝属预期观测结果）；1 = 凭据/客户端构造失败。

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"itsm-backend/service/cloud"
)

// result 是一条 OSS API 验证结果，同时作为 -json-out 的结构化证据。
type result struct {
	Group      string `json:"group"`
	API        string `json:"api"`
	OK         bool   `json:"ok"`
	HTTPStatus int    `json:"http_status"`
	RequestID  string `json:"request_id,omitempty"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Bucket     string `json:"bucket,omitempty"`
	Region     string `json:"region,omitempty"`
	Items      int    `json:"items,omitempty"`
	ElapsedMS  int64  `json:"elapsed_ms"`
	Verdict    string `json:"verdict"`
	Note       string `json:"note,omitempty"`
}

func main() {
	region := flag.String("region", "cn-hangzhou", "列举 Bucket 用的服务端点地域（服务级列举与地域无关）")
	jsonOut := flag.String("json-out", "", "结构化结果输出路径（留空则只打印）")
	flag.Parse()

	fmt.Println("阿里云 OSS SDK 只读验证：ListBuckets + 按 Bucket 地域的 ListObjects，不写任何资源")
	fmt.Printf("列举端点地域: %s\n\n", *region)

	cred, err := cloud.ResolveAliyunCredential(context.Background(), "env://ITSM_ALIYUN")
	if err != nil {
		fmt.Fprintf(os.Stderr, "凭据解析失败（env://ITSM_ALIYUN）: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[OK  ] 凭据解析（env://ITSM_ALIYUN），AK/SK 不回显")

	endpoint := fmt.Sprintf("https://oss-%s.aliyuncs.com", *region)
	client, err := oss.New(endpoint, cred.AccessKeyID, cred.AccessKeySecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "构造 OSS 客户端失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[OK  ] OSS 客户端构造成功（endpoint=%s）\n", endpoint)

	var results []result

	// ① ListBuckets：服务级，验证 OSS 签名与权限。
	start := time.Now()
	listRes, err := client.ListBuckets(oss.MaxKeys(100))
	r := result{
		Group: "read-only", API: "ListBuckets", Region: *region,
		ElapsedMS: time.Since(start).Milliseconds(),
		Note:      "服务级列举，与地域无关；成功即证明 OSS 签名与 oss:ListBuckets 权限成立",
	}
	if err != nil {
		applyServiceError(&r, err)
	} else {
		r.OK = true
		r.Verdict = "ok"
		r.HTTPStatus = 200
		r.Items = len(listRes.Buckets)
	}
	results = append(results, r)
	printResult(r)

	buckets := listRes.Buckets
	if err != nil {
		buckets = nil
	}

	// ② ListObjects：**按 Bucket 自己的 location 路由**。
	for _, b := range buckets {
		loc := strings.TrimPrefix(b.Location, "oss-")
		bucketClient, cerr := oss.New(fmt.Sprintf("https://oss-%s.aliyuncs.com", loc),
			cred.AccessKeyID, cred.AccessKeySecret)
		if cerr != nil {
			results = append(results, result{
				Group: "read-only", API: "ListObjects", Bucket: b.Name, Region: loc,
				OK: false, Verdict: "client-init-failed", Message: cerr.Error(),
				Note: "按 Bucket 地域构造客户端失败",
			})
			continue
		}
		bucket, berr := bucketClient.Bucket(b.Name)
		if berr != nil {
			results = append(results, result{
				Group: "read-only", API: "ListObjects", Bucket: b.Name, Region: loc,
				OK: false, Verdict: "client-init-failed", Message: berr.Error(),
			})
			continue
		}
		s := time.Now()
		objs, oerr := bucket.ListObjects(oss.MaxKeys(1))
		rr := result{
			Group: "read-only", API: "ListObjects", Bucket: b.Name, Region: loc,
			ElapsedMS: time.Since(s).Milliseconds(),
			Note:      "按 Bucket 所在地域路由的桶级读权限与数据面签名",
		}
		if oerr != nil {
			applyServiceError(&rr, oerr)
		} else {
			rr.OK = true
			rr.Verdict = "ok"
			rr.HTTPStatus = 200
			rr.Items = len(objs.Objects)
		}
		results = append(results, rr)
		printResult(rr)
	}

	// ③ 负向用例：错误形态是后续错误分类的依据。
	negative(client, cred, *region, &results)

	// 汇总。
	fmt.Println()
	readOK, readTotal := 0, 0
	for _, item := range results {
		if item.Group == "read-only" {
			readTotal++
			if item.OK {
				readOK++
			}
		}
	}
	fmt.Printf("汇总：只读面 %d/%d 通过；负向用例 %d 项按预期观测。\n", readOK, readTotal, len(results)-readTotal)

	if *jsonOut != "" {
		payload, err := json.MarshalIndent(struct {
			Results []result `json:"results"`
		}{Results: results}, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化结果失败: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*jsonOut, append(payload, '\n'), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "写入结果失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("结构化证据已写入 %s\n", *jsonOut)
	}
}

// negative 探测错误形态。这里刻意用「不存在的 Bucket」而不是「不存在的对象」：
// 不存在的对象在存在的 Bucket 上会返回 NoSuchKey，混在真实 Bucket 的错误里
// 不利于分辨。
func negative(client *oss.Client, cred *cloud.ResolvedCredential, region string, results *[]result) {
	fmt.Println()
	fmt.Println("--- 负向用例（错误形态观测）---")

	probe := func(name, api string, fn func() error, note string) {
		start := time.Now()
		err := fn()
		r := result{
			Group: "negative", API: api, Region: region,
			ElapsedMS: time.Since(start).Milliseconds(), Note: note,
		}
		if err == nil {
			r.OK = true
			r.Verdict = "unexpected-success"
			r.Message = "非法参数竟返回成功，需复核云侧校验"
		} else {
			applyServiceError(&r, err)
		}
		*results = append(*results, r)
		printResult(r)
	}

	probe("missing-bucket", "ListObjects[不存在的 Bucket]", func() error {
		bucket, err := client.Bucket("itsm-verify-not-exist-bucket")
		if err != nil {
			return err
		}
		_, err = bucket.ListObjects(oss.MaxKeys(1))
		return err
	}, "期望 404 NoSuchBucket")

	probe("wrong-region", "ListBuckets[错地域]", func() error {
		other := "cn-beijing"
		if region == other {
			other = "cn-shanghai"
		}
		otherClient, err := oss.New(fmt.Sprintf("https://oss-%s.aliyuncs.com", other),
			cred.AccessKeyID, cred.AccessKeySecret)
		if err != nil {
			return err
		}
		_, err = otherClient.ListBuckets(oss.MaxKeys(1))
		return err
	}, "服务级列举与地域无关，预期成功")

	probe("bad-credential", "ListBuckets[错凭据]", func() error {
		badClient, err := oss.New(fmt.Sprintf("https://oss-%s.aliyuncs.com", region),
			"LTAI-invalid-key-for-itsm-probe", "invalid-secret-for-itsm-probe")
		if err != nil {
			return err
		}
		_, err = badClient.ListBuckets(oss.MaxKeys(1))
		return err
	}, "期望 403 InvalidAccessKeyId")
}

// applyServiceError 把 OSS 的结构化错误填进结果。OSS 错误是 XML
// <Error><Code>…</Code></Error>，错误码空间与 ECS/OpenAPI 完全不同
// （NoSuchBucket / AccessDenied / InvalidAccessKeyId / BucketAlreadyExists …），
// 分类必须按 OSS 自己的码表来。
func applyServiceError(r *result, err error) {
	r.OK = false
	var se oss.ServiceError
	if asServiceError(err, &se) {
		r.Code = se.Code
		r.HTTPStatus = se.StatusCode
		r.RequestID = se.RequestID
		r.Message = se.Message
		r.Verdict = classifyOSSCode(se.Code, se.StatusCode)
		return
	}
	r.Message = truncate(err.Error(), 240)
	r.Verdict = "transport-error"
}

// classifyOSSCode 把 OSS 错误码归到可行动结论。
func classifyOSSCode(code string, status int) string {
	switch code {
	case "NoSuchBucket", "NoSuchKey", "NoSuchUpload", "NoSuchWebsite", "NoSuchLifecycle", "NoSuchCORS", "NoSuchLiveChannel":
		return "not-found"
	case "InvalidAccessKeyId", "SignatureDoesNotMatch", "InvalidSecurityToken":
		return "bad-credential"
	case "AccessDenied", "SignatureDoesNotMatch2":
		return "access-denied"
	case "BucketAlreadyExists":
		return "already-exists"
	case "InvalidBucketName", "InvalidObjectName", "InvalidArgument", "MalformedXML":
		return "invalid-parameter"
	case "TooManyRequests", "ServiceUnavailable", "RequestTimeout", "InternalError":
		return "throttled"
	default:
		if status == 0 {
			return "transport-error"
		}
		return "cloud-error"
	}
}

// asServiceError 从错误链里取出 oss.ServiceError。
func asServiceError(err error, target *oss.ServiceError) bool {
	for err != nil {
		if se, ok := err.(oss.ServiceError); ok {
			*target = se
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func printResult(r result) {
	status := "OK  "
	if !r.OK {
		status = "FAIL"
	}
	extra := ""
	if r.Code != "" {
		extra = fmt.Sprintf(" code=%s", r.Code)
	}
	fmt.Printf("[%s] %-34s bucket=%-38s region=%-12s http=%-4d items=%-3d %5dms  %s%s\n",
		status, r.API, truncate(r.Bucket, 38), r.Region, r.HTTPStatus, r.Items,
		r.ElapsedMS, r.Verdict, extra)
	if r.Message != "" && !r.OK {
		fmt.Printf("       └─ %s\n", truncate(r.Message, 200))
	}
}

func truncate(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
