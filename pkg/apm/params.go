package apm

import "net/url"

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func setCommonParams(q url.Values, env, start, end, kuery string) {
	q.Set("environment", orDefault(env, "ENVIRONMENT_ALL"))
	q.Set("start", start)
	q.Set("end", end)
	q.Set("kuery", kuery)
}
