# KOSIS CLI TUI 키바인딩 및 네비게이션 테스트 보고서 (Tester4)

**작성일**: 2026-04-01
**테스터**: TUI Tester4
**테스트 범위**: 키바인딩, 네비게이션, 패널 전환
**코드 리뷰 대상**: `internal/tui/app.go`, `internal/tui/keys.go`

---

## 발견된 문제 목록

### 1. ⚠️ Esc 키에서 상태 복원 누락 (HIGH)
**코드 위치**: `internal/tui/app.go:285-287`

```go
case KeyEscape:
    m.typing = false
    m.statusMsg = m.getPanelStatusMsg()
```

**문제점**:
- Esc 키로 검색 입력을 취소할 때 `m.searchInput`을 초기화하지 않음
- 사용자가 검색어를 입력하다가 Esc로 취소해도 입력창에 이전 검색어가 남아있음
- `m.activePanel`도 PanelSearch에서 다른 패널로 복원해야 하는데, 그렇지 않음

**설계서 기준**: 설계서 섹션 5.3 키바인딩에서 `Esc`는 검색을 취소하는 기능으로 명시됨

**권장 수정**:
```go
case KeyEscape:
    m.typing = false
    m.searchInput = ""  // 입력창 초기화
    m.statusMsg = m.getPanelStatusMsg()
    // 이전 패널로 돌아가기 (필요시)
```

---

### 2. ⚠️ typing 모드에서 Tab/q/↑↓ 키 차단 불완전 (HIGH)
**코드 위치**: `internal/tui/app.go:274-300`

**문제점**:
- `typing == true`일 때 키 처리가 매우 제한적임
- Enter, Escape, backspace, 문자 입력만 처리됨
- **하지만 `case KeyTab`이나 `case KeyUp/KeyDown` 또는 `case KeyQuit`를 명시적으로 차단하지 않음**
- 만약 사용자가 실수로 이들 키를 누르면 어떻게 되는지 명확하지 않음 (함수 끝에서 `return m, nil`이므로 무시되지만, 코드가 자명하지 않음)

**권장 개선**:
```go
if m.typing {
    switch msg.String() {
    case KeyEnter:
        // ...
    case KeyEscape:
        // ...
    case "backspace":
        // ...
    default:
        // 한글, 영문 등 모든 문자 입력 지원
        if len(msg.Runes) > 0 {
            m.searchInput += string(msg.Runes)
        }
    }
    return m, nil  // typing 모드는 항상 여기서 반환
}
```

현재 구조도 동작하지만 **문서화 주석이 필요함**: "typing 모드에서는 Enter/Esc/backspace만 처리하고, 다른 모든 입력은 문자로 취급"

---

### 3. ❌ typing 모드에서 q 키: 문자 입력 vs 종료 문제 (CRITICAL)
**코드 위치**: `internal/tui/app.go:274-300` (typing 모드) vs `internal/tui/app.go:305-306` (일반 모드)

**문제점**:
- typing 모드(`m.typing == true`)에서 `q` 키를 누르면 **문자로 입력됨** (default case에서 msg.Runes 처리)
- 일반 모드에서만 `q` 키가 종료 명령으로 작동
- **사용자가 검색 중에 실수로 누른 q가 종료가 아닌 검색어 입력으로 처리됨** (설계서와 모순)

**설계서 기준**: 설계서 섹션 5.3에서 `q`는 "종료"로 명시됨

**권장 수정**: Ctrl+C는 어디서나 종료하도록 하거나, q도 typing 모드에서 명시적으로 차단:
```go
if m.typing {
    switch msg.String() {
    case KeyEnter, KeyEscape, "backspace":
        // ...
    case KeyQuit:  // q는 typing 모드에서도 입력 불가
        return m, nil  // 무시
    default:
        if len(msg.Runes) > 0 {
            m.searchInput += string(msg.Runes)
        }
    }
}
```

**대안**: 설계 의도가 "q는 무조건 종료"라면:
```go
case KeyQuit, "ctrl+c":
    m.quitting = true
    return m, tea.Quit
```
를 typing 모드 **위에** 먼저 배치

---

### 4. ❌ Tab 키 순환 검증 불완전 (MEDIUM)
**코드 위치**: `internal/tui/app.go:308-311`

```go
case KeyTab:
    // 패널 순환: Search → Meta → Params → Result → Search
    m.activePanel = (m.activePanel + 1) % 4
    m.statusMsg = m.getPanelStatusMsg()
```

**문제점**:
- 주석은 "Search → Meta → Params → Result → Search"라고 명시됨
- 하지만 Panel enum 정의 (`internal/tui/app.go:20-25`)를 보면:
  ```go
  const (
      PanelSearch Panel = iota    // 0
      PanelMeta                   // 1
      PanelParams                 // 2
      PanelResult                 // 3
  )
  ```
  순서는 맞음 ✓
- 그러나 **이 로직이 typing 모드를 고려하지 않음**:
  - typing 모드에서 Tab을 누르면 패널 전환이 됨 (의도: Tab은 검색 입력 중에 차단되어야 함)

**권장 수정**: typing 모드에서 Tab 차단:
```go
if m.typing {
    switch msg.String() {
    case KeyEnter, KeyEscape, "backspace":
        // ...
    case KeyTab:  // typing 중에는 Tab 입력 가능하게?
        // 또는 무시:
        return m, nil
    // ...
    }
    return m, nil
}
```

---

### 5. ⚠️ Enter 키: 패널별 동작 분기 불완전 (HIGH)
**코드 위치**: `internal/tui/app.go:350-411`

```go
case KeyEnter:
    // 검색 결과에서 통계표 선택 → 메타 조회
    if (m.activePanel == PanelSearch || m.activePanel == PanelMeta) && len(m.searchResults) > 0 && m.cursor < len(m.searchResults) {
        // ...
    } else if m.activePanel == PanelParams {
        // ...
    }
    // PanelResult에 대한 처리 없음!
```

