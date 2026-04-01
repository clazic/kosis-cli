# KOSIS CLI TUI 테스트 보고서 - 테스터2 (통계표 선택 → 메타 → 데이터 흐름)

**테스트 날짜**: 2026-04-01
**테스터**: TUI 테스터2
**포커스**: 선택 → 메타 로드 → 데이터 조회 흐름
**상태**: 10개 문제 발견

---

## 발견된 문제 목록

### 1. 검색 결과에서 Enter로 통계표 선택 시 메타가 자동 로드되는가

**상태**: ✅ 정상 작동

**분석**:
- `app.go:350-357`: KeyEnter 처리에서 `PanelSearch` 또는 `PanelMeta` 패널일 때 선택된 통계표로 메타 조회 커맨드 실행
- `doMeta()` 함수 호출로 메타 로드 시작
- 상태 메시지: "📋 메타 로딩 중..."

**코드 발췌** (app.go:350-357):
```go
case KeyEnter:
    if (m.activePanel == PanelSearch || m.activePanel == PanelMeta) && len(m.searchResults) > 0 && m.cursor < len(m.searchResults) {
        selected := m.searchResults[m.cursor]
        m.selectedTable = &TableInfo{OrgID: selected.OrgID, TblID: selected.TblID, TblNM: selected.TblNM}
        m.loading = true
        m.statusMsg = "📋 메타 로딩 중..."
        return m, m.doMeta(selected.OrgID, selected.TblID)
```

---

### 2. 메타 로드 후 자동으로 데이터 조회가 시작되는가

**상태**: ✅ 정상 작동

**분석**:
- `app.go:425-478`: `metaResultMsg` 처리에서 메타 로드 완료 후 자동으로 데이터 조회 시작
- `m.selectedTable != nil` 확인 후 기본 파라미터 설정하여 `doData()` 호출
- 자동 파라미터: ALL, ALL, 자동 주기 감지, 최근 5개

**코드 발췌** (app.go:433-474):
```go
if m.selectedTable != nil {
    // 기본 파라미터: ALL, ALL, 자동 주기, 최근 5개
    // ...
    firstItem := "T10"  // 기본값
    if len(m.metaInfo.Items) > 0 {
        firstItem = m.metaInfo.Items[0].Code
    }
    // ...
    m.loading = true
    m.activePanel = PanelResult
    m.statusMsg = "📊 데이터 조회 중... (ALL, ALL, 최근 5개)"
    return m, m.doData(m.selectedTable.OrgID, m.selectedTable.TblID, opts)
```

---

### 3. 분류 그룹 수(NumClassGroups) 계산이 정확한가 (1개, 2개, 3개 분류)

**상태**: ⚠️ 잠재적 문제 - 분류가 0개일 때 강제로 1로 설정하나 로직이 불명확함

**분석**:
- `app.go:201-211`: `doMeta()` 함수에서 분류 그룹 수를 계산
- `objIDSet` 맵에 `ObjID`를 수집하여 고유값 개수 계산
- **문제**: 메타 API 응답이 `ObjID` 필드를 항상 포함하지 않을 수 있음
- 분류가 없으면 `NumClassGroups == 0`이 되고, 이를 1로 강제 설정

**코드 발췌** (app.go:201-211):
```go
objIDSet := map[string]bool{}
for _, c := range summary.Classifications {
    if c.ObjID != "" {
        objIDSet[c.ObjID] = true
    }
}
meta.NumClassGroups = len(objIDSet)
if meta.NumClassGroups == 0 {
    meta.NumClassGroups = 1  // 강제로 1로 설정
}
```

**발견된 문제**:
- `summary.Classifications`이 비어있을 경우: 계산되지 않고 1로 설정
- 메타 응답 구조가 명확하지 않으면 정확한 분류 그룹 수를 계산할 수 없음
- 설계서에 분류 그룹 계산 방식이 정의되어 있는지 확인 필요

**추천**:
- 메타 응답 구조 명확화
- 테스트: 분류 1개, 2개, 3개인 통계표로 정확한 계산 확인

---

### 4. 분류 그룹이 0일 때 크래시 방지

**상태**: ⚠️ 부분 보호됨 - 강제 1로 설정하지만 다른 곳에서 크래시 가능

**분석**:
- `app.go:462-470`: `NumClassGroups` 기반으로 Class1~Class8 설정
- `NumClassGroups == 0`이면 어떤 분류도 설정 안 함
- **문제**: API에 분류를 전송하지 않으면 실패할 수 있음

**코드 발췌** (app.go:462-470):
```go
n := m.metaInfo.NumClassGroups
if n >= 1 { opts.Class1 = "ALL" }
if n >= 2 { opts.Class2 = "ALL" }
// ...
```

