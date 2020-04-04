// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package internal

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	commonpb "go.temporal.io/temporal-proto/common"
	decisionpb "go.temporal.io/temporal-proto/decision"
	eventpb "go.temporal.io/temporal-proto/event"
	"go.uber.org/zap"
)

const (
	// assume this is some error reason defined by activity implementation.
	customErrReasonA = "CustomReasonA"
)

type testStruct struct {
	Name string
	Age  int
}

type testStruct2 struct {
	Name      string
	Age       int
	Favorites *[]string
}

var (
	testErrorDetails1 = "my details"
	testErrorDetails2 = 123
	testErrorDetails3 = testStruct{"a string", 321}
	testErrorDetails4 = testStruct2{"a string", 321, &[]string{"eat", "code"}}
)

func Test_GenericError(t *testing.T) {
	// test activity error
	errorActivityFn := func() error {
		return errors.New("error:foo")
	}
	s := &WorkflowTestSuite{}
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(errorActivityFn)
	_, err := env.ExecuteActivity(errorActivityFn)
	require.Error(t, err)
	require.Equal(t, &GenericError{"error:foo"}, err)

	// test workflow error
	errorWorkflowFn := func(ctx Context) error {
		return errors.New("error:foo")
	}
	wfEnv := s.NewTestWorkflowEnvironment()
	wfEnv.RegisterWorkflow(errorWorkflowFn)
	wfEnv.ExecuteWorkflow(errorWorkflowFn)
	err = wfEnv.GetWorkflowError()
	require.Error(t, err)
	require.Equal(t, &GenericError{"error:foo"}, err)
}

func Test_ActivityNotRegistered(t *testing.T) {
	registeredActivityFn, unregisteredActivitFn := "RegisteredActivity", "UnregisteredActivityFn"
	s := &WorkflowTestSuite{}
	s.SetLogger(zap.NewNop())
	env := s.NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(func() error { return nil }, RegisterActivityOptions{Name: registeredActivityFn})
	_, err := env.ExecuteActivity(unregisteredActivitFn)
	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("unable to find activityType=%v", unregisteredActivitFn))
	require.Contains(t, err.Error(), registeredActivityFn)
}

func Test_TimeoutError(t *testing.T) {
	timeoutErr := NewTimeoutError(eventpb.TimeoutType_ScheduleToStart)
	require.False(t, timeoutErr.HasDetails())
	var data string
	require.Equal(t, ErrNoData, timeoutErr.Details(&data))

	heartbeatErr := NewHeartbeatTimeoutError(testErrorDetails1)
	require.True(t, heartbeatErr.HasDetails())
	require.NoError(t, heartbeatErr.Details(&data))
	require.Equal(t, testErrorDetails1, data)
}

func Test_TimeoutError_WithDetails(t *testing.T) {
	testTimeoutErrorDetails(t, eventpb.TimeoutType_Heartbeat)
	testTimeoutErrorDetails(t, eventpb.TimeoutType_ScheduleToClose)
	testTimeoutErrorDetails(t, eventpb.TimeoutType_StartToClose)
}

func testTimeoutErrorDetails(t *testing.T, timeoutType eventpb.TimeoutType) {
	context := &workflowEnvironmentImpl{
		decisionsHelper: newDecisionsHelper(),
		dataConverter:   getDefaultDataConverter(),
	}
	h := newDecisionsHelper()
	var actualErr error
	activityID := "activityID"
	context.decisionsHelper.scheduledEventIDToActivityID[5] = activityID
	di := h.newActivityDecisionStateMachine(
		&decisionpb.ScheduleActivityTaskDecisionAttributes{ActivityId: activityID})
	di.state = decisionStateInitiated
	di.setData(&scheduledActivity{
		callback: func(r []byte, e error) {
			actualErr = e
		},
	})
	context.decisionsHelper.addDecision(di)
	encodedDetails1, _ := context.dataConverter.ToData(testErrorDetails1)
	event := createTestEventActivityTaskTimedOut(7, &eventpb.ActivityTaskTimedOutEventAttributes{
		Details:          encodedDetails1,
		ScheduledEventId: 5,
		StartedEventId:   6,
		TimeoutType:      timeoutType,
	})
	weh := &workflowExecutionEventHandlerImpl{context, nil}
	_ = weh.handleActivityTaskTimedOut(event)
	err, ok := actualErr.(*TimeoutError)
	require.True(t, ok)
	require.True(t, err.HasDetails())
	data := ""
	require.NoError(t, err.Details(&data))
	require.Equal(t, testErrorDetails1, data)
}

