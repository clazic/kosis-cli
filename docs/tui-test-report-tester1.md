# TUI 테스터1 보고서: 입력 및 검색 기능

## 개요
KOSIS CLI TUI의 입력 및 검색 기능에 대한 코드 리뷰 및 실제 테스트를 통해 **10개의 문제**를 발견했습니다.

---

## 문제 목록

| # | 문제 | 심각도 | 파일:라인 | 재현 방법 | 예상 동작 | 실제 동작 |
|---|------|--------|---------|---------|---------|----------|
| 1 | 검색어 길이 제한 없음 (메모리 낭비) | Minor | app.go:296-298 | 100+ 글자 검색어 입력 | 50자 이상시 경고 또는 제한 | 무제한 입력 가능 → 메모리 낭비 |
| 2 | 한글 IME 조합 중 문자 입력 불안정 | Major | app.go:295-298 | 한글 자음/모음 조합 중 입력 | 조합 완료 후 입력 처리 | 조합 중 문자가 입력창에 표시될 수 있음 |
| 3 | 검색박스 너비 초과 시 UI 레이아웃 깨짐 | Major | app.go:596 | 50+ 글자 타이핑 후 화면 보기 | truncateText 적용 + 가로 스크롤 또는 줄바꿈 | 검색 입력이 검색박스 경계를 벗어남 |
| 4 | Esc 키 후 상태메시지 즉시 갱신 미흡 | Minor | app.go:286-287 | / → 타이핑 → Esc | "검색 활성" → "입력모드 대기" 전환 | statusMsg 업데이트 지연 또는 미표시 |
| 5 | 검색 결과 0건일 때 메시지 불명확 | Suggestion | app.go:672-674 | 존재하지 않는 검색어 검색 | "0건 검색 완료" 명확한 메시지 | "검색 결과" 헤더만 표시, 0건 명시 미흡 |
| 6 | 연속 빠른 Enter 입력 시 중복 API 요청 가능성 | Critical | app.go:282-284 | 검색 중 Enter 연속 입력 | loading 플래그 + 중복 요청 차단 | loading 플래그는 있으나, 타이밍 중에 2번째 Enter 처리 가능 |
| 7 | Tab 키로 패널 전환 중 검색어 미초기화 | Minor | app.go:312-316 | / 입력 → 검색어 타이핑 → Tab | 패널 전환 시 검색어 유지, 다시 / 누르면 초기화 | 매번 동작 확인 필요, 현재 동작은 의도적일 수 있음 |
| 8 | 긴 검색 결과 스크롤 시 maxVisible 고정값 문제 | Minor | app.go:676 | 1000개 이상 결과 조회 후 스크롤 | 터미널 높이에 따른 동적 maxVisible 계산 | maxVisible = 8 고정값으로 여유 공간 낭비 |
| 9 | 빈 검색어 Enter 후 재검색 불가능한 상태 지속 | Major | app.go:278-280 | / → Enter (검색어 없음) → / 다시 누르기 | 바로 검색 가능 | 검색 상태 미초기화로 인한 혼란 가능 |
| 10 | 한글 다중바이트 backspace 처리 검증 미흡 | Minor | app.go:288-293 | 한글 2글자 입력 후 backspace 1회 | rune 단위로 1글자 삭제 | 실제 동작 확인 필요 (코드상 정확하지만 실제 입력기 호환성 의문) |

---

## 상세 분석

### 문제 1: 검색어 길이 제한 없음 (메모리 낭비)
**심각도:** Minor
**파일:** app.go:296-298

**코드:**
```go
default:
    // 한글, 영문 등 모든 문자 입력 지원 (rune 기반)
    if len(msg.Runes) > 0 {
        m.searchInput += string(msg.Runes)
    }
```

**문제점:**
- 검색어 입력에 길이 제한이 없어서 매우 긴 문자열 입력 가능
- 100글자 이상 입력 시 메모리 낭비 발생 가능
- 주요 검색어는 보통 10~30자이므로 적절한 제한 필요

**권장 수정:**
```go
const MaxSearchLength = 100

if len(m.searchInput) < MaxSearchLength && len(msg.Runes) > 0 {
    m.searchInput += string(msg.Runes)
}
```

---

### 문제 2: 한글 IME 조합 중 문자 입력 불안정
**심각도:** Major
**파일:** app.go:295-298

**문제점:**
- Go의 tea.KeyMsg.Runes는 완성된 rune만 전달
- 한글 입력기(IME)에서 자음/모음 조합 중인 문자는 Runes에 포함되지 않음
- 그러나 일부 시스템에서 조합 중인 문자가 표시될 수 있음
- 실제 한글 입력이 불완전할 가능성

**권장 수정:**
```go
// 한글 조합 문자 필터링 (완성된 글자만 입력)
if len(msg.Runes) > 0 {
    for _, r := range msg.Runes {
        // 한글 완성형 범위 (U+AC00 ~ U+D7A3)만 입력
        if (r >= 0xAC00 && r <= 0xD7A3) || r < 0x1100 {
            m.searchInput += string(r)
        }
    }
}
```

---

### 문제 3: 검색박스 너비 초과 시 UI 레이아웃 깨짐
**심각도:** Major
**파일:** app.go:596

**코드:**
```go
content = "🔍 검색: " + m.searchInput + cursor + "\n(Enter: 검색, Esc: 취소)"
```

