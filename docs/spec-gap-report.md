# 설계서 기준 구현 현황

기준 문서:
- `../../docs/superpowers/specs/2026-03-31-kosis-cli-design.md`

판정 기준:
- 구현됨: 코드와 실행 경로가 실제로 존재함
- 부분 구현: 코드나 명령은 있으나 설계서 수준 전체를 충족하지 못함
- 미구현: 설계서에 있으나 현재 코드/파일이 없음

검증 기준:
- `go build ./...` 통과
- `go test ./...` 통과
- `go run . --help` 확인

## 1. 전체 판단

- 구현됨:
  - KOSIS CLI 기본 골격
  - 주요 명령 대부분
  - API 클라이언트 주요 경로
  - 출력 포맷터
  - 설정/즐겨찾기/이력
  - quick 규칙 기반/AI 옵션
  - 크로스 플랫폼 바이너리 빌드용 `Makefile`
- 부분 구현:
  - root 무인자 경험
  - 대화형 모드 전반
  - `--help` 설계서 완전 일치
  - quick의 설계서 전체 자연어 범위
- 미구현:
  - 실제 TUI 대시보드
  - Bubble Tea/Bubbles/Lipgloss 기반 화면
  - `.goreleaser.yaml`
  - `cmd/completion.go` 전용 파일

핵심 결론:
- 이 프로젝트는 "CLI는 상당 부분 구현됨"
- 하지만 "설계서 전체 구현 완료"는 아님
- 특히 5장 TUI 대시보드 설계는 아직 구현되지 않음

## 2. 설계서 항목별 판정

### 2.1 12개 API 엔드포인트

- 구현됨
- 근거:
  - `internal/api/list.go`
  - `internal/api/data.go`
  - `internal/api/data_registered.go`
  - `internal/api/bigdata.go`
  - `internal/api/explain.go`
  - `internal/api/meta.go`
  - `internal/api/search.go`
  - `internal/api/indicator.go`
  - `internal/api/splitter.go`

### 2.2 CLI 명령 체계

- 구현됨
- 존재 명령:
  - `search`, `meta`, `data`, `list`, `explain`, `bulk`
  - `indicator`
  - `quick`
  - `config`
  - `bookmark`
  - `history`
  - Cobra 기본 `completion`
- 근거:
  - `cmd/search.go`
  - `cmd/meta.go`
  - `cmd/data.go`
  - `cmd/list.go`
  - `cmd/explain.go`
  - `cmd/bulk.go`
  - `cmd/indicator.go`
  - `cmd/quick.go`
  - `cmd/config.go`
  - `cmd/bookmark.go`
  - `cmd/history.go`

### 2.3 대화형 모드

- 부분 구현
- 구현된 것:
  - `search` 인자 없을 때 프롬프트
  - `meta` 인자 없을 때 검색 후 선택
  - `data` 인자 없을 때 단계형 입력
  - `explain` 인자 없을 때 대화형 진입
- 부족한 것:
  - 설계서가 의도한 대화형 UX는 사실상 구현되지 않았음
  - 현재 있는 것은 일부 명령에서 인자 누락 시 뜨는 단순 프롬프트일 뿐임
  - 설계서의 검색/선택/탐색/후속 액션 연결형 사용자 경험을 제공하지 못함
  - 사람용 대화형 모드가 완성됐다고 볼 수 없음
- 근거:
  - `cmd/search.go`
  - `cmd/meta.go`
  - `cmd/data.go`
  - `cmd/explain.go`
  - `internal/interactive/interactive.go`

### 2.4 quick 명령

- 부분 구현
- 구현된 것:
  - 규칙 기반 매칭
  - AI 도구 호출
  - 하위 `data/search` 실행 연결
- 부족한 것:
  - 설계서가 기대하는 자연어 커버리지 전체를 보장하긴 어려움
  - 외부 AI 도구 성공 여부는 사용자 환경에 의존
- 근거:
  - `cmd/quick.go`
  - `internal/nlp/matcher.go`
  - `internal/nlp/ai.go`