**발견된 문제**:
- `NumClassGroups`가 0으로 설정된 메타일 때, 어떤 분류도 `ALL`로 설정 안 됨
- 이후 `Data()` 호출 시 필수 분류 누락으로 API 에러 가능
- 에러 처리: `app.go:480-488`에서만 에러 메시지 표시

**권장 사항**:
- 메타 로드 후 `NumClassGroups > 0` 확인
- `NumClassGroups == 0`이면 재시도 또는 사용자 안내 필요

---

### 5. 메타 API 실패 시 에러 표시

**상태**: ✅ 정상 작동

**분석**:
- `app.go:425-431`: `metaResultMsg` 에러 처리
- `msg.err != nil`일 때 `m.err = msg.err` 설정하고 상태 메시지 표시
- 상태 메시지: "❌ 메타 조회 오류: ..."

**코드 발췌** (app.go:425-431):
```go
case metaResultMsg:
    m.loading = false
    if msg.err != nil {
        m.err = msg.err
        m.statusMsg = fmt.Sprintf("❌ 메타 조회 오류: %v", msg.err)
    } else {
```

---

### 6. 데이터 API 실패 시 에러 표시 (API 오류 코드별)

**상태**: ⚠️ 부분 작동 - 에러는 표시되지만 API 오류 코드별 구분 없음

**분석**:
- `app.go:480-488`: `dataResultMsg` 에러 처리
- 모든 에러를 동일하게 "❌ 데이터 조회 오류: ..." 로 표시
- API 오류 코드(예: 10, 20, 99 등) 구분 없음
- `client.go:168`: API 응답의 `err` 필드를 감지하면 `API 오류 [ERR_CODE]: MESSAGE` 형식 반환

**코드 발췌** (app.go:480-488):
```go
case dataResultMsg:
    m.loading = false
    if msg.err != nil {
        m.err = msg.err
        m.statusMsg = fmt.Sprintf("❌ 데이터 조회 오류: %v", msg.err)
    } else {
        m.resultData = msg.data
        m.statusMsg = fmt.Sprintf("✓ %d건 조회 완료", len(msg.data))
    }
```

**발견된 문제**:
- 사용자가 API 오류 코드를 확인할 수 없음
- 통계표 선택 오류(OBJ_ID 없음), 필수 파라미터 누락 등을 구분할 수 없음
- TUI에서는 제너릭 에러만 표시 (API 응답의 `err`, `errMsg` 구분 안 함)

**권장 사항**:
- 에러 메시지를 더 자세히 표시 (API 오류 코드 포함)
- 통계표 메타 없음 등 특정 케이스 처리

---

### 7. 결과 0건일 때 UI

**상태**: ✅ 정상 작동

**분석**:
- `app.go:851-852`: `len(m.resultData) == 0`일 때 "📋 결과 데이터\n───────────────────\n(조회를 실행하세요)" 표시
- 데이터 조회 완료 후 0건일 때도 동일한 메시지로 표시 (구분 안 함)

**코드 발췌** (app.go:851-852):
```go
if len(m.resultData) == 0 {
    return sectionStyle.Render("📋 결과 데이터\n───────────────────\n(조회를 실행하세요)")
}
```

**발견된 문제**:
- 조회를 실행하지 않은 상태와 조회 결과 0건을 구분하지 않음
- 사용자가 조회 결과 0건인지 조회 미실행인지 모호함
- 상태 메시지 (`statusMsg`)에는 "✓ 0건 조회 완료"라고 표시되지만, 결과 영역에서는 모호함

**권장 사항**:
- 플래그 추가: `hasQueryRun` bool
- 조회 완료 후 0건 시 "✓ 조회 완료 (결과 0건)" 등으로 명확히 표시

---

### 8. 결과 데이터 5행 이상일 때 나머지가 "..."로 표시되는가

**상태**: ✅ 정상 작동

**분석**:
- `app.go:861-865`: 최대 5개 행만 표시하고 초과 시 "..." 추가
- `renderResultSection()` 함수에서 구현

**코드 발췌** (app.go:861-865):
```go
for i, row := range m.resultData {
    if i >= 5 {  // 최대 5개 행만 표시
        lines = append(lines, "...")
        break
    }
```

---

### 9. 결과 데이터의 필드가 비어있을 때 (C1_NM이 empty)

**상태**: ⚠️ 잠재적 문제 - 빈 필드를 빈 문자열로 표시함

**분석**:
- `app.go:866-871`: 필드 값을 `truncateText()`로 처리하지만, 비어있으면 공백만 반환
- `data.go:256-262`: API 응답의 필드를 직접 매핑 (`r.C1NM`, `r.ItmNM` 등)
- 필드가 없으면 JSON Unmarshal에서 빈 문자열 할당

