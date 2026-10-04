#!/usr/bin/env bash
# Anti-pattern guard for legacy Ant Design v4/v5 APIs (v6 migration).
# Run from anywhere; resolves its own repo root.
#
# 两层检测：
#   1. 全局模式 —— 出现即违规（Space.direction / Tabs.TabPane / Form 复合组件）。
#   2. 组件归属模式 —— 仅当 JSX 标签是本文件从 'antd' 具名导入（含 `as` 别名）
#      的 antd 组件时才判定违规。项目自有组件可以合法拥有同名 prop
#      （visible/overlay/...），归属由 import 决定，不做全局文本误伤。
#      弃用面已逐一对照 node_modules/antd 的 .d.ts @deprecated 注释核实
#      （Statistic/Spin/Alert 三项按 6.3.1 核实，其余按 6.2.2）：
#        visible            Modal/Drawer（v6 已移除，须用 open）
#        destroyOnClose     Modal/Drawer（改 destroyOnHidden）
#        bodyStyle          Card/Modal/Drawer（改 styles.body）
#        overlay            Dropdown（v6 已移除，须用 menu）
#        dropdownRender     Select/TreeSelect/AutoComplete/Cascader/Dropdown（改 popupRender）
#        onDropdownVisibleChange Select/TreeSelect/AutoComplete/Cascader（改 onOpenChange）
#        valueStyle         Statistic（改 styles.content）
#        Spin.tip           Spin（改 description）
#        Alert.message      Alert（`mergedTitle = title ?? message`，仍能渲染但每次打
#                           dev 弃用告警；改 title）
#        Alert.onClose      Alert（改 closable={{ onClose }}）
#
# 基线棘轮：各类别命中数与 tools/antd-legacy-baseline.txt 比较——
#   超过基线 → exit 1；低于基线 → 提示收紧基线（只许降不许升）。
#   `--list` 打印全部命中的 file:line 明细（用于收紧基线前核对）。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC_DIR="$REPO_ROOT/src"
BASELINE_FILE="$SCRIPT_DIR/antd-legacy-baseline.txt"
LIST_ALL=0
if [ "${1:-}" = "--list" ]; then
  LIST_ALL=1
  export ANTD_LEGACY_LIST=1
fi

if [ ! -d "$SRC_DIR" ]; then
  echo "✗ src/ not found at $SRC_DIR"
  exit 1
fi

# Collect matching ts/tsx files once (POSIX-compatible, bash 3.2-safe).
FILES=$(find "$SRC_DIR" -type f \( -name '*.ts' -o -name '*.tsx' \) 2>/dev/null)
if [ -z "$FILES" ]; then
  echo "✗ No source files found under $SRC_DIR"
  exit 1
fi

if [ ! -f "$BASELINE_FILE" ]; then
  echo "✗ baseline file missing: $BASELINE_FILE"
  exit 1
fi

ANTD_LEGACY_BASELINE="$BASELINE_FILE" perl -e '
use strict; use warnings;

my $LIST_ALL = ($ENV{ANTD_LEGACY_LIST} || 0) eq "1";

# [类别, 组件清单, prop 正则] —— 在剥离花括号/引号表达式后的裸属性名上判定，
# 因此布尔简写（destroyOnClose）与值形式（visible={x}）都能命中，
# 而表达式内部的同名变量（title={visible ? ...}）不会误报。
my @ATTR = (
  ["visible",                 ["Modal","Drawer"],                                              qr/(?<![.\w])visible\b/],
  ["destroyOnClose",          ["Modal","Drawer"],                                              qr/(?<![.\w])destroyOnClose\b/],
  ["bodyStyle",               ["Card","Modal","Drawer"],                                       qr/(?<![.\w])bodyStyle\b/],
  ["overlay",                 ["Dropdown"],                                                    qr/(?<![.\w])overlay\b/],
  ["dropdownRender",          ["Select","TreeSelect","AutoComplete","Cascader","Dropdown"],    qr/(?<![.\w])dropdownRender\b/],
  ["onDropdownVisibleChange", ["Select","TreeSelect","AutoComplete","Cascader"],               qr/(?<![.\w])onDropdownVisibleChange\b/],
  ["valueStyle",              ["Statistic"],                                                   qr/(?<![.\w])valueStyle\b/],
  ["Spin.tip",                ["Spin"],                                                        qr/(?<![.\w])tip\b/],
  ["Alert.message",           ["Alert"],                                                       qr/(?<![.\w])message\b/],
  ["Alert.onClose",           ["Alert"],                                                       qr/(?<![.\w])onClose\b/],
);

# 历史全局模式（保持原行级语义，不加归属判定）。
my @GLOBAL = (
  ["Space.direction", qr/<Space\b[^>]*\bdirection\s*=/],
  ["Tabs.TabPane",    qr/<Tabs\.TabPane\b/],
  ["Form.compound",   qr/\bForm\.(Input|TextArea|Select|DatePicker|Radio|Checkbox)\b/],
);

my %count;   # 类别 -> 命中数
my %detail;  # 类别 -> [file:line: 片段]

