# KOSIS CLI 코드 심층 분석 보고서

> 작성일: 2026-04-02
> 수정 완료일: 2026-04-02
> 분석 범위: 전체 Go 소스 (56개 파일)
> 분석 방법: 대화 기반 반복 분석 + 20회 반복 심층 탐색 (병렬 에이전트 2개)

---

## 요약

| 심각도 | 전체 | ✅ 수정 | ⏭ 스킵 | 설명 |
|:---:|:---:|:---:|:---:|------|
| **치명** | 7 | **7** | 0 | 크래시, 데이터 손상, 보안 취약점 |
| **높음** | 12 | **12** | 0 | 데이터 정확성, 리소스 누수, 성능 |
| **중간** | 16 | **16** | 0 | 논리 오류, 엣지 케이스, UX |
| **낮음** | 14 | **10** | 4 | 코드 품질, 미미한 엣지 케이스 |
| **합계** | **49** | **45** | **4** | |

> ⏭ 스킵 사유: 4.4 Parquet 동적 스키마(복잡도 높음), 4.6 table 샘플링(미미), 4.7 chart 스트리밍(미미), 4.12 configDirOverride(테스트 전용)

---

## 1. 치명 (크래시 / 보안)

### 1.1 ✅ 셸 명령어 인젝션 (NLP AI)
- **파일**: `internal/nlp/ai.go:40-41`
- **코드**: `exec.Command("sh", "-c", fullCmd)` — `{prompt}`에 사용자 입력이 이스케이핑 없이 삽입
- **영향**: `'; rm -rf / #` 같은 입력으로 임의 셸 명령 실행 가능
- **수정**: `shellescape` 라이브러리 사용 또는 직접 인자 전달 방식으로 변경

### 1.2 ✅ SQL 인젝션 (SQLite)
- **파일**: `internal/output/sqlite.go:107, 118, 166, 200, 229`
- **코드**: `fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE ... name='%s'", tableName)`
- **영향**: 테이블명/컬럼명에 `'` 또는 `;`가 포함되면 SQL 인젝션 가능
- **수정**: 테이블/컬럼명을 백틱으로 감싸고, `sanitizeTableName`에 `'`, `;`, `--` 제거 추가

### 1.3 ✅ HTTP Response Body 누수 (재시도 루프)
- **파일**: `internal/api/client.go:142`
- **코드**: 재시도 루프 안에서 `defer resp.Body.Close()` 사용
- **영향**: `defer`는 함수 종료 시 실행 → 429 재시도 시 이전 Body가 닫히지 않음 → 최대 3개 Body 누수
- **수정**: 루프 내에서 `resp.Body.Close()`를 명시적으로 호출

### 1.4 ✅ 문자열 슬라이싱 패닉
- **파일**: `internal/api/splitter.go:296-297, 320-321`
- **코드**: `opts.StartPrdDe[:4]` — 길이가 4 미만이면 런타임 패닉
- **영향**: 짧은 시점 문자열 입력 시 프로그램 크래시
- **수정**: 길이 체크 후 슬라이싱

