package ping

type Update struct {
	Target string `json:"target"`
	Up     bool   `json:"up"`
}
