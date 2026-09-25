package pyrandom_test

import (
	"encoding/hex"
	"slices"
	"strings"
	"testing"

	"github.com/wrouesnel/code-card/pkg/pyrandom"
)

// Vectors generated with CPython 3.13:
//
//	r = random.Random(seed)
//	v = [r.getrandbits(32) for _ in range(700)]
//	v[:3], v[623:626], v[699], [r.choice('0123456789') for _ in range(10)]
func TestMatchesCPython(t *testing.T) {
	cases := []struct {
		seedHex string
		first   []uint32
		wrap    []uint32
		last    uint32
		choices string
	}{
		{"", []uint32{4124137760, 1951988028, 2557274880}, []uint32{1170448423, 3353959571, 1911786301}, 3895838982, "0169622606"},
		{"68656c6c6f", []uint32{1519453933, 3618488475, 2848416207}, []uint32{3023429643, 3451637425, 430735067}, 816905372, "9924223265"},
		{
			"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
			[]uint32{4120112502, 1359723207, 3884295436}, []uint32{3344380679, 3883978186, 2315344437}, 59624416, "3639169434",
		},
		// Leading zero bytes shrink the seed integer.
		{"0000616263", []uint32{1217561204, 2009949103, 3408464373}, []uint32{1950378267, 3937530647, 1347977878}, 1441685493, "4423510133"},
	}

	for _, tc := range cases {
		t.Run(tc.seedHex, func(t *testing.T) {
			seed, err := hex.DecodeString(tc.seedHex)
			if err != nil {
				t.Fatal(err)
			}
			r := pyrandom.New(seed)
			values := make([]uint32, 700)
			for idx := range values {
				values[idx] = r.GetRandBits(32)
			}
			if !slices.Equal(values[:3], tc.first) {
				t.Errorf("first values: got %v, want %v", values[:3], tc.first)
			}
			if !slices.Equal(values[623:626], tc.wrap) {
				t.Errorf("values after state regeneration: got %v, want %v", values[623:626], tc.wrap)
			}
			if values[699] != tc.last {
				t.Errorf("last value: got %v, want %v", values[699], tc.last)
			}

			digits := strings.Split("0123456789", "")
			var choices strings.Builder
			for range 10 {
				choices.WriteString(pyrandom.Choice(r, digits))
			}
			if choices.String() != tc.choices {
				t.Errorf("choices: got %q, want %q", choices.String(), tc.choices)
			}
		})
	}
}