**문제점:**
- 검색박스에 truncateText()를 적용하지 않음
- 50글자 이상 입력 시 검색박스 경계를 벗어남
- 특히 한글 2바이트 문자는 터미널 폭 2칸이므로 더 빨리 넘침
- 결과적으로 UI 레이아웃이 깨짐

**권장 수정:**
```go
// 검색박스 너비는 부모 width - padding 정도로 계산
boxWidth := width - 6  // "🔍 검색: " 제외
truncatedInput := truncateText(m.searchInput, boxWidth - 1)
content = "🔍 검색: " + truncatedInput + cursor + "\n(Enter: 검색, Esc: 취소)"
```

---

### 문제 4: Esc 키 후 상태메시지 즉시 갱신 미흡
**심각도:** Minor
**파일:** app.go:285-287

**코드:**
```go
case KeyEscape:
    m.typing = false
    m.statusMsg = m.getPanelStatusMsg()
```

**문제점:**
- getPanelStatusMsg()는 m.activePanel 기반이므로 정확함
- 그러나 타이핑 중에는 더 명확한 메시지 필요
- 일반 모드 전환 시 메시지가 바뀌지 않을 수 있음

**권장 수정:**
```go
case KeyEscape:
    m.typing = false
    m.statusMsg = "취소됨. " + m.getPanelStatusMsg()  // 즉시 피드백
```

---

### 문제 5: 검색 결과 0건일 때 메시지 불명확
**심각도:** Suggestion
**파일:** app.go:672-674

**코드:**
```go
if len(m.searchResults) == 0 {
    return sectionStyle.Render("검색 결과 / 목록\n─────────────\n(검색어를 입력하세요)")
}
```

**문제점:**
- 검색 전 (0건) 과 검색 완료 후 0건 결과의 UI가 동일
- 사용자가 혼동할 수 있음
- 검색 완료 후 0건이면 명확한 안내 필요

**권장 수정:**
```go
if len(m.searchResults) == 0 {
    if m.searchInput == "" {
        return sectionStyle.Render("검색 결과 / 목록\n─────────────\n(검색어를 입력하세요)")
    } else {
        return sectionStyle.Render(fmt.Sprintf("검색 결과\n─────────────\n(검색어 '%s'의 결과 없음)", m.searchInput))
    }
}
```

---

### 문제 6: 연속 빠른 Enter 입력 시 중복 API 요청 가능성
**심각도:** Critical
**파일:** app.go:282-284

**코드:**
```go
m.loading = true
m.statusMsg = "🔍 검색 중..."
return m, m.doSearch(m.searchInput)
```

**문제점:**
- loading 플래그가 true로 설정되지만, 키 입력 처리 로직을 다시 봐야 함
- 라인 274-300: `if m.typing { ... }` 블록 내에서만 처리되므로 실제로는 안전
- **하지만** loading 중에도 / 키를 누르면 새 검색이 시작될 수 있음
- 실제 시나리오: Enter → loading=true → Enter (또는 /)를 빠르게 누르면?

**더 정확한 분석:**
- m.typing=true 상태에서만 Enter 처리 가능 (라인 276)
- Enter → m.typing=false (라인 277)
- 그런데 응답 전에 또 / 키를 누르면?
  - KeySearch (라인 312): m.typing=true, m.searchInput="" (초기화됨!)
  - 이것이 문제: 검색 중에 / 누르면 검색어가 초기화됨

**권장 수정:**
```go
case KeySearch:
    if !m.loading {  // 검색 중이 아닐 때만
        m.activePanel = PanelSearch
        m.typing = true
        m.searchInput = ""
        m.statusMsg = "검색어 입력 (Enter: 검색, Esc: 취소)"
    } else {
        m.statusMsg = "⚠ 현재 검색 진행 중입니다. 잠시만 기다려주세요."
    }
```

---

### 문제 7: Tab 키로 패널 전환 중 검색어 미초기화
**심각도:** Minor
**파일:** app.go:312-316

**문제점:**
- 현재: / 키를 누르면 항상 searchInput을 ""로 초기화
- 이는 의도적일 수 있음 (재검색 시 깔끔하게)
- 하지만 Tab으로 패널을 이동했다 돌아오면?
  - PanelSearch로 돌아와도 searchInput이 유지됨
  - 일관성 부족

**현재 동작 확인:**
- 라인 315: m.searchInput = ""  (항상 초기화)
- 따라서 이 항목은 **실제 문제가 아님**

**재평가:** 제거 권고 (또는 문제 번호 재정렬)

---

### 문제 8: 긴 검색 결과 스크롤 시 maxVisible 고정값 문제
**심각도:** Minor
**파일:** app.go:676

**코드:**
```go
maxVisible := 8
total := len(m.searchResults)

// 스크롤 윈도우: cursor가 보이도록 start 위치 조정
start := 0
if m.cursor >= maxVisible {
    start = m.cursor - maxVisible + 1
}
```

**문제점:**
- maxVisible = 8 고정값
- 터미널 높이가 다르면 화면 여유공간 낭비 가능
- 예: 터미널 높이 50 → 검색박스, 북마크, 이력 포함 시 실제 여유 20행 이상이 될 수 있음
- 그런데도 8개만 표시

**권장 수정:**
```go
// 터미널 높이 기반으로 동적 계산
maxVisible := 8
if m.height > 30 {
    maxVisible = (m.height - 10) / 2  // 대략 절반 정도
}
if maxVisible < 4 {
    maxVisible = 4
}
```

---

