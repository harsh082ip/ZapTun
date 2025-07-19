package tunnel

type ControlMessage struct {
	Type      string `json:"type"` // http or tcp
	Subdomain string `json:"subdomain,omitempty"`
}
