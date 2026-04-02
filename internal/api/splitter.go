package api

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SplitOptions 분할 조회 옵션
type SplitOptions struct {
	MaxCells    int  // 최대 셀 수 (기본 40000)
	NoAutoSplit bool // 자동 분할 비활성화
}

// PeriodChunk 시점 축으로 분할된 청크
type PeriodChunk struct {
	Start string
	End   string
}

// DataWithAutoSplit 4만 셀 초과 시 자동 분할 조회
// meta API로 분류값/항목 개수를 파악하고, 예상 셀 수를 계산한 후,
// 4만 초과 시 자동으로 시점 축 분할 조회를 수행합니다.
// API 키가 여러 개면 워커 풀 기반 병렬 조회를 수행합니다.
func (c *Client) DataWithAutoSplit(orgID, tblID string, opts DataOptions, splitOpts SplitOptions, progressFn func(current, total int)) ([]DataRow, error) {
	if orgID == "" || tblID == "" {
		return nil, fmt.Errorf("orgId와 tblId는 필수입니다")
	}

	// 기본값 설정
	if splitOpts.MaxCells == 0 {
		splitOpts.MaxCells = 40000
	}

	// 1. Meta API로 예상 셀 수 계산
	summary, summaryErr := c.MetaSummary(orgID, tblID)
	if summaryErr != nil {
		// Meta 조회 실패 시: start/end가 있으면 보수적으로 5개 청크로 분할 시도
		if opts.StartPrdDe != "" && opts.EndPrdDe != "" {
			conservativeChunks := c.splitByPeriod(opts, splitOpts.MaxCells*5+1, splitOpts.MaxCells)
			if len(conservativeChunks) > 0 {
				fmt.Fprintf(os.Stderr, "⚠ 메타 조회 실패로 보수적 분할(%d 청크)을 시도합니다.\n", len(conservativeChunks))
				if len(c.apiKeys) <= 1 {
					return c.dataWithAutoSplitSequential(orgID, tblID, opts, conservativeChunks, progressFn)
				}
				return c.dataWithAutoSplitParallel(orgID, tblID, opts, conservativeChunks, progressFn)
			}
		}
		// start/end도 없으면 안내 메시지와 함께 에러 반환
		return nil, fmt.Errorf("메타 정보 조회에 실패했습니다. --start, --end로 시점 범위를 지정하거나 --latest로 최근 N개만 조회하세요: %w", summaryErr)
	}
	estimatedCells := estimateCellCountFromSummary(summary, opts)

	// 2. 4만 이하면 일반 조회
	if estimatedCells <= splitOpts.MaxCells {
		if progressFn != nil {
			progressFn(1, 1)
		}
		return c.Data(orgID, tblID, opts)
	}

	// 3. NoAutoSplit이 true면 에러 반환
	if splitOpts.NoAutoSplit {
		return nil, fmt.Errorf("조회 데이터가 %d셀로 4만 셀 제한을 초과합니다. --no-auto-split을 제거하거나 범위를 축소하세요", estimatedCells)
	}

	// 4. 20만 초과 시 경고 (지금은 경고만, 실제로는 프롬프트 필요)
	if estimatedCells > 200000 {
		fmt.Fprintf(os.Stderr, "⚠ 예상 셀 수: 약 %d건. 이는 조회 시간이 오래 걸릴 수 있습니다.\n", estimatedCells)
		fmt.Fprintf(os.Stderr, "  범위를 축소하거나 --periods를 사용하여 특정 시점만 조회하세요.\n")
		// 현재 단계에서는 경고만 하고 계속 진행
	}

	// 5. start/end가 없으면 메타에서 추출 (분할에 필요)
	if opts.StartPrdDe == "" || opts.EndPrdDe == "" {
		if summary != nil && len(summary.Periods) > 0 {
			for _, p := range summary.Periods {
				prdCode := strings.ToUpper(strings.TrimSpace(p.PrdSe))
				// 수록주기가 매칭되면 시점 범위를 설정
				optsPrd := strings.ToUpper(opts.PrdSe)
				prdMatch := (optsPrd == "" || prdCode == optsPrd ||
					(optsPrd == "Y" && (prdCode == "년" || prdCode == "Y")) ||
					(optsPrd == "M" && (prdCode == "월" || prdCode == "M")) ||
					(optsPrd == "Q" && (prdCode == "분기" || prdCode == "Q")) ||
					(optsPrd == "H" && (prdCode == "반기" || prdCode == "H")))
				if prdMatch && p.StrtPrdDe != "" && p.EndPrdDe != "" {
					// 시점 형식 정규화 (예: "1995.01" → "199501", "2025 4/4" → "20254")
					start := strings.ReplaceAll(strings.ReplaceAll(p.StrtPrdDe, ".", ""), " ", "")
					end := strings.ReplaceAll(strings.ReplaceAll(p.EndPrdDe, ".", ""), " ", "")
					// 분기 형식 변환: "19951/4" → "19951"
					start = strings.Split(start, "/")[0]
					end = strings.Split(end, "/")[0]

					// --latest N 사용 시: 전체 범위가 아닌 최근 N개 시점만 계산
					if opts.NewEstPrdCnt != "" {
						if n, parseErr := strconv.Atoi(opts.NewEstPrdCnt); parseErr == nil && n > 0 {
							start = c.calcLatestStart(end, n, opts.PrdSe)
						}
					}

					opts.StartPrdDe = start
					opts.EndPrdDe = end
					opts.NewEstPrdCnt = "" // start/end를 쓰므로 latest 제거
					break
				}
			}
		}
	}

	// 시점 축으로 분할
	chunks := c.splitByPeriod(opts, estimatedCells, splitOpts.MaxCells)
	if len(chunks) == 0 {
		// 시점 분할이 불가능하고 예상 셀이 제한 초과이면 에러 반환
		if estimatedCells > splitOpts.MaxCells {
			return nil, fmt.Errorf("예상 셀 수(%d)가 %d셀 제한을 초과하지만 시점 축 분할이 불가능합니다. "+
				"분류값(--class1 등)을 좁히거나 --item으로 항목을 지정하여 조회 범위를 축소하세요", estimatedCells, splitOpts.MaxCells)
		}
		// 제한 이하이면 일반 조회
		return c.Data(orgID, tblID, opts)
	}

	// 6. API 키 개수에 따라 순차 또는 병렬 실행
	if len(c.apiKeys) <= 1 {
		// API 키 1개: 순차 실행 (기존 방식)
		return c.dataWithAutoSplitSequential(orgID, tblID, opts, chunks, progressFn)
	}

	// API 키 2개 이상: 워커 풀 기반 병렬 실행
	return c.dataWithAutoSplitParallel(orgID, tblID, opts, chunks, progressFn)
}

