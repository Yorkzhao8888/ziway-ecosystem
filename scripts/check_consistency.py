#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
知味生态 A1.x 一致性守卫脚本
=================================
扫描全量文档集，校验「07_场景范例/00_一致性约束.md」中的 8 条铁律，
并支持 --fix 自动修复可机械处理的违规。

退出码：
  0  无违规（或 --fix 后全部修复）
  1  发现违规（CI 守卫失败）

设计原则：
  * 纯标准库，python3 直接运行，零依赖
  * 只扫描 .md / .yaml / .yml / .sql / .go / .ts（可配置）
  * 每条规则：id / 名称 / 严重度 / 匹配正则 / [修复函数]
"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Callable, Optional, Tuple

# ---------------------------------------------------------------------------
# 可配置项
# ---------------------------------------------------------------------------

# 需要扫描的根目录（脚本所在目录的上一级 = 合集根）
ROOT = Path(__file__).resolve().parent.parent

# 跳过的目录 / 文件（脚本自身、生成产物、node_modules、.git 等）
SKIP_DIRS = {
    "node_modules", ".git", "__pycache__", ".venv", "venv",
    "dist", "build", ".next", "coverage",
    "scripts",   # 守卫脚本自身含术语说明，不对其校验
}

# 「迁移说明」行关键词：含这些词的行视为有意的历史对照，跳过整行
MIGRATION_KEYWORDS = (
    "旧", "原为", "曾用", "原名", "迁移", "改名", "→", "->",
    "历史", "deprecated", "legacy", "LOCKED", "基线", "A1.",
)

# 扫描的文件后缀
SCAN_SUFFIXES = {".md", ".yaml", ".yml", ".sql", ".go", ".ts", ".js", ".py"}

# ---------------------------------------------------------------------------
# 规则定义
# ---------------------------------------------------------------------------


@dataclass
class Rule:
    rid: str              # 铁律编号，如 R1
    name: str             # 铁律简称
    severity: str         # error / warning
    pattern: str          # 匹配「违规」的正则
    message: str          # 违规说明
    fix: Optional[Callable[[str], Tuple[str, int]]] = None  # (新文本, 替换次数)


# 捕获完整单词，避免误伤（如 "OSA" 在 "OSAMBS" 中也会被单独匹配）
WORD = r"(?<![A-Za-z])"   # 左边界（前面不是字母）
WORD_R = r"(?![A-Za-z])"  # 右边界（后面不是字母）


def _repl_os_ms(m: re.Match) -> str:
    """把 MBS/BOS 风格旧名统一为 MS/OS（按域字母保留）。"""
    return m.group(0)  # 由具体规则决定，此处占位


def _noop(text: str) -> Tuple[str, int]:
    return text, 0


@dataclass
class Finding:
    rule: Rule
    file: Path
    line: int
    matched: str
    snippet: str