**코드 발췌** (app.go:866-871):
```go
c1nm := truncateText(row.Fields["C1_NM"], 10)
itmnm := truncateText(row.Fields["ITM_NM"], 8)
prdde := truncateText(row.Fields["PRD_DE"], 8)
dt := truncateText(row.Fields["DT"], 10)
unit := truncateText(row.Fields["UNIT_NM"], 4)
```

**발견된 문제**:
- `row.Fields["C1_NM"]`이 빈 문자열이면 `truncateText("", 10)` 호출
- `truncateText()` (app.go:905-935)는 빈 문자열에 대해 패딩만 반환
- 테이블 행이 많은 공백으로 차게 됨
- 사용자가 데이터가 없는지 데이터가 정말 비어있는지 구분 불가

**권장 사항**:
- 빈 필드에 "-" 또는 "[없음]" 등 표시
- 또는 경고 메시지 추가

---

### 10. doMeta/doData에서 client가 nil일 때 처리

**상태**: ✅ 정상 작동

**분석**:
- `app.go:191-195` (doMeta): `if m.client == nil` 확인 후 에러 메시지 반환
- `app.go:244-248` (doData): `if m.client == nil` 확인 후 에러 메시지 반환
- `app.go:172-174` (doSearch): `if m.client == nil` 확인 후 에러 메시지 반환

**코드 발췌** (app.go:191-195):
```go
func (m Model) doMeta(orgID, tblID string) tea.Cmd {
    return func() tea.Msg {
        if m.client == nil {
            return metaResultMsg{err: fmt.Errorf("API 클라이언트가 초기화되지 않았습니다")}
        }
```

---

### 11. PeriodInfo 파싱: "prdSe=Y 1950~2100" vs "prdSe=M 202001~202612" vs 빈 값

**상태**: ⚠️ 부분 작동 - 파싱 로직이 명확하지 않고 오류 가능성 높음

**분석**:

#### 메타 로드 후 PeriodInfo 생성 (app.go:235-238):
```go
for _, p := range summary.Periods {
    prdCode := convertPrdSe(p.PrdSe)
    meta.PeriodInfo = "prdSe=" + prdCode + " " + p.StrtPrdDe + "~" + p.EndPrdDe
}
```

#### PeriodInfo 파싱 (app.go:437-444):
```go
period := "Y"
if m.metaInfo != nil && m.metaInfo.PeriodInfo != "" {
    parts := strings.Fields(m.metaInfo.PeriodInfo)
    for _, p := range parts {
        if strings.HasPrefix(p, "prdSe=") {
            period = strings.TrimPrefix(p, "prdSe=")
            break
        }
    }
}
```

**발견된 문제**:

1. **PeriodInfo가 여러 개일 때**: 반복문에서 마지막 주기만 사용됨
   - 년도(Y)와 월(M) 데이터가 모두 있으면 마지막만 사용
   - 예: "prdSe=Y 1950~2100" 이후 "prdSe=M 202001~202612"가 있으면 M만 사용

2. **빈 값**: `strings.Fields()`는 공백 기준 분할
   - `PeriodInfo = "prdSe=Y 1950~2100 prdSe=M 202001~202612"` 형식이면
   - `parts = ["prdSe=Y", "1950~2100", "prdSe=M", "202001~202612"]`
   - 반복에서 처음 "prdSe=Y"를 찾으면 "Y" 반환 (정상)

3. **범위 데이터 손실**: 주기 코드(Y/M/Q/H)만 추출하고 범위(1950~2100) 무시
   - `p.StrtPrdDe`와 `p.EndPrdDe` 정보가 생성되지만 데이터 조회에 사용 안 됨

4. **변환 로직**: `convertPrdSe()` (app.go:977-996)
   - 이미 코드(Y/M 등)를 받으면 그대로 반환
   - 한글 입력(년, 월 등)만 변환

**예상되는 시나리오**:
- "prdSe=Y 1950~2100 prdSe=M 202001~202612" → 파싱: "Y" (실제로는 Y와 M 모두 가능)
- "prdSe=" (빈 값) → 파싱: "" → 기본값 "Y" 사용
- "" (빈 문자열) → 기본값 "Y" 사용

**권장 사항**:
- 여러 주기가 있을 때 선택 로직 명확화 (최신? 기본?)
- 범위 데이터 활용 (옵션)
- 테스트: 년도, 월 데이터 혼합된 통계표 검증

---

### 12. 자동 파라미터 기본값: firstItem이 빈 문자열일 때