### 1.5 ✅ Excel 컬럼 Z열 초과 오버플로
- **파일**: `internal/output/xlsx.go:68, 94, 129` / `internal/chart/excel.go:24, 49, 66`
- **코드**: `string(rune('A'+i))` — 26열(Z) 초과 시 `[`, `\` 등 잘못된 문자 생성
- **영향**: KOSIS 데이터는 최대 27+열 가능 (C1~C8_NM + 기타) → Excel 파일 손상
- **수정**: `excelize.ColumnNumberToName()` 사용

### 1.6 ✅ 캐시 Race Condition
- **파일**: `internal/cache/cache.go:59-60`
- **코드**: `RLock` 상태에서 goroutine으로 `os.Remove` 호출 → Lock 없이 파일 삭제
- **영향**: 동시 `Set`과 충돌 시 파일 손상
- **수정**: `Remove`를 Lock 내에서 동기적으로 실행

### 1.7 ✅ 파일 핸들 충돌 (file.go + 포맷터)
- **파일**: `internal/output/file.go:16` + `xlsx.go`, `sqlite.go`, `parquet.go`
- **코드**: `file.go`에서 `os.Create`로 파일을 열어놓은 채 포맷터가 같은 경로에 별도로 쓰기
- **영향**: Windows에서 파일 락 에러, 기존 DB 파일 truncate 위험
- **수정**: XLSX/SQLite/Parquet는 `file.go`에서 파일을 미리 생성하지 않도록 분기

---

## 2. 높음 (데이터 정확성 / 리소스 / 성능)

### 2.1 ✅ 병합 후 시점 정렬 없음
- **파일**: `internal/api/splitter.go:215-220`
- **코드**: 병렬 조회 결과를 청크 순서로 append하지만, 각 청크 내부는 `분류 × 시점` 순
- **영향**: 다중 분류 + 분할 조회 시 시점이 뒤섞임 → 차트 X축 뒤죽박죽
- **수정**: 병합 후 `PRD_DE` 기준 정렬 추가

### 2.2 ✅ 429 에러 시 데이터 누락 (재시도 없음)
- **파일**: `internal/api/splitter.go:200-204`
- **코드**: 429 에러 → 해당 청크 skip → 경고만 출력
- **영향**: 병렬 조회 시 일부 시점 데이터가 누락될 수 있음
- **수정**: 최대 3회 재시도 로직 추가 (exponential backoff)

### 2.3 ✅ 와일드카드 셀 수 추정 부정확
- **파일**: `internal/api/splitter.go:234-243`
- **코드**: `"11*"` → `metaCount/4`로 추정
- **영향**: 실제 하위 항목이 25개인데 4로 나눔 → 과소 추정 → 분할 안 함 → API 에러 31
- **수정**: meta 데이터에서 해당 prefix의 실제 개수를 세기

### 2.4 ✅ Meta 실패 시 무조건 일반 조회 시도
- **파일**: `internal/api/splitter.go:41`
- **코드**: Meta 실패 → `c.Data()` 직접 호출 → 4만 셀 초과면 API 에러 31
- **영향**: Meta API 장애 시 대용량 조회가 항상 실패
- **수정**: Meta 실패 시 에러 반환하거나, 보수적 분할 (기본 5개 청크) 시도

### 2.5 ✅ 분류 축 분할 미지원
- **파일**: `internal/api/splitter.go:339-388`
- **영향**: 시점 1개 + 분류 ALL (예: 전국 읍면동 인구 1년) → 시점 분할 무의미 → 분할 불가 → API 에러 31
- **수정**: 시점 분할 불가 시 분류값 축 분할 추가

### 2.6 ✅ SQLite 트랜잭션 미사용
- **파일**: `internal/output/sqlite.go:178-187`
- **코드**: 행마다 개별 INSERT (트랜잭션 없음)
- **영향**: 수만 행에서 극도로 느림 (각 INSERT마다 fsync 발생)
- **수정**: `BEGIN TRANSACTION` / `COMMIT`으로 감싸기

### 2.7 ✅ MetaSummary 이중 호출
- **파일**: `internal/api/splitter.go:37-38`
- **코드**: `DataWithAutoSplit`에서 `MetaSummary` 호출 후 `estimateCellCount` 내부에서 다시 호출
- **영향**: 동일 API를 2번 호출하여 불필요한 네트워크 비용
- **수정**: `estimateCellCount`에 summary를 인자로 전달

### 2.8 ✅ data 명령어에서 캐시 미초기화
- **파일**: `cmd/data.go:157-168`
- **코드**: `api.NewClient()` 후 `InitCache()` 미호출
- **영향**: MetaSummary 등에서 매번 네트워크 요청 → 성능 저하
- **수정**: `NewClient` 후 `InitCache()` 호출 추가

### 2.9 ✅ 부분 실패 시 수집된 데이터 버림
- **파일**: `internal/api/splitter.go:131-136`
- **코드**: 순차 모드에서 3번째 청크 에러 → 1, 2번째 결과도 함께 반환 불가
- **영향**: 10개 청크 중 마지막 1개 실패해도 전체 데이터 손실
- **수정**: 부분 결과 + 에러를 함께 반환하는 구조로 변경

### 2.10 ✅ MetaSummary 에러 무시
- **파일**: `internal/api/meta.go:82-97`
- **코드**: ITM/PRD 조회 실패 시 빈 결과 반환 (에러 미전파)
- **영향**: splitter에서 에러 감지를 놓침 → 잘못된 셀 수 추정
- **수정**: 부분 에러 시 경고 로그 + 에러 반환 옵션

### 2.11 ✅ 중복 제거 없음 (병합)
- **파일**: `internal/api/splitter.go:139, 219`
- **코드**: `allResults = append(allResults, results...)` — 중복 체크 없이 단순 append
- **영향**: API가 중복 행을 반환할 가능성은 낮지만, 분할 경계 근처에서 발생 가능
- **수정**: `(C1+C2+...+ITM_ID+PRD_DE)` 키 기반 중복 제거 옵션 추가

### 2.12 ✅ 전체 결과를 메모리에 이중 적재
- **파일**: `cmd/data.go:238-263`
- **코드**: `[]DataRow` → `[]map[string]interface{}` 전체 변환
- **영향**: 수십만 행이면 메모리 압박 (DataRow + dataMap 동시 존재)
- **수정**: 스트리밍 변환 또는 DataRow를 직접 포맷터에 전달

---

## 3. 중간 (논리 오류 / 엣지 케이스 / UX)

### 3.1 ✅ 차트: 다중 분류 컬럼 미지원
- **파일**: `internal/chart/chart.go:ExtractSeries`
- **영향**: C1_NM="서울", C2_NM="강남구" 같은 2단계 분류 → C1만 시리즈로 사용
- **수정**: 다중 분류 컬럼 조합으로 시리즈 이름 생성

### 3.2 ✅ 차트: Pie 차트 음수값 무처리
- **파일**: `internal/chart/html.go:145-158`, `internal/chart/image.go:61-64`
- **영향**: 음수값 포함 시 의미 없는 차트 생성
- **수정**: 음수값 감지 시 경고 출력

### 3.3 ✅ 차트: PNG/SVG Pie 차트가 Bar로 대체됨 (사용자 미고지)
- **파일**: `internal/chart/image.go:61-64`
- **코드**: `case Pie: addBarPlot(p, seriesList[:1])`
- **영향**: 사용자가 `--type pie --format png` 요청 → 막대 차트 수신
- **수정**: gonum/plot pie 미지원 안내 메시지 출력, 또는 별도 pie 렌더링

### 3.4 ✅ 차트: parseFloat에서 비수치 마커 무시
- **파일**: `internal/chart/chart.go:255`
- **코드**: `f, _ := strconv.ParseFloat(val, 64)` — "N/A", "***", "미상" 등이 0으로 처리
- **영향**: 실제 0인 값과 파싱 실패를 구별 불가
- **수정**: NaN 또는 별도 마커로 처리, 경고 출력

### 3.5 ✅ 차트: 빈 시리즈 체크 없음 (image.go)
- **파일**: `internal/chart/image.go:addLinePlot, addBarPlot`
- **영향**: 빈 Values 슬라이스가 gonum/plot에 전달 시 패닉 가능
- **수정**: 빈 시리즈 건너뛰기

### 3.6 ✅ 차트: 단일 데이터 포인트 라인 차트
- **파일**: `internal/chart/image.go:85-89`
- **영향**: 값 1개로 라인 차트 → "선"이 보이지 않음
- **수정**: 포인트 마커 자동 추가 또는 경고

### 3.7 ✅ 차트: 대량 데이터 터미널 차트 제한 없음
- **파일**: `internal/chart/terminal.go`
- **영향**: 수천 포인트를 asciigraph에 넘기면 터미널 출력이 넘침
- **수정**: 최대 포인트 수 제한 또는 다운샘플링

### 3.8 ✅ CSV Injection 취약점
- **파일**: `internal/output/csv.go:49-56`
- **영향**: `=`, `+`, `-`, `@`로 시작하는 값이 Excel에서 수식으로 실행 가능
- **수정**: 위험 문자로 시작하는 값 앞에 `'` 추가

