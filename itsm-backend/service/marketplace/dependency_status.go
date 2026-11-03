package marketplace

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/tenantinstallation"
)

// 存量安装的依赖对齐读面（2026-11-03）。
//
// 依赖闸门此前只作用在「装的那一刻」：安装前校验前置、卸载前拦截被依赖。
// 但存量库里已经存在的安装记录没有经过这道闸——闸门上线前的安装、
// 直接改库的记录、以及只写了 latest_version 却没有版本行的种子商品，
// 都可能带着未满足的前置组件一直显示 active。
//
// 这一面只做读取与判定，不写库、不自动补装：
//   - 「自动补装」需要替租户决定装什么，那是管理员的动作不是后台的特权；
//   - 写状态位则要加新列，而本仓库当前不具备安全的 codegen/迁移路径。
//
// 因此结论按需在读取时算出，界面每次看到的都是与当前目录、当前安装记录
// 一致的结果，不会有一份会过期的「健康状态」字段。

// DependencyFinding 单条已安装组件的依赖判定。
// 不带 json tag：API 契约由 dto.MarketplaceDependencyResponse 承载，
// 这里只是领域结果，避免 service 与 handler 两处各有一套字段名。
type DependencyFinding struct {
	ItemID           int
	ItemName         string
	ItemTitle        string
	ItemType         string
	Status           string
	InstalledVersion string
	InstallStatus    string
	Dependencies     map[string]string
	// Problems 逐项缺口，与安装期闸门的文案同一来源（validateDependencies）。
	Problems []string
}

// 依赖判定结论。三态而非两态：把「没声明」并进「没问题」会让目录缺版本行的
// 商品看起来健康，而那恰恰是最需要先修的一类存量数据。
const (
	DependencyStatusOK                 = "ok"
	DependencyStatusUnsatisfied        = "unsatisfied"
	DependencyStatusDeclarationMissing = "declaration_missing"
)

// DependencyReport 一个租户的存量依赖对齐结果。
type DependencyReport struct {
	TenantID           int
	CheckedAt          time.Time
	Total              int
	Satisfied          int
	Unsatisfied        int
	DeclarationMissing int
	Findings           []DependencyFinding
}

// DependencyStatus 判定租户内每条生效安装（active/disabled）的依赖是否仍然满足。
func (s *Service) DependencyStatus(ctx context.Context, tenantID int) (*DependencyReport, error) {
	installations, err := s.db.TenantInstallation.Query().
		Where(
			tenantinstallation.TenantID(tenantID),
			tenantinstallation.StatusIn(tenantinstallation.StatusActive, tenantinstallation.StatusDisabled),
		).
		WithItem().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant installations: %w", err)
	}

	report := &DependencyReport{
		TenantID:  tenantID,
		CheckedAt: time.Now().UTC(),
		Total:     len(installations),
		Satisfied: 0,
		Findings:  make([]DependencyFinding, 0, len(installations)),
	}
	for _, installation := range installations {
		item, err := installation.Edges.ItemOrErr()
		if err != nil {
			return nil, fmt.Errorf("installation %d has no item: %w", installation.ID, err)
		}
		finding := DependencyFinding{
			ItemID:           item.ID,
			ItemName:         item.Name,
			ItemTitle:        item.Title,
			ItemType:         string(item.Type),
			InstalledVersion: installation.InstalledVersion,
			InstallStatus:    string(installation.Status),
			Status:           DependencyStatusOK,
		}
		hasVersionRow, err := s.hasVersionRow(ctx, item.ID, installation.InstalledVersion)
		if err != nil {
			return nil, err
		}
		if !hasVersionRow {
			// 版本行都不存在，依赖声明无处可查。这既不是「没问题」也不是「有问题」，
			// 而是「目录数据不完整」——修它需要补版本行，不是装前置组件。
			finding.Status = DependencyStatusDeclarationMissing
			finding.Problems = []string{fmt.Sprintf("版本 %s 在目录里没有版本行，依赖声明无法核对", displayVersion(installation.InstalledVersion))}
			report.DeclarationMissing++
			report.Findings = append(report.Findings, finding)
			continue
		}
		deps, err := s.dependenciesForVersion(ctx, item.ID, installation.InstalledVersion)
		if err != nil {
			return nil, err
		}
		finding.Dependencies = deps
		// 复用安装期闸门同一套判定：这里若另写一份版本比较与查找规则，
		// 存量报告与实际放行结果就会不一致，界面说「健康」而安装被拒是最糟的组合。
		if err := s.validateDependencies(ctx, tenantID, item.ID, item.Name, installation.InstalledVersion); err != nil {
			var unsatisfied *DependencyUnsatisfiedError
			if !errors.As(err, &unsatisfied) {
				return nil, err
			}
			finding.Status = DependencyStatusUnsatisfied
			finding.Problems = unsatisfied.Problems
			report.Unsatisfied++
		} else {
			report.Satisfied++
		}
		report.Findings = append(report.Findings, finding)
	}

	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].ItemName != report.Findings[j].ItemName {
			return report.Findings[i].ItemName < report.Findings[j].ItemName
		}
		return report.Findings[i].InstalledVersion < report.Findings[j].InstalledVersion
	})
	return report, nil
}

// hasVersionRow 判断目录里是否存在该版本行（任意状态）。
// 状态是否已发布不影响「有没有声明可查」这个判定。
func (s *Service) hasVersionRow(ctx context.Context, itemID int, version string) (bool, error) {
	if strings.TrimSpace(version) == "" {
		return false, nil
	}
	exists, err := s.db.ItemVersion.Query().
		Where(itemversion.ItemID(itemID), itemversion.Version(strings.TrimSpace(version))).
		Exist(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check version row for item %d version %q: %w", itemID, version, err)
	}
	return exists, nil
}

// displayVersion 空版本号的界面写法，避免展示成看不见的空白。
func displayVersion(version string) string {
	if strings.TrimSpace(version) == "" {
		return "（空）"
	}
	return version
}
