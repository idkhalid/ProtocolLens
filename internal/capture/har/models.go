package har

type File struct {
	Log Log `json:"log"`
}

type Log struct {
	Entries []Entry `json:"entries"`
}

type Entry struct {
	StartedDateTime string   `json:"startedDateTime"`
	Time            float64  `json:"time"`
	Request         Request  `json:"request"`
	Response        Response `json:"response"`
}

type Request struct {
	Method      string      `json:"method"`
	URL         string      `json:"url"`
	Headers     []NameValue `json:"headers"`
	QueryString []NameValue `json:"queryString"`
	PostData    *PostData   `json:"postData"`
}

type Response struct {
	Status  int         `json:"status"`
	Headers []NameValue `json:"headers"`
	Content *Content    `json:"content"`
}

type NameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type Content struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
	Encoding string `json:"encoding"`
}
