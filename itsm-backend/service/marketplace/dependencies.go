package marketplace

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"itsm-backend/ent"
	"itsm-backend/ent/itemversion"
	"itsm-backend/ent/marketplaceitem"
	"itsm-backend/ent/tenantinstallation"
)

// DependencyUnsatisfiedError 描述一次安装被依赖闸门拦下的全部缺口。
//
// 用独立类型而非 fmt.Errorf 拼串：界面需要逐项展示「哪个前置组件缺、为什么」，
// 把它们压成一句自由文本后，管理员只能靠猜，安装失败就退化成一次无信息量的报错。
type DependencyUnsatisfiedError struct {
	Problems []string
}

func (e *DependencyUnsatisfiedError) Error() string {
	if e == nil || len(e.Problems) == 0 {
		return "组件依赖未满足"
	}
	return "组件依赖未满足：" + strings.Join(e.Problems, "；")
}

// Is 让 errors.Is(err, ErrMarketplaceDependencyUnsatisfied) 成立，
// 便于 handler 稳定映射成 409 而不必认识具体实现类型。
func (e *DependencyUnsatisfiedError) Is(target error) bool {
	return target == ErrMarketplaceDependencyUnsatisfied
}

// DependentsBlockUninstallError 列出阻止卸载的已安装依赖方。
//
// 卸载是安装的反向操作，依赖闸门只做一半就会留下「装着的前置被卸掉」的
// 悬空依赖：界面显示前置已卸载，依赖它的组件仍标 active，实际调用全部失败。
type DependentsBlockUninstallError struct {
	Dependents []string
}

func (e *DependentsBlockUninstallError) Error() string {
	if e == nil || len(e.Dependents) == 0 {
		return "组件仍被其他已安装组件依赖"
	}
	return "组件仍被其他已安装组件依赖：" + strings.Join(e.Dependents, "、") + "，请先卸载它们"
}

func (e *DependentsBlockUninstallError) Is(target error) bool {
	return target == ErrMarketplaceItemRequiredByDependents
}

// findDependents 找出当前租户内仍依赖 targetName 的已安装组件。
//
// 依赖声明看的是各依赖方「已安装的那个版本」，不是市场里的最新版本：
// 最新版本新增的依赖与存量安装无关，拿它判定会挡住无关的卸载。
// 只有 active/disabled 算消费依赖——disabled 仍是租户内的已安装组件，
// 与安装期闸门的口径一致；installing/failed 未真正生效，其重试会重新过闸门。
func (s *Service) findDependents(ctx context.Context, tenantID, targetItemID int, targetName string) ([]string, error) {
	if strings.TrimSpace(targetName) == "" {
		return nil, nil
	}

	installations, err := s.db.TenantInstallation.Query().
		Where(
			tenantinstallation.TenantID(tenantID),
			tenantinstallation.ItemIDNEQ(targetItemID),
			tenantinstallation.StatusIn(tenantinstallation.StatusActive, tenantinstallation.StatusDisabled),
		).
		WithItem().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant installations: %w", err)
	}

	var dependents []string
	for _, installation := range installations {
		item, err := installation.Edges.ItemOrErr()
		if err != nil {
			return nil, fmt.Errorf("installation %d has no item: %w", installation.ID, err)
		}
		deps, err := s.dependenciesForVersion(ctx, item.ID, installation.InstalledVersion)
		if err != nil {
			return nil, err
		}
		if _, ok := deps[targetName]; !ok {
			continue
		}
		dependents = append(dependents, fmt.Sprintf("%s（%s %s）", item.Title, item.Name, installation.InstalledVersion))
	}
	sort.Strings(dependents)
	return dependents, nil
}