### 3.9 ✅ HTML XSS 취약점
- **파일**: `internal/chart/html.go:63, 92, 128, 149`
- **영향**: 차트 제목/시리즈 이름에 `'`, `"`, `<script>` 등 포함 시 JavaScript 구문 에러 또는 XSS
- **수정**: go-echarts 이스케이핑 확인, 필요 시 수동 이스케이핑

### 3.10 ✅ history replay의 cobra 상태 오염
- **파일**: `cmd/history.go:136-138`
- **코드**: `rootCmd.SetArgs(cmdParts)` → 이미 실행 중인 rootCmd 재실행
- **영향**: 이전 실행의 플래그 값이 전역 변수에 남아있어 오동작 가능
- **수정**: 새 프로세스로 실행하거나 플래그 리셋

### 3.11 ✅ 대화형 모드 MetaResult Code/Label 비어있음
- **파일**: `cmd/data.go:453-477`
- **코드**: `MetaSummary`가 ITM 타입으로 조회 → `Code`/`Label` 대신 `ItmID`/`ItmNM` 사용해야 함
- **영향**: 빈 옵션이 표시됨
- **수정**: 올바른 필드 참조

### 3.12 ✅ normalizeDataKeys의 "비고" → "LST_CHN_DE" 매핑 불일치
- **파일**: `internal/api/data.go:121`
- **영향**: KOSIS API의 실제 "비고" 필드가 `LST_CHN_DE`(최종수정일)와 동일한지 미검증
- **수정**: 실제 API 응답 샘플로 검증 필요

