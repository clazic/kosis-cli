package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

// buildDataParams 데이터 조회 파라미터 구성 (공통 함수)
// orgID, tblID와 DataOptions를 받아 API 요청용 파라미터 맵을 구성합니다.
func buildDataParams(orgID, tblID string, opts DataOptions) map[string]string {
	params := map[string]string{
		"orgId": orgID,
		"tblId": tblID,
	}

	// Add optional classification parameters
	if opts.Class1 != "" {
		params["objL1"] = opts.Class1
	}
	if opts.Class2 != "" {
		params["objL2"] = opts.Class2
	}
	if opts.Class3 != "" {
		params["objL3"] = opts.Class3
	}
	if opts.Class4 != "" {
		params["objL4"] = opts.Class4
	}
	if opts.Class5 != "" {
		params["objL5"] = opts.Class5
	}
	if opts.Class6 != "" {
		params["objL6"] = opts.Class6
	}
	if opts.Class7 != "" {
		params["objL7"] = opts.Class7
	}
	if opts.Class8 != "" {
		params["objL8"] = opts.Class8
	}

	// Add item and period parameters
	if opts.Item != "" {
		params["itmId"] = opts.Item
	}
	if opts.PrdSe != "" {
		params["prdSe"] = opts.PrdSe
	}

	// Add period parameters
	if opts.StartPrdDe != "" {
		params["startPrdDe"] = opts.StartPrdDe
	}
	if opts.EndPrdDe != "" {
		params["endPrdDe"] = opts.EndPrdDe
	}
	if opts.NewEstPrdCnt != "" {
		params["newEstPrdCnt"] = opts.NewEstPrdCnt
	}
	if opts.PrdInterval != "" {
		params["prdInterval"] = opts.PrdInterval
	}
	if opts.OutputFields != "" {
		params["outputFields"] = opts.OutputFields
	}

	return params
}

// Data retrieves data from statisticsParameterData API with optional parameters.
func (c *Client) Data(orgID, tblID string, opts DataOptions) ([]DataRow, error) {
	if orgID == "" || tblID == "" {
		return nil, fmt.Errorf("orgId와 tblId는 필수입니다")
	}

	params := buildDataParams(orgID, tblID, opts)

	body, err := c.request("Param/statisticsParameterData.do?method=getList", params, true)
	if err != nil {
		return nil, err
	}

	var results []DataRow
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	return results, nil
}

// DataWithPeriods handles non-continuous periods by making multiple requests.
func (c *Client) DataWithPeriods(orgID, tblID string, periods []string, opts DataOptions) ([]DataRow, error) {
	if orgID == "" || tblID == "" {
		return nil, fmt.Errorf("orgId와 tblId는 필수입니다")
	}

	if len(periods) == 0 {
		return nil, fmt.Errorf("기간은 필수입니다")
	}

	var allResults []DataRow

	// Request data for each period separately
	for _, period := range periods {
		// Make a copy of options to avoid mutation
		optsCopy := opts
		optsCopy.StartPrdDe = period
		optsCopy.EndPrdDe = period

		results, err := c.Data(orgID, tblID, optsCopy)
		if err != nil {
			return nil, fmt.Errorf("기간 %s 조회 실패: %w", period, err)
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// ParsePeriods parses period string in various formats.
// Supports: "2020,2022,2025" (non-continuous) or returns as is for continuous ranges.
func ParsePeriods(periodStr string) []string {
	periodStr = strings.TrimSpace(periodStr)
	if periodStr == "" {
		return []string{}
	}

	// Split by comma for non-continuous periods
	parts := strings.Split(periodStr, ",")
	var periods []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			periods = append(periods, trimmed)
		}
	}

	return periods
}

// IsNonContinuousPeriods checks if the period string contains multiple non-continuous values.
func IsNonContinuousPeriods(periodStr string) bool {
	parts := ParsePeriods(periodStr)
	return len(parts) > 1 && strings.Contains(periodStr, ",")
}

// dataWithSpecificKey retrieves data using a specific API key.
// keyIndex >= 0이면 해당 키를 사용, -1이면 라운드로빈 사용 (Data 함수와 동일).
// 설계서 8.5절: 멀티 API 키 병렬 조회에서 각 워커가 특정 키로 요청하도록 함.
func (c *Client) dataWithSpecificKey(orgID, tblID string, opts DataOptions, keyIndex int) ([]DataRow, error) {
	if orgID == "" || tblID == "" {
		return nil, fmt.Errorf("orgId와 tblId는 필수입니다")
	}

	// buildDataParams 함수로 파라미터 구성 (코드 중복 제거)
	params := buildDataParams(orgID, tblID, opts)

	// 데이터 응답은 캐시하지 않음 (설계서 8.5절)
	body, err := c.requestWithKey("Param/statisticsParameterData.do?method=getList", params, true, keyIndex)
	if err != nil {
		return nil, err
	}

	var results []DataRow
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	return results, nil
}