// validateDependencies 校验待安装版本声明的依赖在当前租户内都已安装且版本满足。
//
// 依赖声明存在版本行上（ItemVersion.dependencies，形如 组件名:版本要求）。
// 没有对应版本行时视为「未声明依赖」而非报错：存量与种子商品可能只写了
// MarketplaceItem.latest_version 而没有版本行，把它们一律判成有缺陷会误伤存量安装。
func (s *Service) validateDependencies(ctx context.Context, tenantID, itemID int, itemName, version string) error {
	deps, err := s.dependenciesForVersion(ctx, itemID, version)
	if err != nil {
		return err
	}
	if len(deps) == 0 {
		return nil
	}

	names := make([]string, 0, len(deps))
	for name := range deps {
		names = append(names, name)
	}
	sort.Strings(names) // 稳定输出，测试与界面提示可预期

	var problems []string
	for _, name := range names {
		requirement := strings.TrimSpace(deps[name])
		if name == itemName {
			problems = append(problems, fmt.Sprintf("%s（组件不能依赖自身）", name))
			continue
		}

		depItem, err := s.db.MarketplaceItem.Query().
			Where(marketplaceitem.NameEQ(name)).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				problems = append(problems, fmt.Sprintf("%s（市场中不存在该组件）", name))
				continue
			}
			return fmt.Errorf("failed to resolve dependency %q: %w", name, err)
		}

		// active 与 disabled 都算「已安装」：disabled 是管理员主动停用，
		// 组件本身仍在租户内；把它当成未安装会让依赖链在停用时整体断裂。
		installation, err := s.db.TenantInstallation.Query().
			Where(
				tenantinstallation.TenantID(tenantID),
				tenantinstallation.ItemID(depItem.ID),
				tenantinstallation.StatusIn(tenantinstallation.StatusActive, tenantinstallation.StatusDisabled),
			).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				problems = append(problems, fmt.Sprintf("%s（未安装，需要 %s）", name, displayRequirement(requirement)))
				continue
			}
			return fmt.Errorf("failed to check dependency %q installation: %w", name, err)
		}

		ok, err := versionSatisfies(installation.InstalledVersion, requirement)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s（%v）", name, err))
			continue
		}
		if !ok {
			problems = append(problems, fmt.Sprintf("%s（已安装 %s，需要 %s）", name, installation.InstalledVersion, displayRequirement(requirement)))
		}
	}

	if len(problems) > 0 {
		return &DependencyUnsatisfiedError{Problems: problems}
	}
	return nil
}

// dependenciesForVersion 读取指定版本的依赖声明；版本行不存在时返回 nil。
func (s *Service) dependenciesForVersion(ctx context.Context, itemID int, version string) (map[string]string, error) {
	if strings.TrimSpace(version) == "" {
		return nil, nil
	}
	row, err := s.db.ItemVersion.Query().
		Where(itemversion.ItemID(itemID), itemversion.Version(version)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load item version %q: %w", version, err)
	}
	if len(row.Dependencies) == 0 {
		return nil, nil
	}
	return row.Dependencies, nil
}

func displayRequirement(requirement string) string {
	if requirement == "" {
		return "任意版本"
	}
	return requirement
}

// versionSatisfies 判定已安装版本是否满足要求。
//
// 要求支持 >= > <= < = == 前缀；无前缀按精确匹配处理。
// 采用宽松的 x[.y[.z]] 数字比较并忽略 -prerelease 后缀：市场版本号主要由
// 目录管理员手工填写，强制严格 semver 会把 "1.0" 这类合法写法判成非法，
// 反过来又逼着大家绕过闸门。
func versionSatisfies(installed, requirement string) (bool, error) {
	requirement = strings.TrimSpace(requirement)
	if requirement == "" {
		return true, nil
	}

	op := "="
	for _, candidate := range []string{">=", "<=", "==", ">", "<", "="} {
		if strings.HasPrefix(requirement, candidate) {
			op = candidate
			requirement = strings.TrimSpace(requirement[len(candidate):])
			break
		}
	}
	if requirement == "" {
		return false, fmt.Errorf("版本要求缺少版本号")
	}

	installedParts, ok := parseVersionParts(installed)
	if !ok {
		return false, fmt.Errorf("已安装版本 %q 无法解析", installed)
	}
	requiredParts, ok := parseVersionParts(requirement)
	if !ok {
		return false, fmt.Errorf("版本要求 %q 无法解析", requirement)
	}

	cmp := compareVersionParts(installedParts, requiredParts)
	switch op {
	case ">=":
		return cmp >= 0, nil
	case ">":
		return cmp > 0, nil
	case "<=":
		return cmp <= 0, nil
	case "<":
		return cmp < 0, nil
	default: // "=" 与 "=="
		return cmp == 0, nil
	}
}

// parseVersionParts 把 v1.2.3-rc.1 解析成 [1,2,3]；允许缺省 patch，缺省按 0。
func parseVersionParts(raw string) ([]int, bool) {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	if v == "" {
		return nil, false
	}
	if idx := strings.IndexByte(v, '-'); idx >= 0 {
		v = v[:idx]
	}
	segments := strings.Split(v, ".")
	if len(segments) > 3 {
		return nil, false
	}
	parts := make([]int, 3)
	for i, segment := range segments {
		if segment == "" {
			return nil, false
		}
		n, err := strconv.Atoi(segment)
		if err != nil || n < 0 {
			return nil, false
		}
		parts[i] = n
	}
	return parts, true
}

func compareVersionParts(a, b []int) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