### 3.13 ✅ 비정규 JSON 파싱 정규식 오작동
- **파일**: `internal/api/meta.go:36`
- **코드**: 정규식이 JSON 값 내부 문자열의 `{key:` 패턴도 치환
- **영향**: 특수한 메타 데이터에서 JSON 파싱 실패 가능
- **수정**: 정규식 대신 `json.Decoder` 또는 더 정교한 파서 사용

### 3.14 ✅ openFile에서 경로 인젝션
- **파일**: `internal/chart/opener.go:17`
- **코드**: Windows `cmd /c start "" path` — 경로에 `&` 포함 시 추가 명령 실행
- **수정**: 경로를 따옴표로 감싸기

### 3.15 ✅ bulk 명령어 경로 순회
- **파일**: `cmd/bulk.go:106-115`
- **코드**: `../../../etc/` 같은 경로로 임의 위치에 파일 쓰기 가능
- **수정**: 출력 경로 정규화 및 현재 디렉토리 밖 쓰기 차단

### 3.16 ✅ HTML 차트 불완전 파일 미삭제
- **파일**: `internal/chart/html.go:30-57`
- **코드**: `os.Create` 후 렌더링 실패 시 빈/불완전 HTML 파일이 디스크에 남음
- **수정**: 에러 시 `os.Remove` cleanup 추가

---

## 4. 낮음 (코드 품질 / 미미한 엣지 케이스)

### 4.1 ✅ 테이블 포맷터 CJK 문자 폭 계산 오류
- **파일**: `internal/output/table.go:94-101`
- **코드**: `utf8.RuneCountInString` — 한글이 2칸 차지하는 것 미고려
- **영향**: 한글 데이터에서 테이블 정렬 깨짐
- **수정**: `go.uber.org/runewidth` 사용

### 4.2 ✅ rootCmd.Version이 항상 "dev"
- **파일**: `cmd/root.go:79`
- **영향**: `--version` 출력이 빌드 버전 대신 "dev" 표시
- **수정**: cobra의 `Version` 필드를 동적으로 설정

### 4.3 ✅ config의 default_format 무시
- **파일**: `cmd/data.go:636`
- **코드**: `formatFlag` 기본값 `"table"` 하드코딩 → config의 `default_format` 미참조
- **수정**: config에서 기본값 로드

