# kosis-cli

KOSIS(국가통계포털) OpenAPI CLI/TUI 도구 — 한국 통계 데이터를 터미널에서 검색·조회·시각화합니다.

```bash
kosis s "미분양"                              # 통계표 검색
kosis m 116 DT_MLTM_2086                     # 메타 확인 (분류코드, 수록주기)
kosis d 116 DT_MLTM_2086 -c1 ALL -i ALL -p Y -l 5   # 데이터 조회
kosis q "서울 미분양 최근 5년"                # 자연어 한 줄 조회
```

---

## 설치

> 모든 방법은 **sudo/관리자 권한 없이** user scope에 설치됩니다.

### 방법 1: Claude 플러그인 (Claude Code 사용자 권장)

Claude Code의 채팅창에서:

```
/plugin install kosis@clazic
```

설치 완료 후 Claude가 자동으로 `kosis` 스킬을 인식합니다.
바이너리는 첫 번째 `kosis` 명령 실행 시 자동으로 다운로드됩니다.

> 마켓플레이스 등록이 안 된 경우 먼저 추가:
> ```
> /plugin marketplace add clazic/kosis-cli
> /plugin install kosis@clazic
> ```

---

### 방법 2: npm

Node.js 14 이상이 설치된 환경:

```bash
npm install -g @clazic/kosis
```

설치 시 자동으로:
- `~/.claude/skills/kosis/` 에 스킬 파일 설치 (SKILL.md, docs, templates)
- `~/.codex/skills/kosis/` 에 동일하게 설치
- 해당 OS 바이너리 다운로드 및 배치
- macOS/Linux: `~/.local/bin/kosis` symlink 생성

---

### 방법 3: macOS / Linux (curl)

```bash
curl -fsSL https://raw.githubusercontent.com/clazic/kosis-cli/master/skills/kosis/scripts/install.sh | sh
```

설치 위치:
- 스킬 파일: `~/.claude/skills/kosis/`, `~/.codex/skills/kosis/`
- 바이너리: `~/.claude/skills/kosis/apps/kosis`
- PATH 등록: `~/.local/bin/kosis` symlink

PATH가 등록되지 않은 경우 다음을 `~/.zshrc` 또는 `~/.bashrc`에 추가:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

특정 버전을 설치하려면:
```bash
KOSIS_VERSION=v0.3.0 curl -fsSL https://raw.githubusercontent.com/clazic/kosis-cli/master/skills/kosis/scripts/install.sh | sh
```

---

### 방법 4: Windows (PowerShell)

PowerShell에서:

```powershell
irm https://raw.githubusercontent.com/clazic/kosis-cli/master/skills/kosis/scripts/install.ps1 | iex
```

설치 위치:
- 스킬 파일: `~\.claude\skills\kosis\`, `~\.codex\skills\kosis\`
- 바이너리: `~\.claude\skills\kosis\apps\kosis.exe`
- PATH: 별도 등록 없이 `kosis.cmd` shim이 직접 호출

> 한글이 깨지는 경우 터미널에서 `chcp 65001` 실행

특정 버전 설치:
```powershell
irm https://raw.githubusercontent.com/clazic/kosis-cli/master/skills/kosis/scripts/install.ps1 | iex -Version v0.3.0
```

---

## API 키 설정

KOSIS OpenAPI 키가 필요합니다. [https://kosis.kr/openapi/](https://kosis.kr/openapi/) 에서 무료 발급.

```bash
# 대화형 설정 (권장) — 키 입력 후 자동 검증
kosis config setup

# 직접 입력
kosis config set-key <API_KEY>

# 환경변수 (CI/서버 환경)
export KOSIS_API_KEY="<API_KEY>"   # macOS/Linux
$env:KOSIS_API_KEY = "<API_KEY>"   # Windows PowerShell
```

---

## 빠른 시작

```bash
# 1. 통계표 검색
kosis s "미분양"

# 2. 메타 확인 (분류코드, 항목코드, 수록주기 확인 필수)
kosis m 116 DT_MLTM_2086

# 3. 데이터 조회
kosis d 116 DT_MLTM_2086 -c1 ALL -c2 ALL -i ALL -p Y -l 5

# 자연어 조회
kosis q "서울 미분양 최근 5년"

# 차트 생성 (HTML, 브라우저에서 열기)
kosis d 101 DT_1IN1502 -c1 00 -i T100 -p Y -l 10 --chart line --chart-format html --open

# TUI 대시보드
kosis
```

---

## 주요 명령어

| 명령어 | 별칭 | 설명 |
|--------|------|------|
| `kosis search <키워드>` | `s` | 통계표 검색 |
| `kosis meta <ORG> <TBL>` | `m` | 통계표 메타 확인 (분류/항목/주기) |
| `kosis data <ORG> <TBL>` | `d` | 통계 데이터 조회 |
| `kosis quick <요청>` | `q` | 자연어 조회 |
| `kosis chart` | | 차트 시각화 (파이프/파일 입력) |
| `kosis list` | `ls` | 통계목록 탐색 |
| `kosis explain <ORG> <TBL>` | `ex` | 통계표 설명 |
| `kosis config setup` | | 대화형 API 키 설정 |
| `kosis bookmark` | `bm` | 즐겨찾기 관리 |
| `kosis history` | `hi` | 조회 이력 |
| `kosis` | | TUI 대시보드 |

---

## 출력 형식

```bash
kosis d ... -f table    # 터미널 테이블 (기본)
kosis d ... -f md       # Markdown
kosis d ... -f json     # JSON
kosis d ... -f csv      # CSV
kosis d ... -o data.xlsx   # Excel 저장
kosis d ... -o data.db     # SQLite 저장
```

---

## 지원 플랫폼

| OS | 아키텍처 |
|----|---------|
| macOS | arm64 (Apple Silicon), amd64 (Intel) |
| Linux | amd64, arm64 |
| Windows | amd64 |

---

## 관련 링크

- 스킬 가이드: [skills/kosis/SKILL.md](skills/kosis/SKILL.md)
- 상세 문서: [skills/kosis/docs/](skills/kosis/docs/)
- 키 발급: [https://kosis.kr/openapi/](https://kosis.kr/openapi/)
- 이슈/피드백: [GitHub Issues](https://github.com/clazic/kosis-cli/issues)

---

## 라이선스

MIT