func Test_CustomError(t *testing.T) {
	// test ErrorDetailValues as Details
	var a1 string
	var a2 int
	var a3 testStruct
	err0 := NewCustomError(customErrReasonA, testErrorDetails1)
	require.True(t, err0.HasDetails())
	_ = err0.Details(&a1)
	require.Equal(t, testErrorDetails1, a1)
	a1 = ""
	err0 = NewCustomError(customErrReasonA, testErrorDetails1, testErrorDetails2, testErrorDetails3)
	require.True(t, err0.HasDetails())
	_ = err0.Details(&a1, &a2, &a3)
	require.Equal(t, testErrorDetails1, a1)
	require.Equal(t, testErrorDetails2, a2)
	require.Equal(t, testErrorDetails3, a3)

	// test EncodedValues as Details
	errorActivityFn := func() error {
		return err0
	}
	s := &WorkflowTestSuite{}
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(errorActivityFn)
	_, err := env.ExecuteActivity(errorActivityFn)
	require.Error(t, err)
	err1, ok := err.(*CustomError)
	require.True(t, ok)
	require.True(t, err1.HasDetails())
	var b1 string
	var b2 int
	var b3 testStruct
	_ = err1.Details(&b1, &b2, &b3)
	require.Equal(t, testErrorDetails1, b1)
	require.Equal(t, testErrorDetails2, b2)
	require.Equal(t, testErrorDetails3, b3)

	// test reason and no detail
	require.Panics(t, func() { _ = NewCustomError("temporalInternal:testReason") })
	newReason := "another reason"
	err2 := NewCustomError(newReason)
	require.True(t, !err2.HasDetails())
	require.Equal(t, ErrNoData, err2.Details())
	require.Equal(t, newReason, err2.Reason())
	err3 := NewCustomError(newReason, nil)
	// TODO: probably we want to handle this case when details are nil, HasDetails return false
	require.True(t, err3.HasDetails())

	// test workflow error
	errorWorkflowFn := func(ctx Context) error {
		return err0
	}
	wfEnv := s.NewTestWorkflowEnvironment()
	wfEnv.RegisterWorkflow(errorWorkflowFn)
	wfEnv.ExecuteWorkflow(errorWorkflowFn)
	err = wfEnv.GetWorkflowError()
	require.Error(t, err)
	err4, ok := err.(*CustomError)
	require.True(t, ok)
	require.True(t, err4.HasDetails())
	_ = err4.Details(&b1, &b2, &b3)
	require.Equal(t, testErrorDetails1, b1)
	require.Equal(t, testErrorDetails2, b2)
	require.Equal(t, testErrorDetails3, b3)
}

