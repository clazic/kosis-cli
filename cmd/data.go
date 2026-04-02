package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/clazic/kosis-cli/internal/api"
	"github.com/clazic/kosis-cli/internal/chart"
	"github.com/clazic/kosis-cli/internal/config"
	"github.com/clazic/kosis-cli/internal/interactive"
	"github.com/clazic/kosis-cli/internal/output"
	"github.com/spf13/cobra"
)

var dataCmd = &cobra.Command{
	Use:     "data <ORG_ID> <TBL_ID> [flags]",
	Aliases: []string{"d"},
	Short:   "통계표 데이터 조회",
	Long: `KOSIS 통계표 데이터 조회

통계표의 실제 데이터(수치값)를 조회합니다.
분류, 항목, 시점을 지정하여 원하는 데이터를 가져옵니다.
인자 없이 실행하면 대화형 모드로 단계별 안내합니다.

⚠ 조회 전 반드시 kosis meta로 분류/항목 코드를 확인하세요.
  4만 셀 초과 시 자동으로 분할 조회합니다.

사용법:
  kosis data <ORG_ID> <TBL_ID> [flags]
  kosis d <ORG_ID> <TBL_ID> [flags]
  kosis data                              대화형 모드

파라미터:
  <ORG_ID>              기관 코드 (예: 101, 116, 301)
  <TBL_ID>              통계표 ID (예: DT_1IN1502, DT_MLTM_2086)

필수 플래그 (--user-id 미사용 시):
  -c1, --class1 <값>    분류1 코드 (ALL, "00+11", "11*")
  -i,  --item <값>      항목 코드 (ALL, "T10+T20")
  -p,  --period <코드>  수록주기 (Y=연, M=월, Q=분기, H=반기)

호환성 안내:
  root 명령이 -c1 ~ -c8 입력을 --class1 ~ --class8로 정규화합니다.
  따라서 예제처럼 -c1 형식도 사용할 수 있습니다.

시점 플래그 (택1, 미지정시 최근 1개):
  -s, --start <시점>    시작 시점 (예: 2020, 202401)
  -e, --end <시점>      종료 시점 (예: 2024, 202412)
  -l, --latest <N>      최근 N개 시점
  --periods <시점들>     비연속 시점 지정 (쉼표 구분, 예: "2020,2022,2025")

시점 플래그 사용 방법:
  · --start + --end     연속 범위 (예: 2020~2024 전체)
  · --latest N          최근 N개 시점
  · --periods           원하는 시점만 골라서 (예: 2020,2022,2025)
  · 미지정              최근 1개 시점

선택 플래그:
  -c2 ~ -c8, --class2 ~ --class8  분류2~8 코드
  -f, --format <type>   출력 형식: table(기본), json, csv
  -o, --output <파일>   파일 저장 (.csv/.xlsx/.json/.db/.parquet)
  --fields <필드목록>    출력 필드 선택 ("C1_NM,ITM_NM,PRD_DE,DT")
  --user-id <ID>        자료등록 방식 (userStatsId로 조회, class/item/period 생략 가능)
  --no-auto-split       4만 셀 초과 시 자동 분할 비활성화

값 지정 규칙:
  ALL                   전체 선택
  "00+11+21"            복수 선택 (+로 구분)
  "11*"                 하위 전체 (* 접미사)`,

	Example: `  # 서울 미분양 최근 6개월
  kosis d 116 DT_MLTM_2086 -c1 "11" -i T10 -p M -l 6

  # 전국+서울 총인구 2020~2024 (연별)
  kosis d 101 DT_1IN1502 -c1 "00+11" -i T100 -p Y -s 2020 -e 2024

  # 특정 연도만 골라서 조회 (비연속)
  kosis d 101 DT_1IN1502 -c1 "00" -i T100 -p Y --periods "2020,2022,2025"

  # 특정 월만 골라서 조회
  kosis d 101 DT_1J20001 -c1 "0" -i T10 -p M --periods "202401,202407,202501"

  # GDP 최근 5년, 엑셀로 저장
  kosis d 301 DT_200Y001 -c1 "10100" -i T01 -p Y -l 5 -o gdp.xlsx

  # 소비자물가 월별 JSON → jq 파이프
  kosis d 101 DT_1J20001 -c1 "0" -i T10 -p M -l 12 -f json | jq '.[].DT'

  # 대용량: 전국 읍면동 인구 전체 → SQLite
  kosis d 101 DT_1IN1502 -c1 ALL -i ALL -p Y -s 2015 -e 2024 -o 인구.db

  # 대화형 모드 (단계별 안내)
  kosis data

다음 단계:
  조회 결과를 저장하거나 후속 분석하려면:
  kosis d <ORG_ID> <TBL_ID> ... -o result.xlsx
  kosis d <ORG_ID> <TBL_ID> ... -f json | jq '.'`,

	Run: func(cmd *cobra.Command, args []string) {
		// 대화형 모드: 인자가 없으면 단계별 입력
		if len(args) == 0 {
			runDataInteractive()
			return
		}

		// CLI 모드: 모든 인자가 제공되어야 함
		if len(args) < 2 {
			fmt.Println("오류: ORG_ID와 TBL_ID는 필수입니다")
			fmt.Println()
			cmd.Help()
			return
		}

		// 사용자가 --format을 명시하지 않았으면 config의 default_format 적용
		if !cmd.Flags().Changed("format") {
			if cfg, err := config.Load(); err == nil && cfg.DefaultFormat != "" {
				formatFlag = cfg.DefaultFormat
			}
		}

		orgID := args[0]
		tblID := args[1]

		periodsValue := strings.TrimSpace(periodsFlag)
		hasRange := startFlag != "" || endFlag != ""
		hasLatest := cmd.Flags().Changed("latest")
		hasPeriods := periodsValue != ""

		if userIDFlag == "" && (class1Flag == "" || itemFlag == "" || periodFlag == "") {
			fmt.Println("오류: -c1/--class1, --item, --period는 필수입니다 (또는 --user-id 사용)")
			fmt.Println()
			cmd.Help()
			return
		}
		if hasLatest && latestFlag <= 0 {
			fmt.Println("오류: --latest는 1 이상의 정수여야 합니다.")
			fmt.Println()
			cmd.Help()
			return
		}
		if (startFlag == "") != (endFlag == "") {
			fmt.Println("오류: --start와 --end는 함께 사용해야 합니다.")
			fmt.Println()
			cmd.Help()
			return
		}
		if hasPeriods && (hasLatest || hasRange) {
			fmt.Println("오류: --periods는 --latest 또는 --start/--end와 함께 사용할 수 없습니다.")
			fmt.Println()
			cmd.Help()
			return
		}
		if hasLatest && hasRange {
			fmt.Println("오류: --latest와 --start/--end는 함께 사용할 수 없습니다.")
			fmt.Println()
			cmd.Help()
			return
		}

		// API 클라이언트 생성 (캐시 초기화 포함)
		client, err := NewAPIClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "오류: %v\n", err)
			return
		}

		// 플래그에서 옵션 생성
		opts := api.DataOptions{
			Class1:     class1Flag,
			Class2:     class2Flag,
			Class3:     class3Flag,
			Class4:     class4Flag,
			Class5:     class5Flag,
			Class6:     class6Flag,
			Class7:     class7Flag,
			Class8:     class8Flag,
			Item:       itemFlag,
			PrdSe:      periodFlag,
			StartPrdDe: startFlag,
			EndPrdDe:   endFlag,
			// OutputFields는 API에 전달하지 않음 (클라이언트 사이드 필터링으로 처리)
		}
		if hasLatest && latestFlag > 0 {
			opts.NewEstPrdCnt = strconv.Itoa(latestFlag)
		}

		var results []api.DataRow

		// 실행 로직: --user-id, --periods, 기본
		if userIDFlag != "" {
			// 1. --user-id가 있으면 client.DataRegistered() 사용
			results, err = client.DataRegistered(userIDFlag, opts)
		} else if periodsValue != "" {
			// 2. --periods가 있으면 client.DataWithPeriods() 사용
			periods := strings.Split(periodsValue, ",")
			for i := range periods {
				periods[i] = strings.TrimSpace(periods[i])
			}
			results, err = client.DataWithPeriods(orgID, tblID, periods, opts)
		} else {
			// 3. 그 외 → client.DataWithAutoSplit() 사용 (4만 셀 초과 시 자동 분할)
			splitOpts := api.SplitOptions{
				MaxCells:    40000,
				NoAutoSplit: noAutoSplitFlag,
			}
			var progressFn func(int, int)
			if formatFlag == "table" {
				progressFn = func(current, total int) {
					if total > 1 {
						fmt.Fprintf(os.Stderr, "[%d/%d] 조회 중...\n", current, total)
					}
				}
			}
			results, err = client.DataWithAutoSplit(orgID, tblID, opts, splitOpts, progressFn)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "오류: %v\n", err)
			return
		}

		// 메타 조회하여 실제 컬럼명 가져오기
		var colMeta *api.ColumnMeta
		var colMetaFull *api.ColumnMeta // 필터링 전 전체 메타 (--fields 변환용)
		if orgID != "" && tblID != "" {
			if summary, err := client.MetaSummary(orgID, tblID); err == nil {
				colMetaFull = summary.BuildColumnMeta()
				colMeta = colMetaFull
				if len(results) > 0 {
					colMeta = colMetaFull.FilterByData(results)
				}
			}
		}

		// 데이터를 동적 컬럼명으로 map 슬라이스로 변환
		dataMap := make([]map[string]interface{}, len(results))
		for i, row := range results {
			m := make(map[string]interface{})
			if colMeta != nil {
				for _, col := range colMeta.Columns {
					m[col.Label] = row.GetField(col.Key)
				}
			} else {
				// 폴백: 기존 고정 키
				m["분류값명1"] = row.C1NM
				m["분류값명2"] = row.C2NM
				m["분류값명3"] = row.C3NM
				m["분류값명4"] = row.C4NM
				m["분류값명5"] = row.C5NM
				m["분류값명6"] = row.C6NM
				m["분류값명7"] = row.C7NM
				m["분류값명8"] = row.C8NM
				m["항목명"] = row.ItmNM
				m["수록주기"] = row.PrdSe
				m["수록시점"] = row.PrdDe
				m["수치값"] = row.DT
				m["단위"] = row.UnitNM
				m["비고"] = row.LstChn
			}
			dataMap[i] = m
		}

		// 포맷 옵션 생성 - 동적 컬럼 순서 지정
		formatOpts := output.FormatOptions{
			Korean: true,
		}
		if colMeta != nil {
			for _, col := range colMeta.Columns {
				formatOpts.Columns = append(formatOpts.Columns, col.Label)
			}
		}

		// --fields가 있으면 사용자 지정 컬럼으로 덮어쓰기
		// API 키(C1_NM, DT 등)와 한글 라벨(시도별, 수치값 등) 모두 지원
		if fieldsFlag != "" {
			fields := strings.Split(fieldsFlag, ",")
			// 변환용 메타: 필터링 전 전체 메타 사용 (필터링 후에는 매핑이 빠질 수 있음)
			lookupMeta := colMetaFull
			if lookupMeta == nil {
				lookupMeta = colMeta
			}
			for i := range fields {
				fields[i] = strings.TrimSpace(fields[i])
				// API 키가 입력된 경우 한글 라벨로 변환
				if lookupMeta != nil {
					if label := lookupMeta.GetLabel(fields[i]); label != fields[i] {
						fields[i] = label
					}
				}
			}
			formatOpts.Columns = fields
		}

		// --chart가 있으면 차트 렌더링
		if dataChartFlag != "" {
			chartType, err := chart.ParseChartType(dataChartFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "오류: %v\n", err)
				return
			}

			chartFmt := chart.FormatTerminal
			if dataChartFmtFlag != "" {
				chartFmt, err = chart.ParseChartFormat(dataChartFmtFlag)
				if err != nil {
					fmt.Fprintf(os.Stderr, "오류: %v\n", err)
					return
				}
			}

			seriesList, axisInfo := chart.ExtractSeriesWithAxis(dataMap)
			if len(seriesList) == 0 {
				fmt.Fprintln(os.Stderr, "오류: 차트를 생성할 수 있는 데이터가 없습니다")
				return
			}

			chartOpts := chart.Options{
				Type:   chartType,
				Format: chartFmt,
				Output: outputFlag,
				Open:   dataOpenFlag,
				XLabel: axisInfo.XLabel,
				YLabel: axisInfo.YLabel,
			}

			if err := chart.Render(seriesList, chartOpts); err != nil {
				fmt.Fprintf(os.Stderr, "차트 오류: %v\n", err)
				return
			}

			if outputFlag != "" && chartFmt != chart.FormatTerminal {
				fmt.Fprintf(os.Stderr, "✓ 차트가 %s로 저장되었습니다.\n", outputFlag)
			}
			return
		}

		// --output이 있으면 WriteToFile() 사용
		if outputFlag != "" {
			err = output.WriteToFile(dataMap, outputFlag, formatOpts)
			if err != nil {
				fmt.Printf("파일 저장 오류: %v\n", err)
				return
			}
			fmt.Printf("✓ 데이터가 %s로 저장되었습니다.\n", outputFlag)
		} else {
			// 없으면 NewFormatter(format)으로 stdout 출력
			formatter, err := output.NewFormatter(formatFlag)
			if err != nil {
				fmt.Printf("포맷 오류: %v\n", err)
				return
			}
			err = formatter.Format(dataMap, formatOpts)
			if err != nil {
				fmt.Printf("포맷팅 오류: %v\n", err)
				return
			}
		}
	},
}