// dataWithAutoSplitSequential 순차 실행 (API 키 1개일 때)
func (c *Client) dataWithAutoSplitSequential(orgID, tblID string, opts DataOptions, chunks []PeriodChunk, progressFn func(current, total int)) ([]DataRow, error) {
	var allResults []DataRow
	for i, chunk := range chunks {
		if progressFn != nil {
			progressFn(i+1, len(chunks))
		}

		// 분할된 시점으로 옵션 생성
		chunkOpts := opts
		chunkOpts.StartPrdDe = chunk.Start
		chunkOpts.EndPrdDe = chunk.End

		results, err := c.Data(orgID, tblID, chunkOpts)
		if err != nil {
			// API 오류 코드 30: "데이터가 존재하지 않습니다" → 해당 구간은 건너뜀
			if strings.Contains(err.Error(), "API 오류 [30]") {
				continue
			}
			return allResults, fmt.Errorf("분할 조회 [%s~%s] 실패: %w", chunk.Start, chunk.End, err)
		}

		allResults = append(allResults, results...)
	}

	// 정렬 및 중복 제거
	sortByPeriod(allResults)
	allResults = deduplicateRows(allResults)

	return allResults, nil
}

// dataWithAutoSplitParallel 워커 풀 기반 병렬 실행 (API 키 2개 이상일 때)
// 429 에러 발생 시 해당 청크를 건너뛰고 계속 진행합니다 (다른 키로 재시도할 수 있도록).
func (c *Client) dataWithAutoSplitParallel(orgID, tblID string, opts DataOptions, chunks []PeriodChunk, progressFn func(current, total int)) ([]DataRow, error) {
	type chunkResult struct {
		Index int
		Data  []DataRow
		Err   error
	}

	numWorkers := len(c.apiKeys)
	if numWorkers == 0 {
		numWorkers = 1
	}

	// 결과 채널과 동시성 제한 세마포어
	results := make(chan chunkResult, len(chunks))
	sem := make(chan struct{}, numWorkers)

	// 각 청크를 워커에 분배하여 병렬 실행 (429 에러 시 최대 3회 지수 백오프 재시도)
	for i, chunk := range chunks {
		sem <- struct{}{} // 동시성 제한 획득
		go func(idx int, chk PeriodChunk, keyIdx int) {
			defer func() { <-sem }() // 동시성 제한 해제

			// 분할된 시점으로 옵션 생성
			chunkOpts := opts
			chunkOpts.StartPrdDe = chk.Start
			chunkOpts.EndPrdDe = chk.End

			// 특정 API 키를 사용하여 요청 (429 시 최대 3회 재시도)
			const maxRetries = 3
			var data []DataRow
			var err error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				data, err = c.dataWithSpecificKey(orgID, tblID, chunkOpts, keyIdx)
				if err == nil || !strings.Contains(err.Error(), "429") {
					break
				}
				if attempt < maxRetries {
					// 지수 백오프: 1s, 2s, 4s
					backoff := time.Duration(1<<uint(attempt)) * time.Second
					time.Sleep(backoff)
				}
			}
			results <- chunkResult{Index: idx, Data: data, Err: err}
		}(i, chunk, i%numWorkers)
	}

	// 결과 수집 (순서대로 정렬하기 위해 미리 할당)
	allResults := make([]chunkResult, len(chunks))
	completedCount := 0
	var lastErr error

	for i := 0; i < len(chunks); i++ {
		r := <-results
		allResults[r.Index] = r
		completedCount++

		// 진행률 업데이트
		if progressFn != nil {
			progressFn(completedCount, len(chunks))
		}

		if r.Err != nil {
			// API 오류 코드 30: "데이터가 존재하지 않습니다" → 해당 구간은 건너뜀
			if strings.Contains(r.Err.Error(), "API 오류 [30]") {
				continue
			}
			// 429 에러 발생 시 저장하고 계속 진행 (다른 키로 재시도 가능)
			if strings.Contains(r.Err.Error(), "429") {
				lastErr = r.Err
				continue
			}
			// 다른 에러 발생 시 즉시 반환
			return nil, fmt.Errorf("분할 조회 [%d] 실패: %w", r.Index, r.Err)
		}
	}

	// 429 에러만 발생했을 경우 경고하고 계속 진행
	if lastErr != nil {
		fmt.Fprintf(os.Stderr, "경고: 일부 청크에서 %v 발생. 재시도 중입니다.\n", lastErr)
	}

	// 순서대로 결과 병합
	var merged []DataRow
	for _, r := range allResults {
		merged = append(merged, r.Data...)
	}

	// 정렬 및 중복 제거
	sortByPeriod(merged)
	merged = deduplicateRows(merged)

	return merged, nil
}

