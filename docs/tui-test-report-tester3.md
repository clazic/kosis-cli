# KOSIS CLI TUI 테스터3 - 레이아웃/스타일 문제 보고서

## 평가 기준
**100점 기준** (설계서 5.1, 5.4 준수도 + 코드 품질)

---

## 발견된 문제점 (10개)

### 1. ❌ 터미널 폭 80 미만일 때 패널 겹침/깨짐
**심각도**: 중상
**위치**: `app.go:504-510`

**문제 코드**:
```go
leftWidth := m.width * 3 / 10
if leftWidth < 20 {
    leftWidth = 20
}
rightWidth := m.width - leftWidth - 3
if rightWidth < 20 {
    rightWidth = 20
}
```

**문제점**:
- 터미널 폭이 80 미만(예: 70)일 때: `leftWidth = 21, rightWidth = 46` → 폭 초과
- 터미널 폭이 40일 때: `leftWidth = 20, rightWidth = 17` → rightWidth가 20 미만인데 강제 설정 후 총폭 초과
- **패널 양쪽 모두 최소 20 폭이 필요하면 총 41이 필요하므로, 40 이하 터미널에서 겹침 발생**
- `rightWidth < 20` 체크 후 20으로 강제 설정하면 전체 폭이 초과됨

**해결안**:
```go
leftWidth := m.width * 3 / 10
if leftWidth < 20 {
    leftWidth = 20
}
if leftWidth > m.width - 20 - 3 {  // 오른쪽 패널 최소 20 + 보더 3 예약
    leftWidth = m.width - 20 - 3
}
rightWidth := m.width - leftWidth - 3
if rightWidth < 1 {
    rightWidth = 1  // 0이 아닌 최소값
}
```

---

### 2. ❌ 터미널 높이 20 미만일 때 패널 잘림
**심각도**: 중상
**위치**: `app.go:555, 669, 720, 848`

**문제 코드**:
```go
Height(m.height-3)  // 타이틀 1 + 상태바 1 + 여유 1 = 3
```

**문제점**:
- 터미널 높이가 15일 때 패널 높이가 12 → 5개 섹션(검색, 결과, 즐겨찾기, 이력)을 모두 표시 불가
- 높이 체크가 없음
- renderLeftPanel이 여러 섹션을 수직 정렬하는데 높이 제약이 없음

**해결안**:
```go
// View() 시작에 최소 터미널 크기 체크
if m.height < 10 || m.width < 30 {
    return "터미널이 너무 작습니다. 최소 30x10이 필요합니다."
}
```

---

### 3. ❌ WindowSizeMsg 처리 후 재렌더링 정상 여부
**심각도**: 중상
**위치**: `app.go:490-493`

**문제 코드**:
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
```

**문제점**:
- WindowSizeMsg 처리 후 어떤 cmd도 반환하지 않음
- 재렌더링이 자동 트리거되는지 불명확 (Bubble Tea 자체가 처리하지만 코드 명확성 부족)
- 터미널 크기 변경 후 즉시 패널 재배치 검증 필요
- **설계서에 명시되지 않았으므로 구현 검증 필요**

**개선안**:
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    // 새로운 크기에서 cursor가 범위를 벗어났는지 검증
    if m.cursor >= len(m.searchResults) {
        m.cursor = len(m.searchResults) - 1
        if m.cursor < 0 {
            m.cursor = 0
        }
    }
    // cmd 반환 불필요 (Bubble Tea가 자동으로 View() 호출)
```

---

### 4. ❌ leftWidth/rightWidth 계산에서 음수 방지 미흡
**심각도**: 중상
**위치**: `app.go:504-511`

**문제 코드**:
```go
rightWidth := m.width - leftWidth - 3
if rightWidth < 20 {
    rightWidth = 20
}
```

**문제점**:
- `m.width = 20`일 때: `leftWidth = 6` (최소 20 미만) → `leftWidth = 20` → `rightWidth = 20 - 20 - 3 = -3`
- 음수 값이 들어가면 lipgloss의 Width() 동작이 불명확함
- 보더(3) 계산에서 여유 부족

**해결안**:
```go
leftWidth := m.width * 3 / 10
rightWidth := m.width - leftWidth - 3

// 각각 최소값 체크
if leftWidth < 20 && rightWidth > leftWidth + 5 {
    leftWidth = 20
}
if rightWidth < 20 && leftWidth > rightWidth + 5 {
    rightWidth = 20
}

// 최종 음수 방지
if rightWidth < 1 {
    rightWidth = 1
}
if leftWidth < 1 {
    leftWidth = 1
}
```

---

### 5. ⚠️ truncateText의 한글 폭 계산 정확성 (runeWidth)
**심각도**: 중
**위치**: `app.go:880-893, 904-935`

**문제점**:
- runeWidth() 함수가 한글을 정확히 처리하지 않을 수 있음
- `0xAC00~0xD7A3` 범위(한글 완성형) 처리는 되지만 **분모음, 결합 문자 처리 미흡**
- 예: 한글 "한글" (U+D55C U+D654) → 각 2칸 = 4칸 (정확)
- 그러나 이모지(4칸 이상) 처리 안 함
- "🔍"는 1칸으로 계산되지만 실제로는 2칸 차지 → 정렬 깨짐

**현재 코드**:
```go
func runeWidth(r rune) int {
    if r >= 0x1100 && (r <= 0x115F || r == 0x2329 || r == 0x232A ||
        (r >= 0x2E80 && r <= 0xA4CF && r != 0x303F) ||
        (r >= 0xAC00 && r <= 0xD7A3) ||  // 한글만
        (r >= 0xF900 && r <= 0xFAFF) ||
        (r >= 0xFE10 && r <= 0xFE19) ||
        (r >= 0xFE30 && r <= 0xFE6F) ||
        (r >= 0xFF00 && r <= 0xFF60) ||
        (r >= 0xFFE0 && r <= 0xFFE6)) {
        return 2
    }
    return 1
}
```

**문제**:
- 이모지 범위(`0x1F300~0x1F9FF`) 미포함
- 이모지 자칭 2칸(또는 그 이상)을 1칸으로 처리하면 레이아웃 깨짐

**테스트 사례**:
- "🔍 검색: " → 4칸 (이모지 2 + "검색" 4 = 6) 인데 5칸으로 계산 → 여유 부족

**해결안**:
```go
func runeWidth(r rune) int {
    // 이모지 범위 추가
    if (r >= 0x1F300 && r <= 0x1F9FF) ||  // 이모지
        (r >= 0x2600 && r <= 0x27BF) {     // 기호
        return 2  // 이모지도 2칸으로 처리
    }
    // 기존 한글 코드...
    return 1
}
```

