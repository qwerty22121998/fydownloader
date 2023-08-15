package core

func SplitChunk(length int, size int) []FileChunk {
	roundSize := length / size
	mod := length % size
	sizes := make([]FileChunk, 0, size)
	from := 0
	for i := 0; i < size; i++ {
		curSize := roundSize
		if i == size-1 {
			curSize += mod
		}
		sizes = append(sizes, FileChunk{
			Index: i,
			From:  from,
			To:    from + curSize - 1,
		})

		from += curSize
	}

	return sizes
}
