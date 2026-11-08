package pagination

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

type Page struct {
	NextID        int64 `json:"next_id"`
	NextTimeAtUTC int64 `json:"next_time_at_utc"`
	PageSize      int64 `json:"page_size"`
}

// Encode 返回分页token
func (p Page) Encode() string {
	b, err := json.Marshal(p)
	if err != nil {
		return string("")
	}
	return string(base64.StdEncoding.EncodeToString(b))
}

func (p Page) Invalid() bool {
	return p.NextID < 0 || p.PageSize <= 0 || p.PageSize > 100 || p.NextTimeAtUTC <= 0 || time.Since(time.Unix(p.NextTimeAtUTC, 0)) > 24*time.Hour
}

// Decode 解析分页信息
func Decode(t string) Page {
	var result Page
	if len(t) == 0 {
		return result
	}

	bytes, err := base64.StdEncoding.DecodeString(string(t))
	if err != nil {
		return result
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return result
	}

	return result
}