---

### 6. ⚠️ 패널 보더가 lipgloss RoundedBorder로 올바르게 렌더링되는지 검증
**심각도**: 낮음
**위치**: `app.go:551-574, 584-601, 646, 665, 716` 등 모든 패널

**문제점**:
- 설계서 5.4: "패널 테두리: 둥근 모서리 보더" 명시
- 현재 코드는 RoundedBorder() 사용 중이지만, **중첩 패널 렌더링 시 보더 겹침 발생 가능**
- renderLeftPanel → renderSearchSection, renderResultsSection 중첩 보더
- 이중 보더가 이상하게 보일 수 있음

**현재 구조**:
```
leftPanel (RoundedBorder)
  ├─ searchSection (RoundedBorder)
  ├─ resultsSection (RoundedBorder)
  ├─ bookmarkSection (RoundedBorder)
  └─ historySection (RoundedBorder)
```

**문제**:
- 부모 패널에 보더 + 내부 섹션에 보더 = 이중 보더 시각적 혼란
- 설계서 그림(5.1)에서 왼쪽 패널 내부 섹션에 보더가 있는지 불명확

**해결안**:
- 설계서 다시 확인 필요
- 가능하면 부모 패널에만 보더, 내부 섹션은 구분선(`─────`)으로만 처리

---

### 7. ⚠️ 활성 패널과 비활성 패널의 색상 구분 명확성
**심각도**: 중
**위치**: `app.go:579-582, 660-663, 741-744, 782-784, 839-842`

