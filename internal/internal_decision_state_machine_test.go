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
	"testing"

	"github.com/stretchr/testify/require"
	s "go.temporal.io/temporal/.gen/go/shared"
	"go.temporal.io/temporal/internal/common"
)

func Test_TimerStateMachine_CancelBeforeSent(t *testing.T) {
	timerID := "test-timer-1"
	attributes := &s.StartTimerDecisionAttributes{
		TimerId: common.StringPtr(timerID),
	}
	h := newDecisionsHelper()
	d := h.startTimer(attributes)
	require.Equal(t, decisionStateCreated, d.getState())
	h.cancelTimer(timerID)
	require.Equal(t, decisionStateCompleted, d.getState())
	decisions := h.getDecisions(true)
	require.Equal(t, 0, len(decisions))
}

func Test_TimerStateMachine_CancelAfterInitiated(t *testing.T) {
	timerID := "test-timer-1"
	attributes := &s.StartTimerDecisionAttributes{
		TimerId: common.StringPtr(timerID),
	}
	h := newDecisionsHelper()
	d := h.startTimer(attributes)
	require.Equal(t, decisionStateCreated, d.getState())
	decisions := h.getDecisions(true)
	require.Equal(t, decisionStateDecisionSent, d.getState())
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeStartTimer, decisions[0].GetDecisionType())
	require.Equal(t, attributes, decisions[0].StartTimerDecisionAttributes)
	h.handleTimerStarted(timerID)
	require.Equal(t, decisionStateInitiated, d.getState())
	h.cancelTimer(timerID)
	require.Equal(t, decisionStateCanceledAfterInitiated, d.getState())
	decisions = h.getDecisions(true)
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeCancelTimer, decisions[0].GetDecisionType())
	require.Equal(t, decisionStateCancellationDecisionSent, d.getState())
	h.handleTimerCanceled(timerID)
	require.Equal(t, decisionStateCompleted, d.getState())
}

func Test_TimerStateMachine_CompletedAfterCancel(t *testing.T) {
	timerID := "test-timer-1"
	attributes := &s.StartTimerDecisionAttributes{
		TimerId: common.StringPtr(timerID),
	}
	h := newDecisionsHelper()
	d := h.startTimer(attributes)
	require.Equal(t, decisionStateCreated, d.getState())
	decisions := h.getDecisions(true)
	require.Equal(t, decisionStateDecisionSent, d.getState())
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeStartTimer, decisions[0].GetDecisionType())
	h.cancelTimer(timerID)
	require.Equal(t, decisionStateCanceledBeforeInitiated, d.getState())
	require.Equal(t, 0, len(h.getDecisions(true)))
	h.handleTimerStarted(timerID)
	require.Equal(t, decisionStateCanceledAfterInitiated, d.getState())
	decisions = h.getDecisions(true)
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeCancelTimer, decisions[0].GetDecisionType())
	require.Equal(t, decisionStateCancellationDecisionSent, d.getState())
	h.handleTimerClosed(timerID)
	require.Equal(t, decisionStateCompletedAfterCancellationDecisionSent, d.getState())
}

func Test_TimerStateMachine_CompleteWithoutCancel(t *testing.T) {
	timerID := "test-timer-1"
	attributes := &s.StartTimerDecisionAttributes{
		TimerId: common.StringPtr(timerID),
	}
	h := newDecisionsHelper()
	d := h.startTimer(attributes)
	require.Equal(t, decisionStateCreated, d.getState())
	decisions := h.getDecisions(true)
	require.Equal(t, decisionStateDecisionSent, d.getState())
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeStartTimer, decisions[0].GetDecisionType())
	h.handleTimerStarted(timerID)
	require.Equal(t, decisionStateInitiated, d.getState())
	require.Equal(t, 0, len(h.getDecisions(false)))
	h.handleTimerClosed(timerID)
	require.Equal(t, decisionStateCompleted, d.getState())
}

