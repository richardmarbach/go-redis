package resp

func parseLineLen(line []byte) (length, lengthIndexEnd int, err error) {
	if len(line) < minObjectLen {
		return 0, 0, ErrSyntax
	}

	if line[0] != BulkStringPrefix {
		return 0, 0, ErrSyntax
	}

	// Null bulk string
	if len(line) >= 5 && line[1] == '-' && line[2] == '1' && line[3] == '\r' && line[4] == '\n' {
		return -1, 4, nil
	}

	var n int
	for i, b := range line[1:] {
		if b < '0' || b > '9' {
			if b == '\r' && len(line) > i+2 && line[i+2] == '\n' {
				return n, i + 2, nil
			}
			return 0, 0, ErrSyntax
		}

		n = (n * 10) + int(b-'0')
	}

	return 0, 0, ErrSyntax
}