// countFilterValues는 필터 값에서 실제 선택된 개수를 추정합니다.
// "ALL" → metaCount, "" → 1, "00+11+21" → 3, "11*" → metaCount/4 (추정)
func countFilterValues(filterVal string, metaCount int) int {
	if filterVal == "" {
		return 1
	}
	upper := strings.ToUpper(filterVal)
	if upper == "ALL" {
		return metaCount
	}
	if strings.HasSuffix(filterVal, "*") {
		// 하위 전체 선택: 전체의 1/2로 추정 (과소추정 방지를 위해 보수적으로 계산)
		est := metaCount / 2
		if est < 1 {
			est = 1
		}
		return est
	}
	// "00+11+21" → + 로 구분된 개수
	return len(strings.Split(filterVal, "+"))
}

// estimateCellCountFromSummary는 이미 조회된 MetaSummaryResult를 사용하여 예상 셀 수를 계산합니다.
// MetaSummary를 중복 호출하지 않도록 리팩토링된 함수입니다.
func estimateCellCountFromSummary(summary *MetaSummaryResult, opts DataOptions) int {
	// 분류 그룹별 항목 수 계산
	objGroups := map[string]int{}
	objOrder := []string{}
	for _, cl := range summary.Classifications {
		if cl.ObjID != "" {
			if _, exists := objGroups[cl.ObjID]; !exists {
				objOrder = append(objOrder, cl.ObjID)
			}
			objGroups[cl.ObjID]++
		}
	}

	// 각 분류 그룹의 실제 선택 개수 계산
	classFilters := []string{opts.Class1, opts.Class2, opts.Class3, opts.Class4, opts.Class5, opts.Class6, opts.Class7, opts.Class8}
	classCount := 1
	for i, objID := range objOrder {
		metaCount := objGroups[objID]
		filterVal := ""
		if i < len(classFilters) {
			filterVal = classFilters[i]
		}
		classCount *= countFilterValues(filterVal, metaCount)
	}
	if classCount == 0 {
		classCount = 1
	}

	// 항목 개수: 실제 필터 반영
	itemCount := countFilterValues(opts.Item, len(summary.Items))
	if itemCount == 0 {
		itemCount = 1
	}

	// 시점 개수 계산
	periodCount := 1
	// --latest가 있으면 그 값을 사용
	if opts.NewEstPrdCnt != "" {
		if n, err := strconv.Atoi(opts.NewEstPrdCnt); err == nil && n > 0 {
			periodCount = n
		}
	} else if opts.StartPrdDe != "" && opts.EndPrdDe != "" {
		// start~end 범위 (문자열이 4자 미만이면 연도 파싱 불가하므로 기본값 유지)
		if len(opts.StartPrdDe) >= 4 && len(opts.EndPrdDe) >= 4 {
			startY, _ := strconv.Atoi(opts.StartPrdDe[:4])
			endY, _ := strconv.Atoi(opts.EndPrdDe[:4])
			if endY >= startY {
				periodCount = endY - startY + 1
				prd := strings.ToUpper(opts.PrdSe)
				if prd == "M" { periodCount *= 12 }
				if prd == "Q" { periodCount *= 4 }
				if prd == "H" { periodCount *= 2 }
			}
		}
	} else {
		// 시점 정보 없으면 메타에서 추정
		for _, p := range summary.Periods {
			prdCode := strings.ToUpper(strings.TrimSpace(p.PrdSe))
			optsPrd := strings.ToUpper(opts.PrdSe)
			prdMatch := (optsPrd == "" || prdCode == optsPrd ||
				(optsPrd == "Y" && (prdCode == "년" || prdCode == "Y")) ||
				(optsPrd == "M" && (prdCode == "월" || prdCode == "M")) ||
				(optsPrd == "Q" && (prdCode == "분기" || prdCode == "Q")) ||
				(optsPrd == "H" && (prdCode == "반기" || prdCode == "H")))
			if prdMatch && p.StrtPrdDe != "" && p.EndPrdDe != "" {
				start := strings.ReplaceAll(strings.ReplaceAll(p.StrtPrdDe, ".", ""), " ", "")
				end := strings.ReplaceAll(strings.ReplaceAll(p.EndPrdDe, ".", ""), " ", "")
				start = strings.Split(start, "/")[0]
				end = strings.Split(end, "/")[0]
				// 문자열이 4자 미만이면 연도 파싱 불가하므로 건너뜀
				if len(start) < 4 || len(end) < 4 {
					break
				}
				startY, _ := strconv.Atoi(start[:4])
				endY, _ := strconv.Atoi(end[:4])
				if endY > startY {
					periodCount = endY - startY + 1
					if prdCode == "월" || prdCode == "M" { periodCount *= 12 }
					if prdCode == "분기" || prdCode == "Q" { periodCount *= 4 }
					if prdCode == "반기" || prdCode == "H" { periodCount *= 2 }
				}
				break
			}
		}
	}

	// 예상 행 수 = 분류 × 항목 × 시점, 셀 수 = 행 × 컬럼(~14)
	estimatedCells := classCount * itemCount * periodCount * 14

	return estimatedCells
}