func Test_CustomError_Pointer(t *testing.T) {
	a1 := testStruct2{}
	err1 := NewCustomError(customErrReasonA, testErrorDetails4)
	require.True(t, err1.HasDetails())
	err := err1.Details(&a1)
	require.NoError(t, err)
	require.Equal(t, testErrorDetails4, a1)

	a2 := &testStruct2{}
	err2 := NewCustomError(customErrReasonA, &testErrorDetails4) // // pointer in details
	require.True(t, err2.HasDetails())
	err = err2.Details(&a2)
	require.NoError(t, err)
	require.Equal(t, &testErrorDetails4, a2)

	// test EncodedValues as Details
	errorActivityFn := func() error {
		return err1
	}
	s := &WorkflowTestSuite{}
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(errorActivityFn)
	_, err = env.ExecuteActivity(errorActivityFn)
	require.Error(t, err)
	err3, ok := err.(*CustomError)
	require.True(t, ok)
	require.True(t, err3.HasDetails())
	b1 := testStruct2{}
	require.NoError(t, err3.Details(&b1))
	require.Equal(t, testErrorDetails4, b1)

	errorActivityFn2 := func() error {
		return err2 // pointer in details
	}
	env.RegisterActivity(errorActivityFn2)
	_, err = env.ExecuteActivity(errorActivityFn2)
	require.Error(t, err)
	err4, ok := err.(*CustomError)
	require.True(t, ok)
	require.True(t, err4.HasDetails())
	b2 := &testStruct2{}
	require.NoError(t, err4.Details(&b2))
	require.Equal(t, &testErrorDetails4, b2)

	// test workflow error
	errorWorkflowFn := func(ctx Context) error {
		return err1
	}
	wfEnv := s.NewTestWorkflowEnvironment()
	wfEnv.RegisterWorkflow(errorWorkflowFn)
	wfEnv.ExecuteWorkflow(errorWorkflowFn)
	err = wfEnv.GetWorkflowError()
	require.Error(t, err)
	err5, ok := err.(*CustomError)
	require.True(t, ok)
	require.True(t, err5.HasDetails())
	_ = err5.Details(&b1)
	require.NoError(t, err5.Details(&b1))
	require.Equal(t, testErrorDetails4, b1)

	errorWorkflowFn2 := func(ctx Context) error {
		return err2 // pointer in details
	}
	wfEnv = s.NewTestWorkflowEnvironment()
	wfEnv.RegisterWorkflow(errorWorkflowFn2)
	wfEnv.ExecuteWorkflow(errorWorkflowFn2)
	err = wfEnv.GetWorkflowError()
	require.Error(t, err)
	err6, ok := err.(*CustomError)
	require.True(t, ok)
	require.True(t, err6.HasDetails())
	_ = err6.Details(&b2)
	require.NoError(t, err6.Details(&b2))
	require.Equal(t, &testErrorDetails4, b2)
}

func Test_CanceledError(t *testing.T) {
	// test ErrorDetailValues as Details
	var a1 string
	var a2 int
	var a3 testStruct
	err0 := NewCanceledError(testErrorDetails1)
	require.True(t, err0.HasDetails())
	_ = err0.Details(&a1)
	require.Equal(t, testErrorDetails1, a1)
	a1 = ""
	err0 = NewCanceledError(testErrorDetails1, testErrorDetails2, testErrorDetails3)
	require.True(t, err0.HasDetails())
	_ = err0.Details(&a1, &a2, &a3)
	require.Equal(t, testErrorDetails1, a1)
	require.Equal(t, testErrorDetails2, a2)
	require.Equal(t, testErrorDetails3, a3)

	// test EncodedValues as Details
	errorActivityFn := func() error {
		return err0
	}
	s := &WorkflowTestSuite{}
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(errorActivityFn)
	_, err := env.ExecuteActivity(errorActivityFn)
	require.Error(t, err)
	err1, ok := err.(*CanceledError)
	require.True(t, ok)
	require.True(t, err1.HasDetails())
	var b1 string
	var b2 int
	var b3 testStruct
	_ = err1.Details(&b1, &b2, &b3)
	require.Equal(t, testErrorDetails1, b1)
	require.Equal(t, testErrorDetails2, b2)
	require.Equal(t, testErrorDetails3, b3)

	err2 := NewCanceledError()
	require.False(t, err2.HasDetails())

	// test workflow error
	errorWorkflowFn := func(ctx Context) error {
		return err0
	}
	wfEnv := s.NewTestWorkflowEnvironment()
	wfEnv.RegisterWorkflow(errorWorkflowFn)
	wfEnv.ExecuteWorkflow(errorWorkflowFn)
	err = wfEnv.GetWorkflowError()
	require.Error(t, err)
	err3, ok := err.(*CanceledError)
	require.True(t, ok)
	require.True(t, err3.HasDetails())
	_ = err3.Details(&b1, &b2, &b3)
	require.Equal(t, testErrorDetails1, b1)
	require.Equal(t, testErrorDetails2, b2)
	require.Equal(t, testErrorDetails3, b3)
}

