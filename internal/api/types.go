// Package api provides KOSIS Open API client and data structures.
package api

// SearchResult represents a search result from statisticsSearch API.
type SearchResult struct {
	OrgID     string `json:"ORG_ID"`
	TblID     string `json:"TBL_ID"`
	TblNM     string `json:"TBL_NM"`
	OrgNM     string `json:"ORG_NM"`
	StatID    string `json:"STAT_ID"`
	StatNM    string `json:"STAT_NM"`
	VwCD      string `json:"VW_CD"`
	MTAtitle  string `json:"MT_ATITLE"`
	StrtPrdDe string `json:"STRT_PRD_DE"`
	EndPrdDe  string `json:"END_PRD_DE"`
	RecTblSe  string `json:"REC_TBL_SE"`
}

// MetaResult represents metadata result from statisticsData.getMeta API.
type MetaResult struct {
	ObjID   string `json:"OBJ_ID"`
	ObjNM   string `json:"OBJ_NM"`
	ObjIDSN string `json:"OBJ_ID_SN"`
	ItmID   string `json:"ITM_ID"`
	ItmNM   string `json:"ITM_NM"`
	UpItmID string `json:"UP_ITM_ID"`
	Level   int    `json:"LEVEL"`
	Type    string `json:"TYPE"`
	Code    string `json:"CODE"`
	Label   string `json:"LABEL"`
	PrdSe     string `json:"PRD_SE"`
	PrdDe     string `json:"PRD_DE"`
	StrtPrdDe string `json:"STRT_PRD_DE"`
	EndPrdDe  string `json:"END_PRD_DE"`
	UnitNM    string `json:"UNIT_NM"`
}

// MetaSummaryResult groups classification, item, and period metadata.
type MetaSummaryResult struct {
	Classifications []MetaResult `json:"classifications"`
	Items           []MetaResult `json:"items"`
	Periods         []MetaResult `json:"periods"`
}

// DataRow represents a single data row from statisticsParameterData API.
type DataRow struct {
	OrgID  string `json:"ORG_ID"`
	TblID  string `json:"TBL_ID"`
	TblNM  string `json:"TBL_NM"`
	C1     string `json:"C1"`
	C1NM   string `json:"C1_NM"`
	C2     string `json:"C2"`
	C2NM   string `json:"C2_NM"`
	C3     string `json:"C3"`
	C3NM   string `json:"C3_NM"`
	C4     string `json:"C4"`
	C4NM   string `json:"C4_NM"`
	C5     string `json:"C5"`
	C5NM   string `json:"C5_NM"`
	C6     string `json:"C6"`
	C6NM   string `json:"C6_NM"`
	C7     string `json:"C7"`
	C7NM   string `json:"C7_NM"`
	C8     string `json:"C8"`
	C8NM   string `json:"C8_NM"`
	ItmID  string `json:"ITM_ID"`
	ItmNM  string `json:"ITM_NM"`
	UnitID string `json:"UNIT_ID"`
	UnitNM string `json:"UNIT_NM"`
	PrdSe  string `json:"PRD_SE"`
	PrdDe  string `json:"PRD_DE"`
	DT     string `json:"DT"`
	LstChn string `json:"LST_CHN_DE"`
}

// StatList represents an item from statisticsList.getList API.
type StatList struct {
	VwCD     string `json:"VW_CD"`
	VwNM     string `json:"VW_NM"`
	ListID   string `json:"LIST_ID"`
	ListNM   string `json:"LIST_NM"`
	OrgID    string `json:"ORG_ID"`
	TblID    string `json:"TBL_ID"`
	TblNM    string `json:"TBL_NM"`
	StatID   string `json:"STAT_ID"`
	SendDe   string `json:"SEND_DE"`
	RecTblSe string `json:"REC_TBL_SE"`
}

// ExplainResult represents explanation data from statisticsExplData API.
type ExplainResult struct {
	StatsNM      string `json:"statsNm"`
	WritingPurps string `json:"writingPurps"`
	StatsPeriod  string `json:"statsPeriod"`
	DataCollect  string `json:"dataCollect"`
	SurveyScope  string `json:"surveyScope"`
	PublishDe    string `json:"publishDe"`
	OrgNM        string `json:"orgNm"`
}

// IndicatorResult represents indicator data from indicator APIs.
type IndicatorResult struct {
	StatJipyoID  string `json:"statJipyoId"`
	StatJipyoNM  string `json:"statJipyoNm"`
	JipyoExplan  string `json:"jipyoExplan"`
	JipyoExplan1 string `json:"jipyoExplan1"`
	DataValue    string `json:"dataValue"`
	DataDE       string `json:"dataDe"`
	UnitNM       string `json:"unitNm"`
	LstUpdate    string `json:"lstUpdate"`
	ListID       string `json:"listId"`
	ListNM       string `json:"listNm"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Err    string `json:"err"`
	ErrMsg string `json:"errMsg"`
}

// SearchOptions contains optional search parameters.
type SearchOptions struct {
	Sort        string
	StartCount  int
	ResultCount int
}

// MetaOptions contains optional metadata parameters.
type MetaOptions struct {
	Type string
}

// DataOptions contains optional data query parameters.
type DataOptions struct {
	Class1       string
	Class2       string
	Class3       string
	Class4       string
	Class5       string
	Class6       string
	Class7       string
	Class8       string
	Item         string
	PrdSe        string
	Periods      string
	StartPrdDe   string
	EndPrdDe     string
	NewEstPrdCnt string
	PrdInterval  string
	OutputFields string
}

// PaginationOptions contains pagination parameters.
type PaginationOptions struct {
	PageNo    int
	NumOfRows int
}

// ListOptions contains list filtering parameters.
type ListOptions struct {
	VwCd     string
	ParentID string
}

// BigDataOptions contains big data query parameters.
type BigDataOptions struct {
	Type string
}