// splitByPeriod 시점 축으로 분할
// startPrdDe와 endPrdDe를 기반으로 청크를 생성합니다.
func (c *Client) splitByPeriod(opts DataOptions, totalCells, maxCells int) []PeriodChunk {
	// 시점 정보가 없으면 분할 불가
	if opts.StartPrdDe == "" && opts.EndPrdDe == "" && opts.NewEstPrdCnt == "" {
		return nil
	}

	// 현재는 startPrdDe와 endPrdDe를 기반으로 분할
	if opts.StartPrdDe == "" || opts.EndPrdDe == "" {
		return nil
	}

	start := opts.StartPrdDe
	end := opts.EndPrdDe

	// 수록주기 파악
	period := opts.PrdSe
	if period == "" {
		period = "Y" // 기본값: 연
	}

	// 필요한 청크 개수 계산
	chunksNeeded := (totalCells + maxCells - 1) / maxCells
	if chunksNeeded <= 1 {
		return nil
	}

	// 연도 분할 (월별은 나중에 확장 가능)
	if period == "Y" || period == "y" {
		return c.splitYearRange(start, end, chunksNeeded)
	}

	// 월별 분할
	if period == "M" || period == "m" {
		return c.splitMonthRange(start, end, chunksNeeded)
	}

	// 분기별 분할
	if period == "Q" || period == "q" {
		return c.splitQuarterRange(start, end, chunksNeeded)
	}

	// 반기별 분할
	if period == "H" || period == "h" {
		return c.splitHalfRange(start, end, chunksNeeded)
	}

	return nil
}

