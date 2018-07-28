package ping2ws

type Update struct {
	Target string `json:"target"`
	Up     bool   `json:"up"`
}
