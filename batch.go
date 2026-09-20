package layaonnx

type modelBatch struct {
	inputIDs      []int64
	attentionMask []int64
	markerPos     []int64
	markerMask    []bool
	questionTypes []int64
	rows          int
	sequence      int
	options       int
}

func (e *Engine) makeBatch(rows []preparedQuestion) modelBatch {
	sequence, options := 0, 0
	for _, row := range rows {
		sequence = max(sequence, len(row.inputIDs))
		options = max(options, len(row.options))
	}
	batch := modelBatch{
		rows:          len(rows),
		sequence:      sequence,
		options:       options,
		inputIDs:      make([]int64, len(rows)*sequence),
		attentionMask: make([]int64, len(rows)*sequence),
		markerPos:     make([]int64, len(rows)*options),
		markerMask:    make([]bool, len(rows)*options),
		questionTypes: make([]int64, len(rows)),
	}
	for index := range batch.inputIDs {
		batch.inputIDs[index] = e.tokens.padding
	}
	for rowIndex, row := range rows {
		copy(batch.inputIDs[rowIndex*sequence:], row.inputIDs)
		for index := 0; index < len(row.inputIDs); index++ {
			batch.attentionMask[rowIndex*sequence+index] = 1
		}
		copy(batch.markerPos[rowIndex*options:], row.markers)
		for index := range row.markers {
			batch.markerMask[rowIndex*options+index] = true
		}
		batch.questionTypes[rowIndex] = row.typeID
	}
	return batch
}

func validateOutput(batch modelBatch, logits []float32, shape []int64) error {
	if len(shape) != 2 || int(shape[0]) != batch.rows || int(shape[1]) != batch.options || len(logits) != batch.rows*batch.options {
		return ErrModelContract
	}
	return nil
}