**문제점**:
- `PanelResult` 패널에서 Enter 키를 누를 때 아무 동작도 정의되지 않음
- 상태 메시지(`getPanelStatusMsg`)에는 "e 내보내기  f 즐겨찾기  i 지표모드  Tab 다음 패널"이라고 나오는데, Enter에 대한 언급이 없음
- 설계 의도가 불명확함 (PanelResult에서 Enter는 다음 행 선택인가? 아니면 데이터 내보내기인가?)

**권장 명확화**:
```go
case KeyEnter:
    if m.activePanel == PanelSearch || m.activePanel == PanelMeta {
        // 통계표 선택
        if len(m.searchResults) > 0 && m.cursor < len(m.searchResults) {
            // ...
        }
    } else if m.activePanel == PanelParams {
        // 데이터 조회 실행
        // ...
    } else if m.activePanel == PanelResult {
        // 현재 정의 없음 - 설계 문서 확인 후 구현
        m.statusMsg = "⚠ Result 패널에서 Enter는 아직 구현되지 않았습니다"
    }
```

---

### 6. ⚠️ e 키 (내보내기): 데이터 없음 시 상태 확인 (MEDIUM)
**코드 위치**: `internal/tui/app.go:327-332`

```go
case KeyExport: // e 키: 내보내기
    if len(m.resultData) > 0 {
        m.statusMsg = "💾 내보내기: 기능 준비 중 (CLI로 사용: kosis d ... -o file.xlsx)"
    } else {
        m.statusMsg = "⚠ 내보낼 데이터가 없습니다. 먼저 데이터를 조회하세요."
    }
```

**상태**: ✓ 구현됨 (문제 없음)

**확인 사항**:
- 데이터가 없을 때: 경고 메시지 표시 ✓
- 데이터가 있을 때: 준비 중 안내 ✓
- **다만 실제 내보내기 기능은 아직 구현되지 않아 TUI에서는 기능하지 않음** (CLI 전용)

---

### 7. ⚠️ f 키 (즐겨찾기): 통계표 미선택 시 안내 (MEDIUM)
**코드 위치**: `internal/tui/app.go:333-347`

```go
case KeyBookmark: // f 키: 즐겨찾기 추가
    if m.selectedTable != nil {
        err := bookmark.Add(m.selectedTable.OrgID, m.selectedTable.TblID, m.selectedTable.TblNM)
        // ...
    } else {
        m.statusMsg = "⚠ 즐겨찾기에 추가할 통계표를 먼저 선택하세요."
    }
```

**상태**: ✓ 구현됨 (문제 없음)

**확인 사항**:
- 통계표 미선택: 경고 메시지 ✓
- 통계표 선택: 즐겨찾기 추가 ✓

---

### 8. ⚠️ i 키 (지표 모드): 현재 구현 상태 (LOW)
**코드 위치**: `internal/tui/app.go:348-349`

```go
case KeyIndicator: // i 키: 주요지표 모드 전환
    m.statusMsg = "📊 주요지표 모드: CLI로 사용 → kosis ind s \"GDP\""
```

**문제점**:
- 실제 지표 모드 전환이 구현되지 않음
- 메시지만 표시하고 아무것도 변경되지 않음
- 설계서 섹션 5.3에서 "통계표/지표 모드 전환"이라고 명시되어 있으나, TUI에서는 비활성화된 상태

**설계 참고**: 현재는 CLI 전용으로 제공되는 기능인 것으로 보임

---

### 9. ⚠️ ? 키 (도움말): 상태바 메시지 변경 동작 확인 (MEDIUM)
**코드 위치**: `internal/tui/app.go:317-318`

```go
case KeyHelp:
    m.statusMsg = KeyBindings()
```

**문제점**:
- 도움말이 상태바에만 표시됨
- 다른 키를 누르면 상태 메시지가 바뀌어 도움말이 사라짐
- **? 를 다시 눌러야 도움말을 볼 수 있음** (모달 창이 아님)

**설계 의도 확인 필요**: 이게 원래 의도인지, 아니면 모달 도움말 창이 있어야 하는지

**`KeyBindings()` 함수**: `internal/tui/keys.go:21-30`에서 정의됨
```go
func KeyBindings() string {
    return KeyHintStyle.Render("/") + KeyDescStyle.Render(" 검색  ") + ...
}
```
✓ 올바르게 구현됨

---

### 10. ↑↓ 키: 검색 결과 범위 체크 (MEDIUM)
**코드 위치**: `internal/tui/app.go:319-326`

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

**상태**: ✓ 구현됨 (경계값 체크 완벽)

**확인 사항**:
- cursor < 0: 방지됨 ✓ (`m.cursor > 0` 조건)
- cursor >= len: 방지됨 ✓ (`m.cursor < len(m.searchResults)-1` 조건)
- **빈 검색 결과일 때**: cursor는 움직이지 않으므로 안전 ✓

---

### 11. ⚠️ Ctrl+C: 언제나 종료 보장 (MEDIUM)
**코드 위치**: `internal/tui/app.go:305-307`

```go
case KeyQuit, "ctrl+c":
    m.quitting = true
    return m, tea.Quit
```

**문제점**:
- 이 케이스는 typing 모드가 **아닐 때**만 실행됨 (if m.typing 블록 밖)
- **typing 모드에서 Ctrl+C를 누르면 문자로 입력될 수 있음** (msg.Runes에 포함될 가능성)

**권장 수정**:
```go
// typing 모드 위에 배치
switch msg.String() {
case "ctrl+c":
    m.quitting = true
    return m, tea.Quit
}

// 그 다음 typing 모드 처리
if m.typing {
    switch msg.String() {
    // ...
    }
}
```

---

