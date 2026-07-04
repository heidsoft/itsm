#!/usr/bin/env node
/**
 * 图标库迁移脚本: @ant-design/icons -> lucide-react
 * 简化版 - 使用 sed 进行直接替换
 */

const fs = require('fs');
const path = require('path');

// 基础映射表 - 最常用的图标
const ICON_MAPPINGS = [
  // 箭头
  ['ArrowLeftOutlined', 'ArrowLeft'],
  ['ArrowRightOutlined', 'ArrowRight'],
  ['ArrowUpOutlined', 'ArrowUp'],
  ['ArrowDownOutlined', 'ArrowDown'],
  ['LeftOutlined', 'ChevronLeft'],
  ['RightOutlined', 'ChevronRight'],
  ['UpOutlined', 'ChevronUp'],
  ['DownOutlined', 'ChevronDown'],
  ['DoubleLeftOutlined', 'ChevronsLeft'],
  ['DoubleRightOutlined', 'ChevronsRight'],

  // 基础操作
  ['SearchOutlined', 'Search'],
  ['FilterOutlined', 'Filter'],
  ['PlusOutlined', 'Plus'],
  ['MinusOutlined', 'Minus'],
  ['CloseOutlined', 'X'],
  ['CheckOutlined', 'Check'],
  ['SaveOutlined', 'Save'],
  ['EditOutlined', 'Pencil'],
  ['DeleteOutlined', 'Trash2'],
  ['CopyOutlined', 'Copy'],
  ['DownloadOutlined', 'Download'],
  ['UploadOutlined', 'Upload'],
  ['ExportOutlined', 'Export'],
  ['ImportOutlined', 'Import'],

  // 用户相关
  ['UserOutlined', 'User'],
  ['TeamOutlined', 'Users'],
  ['UserAddOutlined', 'UserPlus'],
  ['UserDeleteOutlined', 'UserMinus'],
  ['LockOutlined', 'Lock'],
  ['UnlockOutlined', 'Unlock'],
  ['SafetyOutlined', 'Shield'],

  // 文件相关
  ['FileOutlined', 'File'],
  ['FileTextOutlined', 'FileText'],
  ['FolderOutlined', 'Folder'],
  ['FolderOpenOutlined', 'FolderOpen'],
  ['FileAddOutlined', 'FilePlus'],
  ['FileSearchOutlined', 'FileSearch'],
  ['FileDeleteOutlined', 'FileX'],
  ['FolderAddOutlined', 'FolderPlus'],

  // 视图相关
  ['EyeOutlined', 'Eye'],
  ['EyeInvisibleOutlined', 'EyeOff'],
  ['MenuOutlined', 'Menu'],
  ['AppstoreOutlined', 'LayoutGrid'],
  ['DashboardOutlined', 'LayoutDashboard'],
  ['HomeOutlined', 'Home'],
  ['SettingOutlined', 'Settings'],
  ['ToolOutlined', 'Wrench'],
  ['BuildOutlined', 'Hammer'],

  // 业务相关
  ['CalendarOutlined', 'Calendar'],
  ['ClockCircleOutlined', 'Clock'],
  ['HistoryOutlined', 'History'],
  ['SyncOutlined', 'RefreshCw'],
  ['ReloadOutlined', 'RotateCcw'],
  ['UndoOutlined', 'Undo'],
  ['RedoOutlined', 'Redo'],
  ['RollbackOutlined', 'Undo'],
  ['BellOutlined', 'Bell'],
  ['NotificationOutlined', 'Bell'],
  ['MailOutlined', 'Mail'],
  ['MessageOutlined', 'MessageSquare'],
  ['PhoneOutlined', 'Phone'],
  ['MobileOutlined', 'Smartphone'],
  ['GlobalOutlined', 'Globe'],
  ['LinkOutlined', 'Link'],
  ['UnlinkOutlined', 'Unlink'],
  ['ShareAltOutlined', 'Share2'],

  // 状态图标
  ['InfoCircleOutlined', 'Info'],
  ['WarningOutlined', 'AlertTriangle'],
  ['AlertOutlined', 'AlertTriangle'],
  ['QuestionCircleOutlined', 'HelpCircle'],
  ['ExclamationCircleOutlined', 'AlertCircle'],
  ['CheckCircleOutlined', 'CheckCircle'],
  ['CloseCircleOutlined', 'XCircle'],
  ['PlusCircleOutlined', 'PlusCircle'],
  ['MinusCircleOutlined', 'MinusCircle'],

  // 图表
  ['BarChartOutlined', 'BarChart3'],
  ['LineChartOutlined', 'LineChart'],
  ['PieChartOutlined', 'PieChart'],
  ['DotChartOutlined', 'Activity'],
  ['FundOutlined', 'TrendingUp'],
  ['StockOutlined', 'LineChart'],

  // 设备/存储
  ['DatabaseOutlined', 'Database'],
  ['ServerOutlined', 'Server'],
  ['CloudOutlined', 'Cloud'],
  ['DesktopOutlined', 'Monitor'],
  ['LaptopOutlined', 'Laptop'],
  ['TabletOutlined', 'Tablet'],

  // 列表/排序
  ['UnorderedListOutlined', 'List'],
  ['OrderedListOutlined', 'ListOrdered'],
  ['SortAscendingOutlined', 'ArrowUpDown'],
  ['SortDescendingOutlined', 'ArrowUpDown'],

  // 杂项
  ['PrinterOutlined', 'Printer'],
  ['StarOutlined', 'Star'],
  ['StarFilled', 'Star'],
  ['HeartOutlined', 'Heart'],
  ['HeartFilled', 'Heart'],
  ['LikeOutlined', 'ThumbsUp'],
  ['DislikeOutlined', 'ThumbsDown'],
  ['TagOutlined', 'Tag'],
  ['TagsOutlined', 'Tags'],
  ['FlagOutlined', 'Flag'],
  ['KeyOutlined', 'Key'],
  ['BugOutlined', 'Bug'],
  ['CodeOutlined', 'Code'],
  ['ApiOutlined', 'Plug'],
  ['ToolOutlined', 'Wrench'],
  ['ThunderboltOutlined', 'Zap'],
  ['FireOutlined', 'Flame'],
  ['RocketOutlined', 'Rocket'],
  ['ShopOutlined', 'Store'],
  ['GiftOutlined', 'Gift'],
  ['TrophyOutlined', 'Trophy'],
  ['CrownOutlined', 'Crown'],
  ['MedalOutlined', 'Medal'],
  ['SecurityScanOutlined', 'ShieldAlert'],
  ['SafetyCertificateOutlined', 'ShieldCheck'],
  ['InsuranceOutlined', 'Shield'],
  ['FirewallOutlined', 'Shield'],
  ['BlockOutlined', 'Ban'],
  ['FullscreenOutlined', 'Maximize'],
  ['FullscreenExitOutlined', 'Minimize'],
  ['ZoomInOutlined', 'ZoomIn'],
  ['ZoomOutOutlined', 'ZoomOut'],
  ['DragOutlined', 'GripVertical'],
  ['SelectOutlined', 'MousePointer2'],
  ['ControlOutlined', 'Sliders'],
  ['SlidersOutlined', 'Sliders'],
  ['ProjectOutlined', 'Briefcase'],
  ['TableOutlined', 'Table'],
  ['BankOutlined', 'Building2'],
  ['AuditOutlined', 'FileCheck'],
  ['ExperimentOutlined', 'FlaskConical'],
  ['LoginOutlined', 'LogIn'],
  ['LogoutOutlined', 'LogOut'],
  ['ForwardOutlined', 'Forward'],
  ['BackwardOutlined', 'Backward'],
  ['EnterOutlined', 'ArrowRight'],
  ['ExitOutlined', 'ArrowLeft'],
  ['CarryOutOutlined', 'CheckSquare'],
  ['ScheduleOutlined', 'Calendar'],
  ['FieldTimeOutlined', 'Timer'],
  ['ClusterOutlined', 'Network'],
  ['NodeIndexOutlined', 'Network'],
  ['BranchesOutlined', 'GitBranch'],
  ['MergeOutlined', 'GitMerge'],
  ['PullRequestOutlined', 'GitPullRequest'],
  ['PlayCircleOutlined', 'PlayCircle'],
  ['PauseCircleOutlined', 'PauseCircle'],
  ['StopCircleOutlined', 'StopCircle'],
  ['VideoCameraOutlined', 'Video'],
  ['CameraOutlined', 'Camera'],
  ['PictureOutlined', 'Image'],
  ['FileImageOutlined', 'Image'],
  ['InboxOutlined', 'Inbox'],
  ['EnvironmentOutlined', 'MapPin'],
  ['AimOutlined', 'Crosshair'],
  ['CompassOutlined', 'Compass'],
  ['WifiOutlined', 'Wifi'],
  ['CustomerServiceOutlined', 'Headphones'],
  ['ShoppingCartOutlined', 'ShoppingCart'],
  ['WalletOutlined', 'Wallet'],
  ['CreditCardOutlined', 'CreditCard'],
  ['MoneyCollectOutlined', 'DollarSign'],
  ['FundProjectionScreenOutlined', 'Presentation'],
  ['AreaChartOutlined', 'AreaChart'],
  ['RadarChartOutlined', 'Radar'],
  ['HeatMapOutlined', 'Grid3x3'],
  ['BoxPlotOutlined', 'Box'],
  ['ContainerOutlined', 'Container'],
  ['CopyrightOutlined', 'Copyright'],
  ['TrademarkOutlined', 'Trademark'],
  ['RegisteredOutlined', 'Registered'],
  ['AudioOutlined', 'Volume2'],
  ['AudioMutedOutlined', 'VolumeX'],
  ['SoundOutlined', 'Speaker'],
  ['StepBackwardOutlined', 'SkipBack'],
  ['StepForwardOutlined', 'SkipForward'],
  ['FastBackwardOutlined', 'Rewind'],
  ['FastForwardOutlined', 'FastForward'],
  ['CaretUpOutlined', 'CaretUp'],
  ['CaretDownOutlined', 'CaretDown'],
  ['CaretLeftOutlined', 'CaretLeft'],
  ['CaretRightOutlined', 'CaretRight'],
  ['VerticalAlignTopOutlined', 'AlignStartVertical'],
  ['VerticalAlignBottomOutlined', 'AlignEndVertical'],
  ['VerticalAlignMiddleOutlined', 'AlignCenterVertical'],
  ['AlignLeftOutlined', 'AlignLeft'],
  ['AlignCenterOutlined', 'AlignCenter'],
  ['AlignRightOutlined', 'AlignRight'],
  ['BoldOutlined', 'Bold'],
  ['ItalicOutlined', 'Italic'],
  ['UnderlineOutlined', 'Underline'],
  ['StrikethroughOutlined', 'Strikethrough'],
  ['FontSizeOutlined', 'Type'],
  ['NumberOutlined', 'Hash'],
  ['BorderOutlined', 'Square'],
  ['BorderInnerOutlined', 'SquareDashed'],
  ['PicCenterOutlined', 'Scissors'],
  ['ExtensionOutlined', 'Puzzle'],
  ['AttachmentOutlined', 'Paperclip'],
  ['DislinkOutlined', 'Unlink'],
  ['ChargerOutlined', 'Battery'],
  ['OneToOneOutlined', 'UserCheck'],
  ['ContactsOutlined', 'Contact'],
  ['UsergroupAddOutlined', 'UserPlus'],
  ['UsergroupDeleteOutlined', 'UserMinus'],
  ['PartitionOutlined', 'PieChart'],
  ['LayoutOutlined', 'Layout'],
  ['HeaderOutlined', 'Heading'],
  ['ContentOutlined', 'AlignJustify'],
  ['FooterOutlined', 'AlignEndHorizontal'],
  ['SiderMenuOutlined', 'PanelLeft'],
  ['TopMenuOutlined', 'PanelTop'],
  ['SidebarOutlined', 'PanelLeft'],
  ['MenuFoldOutlined', 'PanelLeftClose'],
  ['MenuUnfoldOutlined', 'PanelLeft'],
  ['HighlightOutlined', 'Highlighter'],
  ['ScissorOutlined', 'Scissors'],
  ['ExpandOutlined', 'Maximize2'],
  ['CompressOutlined', 'Minimize2'],
  ['ShrinkOutlined', 'Minimize2'],
  ['ArrowsAltOutlined', 'Maximize2'],
  ['SwapOutlined', 'Swap'],
  ['SwapLeftOutlined', 'ArrowLeftRight'],
  ['TransactionOutlined', 'ArrowLeftRight'],
  ['RetweetOutlined', 'Repeat'],
  ['ProgressOutlined', 'Loader2'],
  ['SkipNextOutlined', 'SkipForward'],
  ['SkipPreviousOutlined', 'SkipBack'],
  ['RadiusSettingOutlined', 'CornerDownRight'],
  ['ToTopOutlined', 'ArrowUp'],
  ['ToBottomOutlined', 'ArrowDown'],
  ['AlignStartOutlined', 'AlignStartHorizontal'],
  ['AlignEndOutlined', 'AlignEndHorizontal'],
  ['VerticalLeftOutlined', 'PanelLeft'],
  ['VerticalRightOutlined', 'PanelRight'],
  ['ColumnWidthOutlined', 'Columns'],
  ['ColumnHeightOutlined', 'Rows'],
  ['InfoCircleFilled', 'Info'],
  ['CheckCircleFilled', 'CheckCircle'],
  ['CloseCircleFilled', 'XCircle'],
  ['WarningFilled', 'AlertTriangle'],
  ['ErrorOutlined', 'XCircle'],
  ['DeleteRowOutlined', 'Trash'],
  ['MinusRowOutlined', 'Minus'],
  ['DragEndOutlined', 'Drag'],
  ['FileProtectOutlined', 'Shield'],
  ['FileDoneOutlined', 'FileCheck'],
  ['FileUnknownOutlined', 'FileQuestion'],
  ['FileExclamationOutlined', 'FileWarning'],
  ['FilePdfOutlined', 'FileText'],
  ['FileJpgOutlined', 'Image'],
  ['CloudServerOutlined', 'Server'],
  ['CloudUploadOutlined', 'CloudUpload'],
  ['CloudDownloadOutlined', 'CloudDownload'],
  ['CloudSyncOutlined', 'Cloud'],
  ['LaptopOutlined', 'Laptop'],
  ['SecurityScanOutlined', 'ShieldAlert'],
  ['SafeOutlined', 'ShieldCheck'],
  ['GoldOutlined', 'Gem'],
  ['IdcardOutlined', 'IdCard'],
  ['Idcard', 'IdCard'],
  ['Environment', 'MapPin'],
  ['NodeIndex', 'Network'],
  ['Contacts', 'Contact'],
  ['ExperimentFill', 'FlaskConical'],
  ['SkinOutlined', 'Palette'],
  ['BgColorsOutlined', 'Palette'],
  ['Css3Outlined', 'Palette'],
  ['StorageOutlined', 'HardDrive'],
  ['LoadingOutlined', 'Loader2'],
  ['RobotOutlined', 'Bot'],
];