// splitYearRange 연도 범위를 청크로 분할
// 예: "2015" ~ "2024" (10년)를 2개 청크로 분할하면 "2015" ~ "2019", "2020" ~ "2024"
func (c *Client) splitYearRange(start, end string, chunksNeeded int) []PeriodChunk {
	startYear, errStart := strconv.Atoi(start)
	endYear, errEnd := strconv.Atoi(end)

	if errStart != nil || errEnd != nil {
		return nil
	}

	totalYears := endYear - startYear + 1
	yearsPerChunk := (totalYears + chunksNeeded - 1) / chunksNeeded

	var chunks []PeriodChunk
	for i := 0; i < chunksNeeded; i++ {
		chunkStart := startYear + (i * yearsPerChunk)
		chunkEnd := chunkStart + yearsPerChunk - 1

		// 마지막 청크는 종료 연도를 endYear로 맞춤
		if i == chunksNeeded-1 {
			chunkEnd = endYear
		}

		if chunkStart <= endYear {
			chunks = append(chunks, PeriodChunk{
				Start: fmt.Sprintf("%d", chunkStart),
				End:   fmt.Sprintf("%d", chunkEnd),
			})
		}
	}

	return chunks
}

// splitMonthRange 월도 범위를 청크로 분할
// 예: "202001" ~ "202412" (24개월)를 2개 청크로 분할하면 "202001" ~ "202012", "202101" ~ "202412"
func (c *Client) splitMonthRange(start, end string, chunksNeeded int) []PeriodChunk {
	// start, end 형식: "YYYYMM"
	if len(start) != 6 || len(end) != 6 {
		return nil
	}

	startYear, _ := strconv.Atoi(start[:4])
	startMonth, _ := strconv.Atoi(start[4:6])
	endYear, _ := strconv.Atoi(end[:4])
	endMonth, _ := strconv.Atoi(end[4:6])

	// 월을 절대값으로 변환
	startTotal := startYear*12 + startMonth
	endTotal := endYear*12 + endMonth

	totalMonths := endTotal - startTotal + 1
	monthsPerChunk := (totalMonths + chunksNeeded - 1) / chunksNeeded

	var chunks []PeriodChunk
	for i := 0; i < chunksNeeded; i++ {
		chunkStartTotal := startTotal + (i * monthsPerChunk)
		chunkEndTotal := chunkStartTotal + monthsPerChunk - 1

		// 마지막 청크는 종료월을 endTotal로 맞춤
		if i == chunksNeeded-1 {
			chunkEndTotal = endTotal
		}

		if chunkStartTotal <= endTotal {
			chunkStartYear := chunkStartTotal / 12
			chunkStartMonth := (chunkStartTotal % 12)
			if chunkStartMonth == 0 {
				chunkStartMonth = 12
				chunkStartYear--
			}

			chunkEndYear := chunkEndTotal / 12
			chunkEndMonth := (chunkEndTotal % 12)
			if chunkEndMonth == 0 {
				chunkEndMonth = 12
				chunkEndYear--
			}

			chunks = append(chunks, PeriodChunk{
				Start: fmt.Sprintf("%04d%02d", chunkStartYear, chunkStartMonth),
				End:   fmt.Sprintf("%04d%02d", chunkEndYear, chunkEndMonth),
			})
		}
	}

	return chunks
}

