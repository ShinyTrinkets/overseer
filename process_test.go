package overseer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSimpleProcess(t *testing.T) {
	assert := assert.New(t)
	c := NewCmd("ls")

	assert.Equal(c.DelayStart, defaultDelayStart, "Wrong default delay start")
	assert.Equal(c.RetryTimes, defaultRetryTimes, "Wrong default retry times")

	c.SetDir("/")
	assert.Equal(c.Dir, "/", "Couldn't set dir")
}

func TestCloneProcess(t *testing.T) {
	var (
		delay uint = 1
		retry uint = 9
	)

	assert := assert.New(t)
	c1 := NewCmd("ls")
	c1.SetDir("/")
	c1.SetDelayStart(delay)
	c1.SetRetryTimes(retry)

	c2 := c1.CloneCmd()

	assert.Equal(c1.Dir, c2.Dir)
	assert.Equal(c1.Env, c2.Env)
	assert.Equal(c1.DelayStart, c2.DelayStart)
	assert.Equal(c1.RetryTimes, c2.RetryTimes)
}