class Guard:
    def __init__(self, root: Path):
        self.root = root
        self.findings: list[Finding] = []
        self.fix_count = 0
        self._build_rules()

    # ---- 规则注册 ---------------------------------------------------------
    def _build_rules(self):
        self.rules: list[Rule] = []

        # R1 层命名：禁用 BOS / MBS / OSA / BS（应分别为 OS / MS / OAS）
        # 注意：要排除 "OAS" 自身、"AMBS"/"EMBS" 等正确复合词中的子串，
        # 采用「整词」匹配 + 白名单排除。
        self.rules.append(Rule(
            rid="R1a", name="禁用旧层名 BOS（应为 OS）",
            severity="error",
            pattern=r"\bBOS\b",
            message="出现旧层名 BOS，应改为 OS（如 CBOS→COS、DBOS→DOS）",
        ))
        self.rules.append(Rule(
            rid="R1b", name="禁用旧层名 MBS（应为 MS）",
            severity="error",
            pattern=r"\bMBS\b",
            message="出现旧层名 MBS，应改为 MS（如 CMBS→CMS、EMBS→EMS）",
        ))
        self.rules.append(Rule(
            rid="R1c", name="禁用 OSA（应为 OAS）",
            severity="error",
            pattern=r"\bOSA\b",
            message="OSA 与 OAS 混用，统一为 OAS（上帝视角技术底座）",
        ))
        self.rules.append(Rule(
            rid="R1d", name="禁用 BS 作为层名（应为 MS）",
            severity="error",
            pattern=r"\bBS\b(?!\s*管控)",
            message="BS 不是规范层名，应用「M 在 MS」表述",
        ))

        # R2 价值链：禁用 PM（应为 TM）
        self.rules.append(Rule(
            rid="R2", name="禁用旧价值链名 PM（应为 TM）",
            severity="error",
            pattern=r"\bPM\b(?!\s*〔|\s*\(|BS|O S|R3|R4|2)",
            message="价值链第一段命名为 TM（技术研发），禁用 PM",
        ))

        # R3 核心原则：「M 在 BS」分裂 → 应为「M 在 MS」
        self.rules.append(Rule(
            rid="R3", name="核心原则应为「X 在 OS、M 在 MS」",
            severity="error",
            pattern=r"M\s*在\s*BS",
            message="核心原则分裂：应为「X 在 OS、M 在 MS」",
        ))

        # R4 Kong 路由旧 MBS 前缀（ambs/pmbs/embs/cmbs/dmbs/fmbs/gmbs/hmbs/imbs/oms/vms/sms/ims）
        # 这些在 URL 路径里是合法的小写，但出现在「路由前缀」语境属违规。
        # 此处仅告警（warning），由人工确认是否真为网关路由。
        self.rules.append(Rule(
            rid="R4", name="Kong 路由疑似使用旧 MBS 前缀",
            severity="warning",
            pattern=r"/[a-z]{1,3}mbs\b",
            message="路由前缀疑似旧 MBS 风格（如 /embs），应改为 /ems 等 X-MS 风格",
        ))

        # R5 四根身份：CU 不得作为根身份（派生角色）
        # 此处只检查「CU 是根身份」的显式表述，不阻断正常 CU 使用
        self.rules.append(Rule(
            rid="R5", name="CU 应为派生角色，非根身份",
            severity="warning",
            pattern=r"CU\s*(作为|是|为)\s*(根|原子|原始)",
            message="CU（消费者）是派生角色，根身份仅为 OU/HU/EU/GU",
        ))

        # R6 「通用表不重复定义」与「列为增量表」的自相矛盾（SC-014）
        self.rules.append(Rule(
            rid="R6", name="SC-014 通用表重复定义风险",
            severity="warning",
            pattern=r"通用表不重复定义.{0,40}(iam\.tenants|iam\.users)",
            message="声明不重复定义通用表，却又把 iam.tenants/iam.users 列为增量表（逻辑矛盾）",
        ))

        # R7 事件幂等键规范提示（非阻断）
        self.rules.append(Rule(
            rid="R7", name="事件幂等键建议",
            severity="warning",
            pattern=r"idempotencyKey",
            message="确认幂等键 = eventType + eventId（72h 超时、死信队列）",
        ))

        # R8 场景五层覆盖（仅在索引文件检查，属结构检查，见 check_coverage）
        # 此处占位，不通过正则
        self.rules.append(Rule(
            rid="R8", name="场景五层覆盖",
            severity="warning",
            pattern=r"(?!x)x",  # 永不匹配，由 check_coverage 处理
            message="SC 场景应覆盖产研/业务/职能/治理/横切五层视角",
        ))

    # ---- 扫描 -------------------------------------------------------------
    def should_scan(self, path: Path) -> bool:
        # 跳过脚本自身与生成目录
        rel = path.relative_to(self.root)
        parts = set(rel.parts)
        if parts & SKIP_DIRS:
            return False
        if any(p.startswith(".") for p in rel.parts):
            return False
        # 跳过「元规范」文档：以 00_ 开头的索引/契约/约束文件会合法引用禁用词
        # （如 00_一致性约束.md、00_场景范例索引.md、00_数据字典.xlsx、00_OpenAPI.yaml）
        if path.stem.startswith("00_"):
            return False
        return path.suffix.lower() in SCAN_SUFFIXES

    def collect(self) -> list[Path]:
        files: list[Path] = []
        for p in self.root.rglob("*"):
            if p.is_file() and self.should_scan(p):
                files.append(p)
        return sorted(files)

    def scan(self, fix: bool = False):
        for f in self.collect():
            try:
                text = f.read_text(encoding="utf-8")
            except (UnicodeDecodeError, OSError):
                continue
            lines = text.split("\n")
            for rule in self.rules:
                if rule.rid == "R8":
                    continue  # 结构检查单独处理
                if rule.pattern == r"(?!x)x":
                    continue
                pat = re.compile(rule.pattern)
                in_ignore_block = False
                for idx, line in enumerate(lines):
                    # 块级忽略：`guard:ignore-start` 到 `guard:ignore-end` 之间整段跳过
                    if "guard:ignore-start" in line:
                        in_ignore_block = True
                        continue
                    if "guard:ignore-end" in line:
                        in_ignore_block = False
                        continue
                    if in_ignore_block:
                        continue
                    # 迁移说明行：含历史对照关键词，视为有意引用，跳过整行
                    if any(kw in line for kw in MIGRATION_KEYWORDS):
                        continue
                    # 内联忽略标记：该行含 `guard:ignore`，跳过整行
                    if "guard:ignore" in line:
                        continue
                    for m in pat.finditer(line):
                        self.findings.append(Finding(
                            rule, f, idx + 1, m.group(0),
                            self._snippet(line, m.start(), m.end()),
                        ))
            # 自动修复（仅对可修复规则，在 scan 后统一替换，避免重复计数）
            if fix and False:
                f.write_text(text, encoding="utf-8")

    # ---- 结构检查（R8 / 场景覆盖） ----------------------------------------
    def check_coverage(self):
        scene_dir = self.root / "07_场景范例"
        if not scene_dir.is_dir():
            return
        scenes = sorted(scene_dir.glob("SC-*.md"))
        for s in scenes:
            text = s.read_text(encoding="utf-8")
            # 五层视角关键词
            layers = {
                "产研(TM)": ["POS", "PMS", "Lab", "创研", "IP"],
                "业务(DM/EM/CM)": ["DOS", "EOS", "COS", "Mall", "Market", "Shop"],
                "职能": ["HOS", "HMS", "FOS", "FMS", "GOS", "GMS"],
                "治理": ["OOS", "OMS", "VOS", "VMS", "Case", "神案"],
                "横切": ["XBUS", "事件", "幂等", "tenant_id", "戴帽"],
            }
            any_layer = [
                "POS", "PMS", "Lab", "创研", "IP",
                "DOS", "EOS", "COS", "Mall", "Market", "Shop",
                "HOS", "HMS", "FOS", "FMS", "GOS", "GMS",
                "OOS", "OMS", "VOS", "VMS", "Case", "神案",
                "XBUS", "事件", "幂等", "tenant_id", "戴帽",
            ]
            if not any(kw in text for kw in any_layer):
                self.findings.append(Finding(
                    rule=Rule("R8", "场景业务归属", "warning", "",
                              "场景未命中任何五层视角关键词，请确认归属"),
                    file=s, line=1, matched="(无)",
                    snippet="建议补充所属平台/视角关键词",
                ))

    # ---- 自动修复（R1/R2/R3 的机械替换） --------------------------------
    def apply_fixes(self) -> int:
        """对可机械修复的规则执行替换，返回修复数量。"""
        fixes = [
            # R1a/b/c/d
            (re.compile(r"\bBOS\b"), self._bos_to_os),
            (re.compile(r"\bMBS\b"), self._mbs_to_ms),
            (re.compile(r"\bOSA\b"), "OAS"),
            # R2：PM（价值链语境）→ TM（仅替换「PM 事业/阶段/段」等）
            (re.compile(r"\bPM(?=\s*(?:事业|阶段|段|研发|产品|Manufacturing))"), "TM"),
            # R3：M 在 BS → M 在 MS（整段替换常见表述）
            (re.compile(r"M\s*在\s*BS"), "M 在 MS"),
            (re.compile(r"X\s*在\s*OS[、，]\s*M\s*在\s*MS"), "X 在 OS、M 在 MS"),
        ]
        count = 0
        for f in self.collect():
            try:
                text = f.read_text(encoding="utf-8")
            except (UnicodeDecodeError, OSError):
                continue
            original = text
            for pat, repl in fixes:
                if isinstance(repl, str):
                    text = pat.sub(repl, text)
                else:
                    text = pat.sub(repl, text)
            if text != original:
                f.write_text(text, encoding="utf-8")
                count += 1
        return count

    @staticmethod
    def _bos_to_os(m: re.Match) -> str:
        # CBOS→COS 等：把 BOS 前一个字母保留，后缀改 OS
        return m.group(0)  # 占位：具体域字母由正则上下文决定，此处保持原样

    @staticmethod
    def _mbs_to_ms(m: re.Match) -> str:
        return m.group(0)  # 占位

    # ---- 报告 -------------------------------------------------------------
    def report(self) -> str:
        lines = []
        by_rule: dict[str, list[Finding]] = {}
        for f in self.findings:
            by_rule.setdefault(f.rule.rid, []).append(f)

        lines.append("=" * 70)
        lines.append("知味生态 一致性守卫报告")
        lines.append(f"扫描根目录：{self.root}")
        lines.append(f"违规总数：{len(self.findings)}")
        lines.append("=" * 70)

        for rid in sorted(by_rule.keys()):
            items = by_rule[rid]
            rule = items[0].rule
            sev = rule.severity.upper()
            lines.append(f"\n[{sev}] {rid} {rule.name} （{len(items)} 处）")
            for it in items[:20]:  # 每规则最多显示 20 条
                rel = it.file.relative_to(self.root)
                lines.append(f"  {rel}:{it.line}  `{it.matched}`")
                if it.snippet and it.snippet != it.matched:
                    lines.append(f"      → {it.snippet}")
            if len(items) > 20:
                lines.append(f"  ... 另有 {len(items) - 20} 处")

        lines.append("\n" + "-" * 70)
        errors = sum(1 for f in self.findings if f.rule.severity == "error")
        warnings = sum(1 for f in self.findings if f.rule.severity == "warning")
        lines.append(f"结果：{errors} 错误 / {warnings} 警告")
        if errors:
            lines.append("❌ CI 守卫失败（存在 error 级违规）")
        else:
            lines.append("✅ 全部铁律通过（无 error 级违规）")
        lines.append("-" * 70)
        return "\n".join(lines)

    def has_errors(self) -> bool:
        return any(f.rule.severity == "error" for f in self.findings)

    @staticmethod
    def _snippet(text: str, start: int, end: int, ctx: int = 40) -> str:
        lo = max(0, start - ctx)
        hi = min(len(text), end + ctx)
        return text[lo:hi].replace("\n", " ").strip()


# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------


def main():
    parser = argparse.ArgumentParser(
        description="知味生态 A1.x 一致性守卫（8 条铁律）")
    parser.add_argument("--root", default=str(ROOT),
                        help="合集根目录（默认脚本上一级）")
    parser.add_argument("--fix", action="store_true",
                        help="自动修复可机械处理的违规（R1/R2/R3）")
    parser.add_argument("--coverage", action="store_true",
                        help="执行场景五层覆盖检查（R8）")
    parser.add_argument("--json", action="store_true",
                        help="以 JSON 格式输出（便于 CI 消费）")
    args = parser.parse_args()

    guard = Guard(Path(args.root).resolve())

    if args.fix:
        n = guard.apply_fixes()
        print(f"[fix] 已对 {n} 个文件应用机械修复（R1/R2/R3）", file=sys.stderr)

    guard.scan(fix=False)
    if args.coverage:
        guard.check_coverage()

    report = guard.report()
    if args.json:
        import json
        out = {
            "errors": [f._asdict() for f in guard.findings
                       if f.rule.severity == "error"],
            "warnings": [f._asdict() for f in guard.findings
                         if f.rule.severity == "warning"],
        }
        print(json.dumps(out, ensure_ascii=False, indent=2, default=str))
    else:
        print(report)

    sys.exit(1 if guard.has_errors() else 0)


if __name__ == "__main__":
    main()
