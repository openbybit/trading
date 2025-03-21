package ghdts

import (
	"bytes"
	"fmt"

	"git.bybit.com/tni_go/dtssdk_go_interface/dtssdk"
)

func GetHeader(headers []dtssdk.RecordHeader, key []byte) []byte {
	for _, hv := range headers {
		if bytes.Equal(hv.Key, key) {
			return hv.Value
		}
	}

	return nil
}

// GetOffset 获取offset
// pos = -1 返回最大offset
// pos = -2 返回最小offset
// pos > 0  返回min_offset+pos
// pos < -2 返回max_offset-|pos|
func GetOffset(topic string, pos int32, timeoutMs int32) (res int64, err error) {
	if pos == dtssdk.MinOffset || pos == dtssdk.MaxOffset {
		return dtssdk.GetOffset(topic, pos, timeoutMs)
	}

	maxOffset, maxErr := dtssdk.GetOffset(topic, dtssdk.MaxOffset, timeoutMs)
	minOffset, minErr := dtssdk.GetOffset(topic, dtssdk.MinOffset, timeoutMs)
	if maxErr != nil || minErr != nil {
		if pos > 0 {
			res = int64(dtssdk.MinOffset)
		} else {
			res = int64(dtssdk.MaxOffset)
		}

		err = fmt.Errorf("GetOffset fail, min_err=%w, max_err=%w", minErr, maxErr)

		return
	}

	if pos >= 0 {
		res = minOffset + int64(pos)
		if res > maxOffset {
			res = maxOffset
		}
	} else {
		res = maxOffset + int64(pos)
		if res < minOffset {
			res = minOffset
		}
	}

	return
}