### 12. ⚠️ typing 모드에서 한글 조합 중 Enter 시 동작 (HIGH)
**코드 위치**: `internal/tui/app.go:276-284`

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
- 한글 입력 중에 Enter를 누르면:
  - Bubble Tea KeyMsg에 Enter가 들어옴
  - `case KeyEnter`로 일치함
  - **하지만 한글 조합이 아직 완료되지 않았을 수 있음**
  - OS의 IME(한글 입력기)가 조합을 완료하기 전에 Enter가 처리될 수 있음

**예시**: "검"을 입력하는 중에 Enter를 누르면 → 받침이 버려질 수 있음

**권장 해결책**:
```go
case KeyEnter:
    m.typing = false
    // 혹은 IME 완료 감지 로직 추가
    if m.searchInput == "" {
        m.statusMsg = "검색어를 입력하세요"
        return m, nil
    }
    // 한글 조합 완료 대기 (Bubble Tea에서 자동 처리되는지 확인 필요)
    m.loading = true
    m.statusMsg = "🔍 검색 중..."
    return m, m.doSearch(m.searchInput)
```

**주의**: 이는 Bubble Tea와 Go의 KeyMsg 처리 방식에 따라 달라질 수 있음. Bubble Tea 문서 확인 필요.

---

### 13. 🔍 typing 모드 진입/탈출 상태 추적 (MEDIUM)
**코드 위치**: `internal/tui/app.go:313-316`

```go
case KeySearch:
    m.activePanel = PanelSearch
    m.typing = true
    m.searchInput = ""
```

**상태**: ✓ 올바르게 구현됨

**확인 사항**:
- `/` 키로 검색 모드 진입: 패널 전환 + typing 플래그 설정 ✓
- `m.searchInput` 초기화 ✓
- 상태 메시지 업데이트 ✓

---

## 요약 테이블

| # | 문제 | 심각도 | 상태 | 권장사항 |
|----|------|--------|------|---------|
| 1 | Esc 키에서 상태 복원 누락 | HIGH | 버그 | m.searchInput 초기화, 이전 패널 복원 |
| 2 | typing 모드 키 차단 불완전 | HIGH | 설계모호 | 명시적 차단 또는 주석 추가 |
| 3 | typing 모드에서 q 키 동작 | **CRITICAL** | **버그** | **Ctrl+C를 먼저 처리하거나 q 명시적 차단** |
| 4 | Tab 키 순환 검증 | MEDIUM | 설계모호 | typing 모드에서 Tab 동작 명확화 |
| 5 | Enter 키 패널별 분기 | HIGH | 불완전 | PanelResult 동작 정의 |
| 6 | e 키 (내보내기) | MEDIUM | ✓ 구현됨 | - |
| 7 | f 키 (즐겨찾기) | MEDIUM | ✓ 구현됨 | - |
| 8 | i 키 (지표 모드) | LOW | 미구현 | CLI 전용 확인 |
| 9 | ? 키 (도움말) | MEDIUM | 설계모호 | 모달 창 여부 확인 |
| 10 | ↑↓ 키 범위 체크 | MEDIUM | ✓ 구현됨 | - |
| 11 | Ctrl+C 언제나 종료 | MEDIUM | 버그 | 우선순위 변경 |
| 12 | 한글 조합 중 Enter | **HIGH** | 잠재 버그 | IME 호환성 테스트 필요 |
| 13 | typing 상태 추적 | MEDIUM | ✓ 구현됨 | - |

---

## 추가 구조적 문제

### A. getPanelStatusMsg() 와 실제 동작 간 괴리 (MEDIUM)
**코드 위치**: `internal/tui/app.go:962-975`

```go
func (m Model) getPanelStatusMsg() string {
    switch m.activePanel {
    case PanelSearch:
        return "/ 검색어 입력  Enter 검색  Tab 다음 패널"
    case PanelMeta:
        return "↑↓ 목록 탐색  Enter 선택  Tab 다음 패널"
    case PanelParams:
        return "파라미터 설정  Enter 조회 실행  Tab 다음 패널"
    case PanelResult:
        return "e 내보내기  f 즐겨찾기  i 지표모드  Tab 다음 패널"
    }
}
```

**문제점**:
- PanelSearch: "/" 입력 불가능 (이미 typing 모드임)
- PanelMeta: "↑↓"는 실제로 cursor 이동에 사용되는데, 어느 패널에서 적용되는지 불명확
- PanelResult: Enter 키가 명시되지 않았는데, 다른 패널에서는 Enter가 중요 동작

**권장 개선**:
```go
case PanelSearch:
    if m.typing {
        return "검색 입력 중... (Enter: 검색, Esc: 취소)"
    }
    return "검색 입력 (/ 또는 Enter 시작)"
```

---

### B. 메타 정보 자동 조회 시 cursor 이동 미처리 (LOW)
**코드 위치**: `internal/tui/app.go:430-478`

```go
case metaResultMsg:
    m.loading = false
    if msg.err != nil {
        // ...
    } else {
        m.metaInfo = msg.meta
        // 메타 로드 완료 → 자동으로 기본 파라미터 설정 후 데이터 조회 시작
        if m.selectedTable != nil {
            // ... 데이터 조회 ...
            m.activePanel = PanelResult  // ← PanelResult로 점프
```

**관찰**:
- 메타 조회 완료 후 자동으로 데이터 조회가 시작되고 PanelResult로 이동
- 이는 UX 상 좋은 설계이나, **사용자가 메타 정보를 보지 못할 수 있음** (화면에 표시되는 시간이 매우 짧음)

**권장**: PanelParams에서 사용자의 명시적 Enter를 기다리는 것이 더 나을 수 있음 (현재 설계 재검토 필요)

---

## 결론 및 우선순위

**즉시 수정 필요 (CRITICAL)**:
1. **#3: typing 모드에서 q 키 동작** - Ctrl+C 우선순위 변경

