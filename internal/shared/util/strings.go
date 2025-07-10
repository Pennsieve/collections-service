package util

func SafeDeref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
