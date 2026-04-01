# KOSIS CLI PM 점수표 v2 (재편성)

## 팀 구성
| 역할 | 모델 | 담당 범위 |
|------|------|----------|
| PM | opus | 업무배정, 점수관리, 최종빌드검증 |
| 개발자1 | haiku | API 인프라: api 패키지 테스트 추가, client/splitter 개선 |
| 개발자2 | haiku | 출력/포맷터: output 패키지 품질, test_output.go 정리 |
| 개발자3 | haiku | CLI 명령어: cmd 패키지 --help 정합성, 동작 검증 |
| 개발자4 | haiku | UX/편의: interactive, nlp, bookmark, history 테스트 추가 |
| 평가자1 | haiku | API/인프라 평가 (api, config, cache, splitter) |
| 평가자2 | haiku | 출력/CLI 평가 (output, cmd 전체) |
| 평가자3 | haiku | UX/테스트 평가 (interactive, nlp, bookmark, history, --help) |

## 작업 항목별 점수표

### 개발자1 담당
| 작업 | 1차 | 2차 | 3차 | 4차 | 최종 | 평가자 | 상태 |
|------|-----|-----|-----|-----|------|--------|------|
| internal/api/ 단위 테스트 추가 (types, data, meta, search 최소) | | | | | | 평가자1 | 미착수 |
| internal/api/client.go 429 재시도 로직 | | | | | | 평가자1 | 미착수 |
| internal/api/splitter.go 데드코드 제거 | | | | | | 평가자1 | 미착수 |
| internal/api/data.go 공통함수 추출 (중복 제거) | | | | | | 평가자1 | 미착수 |

### 개발자2 담당
| 작업 | 1차 | 2차 | 3차 | 4차 | 최종 | 평가자 | 상태 |
|------|-----|-----|-----|-----|------|--------|------|
| cmd/test_output.go 제거 또는 examples/ 이동 | | | | | | 평가자2 | 미착수 |
| internal/output/ 테스트 커버리지 보강 | | | | | | 평가자2 | 미착수 |
| internal/output/parquet.go 설계서 준수 확인 | | | | | | 평가자2 | 미착수 |
| internal/output/xlsx.go 설계서 준수 확인 | | | | | | 평가자2 | 미착수 |

### 개발자3 담당
| 작업 | 1차 | 2차 | 3차 | 4차 | 최종 | 평가자 | 상태 |
|------|-----|-----|-----|-----|------|--------|------|
| cmd/ 전체 --help 설계서 3.0.1 정합성 최종 확인 | | | | | | 평가자2 | 미착수 |
| cmd/quick.go 규칙기반 실행 경로 동작 확인 | | | | | | 평가자2 | 미착수 |
| cmd/data.go 대화형 모드 동작 확인 | | | | | | 평가자2 | 미착수 |
| cmd/config.go 전체 하위명령어 동작 확인 | | | | | | 평가자2 | 미착수 |

### 개발자4 담당
| 작업 | 1차 | 2차 | 3차 | 4차 | 최종 | 평가자 | 상태 |
|------|-----|-----|-----|-----|------|--------|------|
| internal/bookmark/ 단위 테스트 추가 | | | | | | 평가자3 | 미착수 |
| internal/history/ 단위 테스트 추가 | | | | | | 평가자3 | 미착수 |
| internal/interactive/ 품질 개선 | | | | | | 평가자3 | 미착수 |
| internal/nlp/ 매칭 정확도 개선 | | | | | | 평가자3 | 미착수 |

## --help 설계서 정합성 점수

| 명령어 | 1차 | 2차 | 3차 | 최종 | 평가자 | 상태 |
|--------|-----|-----|-----|------|--------|------|
| kosis --help | | | | | 평가자3 | 미착수 |
| kosis search --help | | | | | 평가자3 | 미착수 |
| kosis meta --help | | | | | 평가자3 | 미착수 |
| kosis data --help | | | | | 평가자3 | 미착수 |
| kosis list --help | | | | | 평가자3 | 미착수 |
| kosis explain --help | | | | | 평가자3 | 미착수 |
| kosis indicator --help | | | | | 평가자3 | 미착수 |
| kosis ind search --help | | | | | 평가자3 | 미착수 |
| kosis ind info --help | | | | | 평가자3 | 미착수 |
| kosis ind data --help | | | | | 평가자3 | 미착수 |
| kosis ind list --help | | | | | 평가자3 | 미착수 |
| kosis bulk --help | | | | | 평가자3 | 미착수 |
| kosis quick --help | | | | | 평가자3 | 미착수 |
| kosis config --help | | | | | 평가자3 | 미착수 |
| kosis bookmark --help | | | | | 평가자3 | 미착수 |
| kosis history --help | | | | | 평가자3 | 미착수 |

## 빌드/테스트 기록

| 항목 | 결과 | 날짜 | 비고 |
|------|------|------|------|
| go build ./... | PASS | 2026-03-31 | |
| go vet ./... | PASS | 2026-03-31 | |
| go test ./... | PASS | 2026-03-31 | api/bookmark/history 테스트 없음 |

## 종료 조건
- [ ] 모든 작업 항목 100점
- [ ] 모든 --help 100점
- [ ] go build + go vet + go test 전체 PASS
- [ ] PM이 직접 빌드하여 설계서 대비 최종 확인