**높은 우선순위 (HIGH)**:
1. **#1: Esc 키 상태 복원** - searchInput, panel 초기화
2. **#2: typing 모드 키 차단** - 문서화 강화
3. **#5: Enter 키 PanelResult 동작** - 설계 명확화
4. **#12: 한글 조합 중 Enter** - IME 호환성 테스트

**중간 우선순위 (MEDIUM)**:
1. #4: Tab 키 typing 모드 동작
2. #9: ? 키 도움말 UI
3. #11: Ctrl+C 우선순위
4. #A: getPanelStatusMsg() 동적 업데이트

**선택사항 (LOW)**:
1. #8: i 키 지표 모드 (설계 문서 확인)
2. #B: 메타 조회 UX 최적화

---

## 테스트 권장사항

### 수동 테스트 시나리오

1. **Esc 키 테스트**
   ```
   1. / 누르기 → typing 모드 진입
   2. "인구" 입력
   3. Esc 누르기
   ✓ 예상: 검색어 "인구" 사라짐, 상태 메시지 변경
   ```

2. **q 키 in typing 모드**
   ```
   1. / 누르기
   2. "q" 입력
   ✓ 예상: 문자 "q"가 입력되지 않음 (또는 명시적으로 처리)
   ```

3. **Ctrl+C 항상 종료**
   ```
   1. / 누르기 → typing 모드
   2. Ctrl+C
   ✓ 예상: TUI 즉시 종료 (어느 모드에서든)
   ```

4. **한글 조합 중 Enter**
   ```
   1. / 누르기
   2. 한글 "인" 입력 중 (조합 진행)
   3. Enter 누르기
   ✓ 예상: 조합이 완료된 "인"으로 검색 실행
   ```

5. **Tab 순환**
   ```
   1. 검색 후 결과 획득
   2. Tab 반복 누르기
   ✓ 예상: Search → Meta → Params → Result → Search 순환
   ```

### 자동화 테스트 (권장)
- `msg.String()` 값에 대한 정확한 매칭 테스트
- typing 모드에서 특수 키 필터링 테스트
- cursor 범위 경계값 테스트

---

**작성자**: KOSIS CLI Tester4
**검토 기준**: 설계서 섹션 5.3 + 코드 리뷰
**최종 평가**: 대부분 구현됨, 하지만 **#3 (q 키)과 #1 (Esc)은 반드시 수정 필요**

---

## 추가 발견 (2차)

**테스트 범위**: 상태 관리 + 기능 완성도 + 사용성
**검토 대상**: Model의 17개 필드 상태 간 의존성, 각 패널의 기능 구현 완성도

### 14. ❌ err 필드가 초기화되지 않음 (MEDIUM)
**코드 위치**: `internal/tui/app.go:946-959`

```go
func (m Model) renderStatusBar() string {
    var status string
    if m.loading {
        status = "⏳ 로딩중..."
    } else if m.err != nil {
        status = "❌ " + m.err.Error()
    } else {
        status = m.statusMsg
    }
    return statusStyle.Render(status)
}
```

**문제점**:
- 에러가 한번 발생하면 `m.err`이 계속 유지됨
- 새로운 작업 후에도 이전 에러 메시지가 상태바에 표시됨
- `statusMsg`가 업데이트되어도 `m.err != nil`이 우선되어 에러 메시지만 보임
- 검색 성공 → 메타 실패 → 다시 검색 성공한 경우, 이전 에러가 계속 표시될 수 있음

**권장 수정**:
- 새로운 작업 시작 시 `m.err = nil` 명시적으로 설정
- 또는 성공 메시지가 나올 때 에러 초기화

---

### 15. ⚠️ statusMsg와 err 필드 간 우선순위 불명확 (MEDIUM)
**코드 위치**: `internal/tui/app.go:950-955`

**문제점**:
- `loading` 상태일 때는 statusMsg 무시됨
- 로딩 중 "⏳ 로딩중..."만 표시
- 사용자가 현재 어떤 작업을 하고 있는지 알 수 없음
- statusMsg에 "🔍 검색 중..." 같은 구체적인 메시지가 있어도 무시됨

**개선 제안**:
```go
if m.loading {
    status = "⏳ " + m.statusMsg  // statusMsg 포함
} else if m.err != nil {
    status = "❌ " + m.err.Error()
} else {
    status = m.statusMsg
}
```

---

### 16. ❌ typing 모드에서 Esc로 나갈 때 이전 패널로 복원 안됨 (HIGH)
**코드 위치**: `internal/tui/app.go:285-287`

```go
case KeyEscape:
    m.typing = false
    m.statusMsg = m.getPanelStatusMsg()
```

**문제점**:
- Esc로 검색을 취소해도 `m.activePanel`이 그대로 `PanelSearch`
- 사용자가 PanelMeta에 있다가 검색 시작 → Esc 누르면 여전히 PanelSearch 상태
- 패널 복원 로직이 없음

**권장 수정**:
- Esc 전에 이전 패널을 저장해두거나
- Esc 후 자동으로 이전 패널로 복원

---

### 17. ⚠️ New() 초기화에서 typing=true로 설정 (MEDIUM)
**코드 위치**: `internal/tui/app.go:156-161`

```go
return Model{
    // ...
    typing:      true,  // ← 왜 true?
    bookmarks:   bookmarks,
    histories:   histories,
}
```

**문제점**:
- 앱 시작 시 바로 검색 입력 모드로 시작됨
- 사용자가 준비 없이 키를 누르면 검색어로 입력될 수 있음
- PanelSearch 패널이 활성이고 typing=true이므로, 사용자는 검색 중인지 아닌지 구분 어려움

**권장 수정**:
```go
typing: false,  // 앱 시작 시 일반 모드
```

---

### 18. ❌ 검색 결과에서 선택 후 메타 로딩 중 typing 모드 복원 안됨 (HIGH)
**코드 위치**: `internal/tui/app.go:352-357`

