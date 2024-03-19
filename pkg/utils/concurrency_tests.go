package utils

// TODO: noramlly, these go in utils_tests, right? Why does this cause problems only in pkg/utils?

import (
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	// "github.com/rs/zerolog/log"
)

var _ = Describe("utils/concurrency tests", func() {
	It("SliceOfChannelsReducer works", func() {
		individualResultsChannels := []<-chan int{}
		initialValue := 0
		for i := 0; i < 3; i++ {
			c := make(chan int)
			go func(i int, c chan int) {
				for ii := 1; ii < 4; ii++ {
					c <- (i * ii)
				}
				close(c)
			}(i, c)
			individualResultsChannels = append(individualResultsChannels, c)
		}
		Expect(len(individualResultsChannels)).To(Equal(3))
		finalResultChannel := make(chan int)
		wg := SliceOfChannelsReducer[int, int](individualResultsChannels, finalResultChannel, func(input int, val int) int {
			return val + input
		}, initialValue, true)

		Expect(wg).ToNot(BeNil())

		result := <-finalResultChannel

		Expect(result).ToNot(Equal(0))
		Expect(result).To(Equal(18))
	})

	It("SliceOfChannelsRawMergerWithoutMapping works", func() {
		individualResultsChannels := []<-chan int{}
		for i := 0; i < 3; i++ {
			c := make(chan int)
			go func(i int, c chan int) {
				for ii := 1; ii < 4; ii++ {
					c <- (i * ii)
				}
				close(c)
			}(i, c)
			individualResultsChannels = append(individualResultsChannels, c)
		}
		Expect(len(individualResultsChannels)).To(Equal(3))
		outputChannel := make(chan int)
		wg := SliceOfChannelsRawMergerWithoutMapping(individualResultsChannels, outputChannel, true)
		Expect(wg).ToNot(BeNil())
		outputSlice := []int{}
		for v := range outputChannel {
			outputSlice = append(outputSlice, v)
		}
		Expect(len(outputSlice)).To(Equal(9))
		slices.Sort(outputSlice)
		Expect(outputSlice[0]).To(BeZero())
		Expect(outputSlice[3]).To(Equal(1))
		Expect(outputSlice[8]).To(Equal(6))
	})

	It("SliceOfChannelsTransformer works", func() {
		individualResultsChannels := []<-chan int{}
		for i := 0; i < 3; i++ {
			c := make(chan int)
			go func(i int, c chan int) {
				for ii := 1; ii < 4; ii++ {
					c <- (i * ii)
				}
				close(c)
			}(i, c)
			individualResultsChannels = append(individualResultsChannels, c)
		}
		Expect(len(individualResultsChannels)).To(Equal(3))
		mappingFn := func(i int) string {
			return fmt.Sprintf("$%d", i)
		}

		outputChannels := SliceOfChannelsTransformer(individualResultsChannels, mappingFn)
		Expect(len(outputChannels)).To(Equal(3))
		for ii := 1; ii < 4; ii++ {
			rSlice := make([]string, 3)
			for i := 0; i < 3; i++ {
				rSlice[i] = <-outputChannels[i]
			}
			slices.Sort(rSlice)
			Expect(rSlice[0]).To(Equal("$0"))
			Expect(rSlice[2]).To(Equal(fmt.Sprintf("$%d", 2*ii)))
		}
	})
})
