package main

// 阿里云连接与「服务请求交付就绪度」只读探测。
//
// 用途：真实账号验收的第一步——不创建任何资源，只确认凭据有效、
// 地域可达、交付 RunInstances 需要的四类参数（镜像/交换机/安全组/实例规格）
// 在目标地域真实存在。探测结果直接填进服务请求表单，供完整交付验收使用。
//
// 存在性一律判返回条数，不判 err：阿里云 Describe* 对不存在的 ID 返回
// HTTP 200 + 空列表（2026-11-03 真实账号实测），只看错误会把「不存在」
// 当成「查到」。
//
// 需要连写 API 的参数/权限/配额校验链一起看时，用 cmd/aliyun-sdk-api-verify：
// 它多加 RunInstances(DryRun=true)，同样零费用，能直接区分「参数链问题」
// 与「账号余额问题」。
//
// 用法（凭据经 env://ITSM_ALIYUN 从环境变量读取，值不落命令行）：
//
//	export ITSM_ALIYUN_ACCESS_KEY_ID=... ITSM_ALIYUN_ACCESS_KEY_SECRET=...
//	go run ./cmd/aliyun-connection-check -region cn-beijing
//
// 退出码：0 = 全部就绪；1 = 任一项失败（凭据/连通性/交付参数缺失）。

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	ecs "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/service/cloud"
	"itsm-backend/service/cloud/aliyun"
)

var failures int

func report(ok bool, label, detail string) {
	status := "OK  "
	if !ok {
		status = "FAIL"
		failures++
	}
	fmt.Printf("[%s] %s: %s\n", status, label, detail)
}

