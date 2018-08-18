package overseer

// Credit: https://github.com/matryer/flower

import (
	"math/rand"
	"sync"
	"time"
)

// asciiLetters is a []byte of the characters to choose from when generating random keys.
var asciiLetters = []byte("abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

// seedOnce is the sync.Once used to rand.Seed.
var seedOnce sync.Once

// randomKey generates a random string of given length.
//
// The first time this is called, the rand.Seed will be set to the current time.
func randomKey(length uint) string {

	// randomise the seed
	seedOnce.Do(func() {
		rand.Seed(time.Now().UTC().UnixNano())
	})

	// Credit: http://stackoverflow.com/questions/12321133/golang-random-number-generator-how-to-seed-properly

	bytes := make([]byte, length)
	for i := uint(0); i < length; i++ {
		randInt := randomInt(0, len(asciiLetters))
		bytes[i] = asciiLetters[randInt : randInt+1][0]
	}

	return string(bytes)
}

// randomInt generates a random integer between min and max.
func randomInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
