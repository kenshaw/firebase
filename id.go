package firebase

import (
	"math/rand"
	"sync"
	"time"
)

const (
	// defaultPushIDChars are the lexiographically correct base 64 characters for use in generated PushIDs.
	defaultPushIDChars = "-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"
)

// IDGen holds the information related to generating a Push ID.
type IDGen struct {
	mu sync.Mutex

	// r is the random source.
	r *rand.Rand

	// stamp is the timestamp of the last ID creation, used to prevent
	// collisions if called multiple times during the same millisecond.
	stamp int64

	// stamp is comprised of 72 bits of entropy converted to 12 characters.
	// this is appended to the generated id to prevent collisions.
	// the numeric value is incremented in the event of a collision.
	last [12]int
}

// GeneratePushID generates a unique, 20-character ID for use with Firebase,
// using the default Push ID generator.
var GeneratePushID func() string

// NewPushIDGenerator creates a new Push ID generator.
func NewPushIDGenerator(r *rand.Rand) (*IDGen, error) {
	// make sure rand is good
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	// create generator and set last entropy
	ig := &IDGen{
		r: r,
	}
	for i := 0; i < 12; i++ {
		ig.last[i] = r.Intn(64)
	}

	return ig, nil
}

// GeneratePushID generates a unique, 20-character ID for use with Firebase.
func (ig *IDGen) GeneratePushID() string {
	var i int

	id := make([]byte, 20)

	// grab last characters
	ig.mu.Lock()
	now := time.Now().UTC().UnixNano() / 1e6
	if ig.stamp == now {
		for i = 0; i < 12; i++ {
			ig.last[i]++
			if ig.last[i] < 64 {
				break
			}
			ig.last[i] = 0
		}
	}
	ig.stamp = now

	// set last 12 characters
	for i = 0; i < 12; i++ {
		id[19-i] = defaultPushIDChars[ig.last[i]]
	}
	ig.mu.Unlock()

	// set id to first 8 characters
	for i = 7; i >= 0; i-- {
		id[i] = defaultPushIDChars[int(now%64)]
		now /= 64
	}

	return string(id)
}

func init() {
	// set default id generator
	ig, err := NewPushIDGenerator(nil)
	if err != nil {
		panic(err)
	}

	GeneratePushID = ig.GeneratePushID
}