```go
if (m.activePanel == PanelSearch || m.activePanel == PanelMeta) && len(m.searchResults) > 0 && m.cursor < len(m.searchResults) {
    selected := m.searchResults[m.cursor]
    m.selectedTable = &TableInfo{...}
    m.loading = true
    m.statusMsg = "📋 메타 로딩 중..."
    return m, m.doMeta(selected.OrgID, selected.TblID)
```

**문제점**:
- 메타 로딩 후 `typing = false`가 설정되지 않음
- `m.loading = true`이지만 `m.typing`은 여전히 true일 수 있음
- 메타 로딩 중 키를 누르면 문자가 입력될 수 있음

---

### 19. ❌ selectedTable과 metaInfo 의존성 문제 (MEDIUM)
**코드 위치**: `internal/tui/app.go:358-410`

```go
} else if m.activePanel == PanelParams {
    if m.selectedTable != nil && m.metaInfo != nil {
        // ...
    } else {
        m.statusMsg = "⚠ 먼저 통계표를 선택하고 메타 정보를 로드하세요."
    }
}
```

**문제점**:
- 두 조건이 동시에 필요함 (AND 관계)
- 사용자가 Enter를 누를 때 null check는 되지만, 어느 것이 문제인지 불명확
- selectedTable은 있는데 metaInfo가 없으면? (또는 반대)

**개선 제안**:
```go
if m.selectedTable == nil {
    m.statusMsg = "⚠ 통계표를 먼저 선택하세요."
} else if m.metaInfo == nil {
    m.statusMsg = "⚠ 메타 정보를 먼저 로드하세요 (Enter)."
} else {
    // 데이터 조회 실행
}
```

---

### 20. ❌ resultData가 있어도 PanelResult에서 스크롤 불가 (HIGH)
**코드 위치**: `internal/tui/app.go:860-865`

```go
for i, row := range m.resultData {
    if i >= 5 {  // ← 고정: 최대 5개만 표시
        lines = append(lines, "...")
        break
    }
```

**문제점**:
- 결과 테이블이 항상 5행 고정
- resultData가 10개 이상이어도 처음 5개만 보임
- 나머지 데이터는 "..." 로만 표시
- cursor나 스크롤 메커니즘 없음
- ↑↓ 키를 눌러도 테이블이 스크롤되지 않음

**권장 수정**:
- resultDataCursor 필드 추가
- PanelResult에서 ↑↓ 키로 스크롤 가능하게
- 화면에 맞게 데이터 표시

---

### 21. ❌ 즐겨찾기에서 항목 선택 후 재실행 기능 없음 (HIGH)
**코드 위치**: `internal/tui/app.go:604-629`

```go
// renderBookmarkSection은 즐겨찾기 섹션을 렌더링합니다.
func (m Model) renderBookmarkSection(width int) string {
    // ...
    for i, bm := range m.bookmarks {
        if i >= 3 {
            bookmarkLines = append(bookmarkLines, "  ...")
            break
        }
        bookmarkLines = append(bookmarkLines, "  "+truncateText(bm.Name, width-6)+" ("+bm.TblID+")")
    }
}
```

**문제점**:
- 즐겨찾기가 단순 표시만 됨
- 즐겨찾기 항목을 선택할 수 없음
- 즐겨찾기에서 Enter를 누르면 아무 일도 안 일어남
- "f 키로 추가"하기만 하고, "즐겨찾기에서 선택하면 자동 조회" 기능이 없음

**설계서 확인 필요**: 즐겨찾기 패널이 실제로 필요한가?

---

### 22. ❌ 이력에서 항목 선택 후 재실행 기능 없음 (HIGH)
**코드 위치**: `internal/tui/app.go:631-656`

```go
// renderHistorySection은 이력 섹션을 렌더링합니다.
func (m Model) renderHistorySection(width int) string {
    // ...
    for i, h := range m.histories {
        if i >= 3 {
            historyLines = append(historyLines, "  ...")
            break
        }
        historyLines = append(historyLines, "  "+truncateText(h.Command, width-6))
    }
}
```

**문제점**:
- 이력이 단순 표시만 됨 (최대 3개만 보임)
- 이력에서 항목을 선택할 수 없음
- 이력의 Command는 "orgid_tblid" 같은 문자열인데 그것도 확실하지 않음
- "이력에서 선택하면 자동 재실행" 같은 기능이 없음

**문제**: HistoryEntry의 Command 필드가 뭔지 불명확
- `internal/history/history.go:22` 보면 Command는 문자열이지만 저장되는 형식이 불명확

---

### 23. ⚠️ 검색 입력 중 Tab 키 동작 불명확 (MEDIUM)
**코드 위치**: `internal/tui/app.go:274-300`

```go
if m.typing {
    switch msg.String() {
    case KeyEnter, KeyEscape, "backspace":
        // ...
    default:
        if len(msg.Runes) > 0 {
            m.searchInput += string(msg.Runes)
        }
    }
    return m, nil
}
```

**문제점**:
- typing 모드에서 Tab을 누르면 default case에서 Tab의 runes이 있으면 입력됨
- Tab은 특수 키이므로 Runes가 없음 → 무시됨
- 하지만 코드 의도가 불명확 (명시적 차단이 아님)

---

### 24. ❌ paramClass1~8 필드가 있지만 파라미터 수정 불가 (CRITICAL)
**코드 위치**: `internal/tui/app.go:100-105`

```go
paramClass1   string
paramItem     string
paramPeriod   string
paramLatest   string
```

**문제점**:
- 파라미터 필드가 4개 있음 (Class1만 있고 Class2~8은 동적으로 설정)
- 하지만 TUI에서는 파라미터를 수정할 수 없음
- 파라미터 섹션 렌더링은 읽기 전용:
  ```go
  "분류: " + truncateText(class1Display, textWidth) + "\n" +
  "항목: " + truncateText(itemDisplay, textWidth) + "\n" +
  "주기: " + periodDisplay + "\n" +
  "최근: " + latestDisplay + "\n"
  ```