**상태**: ⚠️ 잠재적 문제 - 기본값 "T10"은 임의적이고 모든 통계표에 유효하지 않을 수 있음

**분석**:
- `app.go:451-454`: 메타의 Items 리스트에서 첫 번째 항목 코드 추출
- Items가 비어있으면 기본값 "T10" 사용

**코드 발췌** (app.go:451-454):
```go
firstItem := "T10"  // 기본값 (문제!)
if len(m.metaInfo.Items) > 0 {
    firstItem = m.metaInfo.Items[0].Code
}
```

**발견된 문제**:

1. **"T10" 하드코딩**: KOSIS 시스템의 공통 항목 코드인지 확인 필요
   - 일부 통계표에서는 존재하지 않을 수 있음
   - API 응답: "OBJ_ID not found" 또는 유사 에러

2. **Items 리스트가 비어있을 때**:
   - `summary.Items`가 없으면 (메타 응답에 ITEM 타입 없으면)
   - 강제로 "T10" 사용 → API 에러 가능성 높음

3. **메타 로드 실패**:
   - `MetaSummary()` 호출에서 에러가 발생해도 부분 결과 반환 가능
   - 메타 로드 실패 시에도 자동 데이터 조회 시도

4. **API 파라미터 구조**:
   - Item 파라미터 필수? 선택?
   - 설계서에서 Item 파라미터 요구사항 확인 필요

**예상 에러**:
```
API 오류 [10]: OBJ_ID not found: T10
```

**권장 사항**:
- Item 파라미터 필수 여부 확인
- 비어있을 때 기본값 사용 대신 조회 건너뛰거나 ALL 사용
- 테스트: Item 없는 통계표로 검증

---

## 추가 발견 사항

### A. PanelMeta와 PanelResult 간 상태 관리

**상황**:
- 메타 로드 직후 `m.activePanel = PanelResult` 설정 (app.go:472)
- 사용자는 메타 정보를 확인할 시간 없음
- 즉시 데이터 조회 시작되고 결과 패널로 전환

**영향**:
- 생성되는 데이터 조회 파라미터를 사용자가 확인할 수 없음
- 에러 발생 시 어떤 파라미터로 조회했는지 알기 어려움

---

### B. 메타 요약 정보 부정확성

**분류 정보 추출** (app.go:213-223):
```go
for _, c := range summary.Classifications {
    code := c.ItmID      // 첫 우선순위
    name := c.ItmNM
    if code == "" {
        code = c.Code    // 두 번째 우선순위
    }
    if name == "" {
        name = c.Label   // 두 번째 우선순위
    }
    meta.Classifications = append(meta.Classifications, ClassInfo{Code: code, Name: name})
}
```

**문제**:
- `ItmID` vs `Code`, `ItmNM` vs `Label` 차이점 불명확
- KOSIS API 응답 구조 이해도 부족할 수 있음
- 메타 데이터 타입 혼동 가능성 (분류 vs 항목 vs 기타)

---

## 추가 발견 (2차)

### 13. 검색 결과 → SearchItem 변환 시 필드 누락

**상태**: ❌ 심각한 문제 - 6개 필드 손실

**분석**:
- `app.go:179-185`: SearchResult → SearchItem 변환에서 4개 필드만 매핑
- SearchResult 구조 (types.go:5-17): StatID, StatNM, VwCD, MTAtitle, StrtPrdDe, EndPrdDe, RecTblSe 총 11개 필드
- SearchItem 구조 (app.go:27-33): OrgID, OrgNM, TblID, TblNM 4개만 정의

**코드**:
```go
var items []SearchItem
for _, r := range results {
    items = append(items, SearchItem{
        OrgID: r.OrgID, OrgNM: r.OrgNM,
        TblID: r.TblID, TblNM: r.TblNM,  // 4개만 매핑
    })
}
```

**문제점**:
1. StatNM (통계명) 손실 → 사용자가 통계표가 무엇인지 명확히 알 수 없음
2. StatID (통계ID) 손실 → 통계 추적 불가
3. VwCD (분류코드) 손실 → 통계분류 정보 불가
4. MTAtitle (영문 제목) 손실
5. StrtPrdDe/EndPrdDe (수록 시점) 손실 → 조회 전 기간 미확인
6. RecTblSe (생성주기) 손실 → 갱신 빈도 미확인

**권장**:
- SearchItem에 StatNM, StatID, VwCD, StrtPrdDe, EndPrdDe 필드 추가
- 검색 결과 화면에 표시 (예: "통계명 | 수록기간")

---

### 14. DataRow 변환 시 필드 누락 (api.DataRow → tui.DataRow)

**상태**: ❌ 심각한 문제 - 10개 이상 필드 손실

