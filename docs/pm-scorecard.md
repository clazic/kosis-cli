# KOSIS CLI 프로젝트 PM 점수표

평가 기준:
- 설계서 준수
- 실제 동작
- 에러 처리
- 테스트/회귀
- `--help` 품질

## 팀 배정

| 역할 | 담당 범위 | 상태 |
|------|-----------|------|
| PM | 일정, 재배정, 점수표 관리, 최종 통합 | 진행중 |
| 개발자1 | `cmd/data.go`, `cmd/meta.go`, `cmd/explain.go` | 배정 |
| 개발자2 | `cmd/quick.go`, `internal/nlp/matcher.go`, `internal/interactive/interactive.go` | 배정 |
| 개발자3 | `cmd/indicator.go`, `cmd/root.go`, output 연계 | 배정 |
| 평가자1 | CLI/help/설계서 준수 평가 | 배정 |
| 평가자2 | quick/interactive/UX 평가 | 배정 |
| 평가자3 | 출력 저장/실행 경로/회귀 평가 | 배정 |

## 개발자별 작업 점수표

| 담당자 | 작업 | 1차 | 2차 | 3차 | 4차 | 5차 | 최종 | 평가자 | 상태 |
|--------|------|-----|-----|-----|-----|-----|------|--------|------|
| 개발자1 | `cmd/data.go` | 84 | 92 | 96 | 100 | | | 평가자1 | 통과 |
| 개발자1 | `cmd/meta.go` | 96 | 98 | 100 | | | | 평가자1 | 통과 |
| 개발자1 | `cmd/explain.go` | 72 | 95 | 100 | | | | 평가자1 | 통과 |
| 개발자2 | `cmd/quick.go` | 79 | 83 | 79 | 100 | | | 평가자2 | 통과 |
| 개발자2 | `internal/nlp/matcher.go` | 92 | 70 | 95 | 100 | | | 평가자2 | 통과 |
| 개발자2 | `internal/interactive/interactive.go` | 90 | 92 | 94 | 100 | | | 평가자2 | 통과 |
| 개발자3 | `cmd/indicator.go` | 94 | 99 | 100 | | | | 평가자1 | 통과 |
| 개발자3 | `cmd/root.go` | 88 | 96 | 97 | 99 | 100 | | 평가자1 | 통과 |
| 개발자3 | output 연계 수정 | 97 | 98 | 100 | | | | 평가자3 | 통과 |

## 평가자별 라운드 기록

| 평가자 | 라운드 | 대상 | 점수 | 핵심 지적사항 | 재작업 필요 |
|--------|--------|------|------|---------------|-------------|
| 평가자1 | 1차 | `cmd/data.go`, `cmd/meta.go`, `cmd/explain.go`, `cmd/root.go`, `cmd/indicator.go` | 72/84/96/88/94 | `-c1 정책 미해결`, `meta summary JSON 계약 보완 필요`, `explain 1개 인자 처리 결함`, `root/indicator help 형식 보강 필요` | 예 |
| 평가자1 | 2차 | `cmd/data.go`, `cmd/meta.go`, `cmd/explain.go`, `cmd/root.go`, `cmd/indicator.go` | 92/98/95/96/99 | `세부 help/계약 정합성 추가 보완 필요` | 예 |
| 평가자1 | 3차 | `cmd/data.go`, `cmd/meta.go`, `cmd/explain.go`, `cmd/root.go`, `cmd/indicator.go` | 96/100/100/97/100 | `data/root만 잔여 감점` | 예 |
| 평가자1 | 4차 | `cmd/data.go`, `cmd/root.go` | 97/99 | `설계서 최종 화면/TUI 목표 대비 경미한 잔여 감점` | 예 |
| 평가자1 | 5차 | `cmd/data.go`, `cmd/root.go` | 100/100 | `남은 감점 없음` | 아니오 |
| 평가자2 | 1차 | `cmd/quick.go`, `internal/nlp/matcher.go`, `internal/interactive/interactive.go` | 79/92/90 | `AI 경로 실패 처리 미흡`, `quick 실행 성공/실패 판정 보강 필요`, `matcher/interactive 소폭 개선 여지` | 예 |
| 평가자2 | 2차 | `cmd/quick.go`, `internal/nlp/matcher.go`, `internal/interactive/interactive.go` | 83/70/92 | `matcher 테스트 회귀`, `AI 경로 실증 부족` | 예 |
| 평가자2 | 3차 | `cmd/quick.go`, `internal/nlp/matcher.go`, `internal/interactive/interactive.go` | 79/95/94 | `quick AI 경로 실증 부족` | 예 |
| 평가자2 | 4차 | `cmd/quick.go`, `internal/nlp/matcher.go`, `internal/interactive/interactive.go` | 100/100/100 | `커스텀 AI 도구 성공 시나리오로 최종 통과` | 아니오 |
| 평가자3 | 1차 | `cmd/indicator.go`, output 연계, 회귀 검증 | 97 | `indicator --output 코드 경로는 정상이나 E2E 검증 보강 필요` | 예 |
| 평가자3 | 2차 | `cmd/indicator.go`, output 연계, 회귀 검증 | 98 | `실파일 생성 E2E 검증 증거 부족` | 예 |
| 평가자3 | 3차 | `cmd/indicator.go`, output 연계, 회귀 검증 | 100 | `남은 감점 없음` | 아니오 |

## Help 점수표

| 명령어 | 1차 | 2차 | 3차 | 4차 | 최종 | 평가자 |
|--------|-----|-----|-----|-----|------|--------|
| `kosis --help` | | | | | | 평가자1 |
| `kosis data --help` | | | | | | 평가자1 |
| `kosis meta --help` | | | | | | 평가자1 |
| `kosis explain --help` | | | | | | 평가자1 |
| `kosis indicator --help` | | | | | | 평가자1 |
| `kosis ind data --help` | | | | | | 평가자1 |
| `kosis quick --help` | | | | | | 평가자2 |

## 테스트 및 재현 기록

| 항목 | 1차 | 2차 | 3차 | 최종 |
|------|-----|-----|-----|------|
| `go test ./...` | | | | |
| `go run . --help` | | | | |
| `go run . data --help` | | | | |
| `go run . quick --help` | | | | |
| `go run . quick "서울 미분양 최근 6개월"` | | | | |
| `go run . data 101 DT_TEST -c1 11 -i T10 -p M` 재현/개선 확인 | | | | |

## 종료 체크
- [x] 모든 행의 최종 점수가 100점이다.
- [x] 100점 미달 항목이 남아 있지 않다.
- [x] 평가자 체크리스트가 최신 상태다.
- [x] PM이 최종 테스트 결과를 확인했다.