### 2.5 즐겨찾기 / 조회 이력 / 설정

- 구현됨
- 근거:
  - `cmd/bookmark.go`, `internal/bookmark/bookmark.go`
  - `cmd/history.go`, `internal/history/history.go`
  - `cmd/config.go`, `internal/config/config.go`

### 2.6 출력 포맷터

- 구현됨
- 지원 형식:
  - table
  - json
  - csv
  - xlsx
  - sqlite
  - parquet
- 근거:
  - `internal/output/formatter.go`
  - `internal/output/table.go`
  - `internal/output/json.go`
  - `internal/output/csv.go`
  - `internal/output/xlsx.go`
  - `internal/output/sqlite.go`
  - `internal/output/parquet.go`

### 2.7 자동 분할 조회 / 멀티 API 키

- 구현됨
- 근거:
  - `internal/api/splitter.go`
  - `internal/api/client.go`
  - `internal/config/config.go`

### 2.8 파이프라인 지원

- 부분 구현
- 구현된 것:
  - json/table/csv 출력
  - 일부 비TTY 고려
- 부족한 것:
  - 설계서가 기대하는 전체 stdout/TTY 정책을 완전 검증했다고 보기 어려움
- 근거:
  - `internal/output/table.go`
  - `internal/output/json.go`
  - `cmd/root.go`

### 2.9 자동완성

- 부분 구현
- 구현된 것:
  - Cobra 기본 `completion` 명령은 존재
- 미흡한 것:
  - 설계서 구조의 전용 `cmd/completion.go`는 없음
- 근거:
  - `go run . --help`에 `completion` 노출
  - `cmd/completion.go` 파일 없음

### 2.10 TUI 대시보드

- 미구현
- 현재 상태:
  - 실제 TUI 코드는 없음
  - `internal/tui/`에는 `AGENTS.md`만 있음
  - `kosis` 무인자 실행은 `cmd/root.go`의 임시 메뉴/빠른 시작 화면
- 설계서와의 차이:
  - Bubble Tea 기반 대시보드 아님
  - 패널 구성 없음
  - 키바인딩 없음
  - Lipgloss 스타일링 없음
- 근거:
  - `internal/tui/AGENTS.md`만 존재
  - `cmd/root.go`의 `runDashboard`

### 2.11 프로젝트 구조 요구

- 부분 구현
- 구현된 것:
  - `cmd/`, `internal/`, `docs/`, `main.go`, `Makefile`
- 미구현/차이:
  - `cmd/completion.go` 없음
  - `internal/tui` 실코드 없음
  - `.goreleaser.yaml` 없음
- 근거:
  - 현재 파일 트리

### 2.12 크로스 컴파일

- 부분 구현
- 구현된 것:
  - `Makefile`
  - `bin/mac/kosis`
  - `bin/linux/kosis`
  - `bin/windows/kosis.exe`
  - `bin/kosis` (mac 복사본)
- 미구현:
  - `.goreleaser.yaml`
- 근거:
  - `Makefile`
  - `bin/` 산출물

## 3. 지금 기준으로 안 된 것

- 설계서 수준의 대화형 UX
- 사람용 대화형 흐름 전반
- 실제 TUI 앱
- Bubble Tea/Bubbles/Lipgloss 적용
- `internal/tui` 실코드
- `internal/tui/panels` 실코드
- `.goreleaser.yaml`
- 설계서 예시와 1:1로 완전히 동일한 최종 help/UI

## 4. 지금 기준으로 된 것

- CLI 명령 체계 대부분
- API 엔드포인트 구현
- 설정/출력/quick/즐겨찾기/이력
- 자동분할
- 멀티 API 키
- 크로스 플랫폼 바이너리 빌드용 `Makefile`
- 전체 빌드/테스트 통과

## 5. 한 줄 결론

설계서 기준으로 보면:
- "CLI 중심 구현"은 많이 됨
- "사람용 대화형 UX와 TUI"는 아직 안 됨
