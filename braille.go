package dotmatrix

// Represents an 8 dot braille pattern using x,y coordinates. Eg:
// +----------+
// |(0,0)(1,0)|
// |(0,1)(1,1)|
// |(0,2)(1,2)|
// |(0,3)(1,3)|
// +----------+
type braille [2][4]int

// codePoint maps each point in braille to a dot identifier and
// calculates the corresponding unicode symbol.
// +------+
// |(1)(4)|
// |(2)(5)|
// |(3)(6)|
// |(7)(8)|
// +------+
// See https://en.wikipedia.org/wiki/Braille_Patterns#Identifying.2C_naming_and_ordering)
func (b braille) codePoint() rune {
	lowEndian := [8]int{b[0][0], b[0][1], b[0][2], b[1][0], b[1][1], b[1][2], b[0][3], b[1][3]}
	var v int
	for i, x := range lowEndian {
		v += int(x) << uint(i)
	}
	return rune(v) + '\u2800'
}

func (b braille) String() string {
	return string(b.codePoint())
}