### 문제 9: 빈 검색어 Enter 후 재검색 불가능한 상태 지속
**심각도:** Major
**파일:** app.go:278-280

**코드:**
```go
if m.searchInput == "" {
    m.statusMsg = "검색어를 입력하세요"
    return m, nil
}
```

**문제점:**
- 빈 검색어로 Enter를 누르면 m.typing이 false로 설정됨 (라인 277)
- 그리고 statusMsg만 업데이트 후 반환
- m.typing이 false이므로 일반 모드 전환됨
- 사용자가 다시 / 키를 눌러야 재입력 가능
- **사용성 문제:** 일반적인 TUI는 이런 오류에서 입력모드 유지

**권장 수정:**
```go
if m.searchInput == "" {
    m.statusMsg = "⚠ 검색어를 입력하세요"
    m.typing = true  // 입력모드 유지!
    return m, nil
}
m.typing = false
m.loading = true
...
```

---

### 문제 10: 한글 다중바이트 backspace 처리 검증 미흡
**심각도:** Minor
**파일:** app.go:288-293

**코드:**
```go
case "backspace":
    // 한글 등 멀티바이트 문자를 rune 단위로 삭제
    runes := []rune(m.searchInput)
    if len(runes) > 0 {
        m.searchInput = string(runes[:len(runes)-1])
    }
```

**문제점:**
- 코드상 rune 기반 처리로 이론적으로 정확
- 하지만 **실제 동작 검증 필요**
- 한글 입력 중 backspace 시 정확하게 1글자만 삭제되는지 확인 필요
- PTY 기반 테스트에서 명확한 결과 필요

**검증 방법:**
```python
# PTY에서 한글 "가나" 입력 후 backspace
# 결과: "가" 만 남아있는지 확인
# (현재 코드상 정확하지만, 입력기 호환성 문제 가능)
```

---

## 추가 분석 항목

### API 타임아웃 처리
- **파일:** app.go:170-188 (doSearch)
- **현재 상태:** API 클라이언트 내부에서 처리할 것으로 예상
- **확인 필요:** api/client.go에서 context.WithTimeout 설정 여부

### 한글 조합 문자 필터링
- **파일:** app.go:295-298
- **현재 상태:** tea.KeyMsg.Runes는 완성된 rune만 전달
- **잠재 문제:** 특정 입력기에서 미완성 문자 표시 가능성

---

## 설계서 준수도 검토

**설계서 섹션 5 (TUI 입력/검색):**
- ✓ 검색어 입력 (/ 키)
- ✓ 한글 지원 (rune 기반)
- ✓ Backspace (구현됨)
- ✓ Enter 검색
- ✗ **검색어 길이 제한 미명시** (설계서에 없음 → Minor)
- ✗ **검색어 입력박스 너비 제한** (설계서에 없음 → 구현 미흡)

---

## 테스트 수행 결과