// 扫描目录找文件
function scanDirectory(dir, files = []) {
  try {
    const items = fs.readdirSync(dir);

    for (const item of items) {
      const fullPath = path.join(dir, item);
      try {
        const stat = fs.statSync(fullPath);

        if (stat.isDirectory()) {
          if (!['node_modules', '.next', '.git', 'dist', 'build', '.turbo'].includes(item)) {
            scanDirectory(fullPath, files);
          }
        } else if (fullPath.endsWith('.tsx') || fullPath.endsWith('.ts')) {
          const content = fs.readFileSync(fullPath, 'utf8');
          if (content.includes("@ant-design/icons")) {
            files.push(fullPath);
          }
        }
      } catch (e) {
        // 忽略权限错误
      }
    }
  } catch (e) {
    console.error(`Error scanning ${dir}: ${e.message}`);
  }
  return files;
}

// 处理单个文件
function processFile(filePath) {
  let content = fs.readFileSync(filePath, 'utf8');
  let modified = false;

  // 1. 提取所有 @ant-design/icons 的导入
  const importRegex = /import\s+\{([^}]+)\}\s+from\s+['"]@ant-design\/icons['"]/g;
  let match;
  const allImports = new Set();

  while ((match = importRegex.exec(content)) !== null) {
    const imports = match[1].split(',').map(s => {
      // 处理 "Icon as Alias" 格式
      const parts = s.trim().split(/\s+as\s+/);
      return parts[0].trim();
    });

    for (const icon of imports) {
      allImports.add(icon);
    }
  }

  if (allImports.size === 0) {
    return false;
  }

  // 2. 创建新的导入 - 只保留被映射的图标
  const newIcons = [];
  for (const [original, mapped] of ICON_MAPPINGS) {
    if (allImports.has(original)) {
      newIcons.push(mapped);
    }
  }

  // 3. 替换 import 语句
  const uniqueNewIcons = [...new Set(newIcons)];
  if (uniqueNewIcons.length > 0) {
    const newImport = `import { ${uniqueNewIcons.join(', ')} } from 'lucide-react';`;
    content = content.replace(importRegex, newImport);
    modified = true;
  }

  // 4. 替换 JSX 中的图标使用
  for (const [original, mapped] of ICON_MAPPINGS) {
    // 替换 <Original ... />
    const componentRegex = new RegExp(`<(${original})(\\s|/|>)`, 'g');
    if (componentRegex.test(content)) {
      content = content.replace(componentRegex, `<${mapped}$2`);
      modified = true;
    }

    // 替换 {Original} 用法
    const usageRegex = new RegExp(`\\b(${original})\\b`, 'g');
    if (usageRegex.test(content)) {
      content = content.replace(usageRegex, mapped);
      modified = true;
    }
  }

  if (modified) {
    fs.writeFileSync(filePath, content, 'utf8');
    return true;
  }

  return false;
}

// 构建图标列表
const allIcons = ICON_MAPPINGS.map(([original, mapped]) => mapped);

// 主函数
function main() {
  const srcDir = path.join(__dirname, '..', 'src');

  console.log('🔍 扫描使用 @ant-design/icons 的文件...');
  const files = scanDirectory(srcDir);
  console.log(`📁 找到 ${files.length} 个文件\n`);

  let processed = 0;
  let failed = 0;

  for (const file of files) {
    try {
      const relativePath = path.relative(__dirname, file);
      if (processFile(file)) {
        console.log(`✅ ${relativePath}`);
        processed++;
      }
    } catch (error) {
      console.error(`❌ ${path.relative(__dirname, file)}: ${error.message}`);
      failed++;
    }
  }

  console.log(`\n📊 完成! 成功处理 ${processed} 个文件, 失败 ${failed} 个文件`);
  console.log('\n⚠️  请手动检查以下内容:');
  console.log('   1. 运行 npm run type-check 检查类型错误');
  console.log('   2. 运行 npm run lint 检查代码问题');
  console.log('   3. 确认图标显示正常');
}

main();