- "Enter로 조회"는 써있지만, 파라미터 값을 미리 입력할 수 없음
- 메타 로드 후 자동으로 ALL, ALL, Y, 5가 설정됨 (고정)

**권장 수정**:
- PanelParams에서 파라미터 수정 인터페이스 제공
- ↑↓↑↓로 파라미터 이동, Enter로 수정 등

---

### 25. ❌ resultDataCursor 필드 없음 (MEDIUM)
**코드 위치**: `internal/tui/app.go:84-105`

```go
type Model struct {
    cursor        int          // 현재 선택 인덱스 (← 검색 결과용)
    // resultDataCursor는 없음!
}
```

**문제점**:
- cursor 필드가 검색 결과(searchResults)의 인덱스로만 사용됨
- resultData의 행을 선택할 메커니즘이 없음
- 결과 테이블에서 특정 행을 선택하거나 스크롤할 수 없음

---

### 26. ⚠️ 로딩 상태에서 다른 키 입력 시 상태 검증 없음 (MEDIUM)
**코드 위치**: `internal/tui/app.go:270-411`

```go
// Update 함수가 loading=true일 때 키 입력을 처리하는 로직 없음
```

**문제점**:
- 검색 중(loading=true)에 사용자가 다른 키를 누르면?
  - Tab: 패널 전환 (loading 중인데 전환됨!)
  - ↑↓: cursor 이동 (로딩 중 결과 목록이 변경될 수 있음!)
  - Enter: PanelParams에서 또 다른 데이터 조회 시작 (중복 요청!)
- loading=true일 때 키 입력을 차단해야 함

**권장 수정**:
```go
if m.loading {
    // loading 중에는 특정 키만 처리 (Ctrl+C 등)
    switch msg.String() {
    case "ctrl+c":
        m.quitting = true
        return m, tea.Quit
    }
    return m, nil  // 다른 키는 무시
}
```

---

### 27. ❌ searchResults가 비어있을 때 PanelMeta로 전환되면 crash 위험 (MEDIUM)
**코드 위치**: `internal/tui/app.go:319-326`

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

**문제점**:
- PanelMeta에서 cursor를 사용하는 코드는 없음
- 하지만 만약 누군가 "메타 정보 목록 위/아래 탐색" 기능을 추가한다면?
- cursor는 검색 결과 길이 기준이므로, searchResults가 비면 cursor는 의미가 없어짐

---

### 28. ⚠️ 에러 메시지 길이가 상태바 폭을 초과할 때 처리 없음 (MEDIUM)
**코드 위치**: `internal/tui/app.go:950-958`

```go
if m.loading {
    status = "⏳ 로딩중..."
} else if m.err != nil {
    status = "❌ " + m.err.Error()  // ← 길이 제한 없음
} else {
    status = m.statusMsg
}
```

**문제점**:
- API 에러 메시지가 매우 길 수 있음
- 상태바가 한 줄이므로 긴 메시지는 잘림
- 사용자가 전체 에러 메시지를 볼 수 없음

**권장 수정**:
```go
const maxStatusLength = 100
status = m.err.Error()
if len(status) > maxStatusLength {
    status = status[:maxStatusLength-3] + "..."
}
status = "❌ " + status
```

---

### 29. ❌ 검색 후 결과 목록에 커서 포커스 자동 이동 안됨 (LOW)
**코드 위치**: `internal/tui/app.go:414-423`

```go
case searchResultMsg:
    m.loading = false
    if msg.err != nil {
        m.err = msg.err
        m.statusMsg = fmt.Sprintf("❌ 검색 오류: %v", msg.err)
    } else {
        m.searchResults = msg.results
        m.cursor = 0
        m.statusMsg = fmt.Sprintf("✓ %d건 검색 완료 (↑↓ 선택, Enter 확인)", len(msg.results))
    }
```

**문제점**:
- 검색 결과가 나오면 자동으로 PanelMeta로 전환되지 않음
- 사용자가 수동으로 Tab을 눌러야 PanelMeta에서 결과를 볼 수 있음
- "↑↓ 선택, Enter 확인" 메시지는 있지만, 패널 자동 전환이 없어 사용성 낮음

**UX 개선**:
```go
m.searchResults = msg.results
m.cursor = 0
m.activePanel = PanelMeta  // 자동으로 결과 패널로 전환
```

---

### 30. ⚠️ 메타 정보 로딩 후 자동 데이터 조회로 인한 UX 혼란 (MEDIUM)
**코드 위치**: `internal/tui/app.go:425-478`

```go
case metaResultMsg:
    m.loading = false
    if msg.err != nil {
        // ...
    } else {
        m.metaInfo = msg.meta
        if m.selectedTable != nil {
            // 메타 로드 → 자동으로 기본 파라미터 설정 → 자동 데이터 조회 → PanelResult로 점프
            opts := api.DataOptions{...}
            m.loading = true
            m.activePanel = PanelResult
            m.statusMsg = "📊 데이터 조회 중... (ALL, ALL, 최근 5개)"
            return m, m.doData(...)
        }
    }
```

**문제점**:
- 메타 조회 완료 → 자동으로 데이터 조회 시작
- 사용자가 메타 정보를 볼 시간이 거의 없음 (화면에 0.5초?)
- PanelMeta에서 뭔가 보려고 해도 이미 PanelResult로 이동함
- 사용자가 파라미터를 수정할 기회가 없음 (ALL, ALL 고정)

**UX 개선**:
```go
case metaResultMsg:
    m.metaInfo = msg.meta
    m.activePanel = PanelParams  // PanelParams로 먼저 전환
    m.statusMsg = "✓ 메타 정보 로드 완료. Enter로 데이터 조회"
    // 자동 데이터 조회 제거
```

---

### 31. ⚠️ 상태바 너비가 터미널 전체를 차지하지 않을 수 있음 (LOW)
**코드 위치**: `internal/tui/app.go:944-958`