foreach my $file (@ARGV) {
  open my $fh, "<", $file or next;
  local $/; my $src = <$fh>; close $fh;

  # 全局模式：按行，保持与旧版 grep -E 一致的语义。
  my @lines = split /\n/, $src, -1;
  for my $i (0 .. $#lines) {
    for my $g (@GLOBAL) {
      my ($key, $re) = @$g;
      if ($lines[$i] =~ $re) {
        (my $s = $lines[$i]) =~ s/^\s+|\s+$//g;
        push @{$detail{$key}}, "$file:" . ($i + 1) . ": $s";
        $count{$key}++;
      }
    }
  }

  # 本文件从 antd 具名导入的组件：别名 -> 真名（type import 亦算）。
  my %alias2real;
  while ($src =~ /import\s+(?:type\s+)?\{([^}]+)\}\s*from\s*[\x27"]antd[\x27"]/g) {
    for my $part (split /,/, $1) {
      $part =~ s{//[^\n]*}{};
      $part =~ s{/\*.*?\*/}{}gs;
      $part =~ s/^\s+//; $part =~ s/\s+$//;
      next unless length $part;
      if ($part =~ /^([A-Za-z_]\w*)\s+as\s+([A-Za-z_]\w*)$/) { $alias2real{$2} = $1; }
      elsif ($part =~ /^[A-Za-z_]\w*$/) { $alias2real{$part} = $part; }
    }
  }
  next unless %alias2real;

  my %real2aliases;
  while (my ($a, $r) = each %alias2real) { push @{$real2aliases{$r}}, $a; }

  for my $chk (@ATTR) {
    my ($key, $comps, $re) = @$chk;
    for my $comp (@$comps) {
      my $aliases = $real2aliases{$comp} or next;
      for my $alias (@$aliases) {
        # 逐个开标签扫描：按字符走一遍标签开括号区域，{} 深度与引号串都要跟踪，
        # 只有深度 0 的 > 才是标签结束。之前的「找第一个 >」或「两层花括号」写法
        # 会被 `onClose={() => x}` 里的 `=>` 与 `extra={<Space><Button/>…}` 这种
        # 深层 JSX 提前截断，于是同一标签后面的弃用属性永远看不见——实测因此
        # 漏过 valueStyle 9 处、destroyOnClose 2 处（台账 E4-34）。
        my $cursor = 0;
        while ((my $tag_start = index($src, "<$alias", $cursor)) >= 0) {
          $cursor = $tag_start + 1;
          my $name_end = $tag_start + 1 + length($alias);
          my $nx = substr($src, $name_end, 1);
          next if defined $nx && $nx =~ /[\w\-]/;   # <AlertDialog 等同名前缀组件不算
          my $i = $name_end;
          my ($depth, $quote, $attrs) = (0, "", "");
          while ($i < length($src)) {
            my $ch = substr($src, $i, 1);
            if    ($quote ne "") { $quote = "" if $ch eq $quote; }
            elsif ($ch eq "\x22" or $ch eq "\x27") { $quote = $ch; }
            elsif ($ch eq "{") { $depth++; }
            elsif ($ch eq "}") { $depth--; $depth = 0 if $depth < 0; }
            elsif ($ch eq ">" and $depth == 0) { last; }
            elsif ($depth == 0) { $attrs .= $ch; }
            $i++;
          }
          $cursor = $i + 1 if $i > $cursor;
          if ($attrs =~ $re) {
            my $line = 1 + (substr($src, 0, $tag_start) =~ tr/\n//);
            my $snip = substr($src, $tag_start, 100);
            $snip =~ s/\s+/ /g;
            push @{$detail{$key}}, "$file:$line: $snip";
            $count{$key}++;
          }
        }
      }
    }
  }
}

# 基线：每行 "类别 数值"，# 开头为注释；缺类别按 0 处理（新类别命中即红，须登记）。
my %base;
if (open my $bf, "<", $ENV{ANTD_LEGACY_BASELINE}) {
  while (my $l = <$bf>) {
    chomp $l; $l =~ s/^\s+|\s+$//g;
    next unless length $l; next if $l =~ /^#/;
    my ($k, $v) = split /\s+/, $l;
    $base{$k} = $v + 0 if defined $v;
  }
  close $bf;
} else {
  print "✗ 无法读取基线文件: $ENV{ANTD_LEGACY_BASELINE}\n";
  exit 1;
}

my @ALL_KEYS = ((map { $_->[0] } @ATTR), (map { $_->[0] } @GLOBAL));
my $exit = 0;
for my $key (sort @ALL_KEYS) {
  my $b = exists $base{$key} ? $base{$key} : 0;
  my $c = $count{$key} || 0;
  my $status = $c > $b ? "✗ 超基线" : "✓";
  print "$key: $c/$b $status\n";
  if ($c > $b) { $exit = 1; }
  if ($LIST_ALL || $c > $b) {
    print "  $_\n" for @{$detail{$key} || []};
  }
}
for my $key (sort keys %base) {
  my $c = $count{$key} || 0;
  if ($c < $base{$key}) {
    print "△ $key: 现命中 $c 低于基线 $base{$key}，请收紧 tools/antd-legacy-baseline.txt\n";
  }
}

if ($exit == 0) {
  print $LIST_ALL ? "" : "✓ No legacy antd APIs detected (存量均在基线内)\n";
}
exit $exit;
' $FILES