// splitQuarterRange 분기 범위를 청크로 분할
// 예: "20151" ~ "20244" (10년 4분기)를 2개 청크로 분할
func (c *Client) splitQuarterRange(start, end string, chunksNeeded int) []PeriodChunk {
	// start, end 형식: "YYYYQ" (Q=1~4)
	if len(start) != 5 || len(end) != 5 {
		return nil
	}

	startYear, _ := strconv.Atoi(start[:4])
	startQuarter, _ := strconv.Atoi(start[4:5])
	endYear, _ := strconv.Atoi(end[:4])
	endQuarter, _ := strconv.Atoi(end[4:5])

	// 분기를 절대값으로 변환
	startTotal := startYear*4 + startQuarter
	endTotal := endYear*4 + endQuarter

	totalQuarters := endTotal - startTotal + 1
	quartersPerChunk := (totalQuarters + chunksNeeded - 1) / chunksNeeded

	var chunks []PeriodChunk
	for i := 0; i < chunksNeeded; i++ {
		chunkStartTotal := startTotal + (i * quartersPerChunk)
		chunkEndTotal := chunkStartTotal + quartersPerChunk - 1

		// 마지막 청크는 종료분기를 endTotal로 맞춤
		if i == chunksNeeded-1 {
			chunkEndTotal = endTotal
		}

		if chunkStartTotal <= endTotal {
			chunkStartYear := chunkStartTotal / 4
			chunkStartQuarter := (chunkStartTotal % 4)
			if chunkStartQuarter == 0 {
				chunkStartQuarter = 4
				chunkStartYear--
			}

			chunkEndYear := chunkEndTotal / 4
			chunkEndQuarter := (chunkEndTotal % 4)
			if chunkEndQuarter == 0 {
				chunkEndQuarter = 4
				chunkEndYear--
			}

			chunks = append(chunks, PeriodChunk{
				Start: fmt.Sprintf("%04d%d", chunkStartYear, chunkStartQuarter),
				End:   fmt.Sprintf("%04d%d", chunkEndYear, chunkEndQuarter),
			})
		}
	}

	return chunks
}

