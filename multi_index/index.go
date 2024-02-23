package multiindex

func splitByLength(str string, segment int) []string {
	i := 0
	out := []string{}
	for i < len(str) {
		out = append(out, str[i:i+segment])
		i += segment
	}

	return out
}