var (
	class1Flag      string
	class2Flag      string
	class3Flag      string
	class4Flag      string
	class5Flag      string
	class6Flag      string
	class7Flag      string
	class8Flag      string
	itemFlag        string
	periodFlag      string
	startFlag       string
	endFlag         string
	latestFlag      int
	periodsFlag     string
	formatFlag      string
	outputFlag      string
	fieldsFlag      string
	userIDFlag      string
	noAutoSplitFlag bool
	dataChartFlag   string
	dataChartFmtFlag string
	dataOpenFlag    bool
)

// runDataInteractive implements interactive mode for data command.
func runDataInteractive() {
	// API 클라이언트 생성 (캐시 초기화 포함)
	client, err := NewAPIClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "오류: %v\n", err)
		return
	}

	// 1. 검색어 입력
	keyword := interactive.Prompt("? 검색어를 입력하세요:")
	if keyword == "" {
		fmt.Println("검색어가 입력되지 않았습니다.")
		return
	}

	// 2. 검색 실행
	fmt.Print("  📋 검색 중...\n")
	searchOpts := api.SearchOptions{ResultCount: 20}
	searchResults, err := client.Search(keyword, searchOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "오류: %v\n", err)
		return
	}

	if len(searchResults) == 0 {
		fmt.Println("검색 결과가 없습니다.")
		return
	}

	// 통계표 선택 옵션 구성
	var tableOptions []string
	for _, r := range searchResults {
		tableOptions = append(tableOptions, fmt.Sprintf("%s (%s/%s)", r.TblNM, r.OrgID, r.TblID))
	}

	// 3. 통계표 선택
	selectedIdx, _ := interactive.Select("? 통계표를 선택하세요:", tableOptions)
	if selectedIdx < 0 {
		fmt.Println("통계표를 선택해야 합니다.")
		return
	}

	selectedTable := searchResults[selectedIdx]
	orgID := selectedTable.OrgID
	tblID := selectedTable.TblID

	// 4. 메타 데이터 로드
	fmt.Print("  📋 메타 로딩 중...\n")
	metaSummary, err := client.MetaSummary(orgID, tblID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "오류: %v\n", err)
		return
	}

	// 5. 지역(분류1) 선택
	var classOptions []string
	classCodeMap := make(map[int]string)
	for i, c := range metaSummary.Classifications {
		code := c.Code
		if code == "" {
			code = c.ItmID
		}
		label := c.Label
		if label == "" {
			label = c.ItmNM
		}
		classOptions = append(classOptions, fmt.Sprintf("%s (%s)", label, code))
		classCodeMap[i] = code
	}

	selectedClassIndices := interactive.MultiSelect("? 지역(분류)을 선택하세요 (공백 구분 번호):", classOptions)
	if len(selectedClassIndices) == 0 {
		fmt.Println("분류를 선택해야 합니다.")
		return
	}

	var selectedClassCodes []string
	for _, idx := range selectedClassIndices {
		selectedClassCodes = append(selectedClassCodes, classCodeMap[idx])
	}
	class1Value := strings.Join(selectedClassCodes, "+")

	// 6. 항목 선택
	var itemOptions []string
	itemCodeMap := make(map[int]string)
	for i, it := range metaSummary.Items {
		code := it.Code
		if code == "" {
			code = it.ItmID
		}
		label := it.Label
		if label == "" {
			label = it.ItmNM
		}
		itemOptions = append(itemOptions, fmt.Sprintf("%s (%s)", label, code))
		itemCodeMap[i] = code
	}

	itemIdx, _ := interactive.Select("? 항목을 선택하세요:", itemOptions)
	if itemIdx < 0 {
		fmt.Println("항목을 선택해야 합니다.")
		return
	}

	itemValue := itemCodeMap[itemIdx]

	// 7. 수록주기 자동 감지 (또는 사용자 선택)
	// 간단히 처음 주기를 사용하거나 사용자에게 물을 수 있음
	var periodValue string
	if len(metaSummary.Periods) > 0 {
		periodValue = metaSummary.Periods[0].Code
	} else {
		periodValue = "M" // 기본값
	}

	// 8. 기간: 최근 몇 개 시점?
	latestCount := interactive.SelectInt("? 기간: 최근 몇 개 시점? (기본 5):", 5)

	// 9. 데이터 조회
	fmt.Print("  📊 데이터 조회 중...\n")
	opts := api.DataOptions{
		Class1:       class1Value,
		Item:         itemValue,
		PrdSe:        periodValue,
		NewEstPrdCnt: fmt.Sprintf("%d", latestCount),
	}

	splitOpts := api.SplitOptions{
		MaxCells:    40000,
		NoAutoSplit: false,
	}

	results, err := client.DataWithAutoSplit(orgID, tblID, opts, splitOpts, func(current, total int) {
		if total > 1 {
			fmt.Fprintf(os.Stderr, "[%d/%d] 조회 중...\n", current, total)
		}
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "오류: %v\n", err)
		return
	}

	fmt.Printf("✓ %d건 조회 완료\n", len(results))

	// 10. 재사용 명령어 표시
	fmt.Printf("\n💡 다음에 같은 조회:\n   kosis d %s %s -c1 \"%s\" -i %s -p %s -l %d\n\n",
		orgID, tblID, class1Value, itemValue, periodValue, latestCount)

	// 11. 내보내기 여부
	if interactive.Confirm("? 내보내기 하시겠습니까?", false) {
		exportChoices := []string{
			"CSV",
			"Excel (XLSX)",
			"JSON",
			"SQLite",
			"Parquet",
		}
		exportIdx, _ := interactive.Select("? 내보내기 형식을 선택하세요:", exportChoices)

		if exportIdx >= 0 {
			var ext string
			switch exportIdx {
			case 0:
				ext = ".csv"
			case 1:
				ext = ".xlsx"
			case 2:
				ext = ".json"
			case 3:
				ext = ".db"
			case 4:
				ext = ".parquet"
			}

			filename := fmt.Sprintf("%s_%s%s", orgID, tblID, ext)
			filename = interactive.Prompt(fmt.Sprintf("? 파일명 [기본: %s]:", filename))
			if filename == "" {
				filename = fmt.Sprintf("%s_%s%s", orgID, tblID, ext)
			}

			// 데이터를 map 슬라이스로 변환
			dataMap := make([]map[string]interface{}, len(results))
			for i, row := range results {
				dataMap[i] = map[string]interface{}{
					"C1_NM":  row.C1NM,
					"C2_NM":  row.C2NM,
					"C3_NM":  row.C3NM,
					"C4_NM":  row.C4NM,
					"C5_NM":  row.C5NM,
					"C6_NM":  row.C6NM,
					"C7_NM":  row.C7NM,
					"C8_NM":  row.C8NM,
					"ITM_NM": row.ItmNM,
					"PRD_DE": row.PrdDe,
					"DT":     row.DT,
					"CMMT":   row.LstChn,
					"UNIT":   row.UnitNM,
					"PRD_SE": row.PrdSe,
				}
			}

			formatOpts := output.FormatOptions{Korean: true}
			err := output.WriteToFile(dataMap, filename, formatOpts)
			if err != nil {
				fmt.Printf("파일 저장 오류: %v\n", err)
				return
			}
			fmt.Printf("✓ 데이터가 %s로 저장되었습니다.\n", filename)
		}
	} else {
		// 내보내기하지 않으면 테이블 형식으로 출력
		dataMap := make([]map[string]interface{}, len(results))
		for i, row := range results {
			dataMap[i] = map[string]interface{}{
				"C1_NM":  row.C1NM,
				"ITM_NM": row.ItmNM,
				"PRD_DE": row.PrdDe,
				"DT":     row.DT,
			}
		}

		formatter, err := output.NewFormatter("table")
		if err != nil {
			fmt.Printf("포맷 오류: %v\n", err)
			return
		}
		formatOpts := output.FormatOptions{Korean: true}
		err = formatter.Format(dataMap, formatOpts)
		if err != nil {
			fmt.Printf("포맷팅 오류: %v\n", err)
			return
		}
	}
}