// splitHalfRange 반기 범위를 청크로 분할
// 예: "20151" ~ "20242" (10년 반기)를 2개 청크로 분할
func (c *Client) splitHalfRange(start, end string, chunksNeeded int) []PeriodChunk {
	// start, end 형식: "YYYYH" (H=1~2)
	if len(start) != 5 || len(end) != 5 {
		return nil
	}

	startYear, _ := strconv.Atoi(start[:4])
	startHalf, _ := strconv.Atoi(start[4:5])
	endYear, _ := strconv.Atoi(end[:4])
	endHalf, _ := strconv.Atoi(end[4:5])

	// 반기를 절대값으로 변환
	startTotal := startYear*2 + startHalf
	endTotal := endYear*2 + endHalf

	totalHalves := endTotal - startTotal + 1
	halvesPerChunk := (totalHalves + chunksNeeded - 1) / chunksNeeded

	var chunks []PeriodChunk
	for i := 0; i < chunksNeeded; i++ {
		chunkStartTotal := startTotal + (i * halvesPerChunk)
		chunkEndTotal := chunkStartTotal + halvesPerChunk - 1

		// 마지막 청크는 종료반기를 endTotal로 맞춤
		if i == chunksNeeded-1 {
			chunkEndTotal = endTotal
		}

		if chunkStartTotal <= endTotal {
			chunkStartYear := chunkStartTotal / 2
			chunkStartHalf := (chunkStartTotal % 2)
			if chunkStartHalf == 0 {
				chunkStartHalf = 2
				chunkStartYear--
			}

			chunkEndYear := chunkEndTotal / 2
			chunkEndHalf := (chunkEndTotal % 2)
			if chunkEndHalf == 0 {
				chunkEndHalf = 2
				chunkEndYear--
			}

			chunks = append(chunks, PeriodChunk{
				Start: fmt.Sprintf("%04d%d", chunkStartYear, chunkStartHalf),
				End:   fmt.Sprintf("%04d%d", chunkEndYear, chunkEndHalf),
			})
		}
	}

	return chunks
}

// calcLatestStart는 종료 시점에서 N개 시점 전의 시작 시점을 계산합니다.
// --latest N 사용 시 전체 범위 대신 최근 N개 시점만 조회하기 위해 사용합니다.
func (c *Client) calcLatestStart(end string, n int, prdSe string) string {
	period := strings.ToUpper(prdSe)
	if period == "" {
		period = "Y"
	}

	switch period {
	case "Y":
		endYear, err := strconv.Atoi(end)
		if err != nil {
			return end
		}
		return fmt.Sprintf("%d", endYear-n+1)
	case "M":
		if len(end) < 6 {
			return end
		}
		endYear, _ := strconv.Atoi(end[:4])
		endMonth, _ := strconv.Atoi(end[4:6])
		total := endYear*12 + endMonth - n + 1
		y := total / 12
		m := total % 12
		if m == 0 {
			m = 12
			y--
		}
		return fmt.Sprintf("%04d%02d", y, m)
	case "Q":
		if len(end) < 5 {
			return end
		}
		endYear, _ := strconv.Atoi(end[:4])
		endQ, _ := strconv.Atoi(end[4:5])
		total := endYear*4 + endQ - n + 1
		y := total / 4
		q := total % 4
		if q == 0 {
			q = 4
			y--
		}
		return fmt.Sprintf("%04d%d", y, q)
	case "H":
		if len(end) < 5 {
			return end
		}
		endYear, _ := strconv.Atoi(end[:4])
		endH, _ := strconv.Atoi(end[4:5])
		total := endYear*2 + endH - n + 1
		y := total / 2
		h := total % 2
		if h == 0 {
			h = 2
			y--
		}
		return fmt.Sprintf("%04d%d", y, h)
	}
	return end
}

// sortByPeriod PRD_DE(시점) 기준으로 오름차순 정렬
func sortByPeriod(rows []DataRow) {
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].PrdDe < rows[j].PrdDe
	})
}

// deduplicateRows 중복 행 제거 (C1~C8 + ITM_ID + PRD_DE 조합 기준)
func deduplicateRows(rows []DataRow) []DataRow {
	seen := make(map[string]bool, len(rows))
	result := make([]DataRow, 0, len(rows))
	for _, row := range rows {
		key := row.C1 + "|" + row.C2 + "|" + row.C3 + "|" + row.C4 + "|" +
			row.C5 + "|" + row.C6 + "|" + row.C7 + "|" + row.C8 + "|" +
			row.ItmID + "|" + row.PrdDe
		if !seen[key] {
			seen[key] = true
			result = append(result, row)
		}
	}
	return result
}