```go
func (m Model) renderStatusBar() string {
    statusStyle := StatusBarStyle.
        Width(m.width)  // ← 항상 m.width로 설정
```

**문제점**:
- `m.width`가 제대로 초기화되지 않으면 (예: 0) 상태바가 렌더링 안됨
- 터미널 크기 변경 시 즉시 반응하는지 확인 필요
- tea.WindowSizeMsg 처리 있음 (490-492)이므로 대부분 OK

---

### 32. ❌ 다중 분류(Class2~8) 지원이 코드에는 있으나 UI 없음 (MEDIUM)
**코드 위치**: `internal/tui/app.go:462-470`

```go
n := m.metaInfo.NumClassGroups
if n >= 1 { opts.Class1 = "ALL" }
if n >= 2 { opts.Class2 = "ALL" }
if n >= 3 { opts.Class3 = "ALL" }
// ... 최대 Class8까지
```

**문제점**:
- API는 Class1~8을 지원함
- 하지만 Model에는 paramClass1만 있고, Class2~8은 필드가 없음
- 파라미터 섹션에서는 분류 1개만 보여줌
- 사용자가 Class2~8을 수정할 수 없음

---

### 33. ❌ 키 입력 시 좋지 않은 UX: 현재 어떤 통계표가 선택되었는지 왼쪽에서 표시 안됨 (LOW)
**코드 위치**: `internal/tui/app.go:549-712`

**문제점**:
- 검색 결과 목록(renderResultsSection)에서 현재 선택된 항목은 "▸" 마커로 표시됨 ✓
- 하지만 검색을 수행한 후 다른 패널로 이동하면, 어떤 항목이 선택되었는지 알 수 없음
- 사용자가 메타/파라미터/결과 패널에 있을 때, 왼쪽의 검색 결과에서 어떤 행이 선택되었는지 표시 부족

**개선 제안**:
```go
// renderResultsSection에서
if i == m.cursor {
    marker = "▸ "  // 현재 selectionHighlight
}
// 추가로, 색상 강조 가능
```

---

### 34. ⚠️ 패널 활성화 시각적 피드백 불충분 (LOW)
**코드 위치**: `internal/tui/app.go:578-602`, `internal/tui/app.go:665-712` 등

**문제점**:
- 각 섹션의 보더 색이 바뀜 (ColorAccent vs ColorBorder)
- 하지만 타이틀이나 배경색은 변하지 않음
- 어떤 패널이 "지금" 포커스되어 있는지 한눈에 알기 어려움

**개선 제안**:
- 활성 패널의 배경색 변경
- 또는 음영/명암 조정

---

### 35. ❌ searchInput 필드의 크기 제한 없음 (MEDIUM)
**코드 위치**: `internal/tui/app.go:87`

```go
searchInput   string       // 검색어 입력 (크기 제한 없음)
```

**문제점**:
- 사용자가 매우 긴 검색어를 계속 입력할 수 있음
- searchInput이 화면 폭을 초과할 수 있음
- 렌더링 시 truncateText로 자르지만, 메모리에는 전체 문자열 유지

**권장 수정**:
```go
case "backspace":
    // ...
default:
    if len(msg.Runes) > 0 && len(m.searchInput) < 200 {  // 최대 200자
        m.searchInput += string(msg.Runes)
    }
```

---

### 36. ⚠️ 도움말(?) 키가 한 번만 표시되는 UX (MEDIUM)
**코드 위치**: `internal/tui/app.go:317-318`

```go
case KeyHelp:
    m.statusMsg = KeyBindings()
```

**문제점**:
- "?"를 눌러도 상태바에 키 목록이 2~3초만 표시됨
- 다른 키를 누르면 사라짐
- 사용자가 다시 보려면 "?"를 또 눌러야 함
- 모달 도움말 창이 없음

**권장 개선**:
- 도움말 모드 플래그 추가
- "?"로 도움말 모드 ON/OFF
- 도움말 모드에서는 다른 입력을 처리하지 않음

---

### 37. ⚠️ 실제 파일 저장이 안됨: e 키 (내보내기) (HIGH)
**코드 위치**: `internal/tui/app.go:327-332`

```go
case KeyExport: // e 키: 내보내기
    if len(m.resultData) > 0 {
        m.statusMsg = "💾 내보내기: 기능 준비 중 (CLI로 사용: kosis d ... -o file.xlsx)"
    } else {
        m.statusMsg = "⚠ 내보낼 데이터가 없습니다. 먼저 데이터를 조회하세요."
    }
```

**문제점**:
- "기능 준비 중" 메시지만 표시
- 실제로 파일이 저장되지 않음
- TUI에서 "e" 키로 내보내기 불가능 (설계서에는 명시되었을 수 있음)

---

### 38. ⚠️ 실제 지표 모드 전환 불가: i 키 (MEDIUM)
**코드 위치**: `internal/tui/app.go:348-349`

```go
case KeyIndicator: // i 키: 주요지표 모드 전환
    m.statusMsg = "📊 주요지표 모드: CLI로 사용 → kosis ind s \"GDP\""
```

**문제점**:
- 메시지만 표시되고 아무것도 변하지 않음
- 지표 모드로 실제 전환이 안됨
- 설계서에서 "통계표/지표 모드 전환"이 명시되어 있다면 미구현

---

### 39. ⚠️ 즐겨찾기 추가 후 목록이 다시 렌더링되지 않을 수 있음 (LOW)
**코드 위치**: `internal/tui/app.go:333-347`

```go
case KeyBookmark: // f 키: 즐겨찾기 추가
    if m.selectedTable != nil {
        err := bookmark.Add(...)
        if err != nil {
            // ...
        } else {
            m.statusMsg = "★ 즐겨찾기에 추가: " + ...
            if newBookmarks, err := bookmark.List(); err == nil {
                m.bookmarks = newBookmarks  // ← 새로고침함
            }
        }
    }
```