**분석**:
- `app.go:254-265`: api.DataRow → tui.DataRow 변환에서 5개 필드만 매핑
- api.DataRow 구조 (types.go:45-74): 25개 필드 포함
- tui.DataRow (app.go:62-65): Fields map에 5개만 저장

**코드**:
```go
fields := map[string]string{
    "C1_NM":   r.C1NM,
    "ITM_NM":  r.ItmNM,
    "PRD_DE":  r.PrdDe,
    "DT":      r.DT,
    "UNIT_NM": r.UnitNM,
}
```

**손실된 필드들**:
- 분류: C2, C2NM, C3, C3NM, C4, C4NM, ..., C8, C8NM (14개)
- 기타: OrgID, OrgNM, TblID, TblNM, ItmID, UnitID, PrdSe, LstChn (8개)

**영향**:
1. TUI 결과 테이블에 C1_NM만 표시되고 C2_NM ~ C8_NM 안 보임
2. 대다중분류 통계표 (분류 8개) 일부만 표시
3. CLI로 내보낼 때 완전한 데이터가 아님
4. 통계표/기관 정보 손실로 추적 불가

**권장**:
- tui.DataRow.Fields에 모든 C*_NM 필드 추가
- 결과 화면에 분류별로 표시 (너비 제약은 수평 스크롤로)

---

### 15. TUI에서 DataWithAutoSplit 미사용

**상태**: ❌ 심각한 문제 - 4만 셀 제한 무시

**분석**:
- `app.go:249`: `m.client.Data()` 호출 (일반 조회)
- `cmd/data.go:207`: CLI에서는 `DataWithAutoSplit()` 사용 (자동 분할)
- TUI와 CLI의 동작 불일치

**코드** (app.go:244-253):
```go
func (m Model) doData(orgID, tblID string, opts api.DataOptions) tea.Cmd {
    return func() tea.Msg {
        // ...
        results, err := m.client.Data(orgID, tblID, opts)  // 분할 없음!
        // ...
    }
}
```

**문제점**:
1. TUI에서 4만 셀 초과 조회 시 API 에러 발생
2. estimateCellCount() 함수 활용 안 함
3. 사용자가 "대용량 조회" 경험 불가
4. CLI와 TUI의 기능 차이 (사용자 혼동)

**권장**:
- doData() 함수에 splitOpts 파라미터 추가
- `client.DataWithAutoSplit()` 호출로 변경
- 진행률 콜백 추가 (UI에서 "분할 조회 중... N/M" 표시)

---

### 16. 자동 파라미터: NewEstPrdCnt 고정값 "5" 문제

**상태**: ⚠️ 데이터 손실 위험

**분석**:
- `app.go:459`: `NewEstPrdCnt: "5"` 고정값
- 월별 데이터 조회 시 최근 5개월만 반환
- 년도 데이터 조회 시 최근 5년만 반환

**코드**:
```go
opts := api.DataOptions{
    Item:         firstItem,
    PrdSe:        period,
    NewEstPrdCnt: "5",  // 항상 5개로 고정!
}
```

**문제점**:
1. 분기별 데이터 조회 시 최근 5분기 (1.25년)만 표시
2. 반기별 데이터 조회 시 최근 5반기 (2.5년)만 표시
3. 결과가 완전하지 않음을 사용자가 모름
4. 설계서에서 권장 개수 미확인

**권장**:
- PrdSe에 따라 다른 값 설정 (월: 12, 분기: 8, 반기: 10 등)
- 또는 설계서 기준에 맞춰 조정
- 사용자에게 "최근 N개" 명시 표시

---

### 17. convertPrdSe() 함수: 한글 '반기' 변환 테스트 누락

**상태**: ⚠️ 미검증

**분석**:
- `app.go:977-991`: convertPrdSe() 함수 구현
- `case "반기", "반기별": return "H"` 코드 있음
- 그러나 TUI에서 이 함수를 통해 입력을 받지 않음 (메타 API에서만 사용)

**코드**:
```go
func convertPrdSe(prdSe string) string {
    switch prdSe {
    case "년", "연", "연간": return "Y"
    case "월", "월간": return "M"
    case "분기", "분기별": return "Q"
    case "반기", "반기별": return "H"  // 테스트 필요
    case "Y", "M", "Q", "H": return prdSe
    default:
        if prdSe != "" {
            return prdSe
        }
        return "Y"
    }
}
```

**문제점**:
1. 메타 API 응답에서 PrdSe가 "반기" 형식이면 "H"로 변환하나 확인 필요
2. API 응답 형식이 "반기별"일 수도 있음
3. 영어 입력 (예: "half-year") 처리 안 함

