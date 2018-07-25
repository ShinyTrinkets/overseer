package overseer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSimpleProcess(t *testing.T) {
	assert := assert.New(t)
	c := NewChild("ls")

	assert.Equal(c.DelayStart, DEFAULT_DELAY_START, "Wrong default delay start")
	assert.Equal(c.RetryTimes, DEFAULT_RETRY_TIMES, "Wrong default retry times")

	c.SetDir("/")
	assert.Equal(c.Dir, "/", "Couldn't set dir")
}

func TestCloneProcess(t *testing.T) {
	var (
		delay uint = 1
		retry uint = 9
	)

	assert := assert.New(t)
	c1 := NewChild("ls")
	c1.SetDir("/")
	c1.SetDelayStart(delay)
	c1.SetRetryTimes(retry)

	c2 := c1.CloneChild()

	assert.Equal(c1.Dir, c2.Dir)
	assert.Equal(c1.Env, c2.Env)
	assert.Equal(c1.DelayStart, c2.DelayStart)
	assert.Equal(c1.RetryTimes, c2.RetryTimes)
}
