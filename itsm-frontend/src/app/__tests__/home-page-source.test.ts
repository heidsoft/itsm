import fs from 'node:fs';
import path from 'node:path';

describe('开源首页快速开始契约', () => {
  const source = fs.readFileSync(path.join(process.cwd(), 'src/app/page.tsx'), 'utf8');

  it('使用真实仓库目录和当前开发启动命令', () => {
    expect(source).toContain('$ cd itsm');
    expect(source).toContain('$ cp .env.dev.example .env');
    expect(source).toContain('$ make dev-start-docker');
    expect(source).not.toContain('$ cd tism');
  });

  // 公开落地页此前无条件渲染 admin / <示例密码>，生产构建对未认证访客明文展示。
  // 现在与登录页同款 NODE_ENV 门控；守卫按源码位置判定字面量只能落在开发分支里
  //（把门控删掉，下面的切片断言会立刻变红）。
  it('演示凭据只在开发环境分支渲染，生产不展示明文密码', () => {
    const guard = "process.env.NODE_ENV === 'development'";
    expect(source).toContain(guard);
    expect(source.slice(0, source.indexOf(guard))).not.toContain('admin123');

    const devBranch = source.slice(source.indexOf(guard), source.indexOf('管理员凭据'));
    expect(devBranch).toContain('admin123');
    expect(source.slice(source.indexOf('管理员凭据'))).not.toContain('admin123');
  });

  it('生产管理员凭据口径与 README / prod compose 一致', () => {
    expect(source).toContain('.env.prod');
    expect(source).toContain('ADMIN_PASSWORD');
    expect(source).toContain('itsm-init');
    // bootstrap token 是可选流程且 prod 默认关闭，旧文案把它写成生产默认模型，与实测相反。
    expect(source).not.toContain("ProductionDeployuse一次性 bootstrap token");
    expect(source).not.toContain("不提供DefaultPassword");
  });
});
