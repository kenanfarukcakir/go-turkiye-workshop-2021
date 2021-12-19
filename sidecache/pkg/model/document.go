package model

// CacheData is used for storing cache data in DB
// Warning: If you add/remove/change fields you must run `easyjson -all document.go`.
type CacheData struct {
	Body    []byte
	Headers map[string]string
}
