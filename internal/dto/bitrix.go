package dto

type OutgoingHookDeal struct {
	DocumentID []string `json:"document_id"`
	Auth       Auth     `json:"auth"`
	Timestamp  int64    `json:"ts"`
}

type OutgoingHookComment struct {
	Auth           Auth   `json:"auth"`
	Event          string `json:"event"`
	EventHandlerID int64  `json:"event_handler_id"`
	Data           struct {
		Fields struct {
			ID int64 `json:"ID"`
		} `json:"FIELDS"`
	} `json:"data"`
	Timestamp int64 `json:"ts"`
}

type Auth struct {
	Domain           string `json:"domain"`
	ClientEndpoint   string `json:"client_endpoint"`
	ServerEndpoint   string `json:"server_endpoint"`
	MemberID         string `json:"member_id"`
	ApplicationToken string `json:"application_token"`
}