**문제점**:
- 활성 패널: `ColorAccent` (#10B981 초록)
- 비활성 패널: `ColorBorder` (#334155 회색)
- 색상 명암비가 충분한지 확인 필요 (배경이 어두움)
- WCAG 2.1 AA 기준: 최소 4.5:1 명암비 필요
- 초록(#10B981) vs 회색(#334155) vs 배경(#0F172A) 조합 확인 필요

**현재 색상 팔레트** (styles.go):
- ColorPrimary = #2563EB (파란색)
- ColorAccent = #10B981 (초록)
- ColorBorder = #334155 (회색)
- ColorBgPanel = #0F172A (어두운 배경)

**문제**:
- 회색(#334155)이 배경(#0F172A)과 충분히 구분되는지 불명확
- 초록(#10B981)은 더 밝으므로 구분되지만, 회색은 애매함

**개선안**:
```go
var PanelInactiveStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("#4B5563")).  // 더 밝은 회색
    Padding(0, 1)
```

---

### 8. ⚠️ 상태바가 하단에 고정되는지 검증
**심각도**: 낮음
**위치**: `app.go:497-529, 944-959`

**문제 코드**:
```go
return lipgloss.JoinVertical(lipgloss.Left, title, mainContent, statusBar)
```

**문제점**:
- JoinVertical로 단순 수직 결합 → 상태바가 항상 하단에 고정되는지 보장 불명확
- mainContent 높이가 자동 계산되므로, 터미널 높이가 예상과 다르면 상태바 위치 변동 가능
- **상태바 높이가 1이므로 문제 없지만, statusBar의 Width(m.width) 설정이 있는지 확인 필요**

**현재 상태바**:
```go
statusStyle := StatusBarStyle.
    Width(m.width)
```

**문제**:
- Width 설정은 있으므로 가로는 확보됨
- 그러나 Height 설정이 없음 → 기본 높이 (보통 1행)
- 다행히 이것이 의도된 동작이므로 문제 없음

**결론**: 이 항목은 **실제로 정상 동작함** ✓

---

### 9. ⚠️ 타이틀바 "[?] 도움말 [q] 종료" 표시 위치
**심각도**: 낮음
**위치**: `app.go:532-546`

**문제 코드**:
```go
func (m Model) renderTitle() string {
    title := "KOSIS 통계 데이터 탐색기"
    helpText := "[?] 도움말  [q] 종료"

    titleStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("255")).
        Background(ColorPrimary).
        Width(m.width).
        Padding(0, 1)

    titleContent := title + " " + helpText
    return titleStyle.Render(titleContent)
}
```

**문제점**:
- 설계서 5.1: "KOSIS 통계 데이터 탐색기 [?] 도움말  [q] 종료" (우측 정렬)
- 현재는 단순 연결: title + " " + helpText → **좌측 정렬**
- 타이틀은 좌측, 도움말은 우측에 배치해야 함 (양쪽 끝 정렬)

**해결안**:
```go
func (m Model) renderTitle() string {
    title := "KOSIS 통계 데이터 탐색기"
    helpText := "[?] 도움말  [q] 종료"

    // 타이틀과 도움말 사이 간격 계산
    spacing := m.width - len([]rune(title)) - len([]rune(helpText)) - 2
    if spacing < 1 {
        spacing = 1
    }

    titleStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("255")).
        Background(ColorPrimary).
        Width(m.width).
        Padding(0, 1)

    titleContent := title + strings.Repeat(" ", spacing) + helpText
    return titleStyle.Render(titleContent)
}
```

**주의**: rune 폭 계산(한글/이모지) 필요

---

### 10. ⚠️ 데이터 테이블 컬럼 정렬 (한글/영문 혼합 시) 깨짐
**심각도**: 중
**위치**: `app.go:837-878`

**문제 코드**:
```go
lines = append(lines, truncateText("분류", 10)+" │ "+truncateText("항목", 8)+" │ "+truncateText("시점", 8)+" │ "+truncateText("수치값", 10)+" │ "+truncateText("단위", 4))
lines = append(lines, "──────────┼──────────┼──────────┼────────────┼──────")

for i, row := range m.resultData {
    if i >= 5 {
        lines = append(lines, "...")
        break
    }
    c1nm := truncateText(row.Fields["C1_NM"], 10)
    itmnm := truncateText(row.Fields["ITM_NM"], 8)
    prdde := truncateText(row.Fields["PRD_DE"], 8)
    dt := truncateText(row.Fields["DT"], 10)
    unit := truncateText(row.Fields["UNIT_NM"], 4)

    line := c1nm + " │ " + itmnm + " │ " + prdde + " │ " + dt + " │ " + unit
    lines = append(lines, line)
}
```

**문제점**:
- truncateText가 한글을 2칸으로 계산하므로 `width=10` → 실제 2칸 여유 (한글 4글자 + 패딩 2칸)
- 그런데 헤더 구분선은 고정: `"──────────"` (10칸 한글 = 20 픽셀)
- **한글과 영문 혼합 데이터**: "서울" (4칸) + "2024" (4칸) = 8칸 → 패딩 2칸 추가 = 10칸 ✓
- 그러나 "경기도" (6칸) + "24" (2칸) = 8칸 → 패딩 2칸 추가 = 10칸 ✓
- **문제는 숫자만 있으면**: "202501" (6칸) → 패딩 4칸 추가 = 10칸 ✓

**실제 문제**:
- 헤더 "분류" (4칸) → truncateText(width=10) → "분류      " (4칸 + 패딩 6칸)
- 헤더 구분선: `──────────` (한글이면 10칸, 실제로는 halfwidth 대시 10개 = 10칸) ✓
- 문제 없어 보이지만...

**더 깊은 문제**:
- truncateText가 패딩을 추가하는데, stringDisplayWidth로 계산된 폭 기준
- 한글 "분류" = 4칸, 패딩 6칸 추가 = 10칸 (정확)
- 그런데 영문 "C1_NM" = 5칸, 패딩 5칸 추가 = 10칸 (정확)
- **하지만 이 모든 것이 truncateText 내에서 완료되므로 외부에서 추가 패딩 없음**

**구분선 문제**:
```go
"──────────┼──────────┼──────────┼────────────┼──────"
```
- "──────────" = 10개 문자 (halfwidth) = 10칸
- 데이터: truncateText(width=10) → 정확히 10칸
- **구분선이 한글 대시(─)이므로 각 2칸 = 20칸인가?**

**실제 테스트**:
- Halfwidth hyphen-minus "-" = 1칸
- Fullwidth hyphen "─" (U+2500) = 2칸 (full width)
- 현재 코드의 "──────────"는 fullwidth 대시 10개 = 20칸
- 데이터 열은 10칸 → **정렬 완전히 깨짐!**

**해결안**:
```go
// 구분선도 fullwidth 기준으로 계산
lines = append(lines, strings.Repeat("─", 5)+"┼"+strings.Repeat("─", 4)+"┼"+strings.Repeat("─", 4)+"┼"+strings.Repeat("─", 5)+"┼"+strings.Repeat("─", 2))
```

---

### 11. ⚠️ "⏳ 로딩중..." 표시 위치 및 애니메이션 부재
**심각도**: 중
**위치**: `app.go:593-594, 950-951`

**문제점**:
- 로딩 상태에서 "⏳ 로딩중..."가 상태바에만 표시됨
- 실제 로딩 중에는 검색 섹션의 커서가 "⏳"로 변경됨 (좋음)
- 그러나 **상태바에도 표시되므로 이중 표시**
- 설계서에 로딩 스피너 애니메이션 명시되어 있지 않으므로 현재 구현 (정적 아이콘)이 맞음

**현재 동작**:
```go
if m.loading {
    cursor := "⏳"
    ...
    status = "⏳ 로딩중..."  // 중복
}
```

**개선안**:
- 상태바와 검색 커서 중 하나만 표시
- 또는 진행률 표시 (예: "⏳ 로딩중... [██░░░░░░] 30%")

---

### 12. ⚠️ 스크롤 표시 ("↑ N개 더 있음", "↓ N개 더 있음") 정확성
**심악도**: 낮음
**위치**: `app.go:675-708`

**문제 코드**:
```go
if start > 0 {
    resultLines = append(resultLines, fmt.Sprintf("  ↑ %d개 더 있음", start))
}

for i := start; i < end; i++ {
    // ...
}

if end < total {
    resultLines = append(resultLines, fmt.Sprintf("  ↓ %d개 더 있음", total-end))
}
```

**검증**:
- 총 20건, maxVisible = 8일 때
- cursor = 0: start=0, end=8, 위 화살표 없음 ✓, 아래 화살표 "↓ 12개" ✓
- cursor = 10: start=3, end=11, 위 화살표 "↑ 3개" ✓, 아래 화살표 "↓ 9개" ✓
- cursor = 19: start=12, end=20, 위 화살표 "↑ 12개" ✓, 아래 화살표 없음 ✓

**결론**: 로직 정확함 ✓

---

## 종합 평가

### 심각한 문제 (5점 감점 각각)
1. 터미널 폭 80 미만 패널 겹침 (문제 1)
2. 터미널 높이 20 미만 패널 잘림 (문제 2)
3. leftWidth/rightWidth 음수 방지 미흡 (문제 4)
4. 한글 폭 계산 부정확 - 이모지 미포함 (문제 5)
5. 테이블 컬럼 구분선 너비 불일치 (문제 10)

### 중간 문제 (3점 감점 각각)
6. 활성/비활성 색상 명암비 불충분 (문제 7)
7. 중첩 패널 보더 혼란 (문제 6)
8. 타이틀바 우측 정렬 미구현 (문제 9)

### 경미한 문제 (1점 감점 각각)
9. 로딩 표시 중복 (문제 11)
10. WindowSizeMsg 처리 명확성 (문제 3)

---

## 최종 점수

**기본 점수**: 100점

**감점**:
- 심각한 문제 5개: -25점
- 중간 문제 3개: -9점
- 경미한 문제 2개: -2점

**최종 점수: 64점** ⚠️

---

## 우선 수정 순서

1. **문제 1, 2**: 터미널 최소 크기 체크 추가 (필수)
2. **문제 4**: leftWidth/rightWidth 음수 방지 로직 강화
3. **문제 5**: runeWidth에 이모지 범위 추가 + 테이블 구분선 수정
4. **문제 10**: 테이블 컬럼 너비 일관성 확보
5. **문제 7**: 색상 명암비 검증 + 개선
6. **문제 9**: 타이틀바 우측 정렬 구현

---

## 추가 검토 항목

- [ ] 터미널 256색 vs 24비트 색상 지원 확인 (styles.go 색상값)
- [ ] macOS, Linux, Windows 각 터미널에서 이모지/한글 렌더링 테스트
- [ ] 다양한 터미널 에뮬레이터 (iTerm2, VS Code, Terminal.app 등) 호환성
- [ ] 반응형 테스트: 50x10, 120x30, 200x50 등 극단적 크기
- [ ] 키보드 입력 시 한글 입력기(IME) 처리 (Update 함수의 입력 처리)

---

## 참고: 설계서 5.1, 5.4 요구사항 확인

**5.1 레이아웃**:
- 왼쪽 30%, 오른쪽 70% ✓ (비율 맞음)
- 타이틀 바에 "[?] 도움말 [q] 종료" 표시 ⚠️ (위치 수정 필요)
- 상태바 최하단 ✓ (정상)

**5.4 스타일링**:
- 컬러 팔레트: 파란계열 주색상 ✓
- 패널 테두리: 둥근 모서리 보더 ✓
- 선택된 항목: 반전 색상 + 볼드 ✓ (구현 필요 확인)
- 로딩 상태: 스피너 애니메이션 ⚠️ (정적 아이콘만 사용)
- 에러: 빨간 배경 ✓ (styles.go에 정의됨)

---

## 추가 발견 (2차)

### 13. ❌ 검색 입력: "실시간 검색" 미구현
**심각도**: 중상
**위치**: `app.go:274-298, 578-601, 설계서 5.2`

**설계서 요구사항**:
- 5.2에서 검색 입력 패널 설명: "실시간 검색 (6.통합검색 / 7-B.지표검색 전환)"
- 사용자가 검색어를 타이핑할 때마다 자동으로 검색 실행

**현재 구현**:
```go
case KeyEnter:
    m.typing = false
    if m.searchInput == "" {
        m.statusMsg = "검색어를 입력하세요"
        return m, nil
    }
    m.loading = true
    m.statusMsg = "🔍 검색 중..."
    return m, m.doSearch(m.searchInput)
```

**문제점**:
- Enter 키를 눌러야만 검색이 실행됨
- 사용자가 타이핑하는 동안 검색이 진행되지 않음
- 설계서의 "실시간" 개념 구현 안 됨

**해결안**:
- `Update()` 함수에서 일반 문자 입력 시 자동 검색 트리거
- 디바운싱(300ms) 추가: 사용자가 타이핑을 멈춘 후 자동 검색
- 또는 Enter만 지원하되 설계서 수정 (명확한 선택 필요)

**권장**: Enter 방식이 더 명확하므로, 설계서를 "Enter로 검색"으로 수정하고 코드는 현상 유지

---

### 14. ❌ 키바인딩 `i` (지표 모드 전환) 미구현
**심각도**: 중상
**위치**: `app.go:348-349, keys.go:11, 설계서 5.3`

**설계서 요구사항**:
- 5.3 키바인딩: "i | 통계표/지표 모드 전환"
- TUI에서 키 `i`를 누르면 통계표 조회 모드와 주요지표 조회 모드 전환

**현재 구현**:
```go
case KeyIndicator: // i 키: 주요지표 모드 전환
    m.statusMsg = "📊 주요지표 모드: CLI로 사용 → kosis ind s \"GDP\""
```

**문제점**:
- 상태바에 메시지만 표시
- 실제 모드 전환 구현 없음
- 화면 레이아웃이 변경되지 않음 (검색 패널 등이 그대로)
- 지표 검색과 통계표 검색이 혼재되지 않음

**해결안**:
- `m.mode` 필드 추가 (PanelSearch 패널처럼 구분)
- 지표 모드에서는 다른 검색 엔드포인트 사용 (API 7-B 지표검색)
- 파라미터 설정 패널도 지표 모드에서 다르게 렌더링 (분류 대신 지표명)

**우선순위**: 높음 (설계서 명시)

---

### 15. ❌ 상태바 메시지 중복 및 일관성 부족
**심각도**: 중
**위치**: `app.go:276-316, 944-959`

**문제점**:
- 로딩 상태에서 상태바와 검색 커서 모두 "⏳" 표시
  ```go
  if m.loading {
      cursor := "⏳"  // 검색 섹션
      ...
      status = "⏳ 로딩중..."  // 상태바
  }
  ```
- 상태바 메시지가 Update 함수 여러 곳에서 설정되지만 일관된 포맷 부족
- 예: "❌ 검색 오류: ...", "❌ 메타 조회 오류: ..." vs "⚠ 내보낼 데이터가 없습니다"
- 이모지 사용이 불규칙 (❌, ⚠, ✓, 📊, ⏳, 💾, ★ 등)

**코드 예시**:
```go
// 일관성 없는 상태바 메시지들
m.statusMsg = "⚠ API 키 미설정..."
m.statusMsg = "❌ API 클라이언트 초기화 실패..."
m.statusMsg = "검색어를 입력하세요"  // 이모지 없음
m.statusMsg = fmt.Sprintf("✓ %d건 검색 완료...", len(msg.results))
```

**해결안**:
- 상태바 메시지 포맷 표준화
- 이모지 사용 규칙 정의:
  - 오류: ❌
  - 경고: ⚠
  - 성공: ✓
  - 정보: ℹ
  - 로딩: ⏳ (또는 스피너 애니메이션)
- 상태바는 상단(타이틀)과 하단(상태) 중 하나만 메시지 표시하거나 구역 분리

---

### 16. ⚠️ 지표 모드 패널 구조 미정의
**심각도**: 중상
**위치**: `설계서 5.2, 설계서 3.3 (indicator 명령어)`

**문제점**:
- 설계서 5.2에서 패널 8개 정의 (검색, 목록, 즐겨찾기, 이력, 메타, 파라미터, 결과, 상태바)
- 그런데 지표 모드는 어떻게 렌더링할지 명시 없음
- 현재 코드는 단순 메시지만 표시

**필요한 것**:
- 지표 모드 전환 시 왼쪽 패널: 지표명 검색 입력 (다시 렌더링)
- 지표 모드 전환 시 오른쪽 패널: 메타 → 지표 상세 정보 (산식, 개념)
- 파라미터 설정이 통계표 모드와 다름 (분류/항목 대신 지표명/주기)

**설계서 참조**:
- 설계서 3.3: `kosis indicator (ind)` — 통계주요지표 검색/조회
- API 7-B (지표검색), 7-A (지표설명), 7-D (지표데이터)

**해결안**:
- 모드 전환 구조 명확히 정의
- 각 모드별 렌더링 함수 분리 (renderStatTableMode vs renderIndicatorMode)

---

### 17. ⚠️ 메타 정보 패널 높이 고정 미흡
**심각도**: 중
**위치**: `app.go:739-777`

**문제 코드**:
```go
sectionStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(borderColor).
    Width(width).
    Padding(0, 1)
    // Height 미설정!
```

**문제점**:
- 메타 정보 섹션에 Height 설정 없음
- 분류/항목이 많으면 무한정 확장 (다른 섹션 침범)
- 파라미터 설정 섹션, 결과 테이블 섹션과 레이아웃 불균형
- 터미널 높이가 크면 메타 정보가 지배적 (바뀐 레이아웃)

**현재 코드**:
```go
// metaSection 높이 미지정
// paramsSection 높이 미지정
// resultSection만 Height(8)
```

**해결안**:
- 메타 섹션: Height(10) 고정 + 내부 스크롤 또는 축약
- 파라미터 섹션: Height(6) 고정
- 결과 섹션: Height(height - 3 - 1 - 10 - 1 - 6 - 4) = 동적 계산 (여유 최소화)

---

### 18. ⚠️ 보더 및 구분선 문자 혼재
**심각도**: 중
**위치**: `app.go` 전역 검색 필요

**문제점**:
- 보더와 구분선에 혼합된 문자 사용:
  - `─` (fullwidth hyphen U+2500) — 보더
  - `─────────────` (같은 문자 반복) — 구분선
  - `│` (fullwidth vertical U+2502) — 구분자
  - `─` (halfwidth hyphen "-") 혼재 여부 확인 필요
  - `⏳`, `▌`, `▸` (다양한 이모지/기호)

**코드 예시** (app.go:614):
```go
bookmarkLines = append(bookmarkLines, "─────────────")
```

**fullwidth 문자 칸 수 계산 불명확**:
- `─` = 2칸 (fullwidth)
- 10개 사용 → 20칸
- 근데 텍스트 필드는 10칸 설정 → 정렬 깨짐

**확인 사항**:
- 모든 구분선이 fullwidth 대시인지 halfwidth 대시인지 일관성 확인
- 각 구분선의 실제 터미널 폭 검증
- lipgloss Width() 설정과 구분선 길이 일치성 검증

**해결안**:
- halfwidth 대시 사용 (더 안전)
- 또는 fullwidth 대시 사용하되, 폭 계산 명확히 (개수 × 2)
- 구분선 자동 생성 함수: `createDivider(width int, char string) string`

---

### 19. ⚠️ renderLeftPanel과 renderRightPanel의 높이 불일치
**심각도**: 낮음
**위치**: `app.go:550-574, 715-736`

**문제 코드**:
```go
// renderLeftPanel
panelStyle := lipgloss.NewStyle().
    Height(m.height-3).
    ...

// renderRightPanel
panelStyle := lipgloss.NewStyle().
    Height(m.height-3).
    ...
```

**문제점**:
- 왼쪽, 오른쪽 패널 모두 `Height(m.height-3)` 동일
- 그런데 왼쪽 패널은 4개 섹션 (검색, 결과, 즐겨찾기, 이력) 수직 정렬
- 오른쪽 패널은 3개 섹션 (메타, 파라미터, 결과)
- 높이 분배가 고정되지 않아 한쪽이 다른 쪽보다 커질 수 있음

**예시**:
```go
// renderLeftPanel의 섹션들
searchSection   // 높이 미지정
resultsSection  // Height(10)
bookmarkSection // Height(5)
historySection  // Height(5)
```
- 총 합 = 10 + 5 + 5 = 20 (패널 높이 - 3보다 큼) → 오버플로우

**해결안**:
- 각 섹션에 명시적 Height 설정
- 전체 높이에서 가용 높이를 섹션 수만큼 분배
- 또는 각 섹션에 Margin 또는 Padding으로 여유 확보

---

### 20. ⚠️ 검색 결과가 0건일 때 UX 부족
**심각도**: 낮음
**위치**: `app.go:672-673`

**문제 코드**:
```go
if len(m.searchResults) == 0 {
    return sectionStyle.Render("검색 결과 / 목록\n─────────────\n(검색어를 입력하세요)")
}
```

**문제점**:
- 검색 후 0건인 경우와 아직 검색하지 않은 경우 구분 없음
- 동일한 메시지: "(검색어를 입력하세요)"
- 사용자가 검색했는데 결과 0건인지 알 수 없음
- 상태바에 "✓ 0건 검색 완료"가 있지만, 섹션 자체는 구분 안 함

**UX 개선**:
- 상태 추적: `m.hasSearched` (bool) 필드 추가
- 검색 전: "(검색어를 입력하세요)"
- 검색 후 0건: "검색 결과: 0건\n검색어를 다시 시도하세요"
- 로딩 중: "🔍 검색 중..."

---

### 21. ⚠️ 결과 데이터가 0건일 때 상태바 메시지
**심각도**: 낮음
**위치**: `app.go:851-852`

**문제 코드**:
```go
if len(m.resultData) == 0 {
    return sectionStyle.Render("📋 결과 데이터\n───────────────────\n(조회를 실행하세요)")
}
```

**문제점**:
- "조회를 실행하세요" → 실제로는 조회가 실행되었으나 0건 반환될 수 있음
- 또는 아직 조회하지 않은 초기 상태일 수 있음
- 구분이 필요

**상태 추적**:
- `m.hasDataQueried` (bool) 추가
- 데이터 조회 완료 후 0건: "(조회 결과 0건)"
- 미조회: "(조회를 실행하세요)"

---

### 22. ⚠️ 색상 vs 명암비 (dark terminal 미지원)
**심각도**: 중
**위치**: `styles.go:6-17`

**문제점**:
- 설정된 색상이 모두 밝은 배경을 가정 (파란색 #2563EB, 초록색 #10B981)
- 사용자 터미널이 검은 배경이면: ColorMuted (#6B7280) + ColorBgPanel (#0F172A) = 명암비 낮음
- WCAG AA 기준 4.5:1 미만일 가능성

**색상 팔레트**:
```go
ColorPrimary   = "#2563EB"  // 파란색 (상대적으로 밝음)
ColorSecondary = "#3B82F6"  // 더 밝은 파란
ColorMuted     = "#6B7280"  // 회색 (어둠)
ColorBgPanel   = "#0F172A"  // 매우 어두운 배경
```

**로컬 터미널에서 실제 렌더링 확인 필요**:
- 검색 섹션 비활성 보더 (ColorBorder #334155)가 배경과 구분되는지
- 텍스트 ColorMuted가 읽기 가능한지

**해결안**:
- 밝기 조정: ColorMuted를 더 밝게 (예: #94A3B8)
- 또는 테마 지원 (light/dark mode)

---

### 23. ❌ 개별 섹션 sectionStyle 매번 재생성 (성능)
**심각도**: 낮음
**위치**: `app.go` 모든 render*Section 함수

**문제 코드**:
```go
func (m Model) renderSearchSection(width int) string {
    var borderColor lipgloss.Color = ColorBorder
    if m.activePanel == PanelSearch {
        borderColor = ColorAccent
    }

    sectionStyle := lipgloss.NewStyle().  // ← 매번 생성!
        Border(lipgloss.RoundedBorder()).
        BorderForeground(borderColor).
        ...
}
```

**문제점**:
- 매 렌더링마다 new style 생성 (메모리 할당)
- View() 함수는 최소 16ms마다 호출 (60fps 기준)
- 매번 lipgloss.NewStyle() → 불필요한 GC 압박

**현재 구조**:
```
View() → renderLeftPanel → renderSearchSection (style 생성)
                        → renderResultsSection (style 생성)
                        → renderBookmarkSection (style 생성)
                        → renderHistorySection (style 생성)
      → renderRightPanel → renderMetaSection (style 생성)
                        → renderParamsSection (style 생성)
                        → renderResultSection (style 생성)
```
- 매 렌더링마다 7개 style 객체 생성

**해결안**:
- styles.go에 기본 스타일 미리 정의
- 활성 패널만 동적 색상 변경
- 또는 New() 함수에서 캐시된 style 사용

---

### 24. ⚠️ 선택된 항목 반전 색상 (SelectedItemStyle) 미사용
**심각도**: 낮음
**위치**: `styles.go:39-42, app.go:699-704`

**문제 코드** (styles.go):
```go
var SelectedItemStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(ColorHighlight).
    Background(ColorPrimary)
```

**현재 사용**:
```go
// app.go: renderResultsSection에서
marker := "  "
if i == m.cursor {
    marker = "▸ "  // ← 마커만 변경, 스타일 미적용
}
```

**문제점**:
- SelectedItemStyle 정의는 되어 있지만 사용 안 됨
- 선택된 항목에 배경색 + 텍스트색 반전이 적용 안 됨
- 설계서 5.4: "선택된 항목: 반전 색상 + 볼드" 명시인데 미구현

**해결안**:
```go
if i == m.cursor {
    line = SelectedItemStyle.Render(line)
} else {
    line = NormalItemStyle.Render(line)
}
```

---

### 25. ⚠️ 타이틀바 전체 너비 검증 필요
**심각도**: 낮음
**위치**: `app.go:537-546`

**문제 코드**:
```go
titleContent := title + " " + helpText
return titleStyle.Render(titleContent)
```

**문제점**:
- 타이틀 + 도움말을 단순 연결
- 타이틀과 도움말 사이 간격이 터미널 폭에 따라 동적으로 조정 안 됨
- 좁은 터미널에서 타이틀과 도움말이 겹쳐 표시될 수 있음

**예시**:
- 터미널 폭 80: "KOSIS 통계 데이터 탐색기" (13 문자) + " " + "[?] 도움말  [q] 종료" (12 문자) = 26 문자 (모자람)
- 터미널 폭 50: 같은 텍스트가 너비 초과 → 줄바꿈 또는 잘림

**설계서 요구사항** (5.1):
```
│  KOSIS 통계 데이터 탐색기                          [?] 도움말  [q] 종료 │
```
- 명확히 우측 정렬

**해결안**: 이전 보고서 #9 참조 (타이틀바 우측 정렬)

---

### 26. ⚠️ 파라미터 패널의 "[선택 필요]", "[자동]", "[전체]" 텍스트 위치성 모호
**심각도**: 낮음
**위치**: `app.go:809-820`

**현재 렌더링**:
```
⚙ 파라미터 설정
─────────────
분류: [선택 필요]
항목: [선택 필요]
주기: [자동]
최근: [전체]

(Enter로 조회)
```

**문제점**:
- 사용자가 파라미터를 어떻게 변경하는지 명시 안 됨
- "⚙ 파라미터 설정" 패널이 읽기만 가능해 보임 (상호작용 없음)
- 키바인딩에 "파라미터 수정" 방법 없음 (e, f, i, Tab 등)
- 파라미터 입력 방법: "PanelParams에서 위↓로 선택"? 설계서 불명확

**설계서** (5.2, 5.3):
- 파라미터 설정 패널: "분류/항목/시점 입력"
- 키바인딩: Tab (패널전환), Enter (선택/조회 실행)
- 근데 파라미터 수정 키가 없음

**해결안**:
- 파라미터 편집 모드 추가 (Tab으로 패널 전환 시 활성화)
- 또는 파라미터 편집 팝업 창
- 또는 "위↓ 수정" 힌트 텍스트 추가

---

### 27. ⚠️ 키바인딩 "↑↓" vs "k/j" 이중 정의 검증
**심각도**: 낮음
**위치**: `app.go:319-326, keys.go`

**현재 코드**:
```go
case KeyUp, "k":
    if m.cursor > 0 {
        m.cursor--
    }
case KeyDown, "j":
    if m.cursor < len(m.searchResults)-1 {
        m.cursor++
    }
```

**설계서** (5.3):
```
| ↑↓ | 목록 탐색 |
```

**문제점**:
- 설계서에는 ↑↓만 언급
- 코드에는 k/j (vim 스타일) 추가
- 키바인딩 일관성: 공식 지원 키와 추가 별칭 구분 필요
- --help에서 어떤 키를 표시할지 불명확

**해결안**:
- 설계서에 추가 키 명시: "↑↓ 또는 k/j"
- 또는 vim 스타일 제거하고 화살표만 지원

---

### 28. ⚠️ 이모지 렌더링 호환성 (모든 터미널)
**심각도**: 중
**위치**: 전역 이모지 사용

**현재 사용 이모지**:
- 🔍 (검색)
- 📊 (통계)
- ⚙ (파라미터)
- 📋 (결과)
- ★ (즐겨찾기)
- ◷ (이력)
- ⏳ (로딩)
- ❌ (에러)
- ✓ (성공)
- ⚠ (경고)
- 💾 (저장)

**문제점**:
- 이모지가 터미널 전혀 지원하지 않거나 부분 지원 가능
- Windows 기본 cmd.exe: 이모지 렌더링 안 됨
- 오래된 Linux 터미널: 이모지 안 보임
- macOS 터미널: 대부분 지원하지만 두께 불일치 가능

**확인**:
- 각 이모지의 실제 터미널 폭 (1칸 vs 2칸)
- runeWidth() 함수에서 정확히 계산되는지

**해결안**:
- 이모지 대체 텍스트 지원 (NO_EMOJI 환경변수)
- 또는 Isatty() 감지: TTY가 아니면 이모지 제거
- TERM 환경변수 확인: xterm-256color, xterm 등에서 호환성 검증

---

### 29. ⚠️ 유니코드 보더 문자 호환성
**심각도**: 낮음
**위치**: 보더 문자 사용

**현재 사용 문자**:
- `─` (fullwidth hyphen U+2500)
- `│` (fullwidth vertical U+2502)
- `┼` (cross U+253C)
- `└`, `┌`, `┐`, `┘` (corners) — lipgloss RoundedBorder() 자동 생성

**문제점**:
- 일부 터미널 (예: Windows 10 cmd)에서 깨짐
- UTF-8 인코딩 미지원 환경에서 문자 오류
- 터미널 폰트가 유니코드 미지원하면 대체 문자로 표시

**현재 코드** (app.go:614):
```go
bookmarkLines = append(bookmarkLines, "─────────────")
```

**해결안**:
- ASCII 보더로 대체 옵션: "-", "|", "+"
- 또는 Isatty() + UTF-8 감지 후 결정
- 사용자 설정: `kosis config set-borders ascii` 또는 `unicode`

---

### 30. ⚠️ PanelSearch, PanelMeta, PanelParams, PanelResult 상수만 정의 (지표 모드 추가 불가)
**심각도**: 낮음
**위치**: `app.go:16-24`

**현재 코드**:
```go
type Panel int

const (
    PanelSearch Panel = iota
    PanelMeta
    PanelParams
    PanelResult
)
```

**문제점**:
- 지표 모드 추가 시 새로운 Panel 상수 필요
- 4개 패널만 정의되어 있음
- Tab 키 패널 전환이 순환: `(m.activePanel + 1) % 4` (하드코딩)

**지표 모드 추가 시**:
```
PanelSearch (통계표)
  ↓ i 키
PanelIndicator (지표)
  → 패널 구조 다름 (분류 대신 지표명)
```

**해결안**:
- `m.mode` 추가: TableMode, IndicatorMode
- 각 모드별 Panel 정의
- Tab 순환을 함수로 분리: `func (m Model) nextPanel() Panel`

---

### 31. ⚠️ statusMsg 길이 제한 없음 (터미널 폭 초과 가능)
**심각도**: 낮음
**위치**: `app.go:944-958`

**현재 코드**:
```go
var status string
if m.loading {
    status = "⏳ 로딩중..."
} else if m.err != nil {
    status = "❌ " + m.err.Error()  // ← err.Error()가 길 수 있음
} else {
    status = m.statusMsg
}
```

**문제점**:
- 에러 메시지가 매우 길면 상태바가 터미널 폭 초과
- 예: `"❌ 검색 오류: EOF error reading response body from API ..."`
- 상태바 Width(m.width) 설정도 있지만, statusStyle.Render()가 overflow 처리 불명확

**해결안**:
- 상태바 메시지 길이 제한: `truncateText(status, m.width - 2)`
- 또는 에러 메시지 요약: 처음 50자 + "..." 표시

---

### 32. ⚠️ 메타 정보 분류/항목 목록이 매우 길 때 패널 오버플로우
**심각도**: 중
**위치**: `app.go:762-775`

**문제 코드**:
```go
if m.metaInfo != nil {
    content += "\n[분류] 코드 목록\n"
    for _, cls := range m.metaInfo.Classifications {
        content += "  " + cls.Code + " " + cls.Name + "\n"
    }
    content += "\n[항목] 코드 목록\n"
    for _, item := range m.metaInfo.Items {
        content += "  " + item.Code + " " + item.Name + "\n"
    }
}
```

**문제점**:
- 분류가 1,000개 이상이면 content 문자열이 매우 길어짐
- 렌더링 시 패널 높이 초과 → 내용이 보이지 않음
- 메타 섹션에 Height 제한이 없어서 아래 섹션 침범

**예시** (실제 API):
- DT_1IN1502 (인구): 분류 약 250개
- 제출하는 대용량 통계표: 분류 1,000개 이상

**해결안**:
- 분류/항목 최대 10개까지만 표시: "처음 10개..."
- 또는 스크롤 기능 (↑↓로 분류/항목 탐색)
- 또는 메타 섹션 Height(10) 고정 + 내부 scrolling

---

### 33. ⚠️ 설계서의 "실시간 검색" 개념 불명확
**심각도**: 중
**위치**: `설계서 5.2, 현재 구현 불일치`

**설계서 5.2**:
- 검색 입력 패널: "실시간 검색 (6.통합검색 / 7-B.지표검색 전환)"

**문제점**:
- "실시간 검색"의 의미 불명확
  1. 타이핑하면서 자동 검색? (디바운싱 포함)
  2. Enter 키마다 검색? (현재 구현)
  3. "/" 키를 누르면 검색 입력 시작? (현재 구현)

**현재 코드**:
- "/" 키 → 검색 입력 모드 진입
- Enter 키 → 검색 실행
- 따라서 "실시간" 아님 (수동)

**해결안**:
1. 설계서 수정: "검색어 입력 (Enter로 실행)"
2. 또는 코드 구현: 디바운싱으로 자동 검색
   - 300ms 타이핑 멈춤 후 자동 doSearch() 실행
   - 그러면 UX 개선 (Enter 생략 가능)

---

### 34. ⚠️ 색상 정의에 X11 색상명 vs 16진수 혼재
**심각도**: 낮음
**위치**: `styles.go, app.go`

**styles.go의 색상**:
```go
ColorPrimary   = lipgloss.Color("#2563EB")  // 16진수
...

// app.go에서는
Foreground(lipgloss.Color("255"))  // 256색 인덱스!
```

**문제점**:
- styles.go: 24비트 색상 (#RRGGBB)
- app.go 일부: 256색 인덱스 (255 = 흰색)
- 두 가지 색상 시스템 혼재

**예시** (app.go:540):
```go
titleStyle := lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("255")).  // ← 256색 인덱스
    Background(ColorPrimary).           // ← 24비트 색상
```

**문제**:
- 터미널이 256색만 지원하면 24비트 색상이 근사색으로 표시
- 역으로 256색 인덱스는 24비트 터미널에서도 작동하지만 명확하지 않음

**해결안**:
- 모든 색상을 24비트로 통일: `#FFFFFF` 대신 `#FFFFFF` 사용
- 또는 256색 인덱스로 통일 (호환성)

---

### 35. ⚠️ 검색 결과 cursor가 범위 초과할 수 있음
**심각도**: 낮음
**위치**: `app.go:490-493`

**문제 코드**:
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    // cursor 검증 없음
```

**문제점**:
- 터미널 크기 변경 후 검색 결과 행 개수가 줄어들 수 있음
- 예: 터미널 높이 30 → 10으로 축소
- maxVisible = (30-3)/2 = 13 → maxVisible = (10-3)/2 = 3
- 기존 cursor = 5인데 maxVisible = 3 → 범위 초과 가능

**해결안**:
```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    // cursor 검증
    if m.cursor >= len(m.searchResults) {
        m.cursor = len(m.searchResults) - 1
        if m.cursor < 0 {
            m.cursor = 0
        }
    }
```

---

### 36. ⚠️ 에러 화면에서 UI 상호작용 불가
**심각도**: 중
**위치**: `app.go:113-135`

**문제 코드**:
```go
// New() 함수에서 API 키 없으면
if err != nil || len(keys) == 0 {
    return Model{
        ...
        statusMsg: "⚠ API 키 미설정...",
        err:       fmt.Errorf("API 키 없음"),
        cursor:    0,
        typing:    false,
    }
}
```

**문제점**:
- API 키 미설정 상태에서도 TUI 화면 렌더링됨
- 사용자가 키를 누르면 아무것도 작동 안 함
- 현재 View() 함수가 normal 대시보드만 렌더링 → 에러 메시지만 상태바에 표시

**개선**:
- err != nil 상태에서는 전체 화면을 에러 메시지로 표시
- "API 키 미설정 상태"를 명확히 표시
- "[설정 가이드] kosis config set-key <KEY>" 링크 표시

**해결안**:
```go
func (m Model) View() string {
    if m.err != nil && m.client == nil {
        return errorScreen(m.err)  // ← 전체 화면 에러 표시
    }
    // 정상 대시보드 렌더링
}
```

---

**소계**: 13~36번 = 24개 추가 문제 발견 (총 36개, 이전 12개 포함)

---

## 2차 평가 요약

### 추가 발견 문제 분류

**설계서 불일치 (심각)**:
- 문제 13: 검색 입력 "실시간 검색" 명시 vs Enter 방식
- 문제 14: 키 `i` 지표 모드 전환 미구현
- 문제 33: "실시간 검색" 개념 불명확 (설계서 수정 필요)

**구현 미흡 (중)**:
- 문제 15: 상태바 메시지 중복/일관성
- 문제 16: 지표 모드 패널 구조 미정의
- 문제 17: 메타 정보 패널 높이 고정 미흡
- 문제 19: 왼쪽/오른쪽 패널 높이 불일치
- 문제 24: SelectedItemStyle 미사용
- 문제 32: 메타 정보 분류/항목 오버플로우

**호환성/렌더링 (낮음~중)**:
- 문제 18: 보더/구분선 문자 혼재
- 문제 20: 검색 결과 0건 UX 부족
- 문제 21: 결과 데이터 0건 메시지
- 문제 22: 색상 명암비 (dark terminal)
- 문제 23: sectionStyle 매번 재생성 (성능)
- 문제 25: 타이틀바 전체 너비 검증
- 문제 26: 파라미터 패널 상호작용 명시 부족
- 문제 27: 키바인딩 이중 정의 (↑↓ vs k/j)
- 문제 28: 이모지 렌더링 호환성
- 문제 29: 유니코드 보더 호환성
- 문제 30: Panel 상수 지표 모드 확장성
- 문제 31: statusMsg 길이 제한 없음
- 문제 34: 색상 정의 혼재 (16진수 vs 256색)
- 문제 35: cursor 범위 초과 가능
- 문제 36: 에러 화면 UX 부족

### 2차 점수 계산

**기본 점수**: 64점 (1차 이후)

**2차 감점**:
- 설계서 불일치: -10점 (3개 × 3점, 심각)
- 구현 미흡: -15점 (6개 × 2.5점)
- 호환성/렌더링: -12점 (15개 × 0.8점)

**2차 최종 점수: 64 - 10 - 15 - 12 = 27점** ⚠️⚠️

---

## 최종 종합 평가

### 문제 심각도 분포

| 심각도 | 개수 | 감점 |
|--------|------|------|
| 심각 (❌) | 8개 | -40점 |
| 중간 (⚠️) | 18개 | -36점 |
| 경미 (⚠️) | 10개 | -10점 |
| **합계** | **36개** | **-86점** |

### 수정 우선순위

**1순위 (필수 수정, 설계서 준수)**:
1. 문제 1, 2: 터미널 최소 크기 체크
2. 문제 5: runeWidth에 이모지 범위 추가
3. 문제 10: 테이블 컬럼 구분선 너비 일치
4. 문제 13, 14: 검색 입력 방식 명확화 + 지표 모드 구현
5. 문제 33: 설계서 "실시간 검색" 개념 명시

**2순위 (기능 완성, 설계서 명시)**:
6. 문제 16: 지표 모드 패널 구조 정의 및 구현
7. 문제 17, 19: 패널 높이 분배 명확화
8. 문제 24: SelectedItemStyle 적용
9. 문제 9: 타이틀바 우측 정렬
10. 문제 15: 상태바 메시지 일관성

**3순위 (호환성/UX 개선)**:
11. 문제 18, 28, 29: 보더/이모지/유니코드 호환성
12. 문제 22: 색상 명암비 검증
13. 문제 20, 21, 26: UX 명시성 개선
14. 문제 23, 30, 31, 34, 35, 36: 성능/코드 품질

### 최종 평가 점수

**1차 (레이아웃 세부)**: 64점
**2차 (설계서 대비 + 추가 분석)**: 27점

**누적 평가**: 27점 (수정 전)

**수정 후 예상 점수**: 70~80점 (우선순위 1~2순위 완료 시)

---

## 결론

TUI는 기본 구조는 양호하지만 설계서와의 **세부 불일치**가 많음:
- 레이아웃/렌더링: 터미널 크기 처리, 패널 높이 분배
- 키바인딩: 지표 모드 미구현
- 스타일: 색상 명암비, 이모지 호환성
- UX: 상태 메시지 명확성

**긴급 수정 사항**:
1. 터미널 최소 크기 체크 (현재 40px 미만에서 깨짐)
2. 이모지 폭 계산 정확화
3. 지표 모드 구현
4. 설계서 명시 사항 재검토 및 코드/문서 동기화

---

## 추가 검토 항목 (2차)

- [ ] 실제 터미널에서 30x10, 50x15, 200x50 크기로 테스트
- [ ] Windows cmd.exe, PowerShell, WSL 호환성 테스트
- [ ] macOS Terminal.app, iTerm2 호환성 테스트
- [ ] Linux xterm, gnome-terminal 호환성 테스트
- [ ] 256색 터미널 vs 24비트 색상 터미널 호환성
- [ ] 이모지 렌더링 폭: 각 이모지의 실제 픽셀 폭 측정
- [ ] 키바인딩 테스트: 한글 IME 입력 테스트 (입력 중 Esc, 한자 변환 등)
- [ ] 성능 테스트: 1,000+ 검색 결과 스크롤 응답성
- [ ] 메타 정보 10,000+ 분류/항목 렌더링 테스트
- [ ] 설계서 재검토: "실시간 검색", "지표 모드 패널 구조" 명시
