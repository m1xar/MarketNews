package scraper

type Day struct {
	Date     string  `json:"date"`
	Dateline int64   `json:"dateline"`
	Events   []Event `json:"events"`
}

type Event struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	PrefixedName        string `json:"prefixedName"`
	TrimmedPrefixed     string `json:"trimmedPrefixedName"`
	SoloTitle           string `json:"soloTitle"`
	SoloTitleFull       string `json:"soloTitleFull"`
	SoloTitleShort      string `json:"soloTitleShort"`
	Notice              string `json:"notice"`
	Dateline            int64  `json:"dateline"`
	Country             string `json:"country"`
	Currency            string `json:"currency"`
	ImpactName          string `json:"impactName"`
	ImpactTitle         string `json:"impactTitle"`
	TimeLabel           string `json:"timeLabel"`
	TimeMasked          bool   `json:"timeMasked"`
	Actual              string `json:"actual"`
	Forecast            string `json:"forecast"`
	Previous            string `json:"previous"`
	Revision            string `json:"revision"`
	Date                string `json:"date"`
	URL                 string `json:"url"`
	SoloURL             string `json:"soloUrl"`
	HasNotice           bool   `json:"hasNotice"`
	HasDataValues       bool   `json:"hasDataValues"`
	HasGraph            bool   `json:"hasGraph"`
	HasLinkedThreads    bool   `json:"hasLinkedThreads"`
	ImpactClass         string `json:"impactClass"`
	ActualBetterWorse   int    `json:"actualBetterWorse"`
	RevisionBetterWorse int    `json:"revisionBetterWorse"`
}
