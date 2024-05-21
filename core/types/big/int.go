package big

import (
	"errors"
	"math/big"
	"math/rand"

	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// Int steams math/big/Int with custom Marshall Unmarshall methods,
// which in the byte representation add quotes at the beginning and end of the number.
// Example 123 -> "123".
// Added for nodejs-backend to work with large numbers.
type Int struct {
	big.Int
}

// Validate checks if the Int value is negative and returns an error if it is.
func (z *Int) Validate(_ shim.ChaincodeStubInterface) error {
	if z.Int.Sign() < 0 {
		return errors.New("negative number")
	}

	return nil
}

// SetInt64 sets z to x and returns z.
func (z *Int) SetInt64(x int64) *Int {
	z.Int.SetInt64(x)
	return z
}

// SetUint64 sets z to x and returns z.
func (z *Int) SetUint64(x uint64) *Int {
	z.Int.SetUint64(x)
	return z
}

// NewInt allocates and returns a new Int set to x.
func NewInt(x int64) *Int {
	return new(Int).SetInt64(x)
}

// Set sets z to x and returns z.
func (z *Int) Set(x *Int) *Int {
	z.Int.Set(arg(x))
	return z
}

// SetBits provides raw (unchecked but fast) access to z by setting its
// value to abs, interpreted as a little-endian Word slice, and returning
// z. The result and abs share the same underlying array.
// SetBits is intended to support implementation of missing low-level Int
// functionality outside this package; it should be avoided otherwise.
func (z *Int) SetBits(abs []big.Word) *Int {
	z.Int.SetBits(abs)
	return z
}

// Abs sets z to |x| (the absolute value of x) and returns z.
func (z *Int) Abs(x *Int) *Int {
	z.Int.Abs(arg(x))
	return z
}

// Neg sets z to -x and returns z.
func (z *Int) Neg(x *Int) *Int {
	z.Int.Neg(arg(x))
	return z
}

// Add sets z to the sum x+y and returns z.
func (z *Int) Add(x, y *Int) *Int {
	z.Int.Add(arg(x), arg(y))
	return z
}

// Sub sets z to the difference x-y and returns z.
func (z *Int) Sub(x, y *Int) *Int {
	z.Int.Sub(arg(x), arg(y))
	return z
}

// Mul sets z to the product x*y and returns z.
func (z *Int) Mul(x, y *Int) *Int {
	z.Int.Mul(arg(x), arg(y))
	return z
}

// MulRange sets z to the product of all integers
// in the range [a, b] inclusively and returns z.
// If a > b (empty range), the result is 1.
func (z *Int) MulRange(a, b int64) *Int {
	z.Int.MulRange(a, b)
	return z
}

// Binomial sets z to the binomial coefficient of (n, k) and returns z.
func (z *Int) Binomial(n, k int64) *Int {
	z.Int.Binomial(n, k)
	return z
}

// Quo sets z to the quotient x/y for y != 0 and returns z.
// If y == 0, a division-by-zero run-time panic occurs.
// Quo implements truncated division (like Go); see QuoRem for more details.
func (z *Int) Quo(x, y *Int) *Int {
	z.Int.Quo(arg(x), arg(y))
	return z
}

// Rem sets z to the remainder x%y for y != 0 and returns z.
// If y == 0, a division-by-zero run-time panic occurs.
// Rem implements truncated modulus (like Go); see QuoRem for more details.
func (z *Int) Rem(x, y *Int) *Int {
	z.Int.Rem(arg(x), arg(y))
	return z
}

// QuoRem sets z to the quotient x/y and r to the remainder x%y
// and returns the pair (z, r) for y != 0.
// If y == 0, a division-by-zero run-time panic occurs.
//
// QuoRem implements T-division and modulus (like Go):
//
//	q = x/y      with the result truncated to zero
//	r = x - y*q
//
// (See Daan Leijen, “Division and Modulus for Computer Scientists”.)
// See DivMod for Euclidean division and modulus (unlike Go).
func (z *Int) QuoRem(x, y, r *Int) (*Int, *Int) {
	z.Int.QuoRem(arg(x), arg(y), arg(r))
	return z, r
}

// Div sets z to the quotient x/y for y != 0 and returns z.
// If y == 0, a division-by-zero run-time panic occurs.
// Div implements Euclidean division (unlike Go); see DivMod for more details.
func (z *Int) Div(x, y *Int) *Int {
	z.Int.Div(arg(x), arg(y))
	return z
}

// Mod sets z to the modulus x%y for y != 0 and returns z.
// If y == 0, a division-by-zero run-time panic occurs.
// Mod implements Euclidean modulus (unlike Go); see DivMod for more details.
func (z *Int) Mod(x, y *Int) *Int {
	z.Int.Mod(arg(x), arg(y))
	return z
}

// DivMod sets z to the quotient x div y and m to the modulus x mod y
// and returns the pair (z, m) for y != 0.
// If y == 0, a division-by-zero run-time panic occurs.
//
// DivMod implements Euclidean division and modulus (unlike Go):
//
//	q = x div y  such that
//	m = x - y*q  with 0 <= m < |y|
//
// (See Raymond T. Boute, “The Euclidean definition of the functions
// div and mod”. ACM Transactions on Programming Languages and
// Systems (TOPLAS), 14(2):127-144, New York, NY, USA, 4/1992.
// ACM press.)
// See QuoRem for T-division and modulus (like Go).
func (z *Int) DivMod(x, y, m *Int) (*Int, *Int) {
	z.Int.DivMod(arg(x), arg(y), arg(m))
	return z, m
}

// Cmp compares x and y and returns:
//
//	-1 if x <  y
//	 0 if x == y
//	+1 if x >  y
func (z *Int) Cmp(y *Int) (r int) {
	return z.Int.Cmp(arg(y))
}

// CmpAbs compares the absolute values of x and y and returns:
//
//	-1 if |x| <  |y|
//	 0 if |x| == |y|
//	+1 if |x| >  |y|
func (z *Int) CmpAbs(y *Int) int {
	return z.Int.CmpAbs(arg(y))
}

// SetString sets z to the value of s, interpreted in the given base,
// and returns z and a boolean indicating success. The entire string
// (not just a prefix) must be valid for success. If SetString fails,
// the value of z is undefined but the returned value is nil.
//
// The base argument must be 0 or a value between 2 and MaxBase. If the base
// is 0, the string prefix determines the actual conversion base. A prefix of
// “0x” or “0X” selects base 16; the “0” prefix selects base 8, and a
// “0b” or “0B” prefix selects base 2. Otherwise the selected base is 10.
//
// For bases <= 36, lower and upper case letters are considered the same:
// The letters 'a' to 'z' and 'A' to 'Z' represent digit values 10 to 35.
// For bases > 36, the upper case letters 'A' to 'Z' represent the digit
// values 36 to 61.
func (z *Int) SetString(s string, base int) (*Int, bool) {
	_, ok := z.Int.SetString(s, base)
	if !ok {
		return nil, ok
	}
	return z, ok
}

// SetBytes interprets buf as the bytes of a big-endian unsigned
// integer, sets z to that value, and returns z.
func (z *Int) SetBytes(buf []byte) *Int {
	z.Int.SetBytes(buf)
	return z
}

// Exp sets z = x**y mod |m| (i.e. the sign of m is ignored), and returns z.
// If y <= 0, the result is 1 mod |m|; if m == nil or m == 0, z = x**y.
//
// Modular exponentation of inputs of a particular size is not a
// cryptographically constant-time operation.
func (z *Int) Exp(x, y, m *Int) *Int {
	z.Int.Exp(arg(x), arg(y), arg(m))
	return z
}

// GCD sets z to the greatest common divisor of a and b, which both must
// be > 0, and returns z.
// If x or y are not nil, GCD sets their value such that z = a*x + b*y.
// If either a or b is <= 0, GCD sets z = x = y = 0.
func (z *Int) GCD(x, y, a, b *Int) *Int {
	z.Int.GCD(arg(x), arg(y), arg(a), arg(b))
	return z
}

// Rand sets z to a pseudo-random number in [0, n) and returns z.
//
// As this uses the math/rand package, it must not be used for
// security-sensitive work. Use crypto/rand.Int instead.
func (z *Int) Rand(rnd *rand.Rand, n *Int) *Int {
	z.Int.Rand(rnd, arg(n))
	return z
}

// ModInverse sets z to the multiplicative inverse of g in the ring ℤ/nℤ
// and returns z. If g and n are not relatively prime, the result is undefined.
func (z *Int) ModInverse(g, n *Int) *Int {
	z.Int.ModInverse(arg(g), arg(n))
	return z
}

// Jacobi returns the Jacobi symbol (x/y), either +1, -1, or 0.
// The y argument must be an odd integer.
// func Jacobi(x, y *Int) int {
// 	return big.Jacobi(&x.Int, &y.Int)
// }

// ModSqrt sets z to a square root of x mod p if such a square root exists, and
// returns z. The modulus p must be an odd prime. If x is not a square mod p,
// ModSqrt leaves z unchanged and returns nil. This function panics if p is
// not an odd integer.
func (z *Int) ModSqrt(x, p *Int) *Int {
	y := z.Int.ModSqrt(arg(x), arg(p))
	if y == nil {
		return nil
	}
	return z
}

// Lsh sets z = x << n and returns z.
func (z *Int) Lsh(x *Int, n uint) *Int {
	z.Int.Lsh(arg(x), n)
	return z
}

// Rsh sets z = x >> n and returns z.
func (z *Int) Rsh(x *Int, n uint) *Int {
	z.Int.Rsh(arg(x), n)
	return z
}

// SetBit sets z to x, with x's i'th bit set to b (0 or 1).
// That is, if b is 1 SetBit sets z = x | (1 << i);
// if b is 0 SetBit sets z = x &^ (1 << i). If b is not 0 or 1,
// SetBit will panic.
func (z *Int) SetBit(x *Int, i int, b uint) *Int {
	z.Int.SetBit(arg(x), i, b)
	return z
}

// And sets z = x & y and returns z.
func (z *Int) And(x, y *Int) *Int {
	z.Int.And(arg(x), arg(y))
	return z
}

// AndNot sets z = x &^ y and returns z.
func (z *Int) AndNot(x, y *Int) *Int {
	z.Int.AndNot(arg(x), arg(y))
	return z
}

// Or sets z = x | y and returns z.
func (z *Int) Or(x, y *Int) *Int {
	z.Int.Or(arg(x), arg(y))
	return z
}

// Xor sets z = x ^ y and returns z.
func (z *Int) Xor(x, y *Int) *Int {
	z.Int.Xor(arg(x), arg(y))
	return z
}

// Not sets z = ^x and returns z.
func (z *Int) Not(x *Int) *Int {
	z.Int.Not(arg(x))
	return z
}

// Sqrt sets z to ⌊√x⌋, the largest integer such that z² ≤ x, and returns z.
// It panics if x is negative.
func (z *Int) Sqrt(x *Int) *Int {
	z.Int.Sqrt(arg(x))
	return z
}

// ===================================

// The JSON marshalers are only here for API backward compatibility
// (programs that explicitly look for these two methods). JSON works
// fine with the TextMarshaler only.

// MarshalJSON implements the json.Marshaler interface.
func (z *Int) MarshalJSON() ([]byte, error) {
	out, err := z.MarshalText()
	if err != nil {
		return out, err
	}
	return []byte("\"" + string(out) + "\""), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (z *Int) UnmarshalJSON(text []byte) error {
	// Ignore null, like in the main JSON package.
	text = unquoteIfQuoted(text)
	if string(text) == "null" {
		return nil
	}
	return z.UnmarshalText(text)
}

func unquoteIfQuoted(bytes []byte) []byte {
	if len(bytes) > 2 && bytes[0] == '"' && bytes[len(bytes)-1] == '"' {
		return bytes[1 : len(bytes)-1]
	}
	return bytes
}

func arg(x *Int) *big.Int {
	if x == nil {
		return nil
	}

	return &x.Int
}
