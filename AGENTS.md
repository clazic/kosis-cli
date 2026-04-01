# Module Context

이 디렉토리는 실제 KOSIS CLI 애플리케이션 루트다. `main.go`, `cmd/`, `internal/`, `docs/`가 함께 있으며, 설계서 요구사항을 코드와 운영 문서로 구현한다.

의존 관계:
- 엔트리포인트: `main.go`
- CLI 표면: `cmd/`
- 구현 세부: `internal/`
- 작업 운영 문서: `docs/`

# Tech Stack & Constraints

- Go module: `github.com/clazic/kosis-cli`
- CLI 프레임워크: Cobra
- 설정: Viper
- 출력 포맷: JSON, CSV, XLSX, SQLite, Parquet

Constraints:
- help 문자열은 설계서와 실제 파싱 동작이 일치해야 한다.
- `go test ./...`를 통과하지 못하는 변경은 미완으로 본다.
- API 실호출이 어려운 경우에도 help, 파싱, 파일 저장 경로는 로컬에서 재현 검증한다.

# Implementation Patterns

- 명령어 표면 수정은 먼저 `cmd/`에서 처리하고, 필요한 구현만 `internal/`로 내려보낸다.
- `--output` 동작은 stdout 경로와 파일 저장 경로를 명확히 분리한다.
- help 변경 시 예제 명령이 실제로 파싱되는지 반드시 확인한다.
- 설계서 예시와 Cobra 제약이 충돌하면 루트/명령 전처리 또는 안내 문구로 일관되게 해소한다.

# Testing Strategy

- 전체 테스트: `go test ./...`
- help 검증:
  - `go run . --help`
  - `go run . data --help`
  - `go run . quick --help`
  - `go run . indicator --help`
- 대표 파싱 검증:
  - `go run . data 101 DT_TEST -c1 11 -i T10 -p M`

# Local Golden Rules

Do:
- 변경이 `cmd/` 중심인지 `internal/` 중심인지 먼저 구분한다.
- 작업 후 `docs/pm-scorecard.md` 같은 운영 문서 반영 필요 여부를 확인한다.

Don't:
- 생성 산출물 바이너리(`kosis`, `kosis-cli`, `kosis_final`, `kosis_test`)를 기준 코드처럼 수정하지 않는다.
- `.DS_Store` 같은 비본질 파일을 건드리지 않는다.

# Context Map

- **[CLI 명령어 계층](./cmd/AGENTS.md)** — Cobra 명령, help, 플래그, root 진입 경로 수정 시.
- **[내부 구현 패키지](./internal/AGENTS.md)** — API, 출력, NLP, interactive, config 등 세부 로직 수정 시.
- **[운영 문서와 평가](./docs/AGENTS.md)** — PM/리뷰어 체크리스트와 점수표 수정 시.