func Test_TimerStateMachine_PanicInvalidStateTransition(t *testing.T) {
	timerID := "test-timer-1"
	attributes := &s.StartTimerDecisionAttributes{
		TimerId: common.StringPtr(timerID),
	}
	h := newDecisionsHelper()
	h.startTimer(attributes)
	h.getDecisions(true)
	h.handleTimerStarted(timerID)
	h.handleTimerClosed(timerID)

	panicErr := runAndCatchPanic(func() {
		h.handleCancelTimerFailed(timerID)
	})

	require.NotNil(t, panicErr)
}

func Test_TimerCancelEventOrdering(t *testing.T) {
	timerID := "test-timer-1"
	localActivityID := "test-activity-1"
	attributes := &s.StartTimerDecisionAttributes{
		TimerId: common.StringPtr(timerID),
	}
	h := newDecisionsHelper()
	d := h.startTimer(attributes)
	require.Equal(t, decisionStateCreated, d.getState())
	decisions := h.getDecisions(true)
	require.Equal(t, decisionStateDecisionSent, d.getState())
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeStartTimer, decisions[0].GetDecisionType())
	require.Equal(t, attributes, decisions[0].StartTimerDecisionAttributes)
	h.handleTimerStarted(timerID)
	require.Equal(t, decisionStateInitiated, d.getState())
	m := h.recordLocalActivityMarker(localActivityID, []byte{})
	require.Equal(t, decisionStateCreated, m.getState())
	h.cancelTimer(timerID)
	require.Equal(t, decisionStateCanceledAfterInitiated, d.getState())
	decisions = h.getDecisions(true)
	require.Equal(t, 2, len(decisions))
	require.Equal(t, s.DecisionTypeRecordMarker, decisions[0].GetDecisionType())
	require.Equal(t, s.DecisionTypeCancelTimer, decisions[1].GetDecisionType())
}

func Test_ActivityStateMachine_CompleteWithoutCancel(t *testing.T) {
	activityID := "test-activity-1"
	attributes := &s.ScheduleActivityTaskDecisionAttributes{
		ActivityId: common.StringPtr(activityID),
	}
	h := newDecisionsHelper()

	// schedule activity
	d := h.scheduleActivityTask(attributes)
	require.Equal(t, decisionStateCreated, d.getState())
	decisions := h.getDecisions(true)
	require.Equal(t, decisionStateDecisionSent, d.getState())
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeScheduleActivityTask, decisions[0].GetDecisionType())

	// activity scheduled
	h.handleActivityTaskScheduled(1, activityID)
	require.Equal(t, decisionStateInitiated, d.getState())

	// activity completed
	h.handleActivityTaskClosed(activityID)
	require.Equal(t, decisionStateCompleted, d.getState())
}

func Test_ActivityStateMachine_CancelBeforeSent(t *testing.T) {
	activityID := "test-activity-1"
	attributes := &s.ScheduleActivityTaskDecisionAttributes{
		ActivityId: common.StringPtr(activityID),
	}
	h := newDecisionsHelper()

	// schedule activity
	d := h.scheduleActivityTask(attributes)
	require.Equal(t, decisionStateCreated, d.getState())

	// cancel before decision sent, this will put decision state machine directly into completed state
	h.requestCancelActivityTask(activityID)
	require.Equal(t, decisionStateCompleted, d.getState())

	// there should be no decisions needed to be send
	decisions := h.getDecisions(true)
	require.Equal(t, 0, len(decisions))
}

