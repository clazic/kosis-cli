# PM 업무 관리 체크리스트

평가 기준 문서:
- 설계서: `../../docs/superpowers/specs/2026-03-31-kosis-cli-design.md`
- 사용 가이드: `../../skills/kosis-cli/SKILL.md`

팀 구성:
- PM: Codex
- 개발자1: CLI/help/data 담당
- 개발자2: quick/interactive/nlp 담당
- 개발자3: indicator/root/output 경계 담당
- 평가자1: CLI/help/설계서 준수 평가
- 평가자2: quick/interactive/UX 평가
- 평가자3: 출력 저장/실행 경로/회귀 평가

운영 규칙:
- [ ] 작업 시작 전 설계서와 SKILL.md를 기준 문서로 다시 확인했다.
- [ ] 개발자 3명과 평가자 3명의 담당 범위를 겹치지 않게 분리했다.
- [ ] 각 평가자는 본인 체크리스트 Markdown으로 점검한다.
- [ ] 각 평가자는 점수를 `pm-scorecard.md`에 기록한다.
- [ ] 100점 미만 항목은 해당 개발자에게 재작업 지시한다.
- [ ] 재작업 후 같은 평가자에게 재평가를 요청한다.
- [ ] 모든 담당 항목이 100점이 될 때까지 반복한다.
- [ ] `--help` 출력은 설계서 형식까지 별도 확인한다.
- [ ] 최종 단계에서 `go test ./...`와 주요 CLI 재현 검증을 수행한다.

## 1. 문서 정비
- [ ] `pm-checklist.md` 현행화
- [ ] `pm-scorecard.md` 현행화
- [ ] `reviewer1-checklist.md` 현행화
- [ ] `reviewer2-checklist.md` 현행화
- [ ] `reviewer3-checklist.md` 신규 작성

## 2. 1차 작업 배정

### 개발자1
- [ ] `cmd/data.go`: 설계서와 실제 플래그/예제 불일치 수정
- [ ] `cmd/meta.go`: summary 출력과 `-f json` 계약 불일치 수정
- [ ] `cmd/explain.go`: 설계서 기준 대화형 경로 검토 및 보강
- [ ] 관련 help 문구와 예제 정합성 수정

### 개발자2
- [ ] `cmd/quick.go`: 규칙 기반 실행 경로 구현
- [ ] `cmd/quick.go`: AI 생성 명령 실제 실행 경로 구현
- [ ] `internal/nlp/matcher.go`: 현재 규칙 기반 매칭 요구사항 보강
- [ ] `internal/interactive/interactive.go`: quick/data UX 미비점 보강

### 개발자3
- [ ] `cmd/indicator.go`: `--output` 실제 파일 저장 구현
- [ ] `cmd/root.go`: root help/TUI 플레이스홀더 계약 정리
- [ ] 관련 help 출력과 상위 명령 설명 보강
- [ ] 필요 시 output 연계 코드 정리

## 3. 평가 라운드

### 평가자1
- [ ] CLI/help 설계서 준수 체크
- [ ] `data`, `meta`, `explain`, `root`, `indicator` help 점검
- [ ] 점수 기록

### 평가자2
- [ ] quick 규칙 기반/AI 실행/interactive UX 체크
- [ ] 자연어 예시 재현 체크
- [ ] 점수 기록

### 평가자3
- [ ] `--output` 파일 저장 동작 체크
- [ ] 실제 명령 실행 경로와 회귀 테스트 체크
- [ ] 점수 기록

## 4. 100점 루프
- [ ] 1차 평가 결과 회수
- [ ] 100점 미달 항목 재배정
- [ ] 2차 수정 완료
- [ ] 2차 평가 점수 기록
- [ ] 100점 미달 항목 재반복
- [ ] 최종 100점 확인

## 5. 종료 조건
- [ ] 담당 영역별 최종 점수 100점
- [ ] `pm-scorecard.md` 최종 점수 반영 완료
- [ ] 체크리스트 완료 상태 반영
- [ ] 테스트 및 재현 결과 최종 정리