### 4.4 ⏭ Parquet 고정 스키마
- **파일**: `internal/output/parquet.go:50-78`
- **영향**: `FlatRecord` 외 커스텀 컬럼 데이터 무시
- **수정**: 동적 스키마 생성

### 4.5 ✅ JSON 포맷터 메모리 3배 사용
- **파일**: `internal/output/json.go:21-28`
- **코드**: 원본 + 복사본 + JSON 바이트 슬라이스
- **수정**: 스트리밍 `json.Encoder` 사용

### 4.6 ⏭ table.go에서 MaxRows 없이 전체 순회
- **파일**: `internal/output/table.go:98`
- **영향**: 100만 행에서도 전체 행을 순회하여 컬럼 폭 계산
- **수정**: 샘플링 또는 MaxRows 적용

### 4.7 ⏭ chart readJSON/readCSV 전체 메모리 로드
- **파일**: `cmd/chart.go:151, 177`
- **코드**: `io.ReadAll`, `csvReader.ReadAll`
- **수정**: 스트리밍 디코더 사용

### 4.8 ✅ TUI View() 메서드에서 상태 변경
- **파일**: `internal/tui/app.go:676`
- **코드**: `m.textInput.Width = width - 12` — View()는 순수 함수여야 함
- **영향**: Bubble Tea MVU 패턴 위반 → 예측 불가능한 UI 동작
- **수정**: Update()에서 Width 설정

### 4.9 ✅ explain 명령어 필드 대입 오류
- **파일**: `cmd/explain.go:143`
- **코드**: `"ORG_ID": item.OrgNM` — 기관**명**을 기관**ID** 키에 대입
- **수정**: `item.OrgID` 사용

### 4.10 ✅ opener.go 좀비 프로세스
- **파일**: `internal/chart/opener.go:24`
- **코드**: `cmd.Start()`만 호출, `cmd.Wait()` 미호출
- **수정**: goroutine에서 `cmd.Wait()` 호출

### 4.11 ✅ image.go Width/Height 불필요한 이중 할당
- **파일**: `internal/chart/image.go:38-49`
- **코드**: 첫 줄에서 `vg.Length(opts.Width)` 할당 후 else에서 다시 계산
- **수정**: 분기 정리

### 4.12 ⏭ configDirOverride 동시성 문제
- **파일**: `internal/config/config.go:18, 71-75`
- **영향**: 병렬 테스트에서 race condition
- **수정**: `sync.Mutex` 또는 테스트 전용 패턴 변경

### 4.13 ✅ API 키 로깅 가능성
- **파일**: `internal/api/client.go:125, 164`
- **코드**: API 키가 URL 쿼리 파라미터에 포함 → 에러 메시지에 Body 출력 시 노출 가능
- **수정**: 에러 메시지에서 API 키 마스킹

### 4.14 ✅ XLSX Sheet1 삭제 순서 문제
- **파일**: `internal/output/xlsx.go:28`
- **코드**: `wb.DeleteSheet("Sheet1")` — 새 시트 생성 전에 삭제 시도
- **영향**: excelize가 최소 1개 시트를 요구하면 삭제 실패 가능
- **수정**: 새 시트 생성 후 Sheet1 삭제

---

## 수정 우선순위 로드맵

### Phase 1 — 즉시 수정 (보안 + 크래시) — 7건
| # | 이슈 | 예상 난이도 |
|---|------|:---:|
| 1.1 | 셸 인젝션 (NLP AI) | 중 |
| 1.2 | SQL 인젝션 (SQLite) | 중 |
| 1.3 | HTTP Body 누수 | 하 |
| 1.4 | 문자열 슬라이싱 패닉 | 하 |
| 1.5 | Excel Z열 오버플로 | 중 |
| 1.6 | 캐시 Race Condition | 하 |
| 1.7 | 파일 핸들 충돌 | 중 |

