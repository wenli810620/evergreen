package message

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"strings"

	"github.com/mongodb/grip/level"
	"github.com/stretchr/testify/assert"
)

func TestPopulatedMessageComposerConstructors(t *testing.T) {
	const testMsg = "hello"
	assert := assert.New(t)
	// map objects to output
	cases := map[Composer]string{
		NewString(testMsg):                                                     testMsg,
		NewDefaultMessage(level.Error, testMsg):                                testMsg,
		NewBytes([]byte(testMsg)):                                              testMsg,
		NewBytesMessage(level.Error, []byte(testMsg)):                          testMsg,
		NewError(errors.New(testMsg)):                                          testMsg,
		NewErrorMessage(level.Error, errors.New(testMsg)):                      testMsg,
		NewErrorWrap(errors.New(testMsg), ""):                                  testMsg,
		NewErrorWrapMessage(level.Error, errors.New(testMsg), ""):              testMsg,
		NewFieldsMessage(level.Error, testMsg, Fields{}):                       fmt.Sprintf("[msg='%s']", testMsg),
		NewFields(level.Error, Fields{"test": testMsg}):                        fmt.Sprintf("[test='%s']", testMsg),
		MakeFieldsMessage(testMsg, Fields{}):                                   fmt.Sprintf("[msg='%s']", testMsg),
		MakeFields(Fields{"test": testMsg}):                                    fmt.Sprintf("[test='%s']", testMsg),
		NewFormatted(string(testMsg[0])+"%s", testMsg[1:]):                     testMsg,
		NewFormattedMessage(level.Error, string(testMsg[0])+"%s", testMsg[1:]): testMsg,
		NewLine(testMsg, ""):                                                   testMsg,
		NewLineMessage(level.Error, testMsg, ""):                               testMsg,
		NewLine(testMsg):                                                       testMsg,
		NewLineMessage(level.Error, testMsg):                                   testMsg,
	}

	for msg, output := range cases {
		assert.NotNil(msg)
		assert.NotEmpty(output)
		assert.Implements((*Composer)(nil), msg)
		assert.True(msg.Loggable())
		assert.NotNil(msg.Raw())

		// run the string test to make sure it doesn't change:
		assert.Equal(msg.String(), output)
		assert.Equal(msg.String(), output)

		if msg.Priority() != level.Invalid {
			assert.Equal(msg.Priority(), level.Error)
		}
	}
}

func TestUnpopulatedMessageComposers(t *testing.T) {
	assert := assert.New(t)
	// map objects to output
	cases := []Composer{
		&stringMessage{},
		NewString(""),
		NewDefaultMessage(level.Error, ""),
		&bytesMessage{},
		NewBytes([]byte{}),
		NewBytesMessage(level.Error, []byte{}),
		&ProcessInfo{},
		&SystemInfo{},
		&lineMessenger{},
		NewLine(),
		NewLineMessage(level.Error),
		&formatMessenger{},
		NewFormatted(""),
		NewFormattedMessage(level.Error, ""),
		&stackMessage{},
		NewStack(1, ""),
		NewStackLines(1),
		NewStackFormatted(1, ""),
	}

	for _, msg := range cases {
		assert.False(msg.Loggable())
	}
}

func TestDataCollecterComposerConstructors(t *testing.T) {
	const testMsg = "hello"
	assert := assert.New(t)
	// map objects to output (prefix)
	cases := map[Composer]string{
		NewProcessInfo(level.Error, int32(os.Getpid()), testMsg): "",
		NewSystemInfo(level.Error, testMsg):                      testMsg,
		MakeSystemInfo(testMsg):                                  testMsg,
		CollectProcessInfo(int32(1)):                             "",
		CollectProcessInfoSelf():                                 "",
		CollectSystemInfo():                                      "",
	}

	for msg, prefix := range cases {
		assert.NotNil(msg)
		assert.NotNil(msg.Raw())
		assert.Implements((*Composer)(nil), msg)
		assert.True(msg.Loggable())
		assert.True(strings.HasPrefix(msg.String(), prefix), fmt.Sprintf("%T: %s", msg, msg))
	}

	multiCases := [][]Composer{
		CollectProcessInfoSelfWithChildren(),
		CollectProcessInfoWithChildren(int32(1)),
	}

	for _, group := range multiCases {
		assert.True(len(group) >= 1)
		for _, msg := range group {
			assert.NotNil(msg)
			assert.Implements((*Composer)(nil), msg)
			assert.NotEqual("", msg.String())
			assert.True(msg.Loggable())
		}
	}
}

func TestStackMessages(t *testing.T) {
	const testMsg = "hello"
	const stackMsg = "message/composer_test"
	assert := assert.New(t)
	// map objects to output (prefix)
	cases := map[Composer]string{
		NewStack(1, testMsg):                                       testMsg,
		NewStackLines(1, testMsg):                                  testMsg,
		NewStackLines(1):                                           "",
		NewStackFormatted(1, "%s", testMsg):                        testMsg,
		NewStackFormatted(1, string(testMsg[0])+"%s", testMsg[1:]): testMsg,

		// with 0 frame
		NewStack(0, testMsg):                                       testMsg,
		NewStackLines(0, testMsg):                                  testMsg,
		NewStackLines(0):                                           "",
		NewStackFormatted(0, "%s", testMsg):                        testMsg,
		NewStackFormatted(0, string(testMsg[0])+"%s", testMsg[1:]): testMsg,
	}

	for msg, text := range cases {
		assert.NotNil(msg)
		assert.Implements((*Composer)(nil), msg)
		assert.NotNil(msg.Raw())
		if text != "" {
			assert.True(msg.Loggable())
		}

		diagMsg := fmt.Sprintf("%T: %+v", msg, msg)
		assert.True(strings.Contains(msg.String(), text), diagMsg)
		assert.True(strings.Contains(msg.String(), stackMsg), diagMsg)
	}
}

func TestComposerConverter(t *testing.T) {
	const testMsg = "hello world"
	assert := assert.New(t)

	cases := []interface{}{
		NewLine(testMsg),
		testMsg,
		errors.New(testMsg),
		[]string{testMsg},
		[]interface{}{testMsg},
		[]byte(testMsg),
	}

	for _, msg := range cases {
		comp := ConvertToComposer(level.Error, msg)
		assert.True(comp.Loggable())
		assert.Equal(testMsg, comp.String(), fmt.Sprintf("%T", msg))
	}

	cases = []interface{}{
		nil,
		"",
		[]interface{}{},
		[]string{},
		[]byte{},
		Fields{},
		map[string]interface{}{},
	}

	for _, msg := range cases {
		comp := ConvertToComposer(level.Error, msg)
		assert.False(comp.Loggable())
		assert.Equal("", comp.String(), fmt.Sprintf("%T", msg))
	}

	outputCases := map[string]interface{}{
		"1":         1,
		"2":         int32(2),
		"[msg='3']": Fields{"msg": 3},
		"[msg='4']": map[string]interface{}{"msg": "4"},
	}

	for out, in := range outputCases {
		comp := ConvertToComposer(level.Error, in)
		assert.True(comp.Loggable())
		assert.Equal(out, comp.String(), fmt.Sprintf("%T", in))
	}

}