func Test_ActivityStateMachine_CancelAfterSent(t *testing.T) {
	activityID := "test-activity-1"
	attributes := &s.ScheduleActivityTaskDecisionAttributes{
		ActivityId: common.StringPtr(activityID),
	}
	h := newDecisionsHelper()

	// schedule activity
	d := h.scheduleActivityTask(attributes)
	require.Equal(t, decisionStateCreated, d.getState())
	decisions := h.getDecisions(true)
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeScheduleActivityTask, decisions[0].GetDecisionType())

	// cancel activity
	h.requestCancelActivityTask(activityID)
	require.Equal(t, decisionStateCanceledBeforeInitiated, d.getState())
	require.Equal(t, 0, len(h.getDecisions(true)))

	// activity scheduled
	h.handleActivityTaskScheduled(1, activityID)
	require.Equal(t, decisionStateCanceledAfterInitiated, d.getState())
	decisions = h.getDecisions(true)
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeRequestCancelActivityTask, decisions[0].GetDecisionType())

	// activity canceled
	h.handleActivityTaskCanceled(activityID)
	require.Equal(t, decisionStateCompleted, d.getState())
	require.Equal(t, 0, len(h.getDecisions(false)))
}

func Test_ActivityStateMachine_CompletedAfterCancel(t *testing.T) {
	activityID := "test-activity-1"
	attributes := &s.ScheduleActivityTaskDecisionAttributes{
		ActivityId: common.StringPtr(activityID),
	}
	h := newDecisionsHelper()

	// schedule activity
	d := h.scheduleActivityTask(attributes)
	require.Equal(t, decisionStateCreated, d.getState())
	decisions := h.getDecisions(true)
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeScheduleActivityTask, decisions[0].GetDecisionType())

	// cancel activity
	h.requestCancelActivityTask(activityID)
	require.Equal(t, decisionStateCanceledBeforeInitiated, d.getState())
	require.Equal(t, 0, len(h.getDecisions(true)))

	// activity scheduled
	h.handleActivityTaskScheduled(1, activityID)
	require.Equal(t, decisionStateCanceledAfterInitiated, d.getState())
	decisions = h.getDecisions(true)
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeRequestCancelActivityTask, decisions[0].GetDecisionType())

	// activity completed after cancel
	h.handleActivityTaskClosed(activityID)
	require.Equal(t, decisionStateCompletedAfterCancellationDecisionSent, d.getState())
	require.Equal(t, 0, len(h.getDecisions(false)))
}

func Test_ActivityStateMachine_PanicInvalidStateTransition(t *testing.T) {
	activityID := "test-activity-1"
	attributes := &s.ScheduleActivityTaskDecisionAttributes{
		ActivityId: common.StringPtr(activityID),
	}
	h := newDecisionsHelper()

	// schedule activity
	h.scheduleActivityTask(attributes)

	// verify that using invalid activity id will panic
	err := runAndCatchPanic(func() {
		h.handleActivityTaskClosed("invalid-activity-id")
	})
	require.NotNil(t, err)

	// send schedule decision
	h.getDecisions(true)
	// activity scheduled
	h.handleActivityTaskScheduled(1, activityID)

	// now simulate activity canceled, which is invalid transition
	err = runAndCatchPanic(func() {
		h.handleActivityTaskCanceled(activityID)
	})
	require.NotNil(t, err)
}

func Test_ChildWorkflowStateMachine_Basic(t *testing.T) {
	workflowID := "test-child-workflow-1"
	attributes := &s.StartChildWorkflowExecutionDecisionAttributes{
		WorkflowId: common.StringPtr(workflowID),
	}
	h := newDecisionsHelper()

	// start child workflow
	d := h.startChildWorkflowExecution(attributes)
	require.Equal(t, decisionStateCreated, d.getState())

	// send decision
	decisions := h.getDecisions(true)
	require.Equal(t, decisionStateDecisionSent, d.getState())
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeStartChildWorkflowExecution, decisions[0].GetDecisionType())

	// child workflow initiated
	h.handleStartChildWorkflowExecutionInitiated(workflowID)
	require.Equal(t, decisionStateInitiated, d.getState())
	require.Equal(t, 0, len(h.getDecisions(true)))

	// child workflow started
	h.handleChildWorkflowExecutionStarted(workflowID)
	require.Equal(t, decisionStateStarted, d.getState())
	require.Equal(t, 0, len(h.getDecisions(true)))

	// child workflow completed
	h.handleChildWorkflowExecutionClosed(workflowID)
	require.Equal(t, decisionStateCompleted, d.getState())
	require.Equal(t, 0, len(h.getDecisions(true)))
}

