package apm

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