**권장**:
- 메타 API 응답 PrdSe 필드 형식 확인 테스트
- 대소문자 통일 (입력값 trim + tolower)

---

### 18. NumClassGroups 계산: 더블 카운팅 위험

**상태**: ⚠️ 잠재적 오류

**분석**:
- `app.go:201-211`: objIDSet으로 고유 ObjID 개수 세기
- 문제: Classifications 리스트가 다중 ObjID를 가질 수 있음

**코드**:
```go
objIDSet := map[string]bool{}
for _, c := range summary.Classifications {
    if c.ObjID != "" {
        objIDSet[c.ObjID] = true
    }
}
meta.NumClassGroups = len(objIDSet)
```

**예상 시나리오**:
```
summary.Classifications = [
    {ObjID: "C1_OBJID", ...},  // 분류1의 항목들
    {ObjID: "C1_OBJID", ...},
    {ObjID: "C1_OBJID", ...},
    {ObjID: "C2_OBJID", ...},  // 분류2의 항목들
    {ObjID: "C2_OBJID", ...},
]
// objIDSet = {C1_OBJID, C2_OBJID} → NumClassGroups = 2 ✓
```

**문제점**:
1. API 응답 구조가 명확하지 않으면 잘못된 개수 계산
2. 설계서에서 NumClassGroups 정의 확인 필요
3. ObjID가 항상 분류 그룹 ID를 나타내는지 미확인

**권장**:
- 메타 API 응답 구조 문서화
- 테스트: 분류 1, 2, 3, 8개인 통계표별로 검증

---

### 19. PeriodInfo 여러 주기 때 마지막만 사용

**상태**: ❌ 데이터 손실

**분석**:
- `app.go:235-238`: for 루프에서 PeriodInfo를 반복 덮어쓰기
- 최후의 Periods 항목만 저장됨

**코드**:
```go
for _, p := range summary.Periods {
    prdCode := convertPrdSe(p.PrdSe)
    meta.PeriodInfo = "prdSe=" + prdCode + " " + p.StrtPrdDe + "~" + p.EndPrdDe
}
// 루프 끝 후: 마지막 p만 저장됨!
```

**예상 결과**:
```
summary.Periods = [
    {PrdSe: "년", StrtPrdDe: "2000", EndPrdDe: "2024"},
    {PrdSe: "월", StrtPrdDe: "202401", EndPrdDe: "202412"},
]
// 결과: meta.PeriodInfo = "prdSe=M 202401~202412" (월만 저장됨!)
```

**영향**:
1. 년도/월 데이터 모두 있는 통계표에서 월만 표시
2. 사용자가 년도 조회 불가로 오해
3. 자동 파라미터에서 "월" 주기가 선택됨

**권장**:
- PeriodInfo를 배열로 변경: `[]string`
- 또는 struct로 변경: `[]PeriodInfo{PrdSe, StartPrdDe, EndPrdDe}`
- renderMetaSection()에서 모든 주기 표시

---

### 20. FirstItem 기본값 "T10" 모든 통계표에서 작동하는지 미확인

**상태**: ❌ API 에러 위험

**분석**:
- `app.go:451-454`: Items 비어있으면 "T10" 사용
- "T10"이 항상 유효한지 미확인

**코드**:
```go
firstItem := "T10"
if len(m.metaInfo.Items) > 0 {
    firstItem = m.metaInfo.Items[0].Code
}
```

**문제점**:
1. Items 빈 경우: MetaSummary에서 조회 실패했을 수 있음
2. "T10" 하드코딩: 일부 통계표에서 없을 수 있음
3. API 응답: "OBJ_ID not found: T10" 에러 가능

**테스트 필요 통계표**:
- Items 빈 통계표
- Item 코드가 "T01"부터 시작하는 통계표
- Item 없이 분류만 있는 통계표

**권장**:
- Items 빈 경우 Item 파라미터 생략 (ALL 대신)
- 또는 "ALL"로 설정
- 에러 발생 시 "메타 재로드" 옵션 제공

---

### 21. 결과 테이블: C1_NM만 표시되고 C2~C8 안 보임

**상태**: ❌ 대다중분류 통계표 미지원

**분석**:
- `app.go:867`: `row.Fields["C1_NM"]` 하드코딩
- C2_NM, C3_NM 등은 표시 안 함

**코드** (app.go:855-874):
```go
lines = append(lines, truncateText("분류", 10) + " │ " + truncateText("항목", 8) + " │ " + ...)
for i, row := range m.resultData {
    c1nm := truncateText(row.Fields["C1_NM"], 10)
    itmnm := truncateText(row.Fields["ITM_NM"], 8)
    // ... C2_NM, C3_NM 등 미처리
}
```

