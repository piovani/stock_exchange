package stock

type Quote struct {
	Symbol        string
	ShortName     string
	Price         float64
	Change        float64
	ChangePercent float64
	Timestamp     int64
}

type SearchResult struct {
	Symbol    string
	ShortName string
	Exchange  string
	Type      string
}
