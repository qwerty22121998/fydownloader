package core

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func Test_SplitChunk(t *testing.T) {
	t.Parallel()
	for i := 0; i < 1000; i++ {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			contentLength := rand.Intn(1 << 32)
			chunkNum := rand.Intn(19) + 1
			chunk := SplitChunk(contentLength, chunkNum)
			for i, r := range chunk {
				assert.Equal(t, r.Index, i)
			}
			for i := 1; i < len(chunk); i++ {
				assert.Equal(t, chunk[i-1].To, chunk[i].From-1)
			}
			assert.Equal(t, chunk[len(chunk)-1].To, contentLength-1)
		})
	}

}
