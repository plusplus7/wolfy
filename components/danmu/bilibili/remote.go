package bilibili

type RemoteSignRequest struct {
	ReqJson    string `json:"req_json"`
	AnchorCode string `json:"anchor_code"`
}

type RemoteSignResponse struct {
	Header CommonHeader `json:"signed"`
}
