// vim: sw=8

package quota

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Quota struct {
	Xid            string
	BlockSoftLimit uint64
	BlockHardLimit uint64
	InodeSoftLimit uint64
	InodeHardLimit uint64
}

func Parse(fp io.Reader) ([]Quota, error) {
	scanner := bufio.NewScanner(fp)
	qs := []Quota{}
	num := 0
	for scanner.Scan() {
		num++
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		toks := strings.Fields(line)
		if len(toks) != 5 {
			return nil, fmt.Errorf(
				"line %d: wrong number of fields", num,
			)
		}
		var itoks [5]uint64
		for i := 1; i < len(itoks); i++ {
			var err error
			itoks[i], err = strconv.ParseUint(toks[i], 10, 64)
			if err != nil {
				return nil, fmt.Errorf(
					"line %d: invalid field %d: %v",
					num, i+1, err,
				)
			}
		}

		qs = append(qs, Quota{
			Xid:            toks[0],
			BlockSoftLimit: itoks[1],
			BlockHardLimit: itoks[2],
			InodeSoftLimit: itoks[3],
			InodeHardLimit: itoks[4],
		})
	}
	return qs, nil
}
