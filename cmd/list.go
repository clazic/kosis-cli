package cmd

import (
	"fmt"

	"github.com/clazic/kosis-cli/internal/api"
	"github.com/clazic/kosis-cli/internal/config"
	"github.com/clazic/kosis-cli/internal/output"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "통계목록 탐색",
	Long: `KOSIS 통계목록 트리 탐색

주제별, 기관별, 국제통계 등 통계목록을 계층 구조로 탐색합니다.
상위 목록에서 하위로 따라가며 원하는 통계표를 찾을 수 있습니다.

사용법:
  kosis list [flags]
  kosis ls [flags]

플래그:
  --view <코드>       서비스뷰 (기본: MT_ZTITLE=주제별)
                      MT_OTITLE: 기관별, MT_RTITLE: 국제통계,
                      MT_BUKHAN: 북한통계, MT_GTITLE01: e-지방지표(주제별),
                      MT_GTITLE02: e-지방지표(지역별), MT_CHOSUN_TITLE: 광복이전,
                      MT_HANKUK_TITLE: 대한민국통계연감, MT_STOP_TITLE: 작성중지,
                      MT_TM1_TITLE: 대상별, MT_TM2_TITLE: 이슈별, MT_ETITLE: 영문
  --parent <ID>       상위 목록 ID (기본: A)
  -f, --format <type> 출력 형식: table(기본), json

예제:
  # 주제별 최상위 목록
  kosis ls

  # 기관별 목록
  kosis ls --view MT_OTITLE

  # "A" 하위 탐색
  kosis ls --parent A

관련 명령어:
  kosis search <키워드>   키워드로 직접 검색
  kosis meta <ORG> <TBL>  통계표 메타데이터 확인`,
	RunE: runList,
}

var (
	listView   string
	listParent string
	listFormat string
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listView, "view", "MT_ZTITLE", "서비스뷰 코드")
	listCmd.Flags().StringVar(&listParent, "parent", "A", "상위 목록 ID")
	listCmd.Flags().StringVarP(&listFormat, "format", "f", "table", "출력 형식: table, json")
}

func runList(cmd *cobra.Command, args []string) error {
	// API 키 가져오기
	keys, err := config.GetAPIKeys()
	if err != nil {
		return fmt.Errorf("설정 로드 실패: %w", err)
	}

	if len(keys) == 0 {
		fmt.Println(config.NoAPIKeyMessage())
		return fmt.Errorf("API 키가 설정되지 않았습니다")
	}

	// API 클라이언트 생성
	client, err := api.NewClient(keys)
	if err != nil {
		return fmt.Errorf("클라이언트 생성 실패: %w", err)
	}

	// 통계목록 조회
	results, err := client.List(api.ListOptions{
		VwCd:     listView,
		ParentID: listParent,
	})
	if err != nil {
		return fmt.Errorf("통계목록 조회 실패: %w", err)
	}

	// 결과를 맵 형식으로 변환
	// 목록 레벨: LIST_ID, LIST_NM 사용
	// 통계표 레벨: TBL_ID, TBL_NM, ORG_ID 사용
	data := make([]map[string]interface{}, len(results))
	for i, item := range results {
		// 표시 이름: LIST_NM이 있으면 목록, TBL_NM이 있으면 통계표
		displayNM := item.ListNM
		displayID := item.ListID
		if item.TblNM != "" {
			displayNM = item.TblNM
			displayID = item.TblID
		}
		data[i] = map[string]interface{}{
			"LIST_ID": displayID,
			"LIST_NM": displayNM,
			"ORG_ID":  item.OrgID,
			"TBL_ID":  item.TblID,
			"TBL_NM":  item.TblNM,
			"VW_CD":   item.VwCD,
		}
	}

	// 포맷터 생성
	formatter, err := output.NewFormatter(listFormat)
	if err != nil {
		return fmt.Errorf("포맷터 생성 실패: %w", err)
	}

	// 포맷팅된 출력
	if err := formatter.Format(data, output.FormatOptions{
		Korean: true,
	}); err != nil {
		return fmt.Errorf("출력 생성 실패: %w", err)
	}

	return nil
}
