# 평가자3 체크리스트

담당:
- `cmd/indicator.go`
- output 저장 연계 동작
- 실제 실행 경로 및 회귀 테스트

평가 원칙:
- 문서화된 `--output`은 실제 파일을 생성해야 한다.
- stdout 포맷 전환과 파일 저장을 혼동하면 감점한다.
- 회귀 테스트와 대표 재현 명령이 통과해야 100점이다.
- 설계서/`skills/kosis-cli/SKILL.md`에 나온 사용자 명령 예시를 우선 검증한다.

평가 기준(100점):
- 40점: `indicator data --output` 저장 경로 정확성
- 20점: root/help 문구와 실제 구현 정합성
- 40점: 회귀 검증(테스트 + 대표 실행 시나리오)

감점 규칙:
- `-o/--output`이 파일을 만들지 못하면 해당 라운드는 최대 59점
- help가 실제 동작을 과장하면 항목당 -10점
- 회귀 명령 1개 실패당 -10점

## A. indicator data
- [ ] `-o/--output`이 실제 파일 저장으로 이어진다.
- [ ] 출력 확장자에 따라 포맷이 올바르게 선택된다.
- [ ] stdout 전용 경로와 파일 저장 경로가 명확히 분리된다.
- [ ] 출력 파일 지정 시 stdout에 테이블/JSON 본문이 섞여 출력되지 않는다.
- [ ] 파일 저장 성공/실패 메시지가 사용자에게 명확하다.
- [ ] `.csv/.json/.xlsx/.db/.parquet` 중 지원 범위가 help와 일치한다.

검증 명령(예시):
- `kosis ind d "GDP" -o /tmp/gdp.json`
- `kosis ind d "GDP" -o /tmp/gdp.csv`
- `kosis ind d "GDP" -f json` (stdout 전용 경로 확인)

## B. root / output 연계
- [ ] 상위 명령 설명이 실제 동작과 맞다.
- [ ] 구현 상태와 help 문구가 충돌하지 않는다.
- [ ] `kosis --help`와 `kosis indicator --help`에 미구현 기능이 완료된 것처럼 표기되지 않는다.
- [ ] output 관련 안내가 data/indicator에서 동일한 사용자 기대를 준다.

검증 명령(예시):
- `kosis --help`
- `kosis indicator --help`
- `kosis ind data --help`

## C. 회귀 검증
- [ ] `go test ./...` 통과
- [ ] 주요 help 명령 실행 확인
- [ ] quick/data/indicator 대표 경로 재현 확인
- [ ] 수정이 다른 명령을 깨지 않았는지 점검
- [ ] 실패 시 에러 메시지가 행동 가능한 문장인지 점검

회귀 기준 명령(최소):
- `go test ./...`
- `go run . --help`
- `go run . indicator --help`
- `go run . ind data --help`
- `go run . quick --help`
- `go run . data --help`

## 점수 기록

| 대상 | 1차 | 2차 | 3차 | 4차 | 최종 | 코멘트 |
|------|-----|-----|-----|-----|------|--------|
| `cmd/indicator.go` | | | | | | |
| output 연계 | | | | | | |
| 회귀 검증 | | | | | | |

## 라운드 평가 로그

| 라운드 | 총점(100) | Blocker 여부 | 주요 지적사항 | 재작업 요청 |
|--------|-----------|--------------|---------------|-------------|
| 1차 | | | | |
| 2차 | | | | |
| 3차 | | | | |
| 4차 | | | | |