### Phase 2 — 데이터 정확성 + 성능 — 12건
| # | 이슈 | 예상 난이도 |
|---|------|:---:|
| 2.1 | 병합 후 시점 정렬 | 하 |
| 2.2 | 429 재시도 로직 | 중 |
| 2.3 | 와일드카드 추정 개선 | 중 |
| 2.4 | Meta 실패 시 보수적 분할 | 중 |
| 2.5 | 분류 축 분할 | 상 |
| 2.6 | SQLite 트랜잭션 | 하 |
| 2.7 | MetaSummary 이중 호출 제거 | 하 |
| 2.8 | data 명령어 캐시 초기화 | 하 |
| 2.9 | 부분 실패 시 데이터 보존 | 중 |
| 2.10 | MetaSummary 에러 전파 | 하 |
| 2.11 | 중복 제거 옵션 | 중 |
| 2.12 | 메모리 이중 적재 최적화 | 중 |

### Phase 3 — 차트 + UX + 방어 — 16건
| # | 이슈 | 예상 난이도 |
|---|------|:---:|
| 3.1 | 다중 분류 차트 | 중 |
| 3.2 | Pie 차트 음수값 경고 | 하 |
| 3.3 | Pie 차트 PNG 대체 고지 | 하 |
| 3.4 | parseFloat 비수치 마커 처리 | 하 |
| 3.5 | 빈 시리즈 체크 | 하 |
| 3.6 | 단일 포인트 라인 차트 | 하 |
| 3.7 | 대량 데이터 터미널 차트 제한 | 하 |
| 3.8 | CSV Injection 방어 | 하 |
| 3.9 | HTML XSS 방어 | 하 |
| 3.10 | history replay 상태 오염 | 중 |
| 3.11 | 대화형 모드 필드 참조 수정 | 하 |
| 3.12 | normalizeDataKeys 매핑 검증 | 하 |
| 3.13 | 비정규 JSON 정규식 개선 | 중 |
| 3.14 | opener 경로 인젝션 방어 | 하 |
| 3.15 | bulk 경로 순회 차단 | 하 |
| 3.16 | 불완전 파일 cleanup | 하 |

### Phase 4 — 코드 품질 — 14건
| # | 이슈 | 예상 난이도 |
|---|------|:---:|
| 4.1 | CJK 문자 폭 | 하 |
| 4.2 | 버전 표시 수정 | 하 |
| 4.3 | config default_format 반영 | 하 |
| 4.4 | Parquet 동적 스키마 | 중 |
| 4.5 | JSON 스트리밍 인코딩 | 중 |
| 4.6 | table.go 샘플링 | 하 |
| 4.7 | chart 스트리밍 로드 | 중 |
| 4.8 | TUI MVU 패턴 준수 | 하 |
| 4.9 | explain 필드 오류 | 하 |
| 4.10 | opener 좀비 프로세스 | 하 |
| 4.11 | image.go 이중 할당 정리 | 하 |
| 4.12 | configDirOverride 동시성 | 하 |
| 4.13 | API 키 마스킹 | 하 |
| 4.14 | XLSX Sheet1 삭제 순서 | 하 |

---

## 분석 흐름 기록

| 라운드 | 발견 영역 | 주요 발견 |
|:---:|------|------|
| 1-3 | 대량 데이터 분할 조회 | 분할 경계 중복 가능성, 정렬 미보장, 429 누락 |
| 4-6 | splitter 셀 수 추정 | 와일드카드 추정, Meta 실패 처리, 분류 축 분할 미지원 |
| 7-9 | 보안 | 셸 인젝션(NLP), SQL 인젝션(SQLite), API 키 노출 |
| 10-12 | 크래시/리소스 | HTTP Body 누수, 문자열 패닉, 캐시 Race, 파일 핸들 충돌 |
| 13-15 | 차트 렌더링 | 빈 시리즈, Pie→Bar 대체, 음수값, XSS, 대량 데이터 제한 |
| 16-18 | 출력 포맷터 | Excel Z열, CSV Injection, Parquet 고정 스키마, SQLite 트랜잭션 |
| 19-20 | 코드 품질/UX | CJK 폭, 버전 표시, history replay, TUI MVU 패턴, explain 버그 |