func init() {
	rootCmd.AddCommand(dataCmd)

	// 플래그 정의
	// class 계열 short flag(-c1~-c8)는 root에서 사전 정규화하여 지원합니다.
	dataCmd.Flags().StringVar(&class1Flag, "class1", "", "분류1 코드 (ALL, \"00+11\", \"11*\")")
	dataCmd.Flags().StringVar(&class2Flag, "class2", "", "분류2 코드")
	dataCmd.Flags().StringVar(&class3Flag, "class3", "", "분류3 코드")
	dataCmd.Flags().StringVar(&class4Flag, "class4", "", "분류4 코드")
	dataCmd.Flags().StringVar(&class5Flag, "class5", "", "분류5 코드")
	dataCmd.Flags().StringVar(&class6Flag, "class6", "", "분류6 코드")
	dataCmd.Flags().StringVar(&class7Flag, "class7", "", "분류7 코드")
	dataCmd.Flags().StringVar(&class8Flag, "class8", "", "분류8 코드")
	dataCmd.Flags().StringVarP(&itemFlag, "item", "i", "", "항목 코드 (ALL, \"T10+T20\")")
	dataCmd.Flags().StringVarP(&periodFlag, "period", "p", "", "수록주기 (Y=연, M=월, Q=분기, H=반기)")
	dataCmd.Flags().StringVarP(&startFlag, "start", "s", "", "시작 시점 (예: 2020, 202401)")
	dataCmd.Flags().StringVarP(&endFlag, "end", "e", "", "종료 시점 (예: 2024, 202412)")
	dataCmd.Flags().IntVarP(&latestFlag, "latest", "l", 0, "최근 N개 시점")
	dataCmd.Flags().StringVar(&periodsFlag, "periods", "", "비연속 시점 지정 (쉼표 구분, 예: \"2020,2022,2025\")")
	dataCmd.Flags().StringVarP(&formatFlag, "format", "f", "table", "출력 형식: table(기본), json, csv")
	dataCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "파일 저장 (.csv/.xlsx/.json/.db/.parquet)")
	dataCmd.Flags().StringVar(&fieldsFlag, "fields", "", "출력 필드 선택 (\"C1_NM,ITM_NM,PRD_DE,DT\")")
	dataCmd.Flags().StringVar(&userIDFlag, "user-id", "", "자료등록 방식 (userStatsId로 조회)")
	dataCmd.Flags().BoolVar(&noAutoSplitFlag, "no-auto-split", false, "4만 셀 초과 시 자동 분할 비활성화")
	dataCmd.Flags().StringVar(&dataChartFlag, "chart", "", "차트 타입: line, bar, pie")
	dataCmd.Flags().StringVar(&dataChartFmtFlag, "chart-format", "", "차트 출력 포맷: terminal(기본), png, svg, pdf, html, excel")
	dataCmd.Flags().BoolVar(&dataOpenFlag, "open", false, "차트 생성 후 자동 열기")
}