**상태**: ✓ 구현됨 (새로고침 로직 있음)

**다만 문제**:
- 새로고침 로직은 있지만, renderBookmarkSection에서는 최대 3개만 보여줌
- 즐겨찾기가 3개 이상이면 "..." 표시
- 새로 추가한 즐겨찾기가 화면에 안 보일 수 있음

---

### 40. ❌ 이력 저장이 안됨: 조회 후 이력에 자동 추가 로직 부재 (HIGH)
**코드 위처**: `internal/tui/app.go:414-488`

```go
case searchResultMsg:
    // history.Add() 호출 없음
case metaResultMsg:
    // history.Add() 호출 없음
case dataResultMsg:
    // history.Add() 호출 없음
```

**문제점**:
- 데이터 조회 완료 후 history.Add()를 호출하지 않음
- renderHistorySection에서 "최근 조회" 목록이 항상 비어있음 (또는 New()에서 로드한 것만 보임)
- history.HistoryEntry의 Command 필드가 어떤 형식으로 저장되어야 하는지 불명확
- 조회한 orgID, tblID, 파라미터를 history에 저장해야 함

**권장 수정**:
```go
case dataResultMsg:
    m.loading = false
    if msg.err != nil {
        // ...
    } else {
        m.resultData = msg.data
        // 이력 저장
        if m.selectedTable != nil {
            command := fmt.Sprintf("%s_%s_%s",
                m.selectedTable.OrgID,
                m.selectedTable.TblID,
                m.paramClass1)
            history.Add(command, len(msg.data))  // ← 추가 필요
        }
        m.statusMsg = fmt.Sprintf("✓ %d건 조회 완료", len(msg.data))
    }
```

---

## 요약 (2차 발견 내용)

| # | 문제 | 심각도 | 범주 | 상태 |
|----|------|--------|------|------|
| 14 | err 필드가 초기화되지 않음 | MEDIUM | 상태 관리 | 버그 |
| 15 | statusMsg와 err 간 우선순위 불명확 | MEDIUM | 상태 관리 | 버그 |
| 16 | Esc로 나갈 때 이전 패널로 복원 안됨 | HIGH | 상태 관리 | 버그 |
| 17 | New()에서 typing=true로 초기화 | MEDIUM | UX | 버그 |
| 18 | 메타 로딩 중 typing 모드 복원 안됨 | HIGH | 상태 관리 | 버그 |
| 19 | selectedTable과 metaInfo 의존성 검증 부족 | MEDIUM | 에러 처리 | 개선 필요 |
| 20 | 결과 테이블 스크롤 불가 (5행 고정) | HIGH | 기능 완성도 | 미구현 |
| 21 | 즐겨찾기 항목 선택 불가 | HIGH | 기능 완성도 | 미구현 |
| 22 | 이력 항목 선택 불가 | HIGH | 기능 완성도 | 미구현 |
| 23 | 검색 입력 중 Tab 키 동작 불명확 | MEDIUM | 설계 모호 | 문서화 필요 |
| 24 | 파라미터 수정 불가 (읽기 전용) | **CRITICAL** | 기능 완성도 | **미구현** |
| 25 | resultDataCursor 필드 없음 | MEDIUM | 설계 부족 | 미구현 |
| 26 | loading=true일 때 키 입력 검증 없음 | MEDIUM | 상태 관리 | 버그 |
| 27 | searchResults 비었을 때 PanelMeta 전환 위험 | MEDIUM | 버그 위험 | 잠재 버그 |
| 28 | 에러 메시지 길이 제한 없음 | MEDIUM | 사용성 | 버그 |
| 29 | 검색 후 결과 목록 커서 포커스 미동작 | LOW | 사용성 | 개선 필요 |
| 30 | 메타 로딩 후 자동 데이터 조회 UX 혼란 | MEDIUM | 사용성 | 설계 재검토 |
| 31 | 상태바 너비 문제 | LOW | 사용성 | 거의 해결됨 |
| 32 | Class2~8 UI 미지원 | MEDIUM | 기능 완성도 | 미구현 |
| 33 | 현재 선택된 통계표 시각적 표시 부족 | LOW | 사용성 | 개선 필요 |
| 34 | 패널 활성화 시각적 피드백 부족 | LOW | 사용성 | 개선 필요 |
| 35 | searchInput 크기 제한 없음 | MEDIUM | 보안/성능 | 버그 |
| 36 | 도움말(?) 모달 없음 | MEDIUM | 사용성 | 미구현 |
| 37 | 내보내기(e) 실제 기능 없음 | HIGH | 기능 완성도 | 미구현 |
| 38 | 지표 모드(i) 전환 불가 | MEDIUM | 기능 완성도 | 미구현 |
| 39 | 즐겨찾기 3개 초과 시 새 항목 비표시 | LOW | 사용성 | 미구현 |
| 40 | 이력 저장 로직 부재 | **HIGH** | 기능 완성도 | **미구현** |

---

## 최종 결론

**1차 발견 13개 + 2차 발견 20개 = 총 33개 문제**

### 즉시 수정 필요 (CRITICAL):
1. **#24**: 파라미터 수정 불가 - TUI의 핵심 기능

### 높은 우선순위 (HIGH):
1. **#16**: Esc 패널 복원
2. **#18**: typing 모드 복원
3. **#20**: 결과 테이블 스크롤
4. **#21**: 즐겨찾기 선택
5. **#22**: 이력 선택
6. **#37**: 내보내기 기능
7. **#40**: 이력 저장

### 중간 우선순위 (MEDIUM):
- 에러/statusMsg 우선순위 명확화
- 로딩 중 키 입력 차단
- searchInput 크기 제한
- 메타 로딩 UX 개선

### 낮은 우선순위 (LOW):
- 시각적 피드백 강화
- 도움말 모달
- 기타 UX 개선