**영향**:
1. 분류 2개 이상인 통계표: 첫 분류만 표시
2. 결과 데이터가 완전하지 않아 보임
3. 사용자가 다른 분류 정보 확인 불가

**권장**:
- 필드 매핑 시 모든 C*_NM 포함 (문제 14와 동일)
- 또는 분류별 탭 추가
- 수평 스크롤 지원

---

### 22. 메타 정보 화면: Classifications 빈 경우 처리

**상태**: ⚠️ 혼동 가능

**분석**:
- `app.go:213-223`: Classifications 빈 경우
- 화면에 "분류: (없음)" 표시하지 않고 빈 줄 표시

**코드**:
```go
for _, c := range summary.Classifications {
    // ... 추가
}
// result: []ClassInfo{} (빈 리스트)
```

**renderMetaSection() 부분**:
- Classifications 빈 리스트 순회 → 아무 것도 출력 안 됨
- 사용자가 분류가 정말 없는지, 로드 실패인지 모호

**권장**:
- Classifications 빈 경우: "분류: 없음" 표시
- Items 빈 경우: "항목: 없음" 표시

---

### 23. 에러 메시지: API 오류 코드별 구분 부족

**상태**: ⚠️ 디버깅 어려움

**분석**:
- `app.go:480-488`: 모든 데이터 조회 에러를 동일하게 처리
- "❌ 데이터 조회 오류: ..." 로만 표시

**코드**:
```go
case dataResultMsg:
    m.loading = false
    if msg.err != nil {
        m.err = msg.err
        m.statusMsg = fmt.Sprintf("❌ 데이터 조회 오류: %v", msg.err)
    }
```

**예상되는 에러 종류**:
1. API 오류: "API 오류 [10]: OBJ_ID not found"
2. 네트워크 오류: "API request failed: ..."
3. 파라미터 오류: "orgId와 tblId는 필수입니다"
4. 속도 제한: "API rate limit exceeded (429)"

**권장**:
- 에러 타입별 처리 함수 추가
- "API 오류 [코드]", "네트워크 오류", "파라미터 오류" 등으로 구분
- 사용자에게 해결책 제시 (재시도, 파라미터 확인 등)

---

### 24. 내보내기 (e 키): 기능 미구현

**상태**: ❌ 문제 아님 (설계상 이상 없음)

**분석**:
- `app.go:327-332`: "기능 준비 중" 메시지만 표시
- TUI에서는 내보내기 미지원, CLI에서만 지원

**코드**:
```go
case KeyExport:
    if len(m.resultData) > 0 {
        m.statusMsg = "💾 내보내기: 기능 준비 중 (CLI로 사용: kosis d ... -o file.xlsx)"
    }
```

**현황**:
- CLI: `-o file.xlsx` 옵션으로 내보내기 가능
- TUI: 기능 미구현 (설계상 "추후 구현" 예상)

**권장**:
- TUI에서 내보내기 기능 추가 (선택 사항)
- 또는 현재 메시지 유지 (설계대로)

---

### 25. 메타 로드 전 데이터 조회 시도 가능성

**상태**: ⚠️ UX 문제

**분석**:
- 메타 로드 전에 검색 결과 선택 후 d 키를 누르면?
- `m.metaInfo == nil` 체크 없음

**코드** (app.go:386-407):
```go
case "d":  // 데이터 조회
    if m.selectedTable != nil {
        if m.metaInfo != nil {
            // 메타 정보가 있으면: 데이터 조회
            // ...
        } else {
            m.statusMsg = "📋 메타 정보를 먼저 로드하세요 (Enter)"
        }
    }
```

**현황**:
- d 키 누르면 메타 정보 확인 후 메시지 표시
- Enter 키: 메타 로드 자동 실행

**권장**:
- 현재 구현 유지 (좋은 UX)
- 또는 d 키에서 자동으로 메타 로드

---

### 26. SearchItem 화면: 통계명/수록기간 미표시

**상태**: ⚠️ UX 문제

**분석**:
- `renderSearchSection()` 미확인 (app.go 출력 부분)
- 검색 결과에 OrgNM, TblNM만 표시 (추정)

**권장**:
- SearchItem 변환 시 필드 추가 (문제 13)
- StatNM (통계명) 표시 추가
- StrtPrdDe ~ EndPrdDe (수록 시점) 표시

---

### 27. 메타 API 응답 일부 필드 미사용

**상태**: ⚠️ 정보 손실

**분석**:
- MetaResult 구조 (types.go): ObjIDSN, Level, Type, Code, Label 등 다수 필드
- doMeta() 함수 (app.go): ItmID, Code, Label만 사용