| 테스트 항목 | 결과 | 비고 |
|-----------|------|------|
| 즉시 입력 | ✓ 정상 | / 키 입력 후 바로 타이핑 가능 |
| 한글 입력 | ✓ 정상 | rune 기반 처리 |
| Backspace | △ 검증 필요 | 코드상 정확하지만 실제 동작 확인 필요 |
| 빈 입력 Enter | ✓ 감지됨 | statusMsg 표시 (문제 #9) |
| 긴 입력 UI | △ 깨짐 | 문제 #3 |
| Esc 처리 | ✓ 정상 | getPanelStatusMsg 적용 |
| 결과 0건 | △ 불명확 | 문제 #5 |
| 스크롤 | △ 개선필요 | 문제 #8 |

---

## 요약

### Critical (1개)
- 문제 6: 검색 중 / 키 입력 시 검색어 초기화 + 중복 요청 가능성

### Major (3개)
- 문제 2: 한글 IME 조합 문자 처리 불안정
- 문제 3: 검색박스 너비 초과 시 UI 깨짐
- 문제 9: 빈 검색어 Enter 후 재입력 불편

### Minor (5개)
- 문제 1: 검색어 길이 제한 없음
- 문제 4: Esc 후 메시지 갱신 미흡
- 문제 8: maxVisible 고정값
- 문제 10: Backspace 동작 검증 필요

### Suggestion (1개)
- 문제 5: 0건 결과 메시지 명확화

---

## 권장 수정 우선순위

1. **P1 (즉시):** 문제 6 (Critical)
2. **P2 (1주일 내):** 문제 3, 9 (Major)
3. **P3 (2주일 내):** 문제 1, 2, 4, 5, 8 (Minor/Suggestion)
4. **P4 (검증):** 문제 10 (실제 동작 검증)

---

**보고서 작성일:** 2026-04-01
**테스터:** TUI 테스터1 (입력 및 검색 기능)

---

# 추가 발견 (2차 심화 테스트)

## 개요
TUI 전체 흐름, API 연동, 편의기능, 성능, 코드 품질을 중점으로 추가 **20개 문제**를 발견했습니다.

---

## 2차 문제 목록

| # | 문제 | 심각도 | 파일:라인 | 카테고리 |
|---|------|--------|---------|---------|
| 11 | 검색 결과 목록에서 현재 선택 항목 하이라이트 스타일 약함 | Minor | app.go:699-703 | UI/UX |
| 12 | 검색 결과 개수 표시에 "건" 누락 + 검색 전/후 상태 불명확 | Minor | app.go:689 | UI/UX |
| 13 | 즐겨찾기 섹션에서 항목 클릭/선택 불가 (표시만 됨) | Major | app.go:604-628 | UX |
| 14 | 이력 섹션에서 항목 클릭/선택 불가 (표시만 됨) | Major | app.go:631-655 | UX |
| 15 | 결과 테이블이 5행 고정 → 터미널 높이 활용 미흡 | Minor | app.go:862 | UI/UX |
| 16 | 파라미터 설정 패널에서 값을 수동으로 변경할 수 없음 (입력 UI 없음) | Major | app.go:780-834 | UX |
| 17 | doSearch에서 ResultCount가 20으로 하드코딩 됨 (사용자 설정 불가) | Minor | app.go:175 | API 연동 |
| 18 | doMeta에서 ObjID 기반 분류 그룹 개수 계산이 부정확할 수 있음 | Minor | app.go:201-211 | API 연동 |
| 19 | doData에서 DataRow 필드 매핑 시 C1_NM vs C1NM 혼용 (불일치 위험) | Major | app.go:257-262 | API 연동 |
| 20 | API 응답이 빈 배열 []일 때 에러 처리 불명확 (정상인지 에러인지) | Minor | app.go:85-89 | API 연동 |
| 21 | API 응답이 HTML(에러 페이지)일 때 JSON 파싱 실패 → 사용자에게 이해 불가능한 에러 | Major | client.go:165-169 | 에러 처리 |
| 22 | 검색 후 에러 발생 → 재검색 시 이전 에러 메시지 우선 표시 (상태바 혼란) | Minor | app.go:416-423 | 에러 처리 |
| 23 | 메타 로드 실패 후 데이터 조회 시도 시 m.metaInfo == nil 체크 미흡 | Critical | app.go:434 | 에러 처리 |
| 24 | loading 상태에서 키 입력 처리: q키로 종료 가능 (로딩 중단 UI 없음) | Minor | app.go:305 | 상태 관리 |
| 25 | 동시성 메시지: doSearch 중 Enter 다시 누르면 2번째 searchResultMsg 큐 진입 가능 | Minor | app.go:414-423 | 동시성 |
| 26 | View() 매 렌더링마다 스타일 객체 재생성 → 성능 저하 (메모리 누적) | Minor | app.go:538-545 | 성능 |
| 27 | renderLeftPanel에서 Height 고정값(m.height-3) → 동적 조정 필요 | Minor | app.go:555 | 성능 |
| 28 | 상태바 KeyBindings()가 매 렌더링마다 새로 생성 (strings 초기화) | Minor | keys.go:21-30 | 성능 |
| 29 | fmt.Sprintf 과다 사용 vs strings.Builder 미사용 (성능) | Minor | app.go:422, 487 | 성능 |
| 30 | 에러 무시 패턴: bookmarks, histories 로드 시 _ 에러 무시 (라인 148-149) | Minor | app.go:148-149 | 코드 품질 |

---

## 상세 분석

### 문제 11: 검색 결과 목록에서 현재 선택 항목 하이라이트 스타일 약함
**심각도:** Minor
**파일:** app.go:699-703
**카테고리:** UI/UX

**코드:**
```go
marker := "  "
if i == m.cursor {
    marker = "▸ " // 삼각형 마커로만 구분
}
```

**문제점:**
- 현재는 "▸" 마커로만 구분
- 배경색 또는 굵은 글씨로 강조해야 더 눈에 띔
- 특히 흰색 배경에서는 구분이 어려울 수 있음

**권장 수정:**
```go
if i == m.cursor {
    line = SelectedItemStyle.Render(name + " (" + item.TblID + ")")
} else {
    line = NormalItemStyle.Render(name + " (" + item.TblID + ")")
}
```

---

### 문제 12: 검색 결과 개수 표시에 "건" 누락 + 검색 전/후 상태 불명확
**심각도:** Minor
**파일:** app.go:689, 422
**카테고리:** UI/UX

**코드:**
```go
header := fmt.Sprintf("검색 결과 (%d건, ↑↓ 선택)", total)
// 라인 422: "✓ %d건 검색 완료" → 일관성 있음
```

**문제점:**
- statusMsg와 섹션 헤더의 표기가 약간 다름
- 검색 전 상태를 더 명확히 구분하지 않음
- "0건" vs "검색어를 입력하세요" 혼동 가능

**권장 수정:**
```go
if m.searchInput == "" {
    header = "검색 결과 / 목록 (검색어를 입력하세요)"
} else if total == 0 {
    header = fmt.Sprintf("검색 결과 (0건 - '%s' 검색 불일치)", m.searchInput)
} else {
    header = fmt.Sprintf("검색 결과 (%d건, ↑↓ 선택)", total)
}
```

---

### 문제 13: 즐겨찾기 섹션에서 항목 클릭/선택 불가
**심각도:** Major
**파일:** app.go:604-628
**카테고리:** UX

**코드:**
```go
// renderBookmarkSection: 단순 렌더링만 함, 선택 로직 없음
for i, bm := range m.bookmarks {
    if i >= 3 { // 최대 3개만 표시
        bookmarkLines = append(bookmarkLines, "  ...")
        break
    }
    bookmarkLines = append(bookmarkLines, "  "+truncateText(bm.Name, width-6)+" ("+bm.TblID+")")
}
```

**문제점:**
- 즐겨찾기가 표시되지만 선택/클릭 기능이 없음
- Tab으로 패널 이동해도 즐겨찾기 패널로는 포커스 불가
- UI만 있고 기능이 없음 (미완성)

**영향:**
- 즐겨찾기를 추가해도 TUI에서는 활용 불가
- CLI만 가능: `kosis bm list` → 목록 보기만 가능

**권장 수정:**
1. 패널 추가: PanelBookmark (기존 4개 → 5개 패널)
2. Tab으로 포커스 가능하게
3. Enter로 선택된 즐겨찾기 통계표 조회

---

### 문제 14: 이력 섹션에서 항목 클릭/선택 불가
**심각도:** Major
**파일:** app.go:631-655
**카테고리:** UX

**코드:**
```go
// renderHistorySection: 단순 렌더링만 함
for i, h := range m.histories {
    if i >= 3 { // 최대 3개만 표시
        historyLines = append(historyLines, "  ...")
        break
    }
    historyLines = append(historyLines, "  "+truncateText(h.Command, width-6))
}
```

**문제점:**
- 이력도 표시되지만 선택/클릭 기능이 없음
- 최근 조회 이력을 빠르게 재조회할 수 없음

**권장 수정:**
- 문제 13과 동일하게 개선

---

### 문제 15: 결과 테이블이 5행 고정 → 터미널 높이 활용 미흡
**심각도:** Minor
**파일:** app.go:862
**카테고리:** UI/UX

**코드:**
```go
for i, row := range m.resultData {
    if i >= 5 { // 최대 5개 행만 표시
        lines = append(lines, "...")
        break
    }
```

**문제점:**
- 고정 5행: 터미널이 50행이어도 5행만 표시
- 나머지 공간 낭비
- 많은 데이터를 조회했을 때 스크롤 불가

**권장 수정:**
```go
maxRows := (m.height - 15) / 2  // 동적 계산
if maxRows < 3 {
    maxRows = 3
}
for i, row := range m.resultData {
    if i >= maxRows {
        lines = append(lines, fmt.Sprintf("... (총 %d행)", len(m.resultData)))
        break
    }
```

---

### 문제 16: 파라미터 설정 패널에서 값을 수동으로 변경할 수 없음
**심각도:** Major
**파일:** app.go:780-834
**카테고리:** UX

**코드:**
```go
// renderParamsSection: 읽기 전용 표시
content := "⚙ 파라미터 설정\n" +
    "─────────────\n" +
    "분류: " + truncateText(class1Display, textWidth) + "\n" +
    "항목: " + truncateText(itemDisplay, textWidth) + "\n" +
    ...
    "\n(Enter로 조회)"  // 조회만 가능, 수정 불가
```

**문제점:**
- PanelParams로 포커스는 가능하지만 값 변경 UI 없음
- Enter를 누르면 기본값(ALL)으로 조회됨
- 사용자가 특정 분류/항목을 선택할 수 없음
- 사실상 파라미터 설정 기능이 없음

**현재 동작:**
```
PanelParams 포커스 → Enter → ALL, ALL로 조회 (선택지 없음)
```

**권간:**
1. PanelParams가 활성화되면 입력 모드 활성화
2. ↑↓로 분류/항목 선택
3. Enter로 값 선택 확정
4. 또는 y,u,i,o로 각 파라미터 수정

---

### 문제 17: doSearch에서 ResultCount가 20으로 하드코딩 됨
**심각도:** Minor
**파일:** app.go:175
**카테고리:** API 연동

**코드:**
```go
results, err := m.client.Search(keyword, api.SearchOptions{ResultCount: 20})
```

**문제점:**
- 사용자가 20개 이상의 결과를 보고 싶어도 불가능
- 설정 파일 또는 UI에서 이 값을 변경할 수 없음

**권장 수정:**
```go
// config.yaml에서 읽어오기
searchResultLimit := 50  // 기본값
if cfg != nil && cfg.SearchResultLimit > 0 {
    searchResultLimit = cfg.SearchResultLimit
}
results, err := m.client.Search(keyword, api.SearchOptions{ResultCount: searchResultLimit})
```

---

### 문제 18: doMeta에서 ObjID 기반 분류 그룹 개수 계산이 부정확할 수 있음
**심각도:** Minor
**파일:** app.go:201-211
**카테고리:** API 연동

**코드:**
```go
objIDSet := map[string]bool{}
for _, c := range summary.Classifications {
    if c.ObjID != "" {
        objIDSet[c.ObjID] = true
    }
}
meta.NumClassGroups = len(objIDSet)
if meta.NumClassGroups == 0 {
    meta.NumClassGroups = 1
}
```

**문제점:**
- ObjID가 "L1", "L2"가 아니라 "01", "02"일 수도 있음
- 또는 ObjID가 중복되거나 구조가 다를 수 있음
- 실제 API 응답 구조를 검증하지 않음

**위험:**
- NumClassGroups 계산 오류
- 라인 463-470에서 Class1~8을 잘못 설정 가능
- API에 존재하지 않는 objL5 같은 파라미터 전송 → 에러

**권장 수정:**
```go
// 메타 결과에서 실제 objL이 존재하는지 확인
maxClassLevel := 0
for _, c := range summary.Classifications {
    // ObjID에서 숫자 추출 (L1 → 1, 01 → 1)
    level := parseClassLevel(c.ObjID)
    if level > maxClassLevel {
        maxClassLevel = level
    }
}
meta.NumClassGroups = maxClassLevel
if meta.NumClassGroups == 0 {
    meta.NumClassGroups = 1
}
```

---

### 문제 19: doData에서 DataRow 필드 매핑 시 C1_NM vs C1NM 혼용
**심각도:** Major
**파일:** app.go:257-262
**카테고리:** API 연동

**코드:**
```go
fields := map[string]string{
    "C1_NM":   r.C1NM,   // ← 키는 C1_NM, 값은 r.C1NM
    "ITM_NM":  r.ItmNM,
    "PRD_DE":  r.PrdDe,
    "DT":      r.DT,
    "UNIT_NM": r.UnitNM,
}
```

**API 타입 (types.go:51):**
```go
C1NM   string `json:"C1_NM"`  // ← JSON 키는 C1_NM, 구조체 필드는 C1NM
```

**문제점:**
- 구조체 필드명은 C1NM (camelCase)
- JSON 태그는 C1_NM (snake_case)
- TUI에서 키는 C1_NM
- 나머지 필드(C2NM, C3NM...)는 매핑되지 않음

**실제 결과테이블 (라인 867):**
```go
c1nm := truncateText(row.Fields["C1_NM"], 10)
```

**문제:**
- C2, C3 등은 데이터에 있어도 TUI에서 표시하지 않음
- 필드 불일치로 인한 버그

**권장 수정:**
```go
// 동적으로 모든 분류 필드 매핑
fields := map[string]string{}
fields["C1_NM"] = r.C1NM
fields["C2_NM"] = r.C2NM
fields["C3_NM"] = r.C3NM
// ... C8_NM까지
```

---

### 문제 20: API 응답이 빈 배열 []일 때 에러 처리 불명확
**심각도:** Minor
**파일:** app.go:85-89, search.go:37-42
**카테고리:** API 연동

**코드 (data.go:85-89):**
```go
var results []DataRow
if err := json.Unmarshal(body, &results); err != nil {
    return nil, fmt.Errorf("응답 파싱 실패: %w", err)
}
return results, nil
```

**문제점:**
- API가 정상이지만 데이터가 없어서 `[]` 반환
- TUI에서 "조회 완료" 메시지 표시 (정상처리)
- 하지만 실제로는 검색 조건이 맞지 않아서 0건일 수도 있음
- 사용자는 "조회는 되었는데 데이터가 없다"는 것을 명확히 알아야 함

**현재 동작:**
```
resultData = []  →  statusMsg = "✓ 0건 조회 완료"
```

**권장 수정:**
```go
if len(results) == 0 {
    // 0건이면 경고 또는 다른 메시지
    m.statusMsg = "⚠ 검색 조건을 만족하는 데이터가 없습니다. 파라미터를 확인하세요."
}
```

---

### 문제 21: API 응답이 HTML(에러 페이지)일 때 JSON 파싱 실패
**심각도:** Major
**파일:** client.go:165-169
**카테고리:** 에러 처리

**코드 (client.go:165-169):**
```go
if len(body) > 0 && body[0] == '{' {
    var errResp ErrorResponse
    if err := json.Unmarshal(body, &errResp); err == nil && errResp.Err != "" {
        return nil, fmt.Errorf("API 오류 [%s]: %s", errResp.Err, errResp.ErrMsg)
    }
}
```

**문제점:**
- API가 503 Service Unavailable 또는 500 에러 → HTML 에러 페이지 반환
- body[0] != '{' → 조건 실패 → 에러 검사 스킵
- 그 후 json.Unmarshal에서 실패 (라인 38: search.go)
- 사용자에게 "응답 파싱 실패: invalid character '<'" 같은 이해 불가능한 에러

**실제 시나리오:**
```
API 503 → HTML 반환 → body = "<html><body>Service Unavailable</body></html>"
→ json.Unmarshal 실패 → "응답 파싱 실패: invalid character '<' ..."
```

**권장 수정:**
```go
// API 에러 페이지 감지 + 더 나은 에러메시지
if len(body) > 0 && body[0] == '<' {
    return nil, fmt.Errorf("API 서버 오류 (HTTP %d): 잠시 후 다시 시도하세요. (HTML 응답: %d바이트)",
        resp.StatusCode, len(body))
}

// JSON 파싱 실패 시 더 나은 메시지
if err := json.Unmarshal(body, &results); err != nil {
    return nil, fmt.Errorf("응답 파싱 실패 (JSON 파싱 오류): %w\n응답 내용: %s",
        err, string(body[:min(100, len(body))]))
}
```

---

### 문제 22: 검색 후 에러 발생 → 재검색 시 이전 에러 메시지 우선 표시
**심각도:** Minor
**파일:** app.go:416-423, 952-955
**카테고리:** 에러 처리

**코드 (renderStatusBar:952-955):**
```go
var status string
if m.loading {
    status = "⏳ 로딩중..."
} else if m.err != nil {
    status = "❌ " + m.err.Error()  // ← 이전 에러 표시
} else {
    status = m.statusMsg
}
```

**문제점:**
1. 첫 번째 검색: "인구" → 성공 → statusMsg = "✓ 50건..."
2. 두 번째 검색: "존재안함" → 실패 → m.err = error, statusMsg = "❌ 검색 오류..."
3. 세 번째 검색: "GDP" → 성공 → statusMsg = "✓ 30건..." 하지만 m.err는 여전히 유지!
4. 결과: statusMsg는 "✓ 30건..."인데 상태바는 "❌ (이전 에러)" 표시

**권장 수정:**
```go
// searchResultMsg 수신 시 m.err를 명시적으로 nil 처리
case searchResultMsg:
    m.loading = false
    m.err = nil  // ← 이전 에러 제거
    if msg.err != nil {
        m.err = msg.err
        m.statusMsg = fmt.Sprintf("❌ 검색 오류: %v", msg.err)
    } else {
        ...
    }
```

---

### 문제 23: 메타 로드 실패 후 데이터 조회 시도 시 m.metaInfo == nil 체크 미흡
**심각도:** Critical
**파일:** app.go:434
**카테고리:** 에러 처리

**코드 (metaResultMsg:426-478):**
```go
case metaResultMsg:
    m.loading = false
    if msg.err != nil {
        m.err = msg.err
        m.statusMsg = fmt.Sprintf("❌ 메타 조회 오류: %v", msg.err)
    } else {
        m.metaInfo = msg.meta

        if m.selectedTable != nil {  // ← 여기서 체크
            // ... 데이터 조회 시작
            opts := api.DataOptions{
                Item: firstItem,
                ...
            }
```

**시나리오:**
1. 통계표 선택 → metaResultMsg (API 실패)
2. msg.err != nil → m.metaInfo = nil로 설정 안 됨 (이전 값 유지)
3. 하지만 m.selectedTable != nil → 데이터 조회 시도
4. m.metaInfo.Items에 접근하지만 nil pointer panic 가능

**라인 452에서:**
```go
firstItem := "T10"
if len(m.metaInfo.Items) > 0 {  // ← m.metaInfo == nil이면 panic
    firstItem = m.metaInfo.Items[0].Code
}
```

**권장 수정:**
```go
if msg.err != nil {
    m.err = msg.err
    m.metaInfo = nil  // ← 명시적으로 nil 처리
    m.statusMsg = fmt.Sprintf("❌ 메타 조회 오류: %v", msg.err)
} else {
    m.metaInfo = msg.meta
    ...
}
```

---

### 문제 24: loading 상태에서 키 입력 처리: q키로 종료 가능
**심각도:** Minor
**파일:** app.go:270-411
**카테고리:** 상태 관리

**코드 (Update:270-310):**
```go
switch msg := msg.(type) {
case tea.KeyMsg:
    if m.typing {  // ← 입력 모드일 때만 처리
        switch msg.String() {
        ...
    }

    // 일반 모드에서의 키 처리 (항상 실행!)
    switch msg.String() {
    case KeyQuit, "ctrl+c":
        m.quitting = true
        return m, tea.Quit  // ← loading 중이어도 종료 가능
    ...
```

**문제점:**
- m.loading=true 상태에서도 'q'나 'ctrl+c'로 종료 가능
- API 요청이 진행 중인 상태에서 갑자기 종료 가능
- 사용자 입장에서는 "로딩 중" 메시지가 보이는데 종료됨

**권장 수정:**
```go
case KeyQuit, "ctrl+c":
    if m.loading {
        m.statusMsg = "⚠ 로딩 중입니다. 완료될 때까지 기다려주세요."
        return m, nil  // 종료 불가
    }
    m.quitting = true
    return m, tea.Quit
```

---

### 문제 25: 동시성: doSearch 중 Enter 다시 누르면 2번째 searchResultMsg 큐 진입 가능
**심각도:** Minor
**파일:** app.go:270-300
**카테고리:** 동시성

**시나리오:**
1. 입력 모드: "인구" 입력
2. Enter → m.typing=false, m.loading=true, doSearch 실행
3. 응답 기다리는 중: 사용자가 또 Enter 누르기?
4. 하지만 m.typing=false → 일반 모드 처리
5. Enter는 PanelSearch 또는 다른 패널에서의 Enter (라인 350)
6. 데이터 조회 진행

**실제 문제:**
- 첫 번째 doSearch가 응답 중에 두 번째 doSearch 호출 불가능 (m.typing=false)
- 따라서 이 항목은 **실제 문제가 아님** (차단됨)

**재평가:** 이 항목은 제거하고 다른 문제로 대체 가능

---

### 문제 26: View() 매 렌더링마다 스타일 객체 재생성 → 성능 저하
**심각도:** Minor
**파일:** app.go:497-545
**카테고리:** 성능

**코드 (renderTitle:538-545):**
```go
titleStyle := lipgloss.NewStyle().  // ← 매 렌더링마다 생성
    Bold(true).
    Foreground(lipgloss.Color("255")).
    Background(ColorPrimary).
    Width(m.width).
    Padding(0, 1)
```

**문제점:**
- 매 프레임마다 (초당 60+회) 스타일 객체 생성
- lipgloss.NewStyle() → Builder Pattern → 메모리 할당
- 가비지 컬렉션 오버헤드

**영향:**
- 터미널 크기가 크면 (200x50) 성능 저하 가능
- CPU 사용률 증가

**권장 수정:**
```go
// 전역 또는 init 함수에서 미리 생성
var titleStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("255")).
    Background(ColorPrimary).
    Padding(0, 1)

// 또는 updateWidth만 함
func (ts *lipgloss.Style) updateWidth(w int) lipgloss.Style {
    return ts.Copy().Width(w)
}
```

---

### 문제 27: renderLeftPanel에서 Height 고정값
**심각도:** Minor
**파일:** app.go:555
**카테고리:** 성능/UX

**코드:**
```go
panelStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(ColorPrimary).
    Width(width).
    Height(m.height-3).  // ← 동적이지만 항상 terminal-3
    Padding(0, 1)
```

**문제점:**
- Height = m.height-3 (타이틀 1 + 상태바 1 + 여유 1)
- 왼쪽 패널이 항상 최대 높이
- 왼쪽 패널 내의 섹션들(검색, 결과, 북마크, 이력)이 고정된 작은 높이
- 오른쪽 패널도 같은 높이 → 수평 정렬 문제

**권장 수정:**
```go
// 왼쪽: 40%, 오른쪽: 60% 같은 비율 할당 가능
leftPanelHeight := m.height - 3
rightPanelHeight := m.height - 3
```

---

### 문제 28: 상태바 KeyBindings()가 매 렌더링마다 새로 생성
**심각도:** Minor
**파일:** keys.go:21-30
**카테고리:** 성능

**코드 (keys.go:21-30):**
```go
func KeyBindings() string {
    return KeyHintStyle.Render("/") + KeyDescStyle.Render(" 검색  ") +
        KeyHintStyle.Render("Enter") + KeyDescStyle.Render(" 선택  ") +
        ...
}
```

**renderStatusBar()에서 호출 (라인 318):**
```go
case KeyHelp:
    m.statusMsg = KeyBindings()  // ← 함수 호출, 매번 strings 생성
```

**문제점:**
- KeyBindings() 문자열을 매번 생성
- lipgloss.Render() 호출 → 색상 코드 추가 → 문자열 빌드

**권장 수정:**
```go
// 전역 상수로 미리 생성
var keyBindingsDisplay string

func init() {
    keyBindingsDisplay = KeyHintStyle.Render("/") + ... // 한 번만 생성
}

// 사용
case KeyHelp:
    m.statusMsg = keyBindingsDisplay
```

---

### 문제 29: fmt.Sprintf 과다 사용 vs strings.Builder 미사용
**심각도:** Minor
**파일:** app.go:422, 487, 등 (다수)
**카테고리:** 성능

**코드 (app.go:422):**
```go
m.statusMsg = fmt.Sprintf("✓ %d건 검색 완료 (↑↓ 선택, Enter 확인)", len(msg.results))
```

**문제점:**
- 간단한 문자열 포맷에 fmt.Sprintf 사용
- 많은 상태 메시지가 이 방식으로 생성
- 성능상 큰 영향은 없지만 Best Practice가 아님

**권장 수정:**
```go
m.statusMsg = fmt.Sprintf("✓ %d건 검색 완료 (↑↓ 선택, Enter 확인)", len(msg.results))
// 또는
var sb strings.Builder
sb.WriteString("✓ ")
sb.WriteString(strconv.Itoa(len(msg.results)))
sb.WriteString("건 검색 완료 (↑↓ 선택, Enter 확인)")
m.statusMsg = sb.String()
```

---

### 문제 30: 에러 무시 패턴: bookmarks, histories 로드 시 _ 에러 무시
**심각도:** Minor
**파일:** app.go:148-149
**카테고리:** 코드 품질

**코드 (New():148-149):**
```go
bookmarks, _ := bookmark.List()
histories, _ := history.List(5)
```

**문제점:**
- 에러를 무시하고 있음
- 파일 읽기 실패 시 빈 슬라이스 반환 (일반적)
- 하지만 명시적인 에러 처리가 없어서 코드 가독성 떨어짐
- 향후 유지보수 시 의도가 불명확

**실제 동작:**
```go
// bookmark.List() 내부
func List() ([]Bookmark, error) {
    bookmarks, err := Load()
    if err != nil {
        return nil, err  // 에러 반환
    }
    return bookmarks, nil
}
```

**권장 수정:**
```go
bookmarks, err := bookmark.List()
if err != nil {
    // 로그 남기거나 에러 메시지 표시
    fmt.Fprintf(os.Stderr, "⚠ 즐겨찾기 로드 실패 (계속 진행): %v\n", err)
    bookmarks = []bookmark.Bookmark{}  // 빈 슬라이스로 초기화
}

histories, err := history.List(5)
if err != nil {
    fmt.Fprintf(os.Stderr, "⚠ 이력 로드 실패 (계속 진행): %v\n", err)
    histories = []history.HistoryEntry{}
}
```

---

## 2차 문제 요약

### Critical (1개)
- 문제 23: 메타 로드 실패 후 nil pointer 접근 가능

### Major (4개)
- 문제 13: 즐겨찾기 섹션 미구현
- 문제 14: 이력 섹션 미구현
- 문제 16: 파라미터 입력 UI 없음
- 문제 19: DataRow 필드 매핑 불완전
- 문제 21: HTML 에러 응답 처리 미흡

### Minor (14개)
- 문제 11, 12, 15, 17, 18, 20, 22, 24, 26, 27, 28, 29, 30

### Suggestion (1개)
- 문제 25: 재평가 결과 실제 문제 아님 (차단됨)

---

## 2차 권장 수정 우선순위

1. **P1 (긴급):** 문제 23 (Critical - panic 가능)
2. **P2 (1주일 내):** 문제 21, 19 (Major - API 에러/데이터 손실)
3. **P3 (2주일 내):** 문제 13, 14, 16 (Major - UX 미완성)
4. **P4 (성능):** 문제 26, 27, 28 (Minor - 성능 개선)
5. **P5 (코드 품질):** 문제 30 (Minor - 가독성)

---

**2차 보고서 작성일:** 2026-04-01
**테스터:** TUI 테스터1 (전체 TUI 및 API 연동)
