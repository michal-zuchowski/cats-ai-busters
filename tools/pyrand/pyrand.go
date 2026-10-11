// Package pyrand reproduces CPython's random.Random (MT19937) for integer
// seeds, so generators stay bit-identical with the former Python tools.
package pyrand

import "math/bits"

const (
	n         = 624
	m         = 397
	matrixA   = 0x9908b0df
	upperMask = 0x80000000
	lowerMask = 0x7fffffff
)

// Random is the Mersenne Twister state behind random.Random.
type Random struct {
	mt  [n]uint32
	idx int
}

func (r *Random) initGenrand(s uint32) {
	r.mt[0] = s
	for i := 1; i < n; i++ {
		r.mt[i] = 1812433253*(r.mt[i-1]^(r.mt[i-1]>>30)) + uint32(i)
	}
	r.idx = n
}

// New is random.Random(seed) for a non-negative seed (init_by_array on the
// 32-bit words of the seed, like CPython).
func New(seed uint64) *Random {
	key := []uint32{uint32(seed)}
	if seed>>32 != 0 {
		key = append(key, uint32(seed>>32))
	}
	r := &Random{}
	r.initGenrand(19650218)
	i, j := 1, 0
	k := n
	if len(key) > k {
		k = len(key)
	}
	for ; k > 0; k-- {
		r.mt[i] = (r.mt[i] ^ ((r.mt[i-1] ^ (r.mt[i-1] >> 30)) * 1664525)) + key[j] + uint32(j)
		i++
		j++
		if i >= n {
			r.mt[0] = r.mt[n-1]
			i = 1
		}
		if j >= len(key) {
			j = 0
		}
	}
	for k = n - 1; k > 0; k-- {
		r.mt[i] = (r.mt[i] ^ ((r.mt[i-1] ^ (r.mt[i-1] >> 30)) * 1566083941)) - uint32(i)
		i++
		if i >= n {
			r.mt[0] = r.mt[n-1]
			i = 1
		}
	}
	r.mt[0] = 0x80000000
	return r
}

func (r *Random) genrand() uint32 {
	if r.idx >= n {
		mag := [2]uint32{0, matrixA}
		var kk int
		for kk = 0; kk < n-m; kk++ {
			y := (r.mt[kk] & upperMask) | (r.mt[kk+1] & lowerMask)
			r.mt[kk] = r.mt[kk+m] ^ (y >> 1) ^ mag[y&1]
		}
		for ; kk < n-1; kk++ {
			y := (r.mt[kk] & upperMask) | (r.mt[kk+1] & lowerMask)
			r.mt[kk] = r.mt[kk+m-n] ^ (y >> 1) ^ mag[y&1]
		}
		y := (r.mt[n-1] & upperMask) | (r.mt[0] & lowerMask)
		r.mt[n-1] = r.mt[m-1] ^ (y >> 1) ^ mag[y&1]
		r.idx = 0
	}
	y := r.mt[r.idx]
	r.idx++
	y ^= y >> 11
	y ^= (y << 7) & 0x9d2c5680
	y ^= (y << 15) & 0xefc60000
	y ^= y >> 18
	return y
}

// Random is random.random(): a float in [0, 1) with 53 random bits.
func (r *Random) Random() float64 {
	a, b := r.genrand()>>5, r.genrand()>>6
	return (float64(a)*67108864.0 + float64(b)) * (1.0 / 9007199254740992.0)
}

// GetRandBits is random.getrandbits(k) for 1 <= k <= 32.
func (r *Random) GetRandBits(k int) int {
	return int(r.genrand() >> (32 - k))
}

// RandBelow is random._randbelow(n) (n >= 1).
func (r *Random) RandBelow(n int) int {
	k := bits.Len(uint(n))
	v := r.GetRandBits(k)
	for v >= n {
		v = r.GetRandBits(k)
	}
	return v
}

// RandInt is random.randint(a, b).
func (r *Random) RandInt(a, b int) int { return a + r.RandBelow(b-a+1) }

// Choice is random.choice(seq) for a sequence of length n: it returns the index.
func (r *Random) Choice(n int) int { return r.RandBelow(n) }
