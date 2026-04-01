package api

import (
	"encoding/json"
	"fmt"
)

// List retrieves statistics list for tree navigation.
func (c *Client) List(opts ListOptions) ([]StatList, error) {
	params := map[string]string{}

	// Add optional parameters
	if opts.VwCd != "" {
		params["vwCd"] = opts.VwCd
	}
	if opts.ParentID != "" {
		params["parentListId"] = opts.ParentID
	}

	body, err := c.request("statisticsList.do?method=getList", params, false)
	if err != nil {
		return nil, err
	}

	var results []StatList
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	return results, nil
}
