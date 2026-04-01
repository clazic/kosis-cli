# Module Context

이 디렉토리는 구현 코드가 아니라 작업 운영 문서 영역이다. PM 체크리스트, 점수표, 평가자 체크리스트를 유지하며, 병렬 작업과 재평가 루프의 상태를 기록한다.

# Tech Stack & Constraints

- Markdown only
- 상태 기록 문서

Constraints:
- 점수는 실제 평가 결과만 기록한다.
- 미통과 항목은 상태를 `재작업`으로 유지한다.
- 모든 항목이 100점이 되면 종료 체크를 완료 상태로 바꾼다.

# Implementation Patterns

- `pm-checklist.md`: 작업 계획과 반복 루프 관리
- `pm-scorecard.md`: 파일별 점수, 라운드별 결과, 종료 체크
- `reviewer*.md`: 평가 기준과 점수 기록

# Testing Strategy

- 문서 변경 후 링크와 파일명이 실제 경로와 맞는지 확인한다.
- 코드 라운드가 끝나면 점수표와 평가 문서가 최신 상태인지 대조한다.

# Local Golden Rules

Do:
- PM이 직접 배정, 점수, 종료 상태를 최신화한다.
- 평가자가 남긴 감점 사유를 다음 라운드 지시문에 반영한다.

Don't:
- 추정 점수나 미확인 결과를 기록하지 않는다.
- 코드 상태와 다른 완료 표시를 남기지 않는다.
