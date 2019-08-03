package utils

import (
	"testing"

	"github.com/pajbot/testhelper"
)

func TestSplitIntoChunks(t *testing.T) {
	data22 := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22"}
	data10 := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}

	chunks := SplitIntoChunks(5, data22)
	testhelper.AssertIntsEqual(t, 5, len(chunks))

	chunks = SplitIntoChunks(1, data10)
	testhelper.AssertIntsEqual(t, 10, len(chunks))
}