func Test_ChildWorkflowStateMachine_CancelSucceed(t *testing.T) {
	domain := "test-domain"
	workflowID := "test-child-workflow"
	runID := ""
	cancellationID := ""
	initiatedEventID := int64(28)
	isChildWorkflowOnly := true
	attributes := &s.StartChildWorkflowExecutionDecisionAttributes{
		WorkflowId: common.StringPtr(workflowID),
	}
	h := newDecisionsHelper()

	// start child workflow
	d := h.startChildWorkflowExecution(attributes)
	// send decision
	decisions := h.getDecisions(true)
	// child workflow initiated
	h.handleStartChildWorkflowExecutionInitiated(workflowID)
	// child workflow started
	h.handleChildWorkflowExecutionStarted(workflowID)

	// cancel child workflow
	h.requestCancelExternalWorkflowExecution(domain, workflowID, runID, cancellationID, isChildWorkflowOnly)
	require.Equal(t, decisionStateCanceledAfterStarted, d.getState())

	// send cancel request
	decisions = h.getDecisions(true)
	require.Equal(t, decisionStateCancellationDecisionSent, d.getState())
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeRequestCancelExternalWorkflowExecution, decisions[0].GetDecisionType())

	// cancel request initiated
	h.handleRequestCancelExternalWorkflowExecutionInitiated(initiatedEventID, workflowID, cancellationID)
	require.Equal(t, decisionStateCancellationDecisionSent, d.getState())

	// cancel request accepted
	h.handleExternalWorkflowExecutionCancelRequested(initiatedEventID, workflowID)
	require.Equal(t, decisionStateCancellationDecisionSent, d.getState())

	// child workflow canceled
	h.handleChildWorkflowExecutionCanceled(workflowID)
	require.Equal(t, decisionStateCompleted, d.getState())
}

func Test_ChildWorkflowStateMachine_InvalidStates(t *testing.T) {
	domain := "test-domain"
	workflowID := "test-workflow-id"
	runID := ""
	attributes := &s.StartChildWorkflowExecutionDecisionAttributes{
		WorkflowId: common.StringPtr(workflowID),
	}
	cancellationID := ""
	initiatedEventID := int64(28)
	isChildWorkflowOnly := true
	h := newDecisionsHelper()

	// start child workflow
	d := h.startChildWorkflowExecution(attributes)
	require.Equal(t, decisionStateCreated, d.getState())

	// invalid: start child workflow failed before decision was sent
	err := runAndCatchPanic(func() {
		h.handleStartChildWorkflowExecutionFailed(workflowID)
	})
	require.NotNil(t, err)

	// send decision
	decisions := h.getDecisions(true)
	require.Equal(t, decisionStateDecisionSent, d.getState())
	require.Equal(t, 1, len(decisions))

	// invalid: child workflow completed before it was initiated
	err = runAndCatchPanic(func() {
		h.handleChildWorkflowExecutionClosed(workflowID)
	})
	require.NotNil(t, err)

	// child workflow initiated
	h.handleStartChildWorkflowExecutionInitiated(workflowID)
	require.Equal(t, decisionStateInitiated, d.getState())

	h.handleChildWorkflowExecutionStarted(workflowID)
	require.Equal(t, decisionStateStarted, d.getState())
	// invalid: cancel child workflow failed before cancel request
	err = runAndCatchPanic(func() {
		h.handleRequestCancelExternalWorkflowExecutionFailed(initiatedEventID, workflowID)
	})
	require.NotNil(t, err)

	// cancel child workflow after child workflow is started
	h.requestCancelExternalWorkflowExecution(domain, workflowID, runID, cancellationID, isChildWorkflowOnly)
	require.Equal(t, decisionStateCanceledAfterStarted, d.getState())

	// send cancel request
	decisions = h.getDecisions(true)
	require.Equal(t, decisionStateCancellationDecisionSent, d.getState())
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeRequestCancelExternalWorkflowExecution, decisions[0].GetDecisionType())

	// invalid: start child workflow failed after it was already started
	err = runAndCatchPanic(func() {
		h.handleStartChildWorkflowExecutionFailed(workflowID)
	})
	require.NotNil(t, err)

	// invalid: child workflow initiated again
	err = runAndCatchPanic(func() {
		h.handleStartChildWorkflowExecutionInitiated(workflowID)
	})
	require.NotNil(t, err)

	// cancel request initiated
	h.handleRequestCancelExternalWorkflowExecutionInitiated(initiatedEventID, workflowID, cancellationID)
	require.Equal(t, decisionStateCancellationDecisionSent, d.getState())

	// child workflow completed
	h.handleChildWorkflowExecutionClosed(workflowID)
	require.Equal(t, decisionStateCompletedAfterCancellationDecisionSent, d.getState())

	// invalid: child workflow canceled after it was completed
	err = runAndCatchPanic(func() {
		h.handleChildWorkflowExecutionCanceled(workflowID)
	})
	require.NotNil(t, err)
}

