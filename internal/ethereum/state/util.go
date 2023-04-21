package state

func Ceil32(n int64) int64 {
	if n >= 0 {
		return ceil32(n)
	}
	return -ceil32(-n)
}

func ceil32(n int64) int64 {
	l := n % 32
	if l == 0 {
		return l
	}
	return n - n%32 + 32
}