**미사용 필드**:
- ObjIDSN: 객체ID 순번 (분류 계층 정보?)
- Level: 분류 계층 깊이
- Type: 메타 타입 (ITM, PRD 등)
- Code: ObjID와 다른 코드?
- UnitNM: 단위명 (Periods에는 있으나 Classifications에도 있을 수 있음)

**권장**:
- API 응답 구조 명확화
- 설계서에 필드별 의미 문서화

---

### 28. 진행률 콜백: TUI에서 미사용

**상태**: ⚠️ 기능 미사용

**분석**:
- `splitter.go`: 분할 조회 시 progressFn 콜백 제공
- `cmd/data.go`: progressFn으로 진행률 출력 (CLI)
- TUI의 doData(): progressFn 미제공

**코드** (app.go:249):
```go
results, err := m.client.Data(orgID, tblID, opts)
// progressFn 없음 → 진행률 미표시
```

**현황**:
- TUI에서 대용량 조회 시 로딩 상태 모름
- DataWithAutoSplit() 미사용이므로 현재는 이슈 없음 (문제 15)

**권장**:
- doData()에서 DataWithAutoSplit() 사용 시 progressFn 추가
- UI에 "분할 조회 중... 1/3 (" 표시

---

### 29. 자동 주기 감지: 여러 주기일 때 기본값

**상태**: ⚠️ 불명확한 우선순위

**분석**:
- `app.go:437-448`: PeriodInfo 파싱해서 period 추출
- 여러 주기 있으면 처음 "prdSe=" 찾은 것만 사용

**코드**:
```go
period := "Y"
if m.metaInfo != nil && m.metaInfo.PeriodInfo != "" {
    parts := strings.Fields(m.metaInfo.PeriodInfo)
    for _, p := range parts {
        if strings.HasPrefix(p, "prdSe=") {
            period = strings.TrimPrefix(p, "prdSe=")
            break
        }
    }
}
```

**문제점**:
- PeriodInfo = "prdSe=Y ... prdSe=M ..."일 때 첫 번째만 사용
- 문제 19의 마지막 주기만 저장되는 문제와 관련

**권장**:
- PeriodInfo 구조 변경 (배열)
- 또는 우선순위 설정 (월 > 분기 > 반기 > 연 등)

---

### 30. StatusMsg 장문 출력: 줄 바꿈 미지원

**상태**: ⚠️ UX 문제

**분석**:
- 상태 메시지가 길면 화면 아래 잘림
- 예: "❌ 데이터 조회 오류: API 오류 [10]: OBJ_ID not found: T10"

**권장**:
- statusMsg 길이 제한 (최대 70자)
- 또는 별도 오류 패널 추가
- 또는 오류 코드만 표시 + 자세한 정보는 'h' 키로 도움말 표시

---

### 31. 키 바인딩: 설명 구현 확인

**상태**: ⚠️ 미확인

**분석**:
- TUI 상단: "s(검색) m(메타) d(데이터) e(내보내기) f(즐겨찾기) h(도움말)" 표시
- `renderHelpLine()` 또는 유사 함수 (app.go:? 확인 필요)

**권장**:
- 모든 키 바인딩 일관성 확인
- 사용자 매뉴얼 업데이트

---

### 32. 데이터 조회 후 메타 패널 접근 불가

**상태**: ⚠️ UX 문제

**분석**:
- `m.activePanel = PanelResult` 설정 후 메타 정보 확인 불가
- m 키를 눌러도 PanelMeta로 전환할 수 없거나 (구현 확인 필요)

**권장**:
- m 키로 PanelMeta 전환 가능하게
- 또는 메타/결과 탭 추가

---

## 종합 평가

**기본 흐름**: ✅ 정상
**에러 처리**: ⚠️ 부분 부족
**파라미터 처리**: ⚠️ 잠재적 문제 다수
**사용자 안내**: ⚠️ 일부 모호함
**데이터 흐름**: ❌ 필드 손실 심각

**우선순위 높은 수정 (2차)**:
1. **검색 결과 필드 손실** (문제 13) - SearchItem에 StatNM, StatID 추가
2. **데이터 결과 필드 손실** (문제 14) - tui.DataRow.Fields에 모든 분류 포함
3. **TUI AutoSplit 미사용** (문제 15) - doData()에 DataWithAutoSplit() 적용
4. **NewEstPrdCnt 고정값** (문제 16) - 주기별 다른 값 적용
5. **PeriodInfo 덮어쓰기** (문제 19) - 배열로 변경
6. **FirstItem "T10"** (문제 20) - Items 빈 경우 처리 명확화
7. **C1_NM만 표시** (문제 21) - 모든 분류 표시 지원

---

**테스트 완료 (2차 추가 20개 문제 발견)**