func Test_ChildWorkflowStateMachine_CancelFailed(t *testing.T) {
	domain := "test-domain"
	workflowID := "test-workflow-id"
	runID := ""
	attributes := &s.StartChildWorkflowExecutionDecisionAttributes{
		WorkflowId: common.StringPtr(workflowID),
	}
	cancellationID := ""
	initiatedEventID := int64(28)
	isChildWorkflowOnly := true
	h := newDecisionsHelper()

	// start child workflow
	d := h.startChildWorkflowExecution(attributes)
	// send decision
	h.getDecisions(true)
	// child workflow initiated
	h.handleStartChildWorkflowExecutionInitiated(workflowID)
	// child workflow started
	h.handleChildWorkflowExecutionStarted(workflowID)
	// cancel child workflow
	h.requestCancelExternalWorkflowExecution(domain, workflowID, runID, cancellationID, isChildWorkflowOnly)
	// send cancel request
	h.getDecisions(true)
	// cancel request initiated
	h.handleRequestCancelExternalWorkflowExecutionInitiated(initiatedEventID, workflowID, cancellationID)

	// cancel request failed
	h.handleRequestCancelExternalWorkflowExecutionFailed(initiatedEventID, workflowID)
	require.Equal(t, decisionStateStarted, d.getState())

	// child workflow completed
	h.handleChildWorkflowExecutionClosed(workflowID)
	require.Equal(t, decisionStateCompleted, d.getState())
}

func Test_MarkerStateMachine(t *testing.T) {
	h := newDecisionsHelper()

	// record marker for side effect
	d := h.recordSideEffectMarker(1, []byte{})
	require.Equal(t, decisionStateCreated, d.getState())

	// send decisions
	decisions := h.getDecisions(true)
	require.Equal(t, decisionStateCompleted, d.getState())
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeRecordMarker, decisions[0].GetDecisionType())
}

func Test_UpsertSearchAttributesDecisionStateMachine(t *testing.T) {
	h := newDecisionsHelper()

	attr := &s.SearchAttributes{}
	d := h.upsertSearchAttributes("1", attr)
	require.Equal(t, decisionStateCreated, d.getState())

	decisions := h.getDecisions(true)
	require.Equal(t, decisionStateCompleted, d.getState())
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeUpsertWorkflowSearchAttributes, decisions[0].GetDecisionType())
}