func main() {
	region := flag.String("region", "cn-beijing", "交付目标地域")
	imageID := flag.String("image", "aliyun_3_x64_20G_alibase_20240819.vhd", "待验证的公共镜像 ID")
	instanceType := flag.String("instance-type", "ecs.e-c1m1.large", "待验证的实例规格")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to initialize logger")
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	adapter := aliyun.NewAliyunECSAdapter(logger.Sugar())
	account := &ent.CloudAccount{Provider: "aliyun", CredentialRef: "env://ITSM_ALIYUN"}
	if err := adapter.ValidateCredential(ctx, account); err != nil {
		fmt.Fprintf(os.Stderr, "Aliyun credential validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[OK  ] 凭据有效（DescribeRegions 通）")

	cred, err := cloud.ResolveAliyunCredential(ctx, account.CredentialRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve credential failed: %v\n", err)
		os.Exit(1)
	}
	client, err := aliyun.NewECSClientForProbe(*region, cred)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build ecs client failed: %v\n", err)
		os.Exit(1)
	}

	// 1) 目标地域在账号可用地域列表内。
	regions, err := adapter.ListRegions(ctx, account)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list regions failed: %v\n", err)
		os.Exit(1)
	}
	regionOK := false
	for _, r := range regions {
		if r == *region {
			regionOK = true
			break
		}
	}
	report(regionOK, "目标地域", fmt.Sprintf("%s（账号可见地域 %d 个）", *region, len(regions)))
	if !regionOK {
		os.Exit(1)
	}

	// 2) 镜像存在（系统公共镜像）。
	imgResp, imgErr := client.DescribeImagesWithOptions(&ecs.DescribeImagesRequest{
		RegionId:        tea.String(*region),
		ImageId:         tea.String(*imageID),
		ImageOwnerAlias: tea.String("system"),
	}, aliyun.ProbeRuntimeOptions())
	if imgErr != nil {
		report(false, "镜像", *imageID+" 查询失败: "+imgErr.Error())
	} else if imgResp.Body == nil || imgResp.Body.Images == nil || len(imgResp.Body.Images.Image) == 0 {
		// 镜像 ID 有生命周期（系统镜像定期下线），不存在时列出最新候选，
		// 让验收不用再去查文档。只推荐 Alibaba Linux 等长期维护的家族。
		sugResp, sugErr := client.DescribeImagesWithOptions(&ecs.DescribeImagesRequest{
			RegionId:        tea.String(*region),
			OSType:          tea.String("linux"),
			ImageOwnerAlias: tea.String("system"),
			PageSize:        tea.Int32(100),
		}, aliyun.ProbeRuntimeOptions())
		hint := "查询候选镜像失败: " + func() string {
			if sugErr != nil {
				return sugErr.Error()
			}
			return "无返回"
		}()
		if sugErr == nil && sugResp.Body != nil && sugResp.Body.Images != nil {
			ids := make([]string, 0, 5)
			for _, img := range sugResp.Body.Images.Image {
				if img == nil || img.ImageId == nil {
					continue
				}
				id := tea.StringValue(img.ImageId)
				// 基础系统镜像（alibase）体积小、启动快，优先推荐。
				if strings.HasPrefix(id, "aliyun_3") && strings.Contains(id, "alibase") {
					ids = append(ids, id)
					if len(ids) >= 5 {
						break
					}
				}
			}
			if len(ids) > 0 {
				hint = "候选: " + strings.Join(ids, ", ")
			}
		}
		report(false, "镜像", *imageID+" 在 "+*region+" 不存在；"+hint)
	} else {
		name := ""
		if imgResp.Body.Images.Image[0] != nil && imgResp.Body.Images.Image[0].OSName != nil {
			name = tea.StringValue(imgResp.Body.Images.Image[0].OSName)
		}
		report(true, "镜像", fmt.Sprintf("%s（%s）", *imageID, name))
	}

	// 3) 交换机：交付表单必填 vSwitchId。
	vswResp, vswErr := client.DescribeVSwitchesWithOptions(&ecs.DescribeVSwitchesRequest{
		RegionId: tea.String(*region),
	}, aliyun.ProbeRuntimeOptions())
	if vswErr != nil {
		report(false, "交换机", "查询失败: "+vswErr.Error())
	} else if vswResp.Body == nil || vswResp.Body.VSwitches == nil || len(vswResp.Body.VSwitches.VSwitch) == 0 {
		report(false, "交换机", *region+" 账号下无可用 vSwitch（需先建 VPC/交换机，或交付表单填默认 VPC 的）")
	} else {
		ids := make([]string, 0, 3)
		for i, vsw := range vswResp.Body.VSwitches.VSwitch {
			if i >= 3 || vsw == nil {
				continue
			}
			ids = append(ids, tea.StringValue(vsw.VSwitchId))
		}
		report(true, "交换机", fmt.Sprintf("可用 %d 个，示例: %s", len(vswResp.Body.VSwitches.VSwitch), strings.Join(ids, ", ")))
	}

	// 4) 安全组：交付表单必填 securityGroupId。
	sgResp, sgErr := client.DescribeSecurityGroupsWithOptions(&ecs.DescribeSecurityGroupsRequest{
		RegionId: tea.String(*region),
	}, aliyun.ProbeRuntimeOptions())
	if sgErr != nil {
		report(false, "安全组", "查询失败: "+sgErr.Error())
	} else if sgResp.Body == nil || sgResp.Body.SecurityGroups == nil || len(sgResp.Body.SecurityGroups.SecurityGroup) == 0 {
		report(false, "安全组", *region+" 账号下无安全组（需先建）")
	} else {
		ids := make([]string, 0, 3)
		for i, sg := range sgResp.Body.SecurityGroups.SecurityGroup {
			if i >= 3 || sg == nil {
				continue
			}
			ids = append(ids, tea.StringValue(sg.SecurityGroupId))
		}
		report(true, "安全组", fmt.Sprintf("可用 %d 个，示例: %s", len(sgResp.Body.SecurityGroups.SecurityGroup), strings.Join(ids, ", ")))
	}

	// 5) 实例规格存在（可用库存随区随量波动，这里只验证规格 ID 合法）。
	itResp, itErr := client.DescribeInstanceTypesWithOptions(&ecs.DescribeInstanceTypesRequest{
		InstanceTypes: []*string{tea.String(*instanceType)},
	}, aliyun.ProbeRuntimeOptions())
	if itErr != nil {
		report(false, "实例规格", *instanceType+" 查询失败: "+itErr.Error())
	} else if itResp.Body == nil || itResp.Body.InstanceTypes == nil || len(itResp.Body.InstanceTypes.InstanceType) == 0 {
		report(false, "实例规格", *instanceType+" 不存在（用 DescribeInstanceTypes 换一个）")
	} else {
		report(true, "实例规格", *instanceType)
	}

	if failures > 0 {
		fmt.Printf("\n交付就绪度: %d 项未通过，先补齐再跑完整交付验收\n", failures)
		os.Exit(1)
	}
	fmt.Println("\n交付就绪度: 全部通过——以上参数可直接填入服务请求表单跑完整验收")
}