func Test_IsCanceledError(t *testing.T) {

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "empty detail",
			err:      NewCanceledError(),
			expected: true,
		},
		{
			name:     "with detail",
			err:      NewCanceledError("details"),
			expected: true,
		},
		{
			name:     "not canceled error",
			err:      errors.New("details"),
			expected: false,
		},
	}

	for _, test := range tests {
		require.Equal(t, test.expected, IsCanceledError(test.err))
	}
}

func TestErrorDetailsValues(t *testing.T) {
	e := ErrorDetailsValues{}
	require.Equal(t, ErrNoData, e.Get())

	e = ErrorDetailsValues{testErrorDetails1, testErrorDetails2, testErrorDetails3}
	var a1 string
	var a2 int
	var a3 testStruct
	require.True(t, e.HasValues())
	_ = e.Get(&a1)
	require.Equal(t, testErrorDetails1, a1)
	_ = e.Get(&a1, &a2, &a3)
	require.Equal(t, testErrorDetails1, a1)
	require.Equal(t, testErrorDetails2, a2)
	require.Equal(t, testErrorDetails3, a3)

	require.Equal(t, ErrTooManyArg, e.Get(&a1, &a2, &a3, &a3))
}

func Test_SignalExternalWorkflowExecutionFailedError(t *testing.T) {
	context := &workflowEnvironmentImpl{
		decisionsHelper: newDecisionsHelper(),
		dataConverter:   getDefaultDataConverter(),
	}
	h := newDecisionsHelper()
	var actualErr error
	var initiatedEventID int64 = 101
	signalID := "signalID"
	context.decisionsHelper.scheduledEventIDToSignalID[initiatedEventID] = signalID
	di := h.newSignalExternalWorkflowStateMachine(
		&decisionpb.SignalExternalWorkflowExecutionDecisionAttributes{},
		signalID,
	)
	di.state = decisionStateInitiated
	di.setData(&scheduledSignal{
		callback: func(r []byte, e error) {
			actualErr = e
		},
	})
	context.decisionsHelper.addDecision(di)
	weh := &workflowExecutionEventHandlerImpl{context, nil}
	event := createTestEventSignalExternalWorkflowExecutionFailed(1, &eventpb.SignalExternalWorkflowExecutionFailedEventAttributes{
		InitiatedEventId: initiatedEventID,
		Cause:            eventpb.WorkflowExecutionFailedCause_UnknownExternalWorkflowExecution,
	})
	require.NoError(t, weh.handleSignalExternalWorkflowExecutionFailed(event))
	_, ok := actualErr.(*UnknownExternalWorkflowExecutionError)
	require.True(t, ok)
}

func Test_ContinueAsNewError(t *testing.T) {
	var a1 = 1234
	var a2 = "some random input"

	continueAsNewWfName := "continueAsNewWorkflowFn"
	continueAsNewWorkflowFn := func(ctx Context, testInt int, testString string) error {
		return NewContinueAsNewError(ctx, continueAsNewWfName, a1, a2)
	}

	header := &commonpb.Header{
		Fields: map[string][]byte{"test": []byte("test-data")},
	}

	s := &WorkflowTestSuite{
		header:             header,
		contextPropagators: []ContextPropagator{NewStringMapPropagator([]string{"test"})},
	}
	wfEnv := s.NewTestWorkflowEnvironment()
	wfEnv.RegisterWorkflowWithOptions(continueAsNewWorkflowFn, RegisterWorkflowOptions{
		Name: continueAsNewWfName,
	})
	wfEnv.ExecuteWorkflow(continueAsNewWorkflowFn, 101, "another random string")
	err := wfEnv.GetWorkflowError()

	require.Error(t, err)
	continueAsNewErr, ok := err.(*ContinueAsNewError)
	require.True(t, ok)
	require.Equal(t, continueAsNewWfName, continueAsNewErr.WorkflowType().Name)

	args := continueAsNewErr.Args()
	intArg, ok := args[0].(int)
	require.True(t, ok)
	require.Equal(t, a1, intArg)
	stringArg, ok := args[1].(string)
	require.True(t, ok)
	require.Equal(t, a2, stringArg)
	require.Equal(t, header, continueAsNewErr.params.header)
}