func Test_CancelExternalWorkflowStateMachine_Succeed(t *testing.T) {
	domain := "test-domain"
	workflowID := "test-workflow-id"
	runID := "test-run-id"
	cancellationID := "1"
	initiatedEventID := int64(28)
	childWorkflowOnly := false
	h := newDecisionsHelper()

	// request cancel external workflow
	decision := h.requestCancelExternalWorkflowExecution(domain, workflowID, runID, cancellationID, childWorkflowOnly)
	require.False(t, decision.isDone())
	d := h.getDecision(makeDecisionID(decisionTypeCancellation, cancellationID))
	require.Equal(t, decisionStateCreated, d.getState())

	// send decisions
	decisions := h.getDecisions(true)
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeRequestCancelExternalWorkflowExecution, decisions[0].GetDecisionType())
	require.Equal(
		t,
		&s.RequestCancelExternalWorkflowExecutionDecisionAttributes{
			Domain:            common.StringPtr(domain),
			WorkflowId:        common.StringPtr(workflowID),
			RunId:             common.StringPtr(runID),
			Control:           []byte(cancellationID),
			ChildWorkflowOnly: common.BoolPtr(childWorkflowOnly),
		},
		decisions[0].RequestCancelExternalWorkflowExecutionDecisionAttributes,
	)

	// cancel request initiated
	h.handleRequestCancelExternalWorkflowExecutionInitiated(initiatedEventID, workflowID, cancellationID)
	require.Equal(t, decisionStateInitiated, d.getState())

	// cancel requested
	h.handleExternalWorkflowExecutionCancelRequested(initiatedEventID, workflowID)
	require.Equal(t, decisionStateCompleted, d.getState())

	// mark the cancel request failed now will make it invalid state transition
	err := runAndCatchPanic(func() {
		h.handleRequestCancelExternalWorkflowExecutionFailed(initiatedEventID, workflowID)
	})
	require.NotNil(t, err)
}

func Test_CancelExternalWorkflowStateMachine_Failed(t *testing.T) {
	domain := "test-domain"
	workflowID := "test-workflow-id"
	runID := "test-run-id"
	cancellationID := "2"
	initiatedEventID := int64(28)
	childWorkflowOnly := false
	h := newDecisionsHelper()

	// request cancel external workflow
	decision := h.requestCancelExternalWorkflowExecution(domain, workflowID, runID, cancellationID, childWorkflowOnly)
	require.False(t, decision.isDone())
	d := h.getDecision(makeDecisionID(decisionTypeCancellation, cancellationID))
	require.Equal(t, decisionStateCreated, d.getState())

	// send decisions
	decisions := h.getDecisions(true)
	require.Equal(t, 1, len(decisions))
	require.Equal(t, s.DecisionTypeRequestCancelExternalWorkflowExecution, decisions[0].GetDecisionType())
	require.Equal(
		t,
		&s.RequestCancelExternalWorkflowExecutionDecisionAttributes{
			Domain:            common.StringPtr(domain),
			WorkflowId:        common.StringPtr(workflowID),
			RunId:             common.StringPtr(runID),
			Control:           []byte(cancellationID),
			ChildWorkflowOnly: common.BoolPtr(childWorkflowOnly),
		},
		decisions[0].RequestCancelExternalWorkflowExecutionDecisionAttributes,
	)

	// cancel request initiated
	h.handleRequestCancelExternalWorkflowExecutionInitiated(initiatedEventID, workflowID, cancellationID)
	require.Equal(t, decisionStateInitiated, d.getState())

	// cancel request failed
	h.handleRequestCancelExternalWorkflowExecutionFailed(initiatedEventID, workflowID)
	require.Equal(t, decisionStateCompleted, d.getState())

	// mark the cancel request succeed now will make it invalid state transition
	err := runAndCatchPanic(func() {
		h.handleExternalWorkflowExecutionCancelRequested(initiatedEventID, workflowID)
	})
	require.NotNil(t, err)
}

func runAndCatchPanic(f func()) (err *PanicError) {
	// panic handler
	defer func() {
		if p := recover(); p != nil {
			topLine := "runAndCatchPanic [panic]:"
			st := getStackTraceRaw(topLine, 7, 0)
			err = newPanicError(p, st) // Fail decision on panic
		}
	}()

	f()
	return nil
}
