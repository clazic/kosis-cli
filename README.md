# kosis-cli

KOSIS(국가통계포털) OpenAPI CLI/TUI 도구 — 한국 통계 데이터를 터미널에서 검색·조회·시각화합니다.

---

## 설치

### npm

```bash
npm install -g kosis
```

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/clazic/kosis-cli/master/skills/kosis/scripts/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/clazic/kosis-cli/master/skills/kosis/scripts/install.ps1 | iex
```

### Claude 플러그인 (Claude Code)

```
/plugin install kosis@clazic
```

> 모든 설치 방법은 sudo/관리자 권한 없이 user scope에 설치됩니다.

---

## API 키 설정

KOSIS OpenAPI 키가 필요합니다. [https://kosis.kr/openapi/](https://kosis.kr/openapi/) 에서 발급하세요.

```bash
# 대화형 설정 (권장)
kosis config setup

# 직접 입력
kosis config set-key <API_KEY>

# 환경변수
export KOSIS_API_KEY="<API_KEY>"
```

---

## 빠른 시작

```bash
# 통계표 검색
kosis s "미분양"

# 메타 확인 (분류코드, 항목코드, 수록주기)
kosis m 116 DT_MLTM_2086

# 데이터 조회
kosis d 116 DT_MLTM_2086 -c1 ALL -c2 ALL -i ALL -p Y -l 5

# 자연어 조회
kosis q "서울 미분양 최근 5년"

# 차트 생성
kosis d 101 DT_1IN1502 -c1 00 -i T100 -p Y -l 10 --chart line --chart-format html --open
```

---

## 주요 명령어

| 명령어 | 설명 |
|--------|------|
| `kosis s <키워드>` | 통계표 검색 |
| `kosis m <ORG> <TBL>` | 통계표 메타 확인 |
| `kosis d <ORG> <TBL>` | 통계 데이터 조회 |
| `kosis q <요청>` | 자연어 조회 |
| `kosis chart` | 차트 시각화 |
| `kosis` | TUI 대시보드 |

자세한 사용법은 [skills/kosis/SKILL.md](skills/kosis/SKILL.md) 또는 `kosis --help` 참조.

---

## 라이선스

MIT
