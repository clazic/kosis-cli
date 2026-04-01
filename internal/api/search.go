package api

import (
	"encoding/json"
	"fmt"
)

// Search searches for statistics tables by keyword.
func (c *Client) Search(keyword string, opts SearchOptions) ([]SearchResult, error) {
	if keyword == "" {
		return nil, fmt.Errorf("검색어는 필수입니다")
	}

	params := map[string]string{
		"searchNm": keyword,
	}

	// Add optional parameters
	if opts.Sort != "" {
		params["sort"] = opts.Sort
	}
	if opts.StartCount > 0 {
		params["startCount"] = fmt.Sprintf("%d", opts.StartCount)
	}
	if opts.ResultCount > 0 {
		params["resultCount"] = fmt.Sprintf("%d", opts.ResultCount)
	} else {
		// Default result count
		params["resultCount"] = "20"
	}

	body, err := c.request("statisticsSearch.do?method=getList", params, false)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	return results, nil
}
