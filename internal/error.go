// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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
	"reflect"

	commonpb "go.temporal.io/temporal-proto/common/v1"
	enumspb "go.temporal.io/temporal-proto/enums/v1"
	failurepb "go.temporal.io/temporal-proto/failure/v1"
)

/*
If activity fails then *ActivityError is returned to the workflow code. The error has important information about activity
and actual error which caused activity failure. This internal error can be unwrapped using errors.Unwrap() or checked using errors.As().
Below are the possible types of internal error:
1) *ApplicationError: (this should be the most common one)
	*ApplicationError can be returned in two cases:
		- If activity implementation returns *ApplicationError by using NewApplicationError() API.
		  The err would contain a message, details, and NonRetryable flag. Workflow code could check this flag and details to determine
		  what kind of error it was and take actions based on it. The details is encoded payload which workflow code could extract
		  to strong typed variable. Workflow code needs to know what the types of the encoded details are before extracting them.
		- If activity implementation returns errors other than from NewApplicationError() API. In this case GetOriginalType()
		  will return orginal type of an error represented as string. Workflow code could check this type to determine what kind of error it was
		  and take actions based on the type. These errors are retryable by default, unless error type is specified in retry policy.
2) *CanceledError:
	If activity was canceled, internal error will be an instance of *CanceledError. When activity cancels itself by
	returning NewCancelError() it would supply optional details which could be extracted by workflow code.
3) *TimeoutError:
	If activity was timed out (several timeout types), internal error will be an instance of *TimeoutError. The err contains
	details about what type of timeout it was.
4) *PanicError:
	If activity code panic while executing, temporal activity worker will report it as activity failure to temporal server.
	The SDK will present that failure as *PanicError. The err contains a string	representation of the panic message and
	the call stack when panic was happen.

Workflow code could handle errors based on different types of error. Below is sample code of how error handling looks like.

err := workflow.ExecuteActivity(ctx, MyActivity, ...).Get(ctx, nil)
if err != nil {
	var applicationErr *ApplicationError
	if errors.As(err, &applicationError) {
		// handle activity errors (created via NewApplicationError() API)
		if !applicationErr.NonRetryable() {
			// manually retry activity
		}
		var detailMsg string // assuming activity return error by NewApplicationError("message", true, "string details")
		applicationErr.Details(&detailMsg) // extract strong typed details

		// handle activity errors (errors created other than using NewApplicationError() API)
		switch err.OriginalType() {
		case "CustomErrTypeA":
			// handle CustomErrTypeA
		case CustomErrTypeB:
			// handle CustomErrTypeB
		default:
			// newer version of activity could return new errors that workflow was not aware of.
		}
	}

	var canceledErr *CanceledError
	if errors.As(err, &canceledErr) {
		// handle cancellation
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		// handle timeout, could check timeout type by timeoutErr.TimeoutType()
        switch err.TimeoutType() {
        case commonpb.ScheduleToStart:
                // Handle ScheduleToStart timeout.
        case commonpb.StartToClose:
                // Handle StartToClose timeout.
        case commonpb.Heartbeat:
                // Handle heartbeat timeout.
        default:
        }
	}

	var panicErr *PanicError
	if errors.As(err, &panicErr) {
		// handle panic, message and stack trace are available by panicErr.Error() and panicErr.StackTrace()
	}
}

Errors from child workflow should be handled in a similar way, except that instance of *ChildWorkflowExecutionError is returned to
workflow code. It will contains *ActivityError, which in turn will contains on of the errors above.
When panic happen in workflow implementation code, SDK catches that panic and causing the decision timeout.
That decision task will be retried at a later time (with exponential backoff retry intervals).

Workflow consumers will get an instance of *WorkflowExecutionError. This error will contains one of errors above.
*/

type (
	// ApplicationError returned from activity implementations with message and optional details.
	ApplicationError struct {
		temporalError
		message      string
		originalType string
		nonRetryable bool
		cause        error
		details      Values
	}

	// TimeoutError returned when activity or child workflow timed out.
	TimeoutError struct {
		temporalError
		timeoutType          enumspb.TimeoutType
		lastHeartbeatDetails Values
		cause                error
	}

	// CanceledError returned when operation was canceled.
	CanceledError struct {
		temporalError
		details Values
	}

	// TerminatedError returned when workflow was terminated.
	TerminatedError struct {
		temporalError
	}

	// PanicError contains information about panicked workflow/activity.
	PanicError struct {
		temporalError
		value      interface{}
		stackTrace string
	}

	// workflowPanicError contains information about panicked workflow.
	// Used to distinguish go panic in the workflow code from a PanicError returned from a workflow function.
	workflowPanicError struct {
		value      interface{}
		stackTrace string
	}

	// ContinueAsNewError contains information about how to continue the workflow as new.
	ContinueAsNewError struct {
		wfn    interface{}
		args   []interface{}
		params *ExecuteWorkflowParams
	}

	// UnknownExternalWorkflowExecutionError can be returned when external workflow doesn't exist
	UnknownExternalWorkflowExecutionError struct{}

	// ServerError can be returned from server.
	ServerError struct {
		temporalError
		message      string
		nonRetryable bool
		cause        error
	}

	// ActivityError is returned from workflow when activity returned an error.
	// Unwrap this error to get actual cause.
	ActivityError struct {
		temporalError
		scheduledEventID int64
		startedEventID   int64
		identity         string
		activityType     *commonpb.ActivityType
		activityID       string
		retryStatus      enumspb.RetryStatus
		cause            error
	}

	// ChildWorkflowExecutionError is returned from workflow when child workflow returned an error.
	// Unwrap this error to get actual cause.
	ChildWorkflowExecutionError struct {
		temporalError
		namespace        string
		workflowID       string
		runID            string
		workflowType     string
		initiatedEventID int64
		startedEventID   int64
		retryStatus      enumspb.RetryStatus
		cause            error
	}

	// WorkflowExecutionError is returned from workflow.
	// Unwrap this error to get actual cause.
	WorkflowExecutionError struct {
		workflowID   string
		runID        string
		workflowType string
		cause        error
	}

	temporalError struct {
		originalFailure *failurepb.Failure
	}

	failureHolder interface {
		setFailure(*failurepb.Failure)
		failure() *failurepb.Failure
	}
)

// ErrNoData is returned when trying to extract strong typed data while there is no data available.
var ErrNoData = errors.New("no data available")

// ErrTooManyArg is returned when trying to extract strong typed data with more arguments than available data.
var ErrTooManyArg = errors.New("too many arguments")

// ErrActivityResultPending is returned from activity's implementation to indicate the activity is not completed when
// activity method returns. Activity needs to be completed by Client.CompleteActivity() separately. For example, if an
// activity require human interaction (like approve an expense report), the activity could return activity.ErrResultPending
// which indicate the activity is not done yet. Then, when the waited human action happened, it needs to trigger something
// that could report the activity completed event to temporal server via Client.CompleteActivity() API.
var ErrActivityResultPending = errors.New("not error: do not autocomplete, using Client.CompleteActivity() to complete")

// NewApplicationError create new instance of *ApplicationError with message and optional details.
func NewApplicationError(message string, nonRetryable bool, cause error, details ...interface{}) *ApplicationError {
	applicationErr := &ApplicationError{
		message:      message,
		originalType: getErrorType(&ApplicationError{}),
		nonRetryable: nonRetryable,
		cause:        cause}

	// When return error to user, use EncodedValues as details and data is ready to be decoded by calling Get
	if len(details) == 1 {
		if d, ok := details[0].(*EncodedValues); ok {
			applicationErr.details = d
			return applicationErr
		}
	}

	// When create error for server, use ErrorDetailsValues as details to hold values and encode later
	applicationErr.details = ErrorDetailsValues(details)
	return applicationErr
}

// NewTimeoutError creates TimeoutError instance.
// Use NewHeartbeatTimeoutError to create heartbeat TimeoutError.
func NewTimeoutError(timeoutType enumspb.TimeoutType, cause error, lastHeatbeatDetails ...interface{}) *TimeoutError {
	timeoutErr := &TimeoutError{
		timeoutType: timeoutType,
		cause:       cause,
	}

	if len(lastHeatbeatDetails) == 1 {
		if d, ok := lastHeatbeatDetails[0].(*EncodedValues); ok {
			timeoutErr.lastHeartbeatDetails = d
			return timeoutErr
		}
	}
	timeoutErr.lastHeartbeatDetails = ErrorDetailsValues(lastHeatbeatDetails)
	return timeoutErr
}

// NewHeartbeatTimeoutError creates TimeoutError instance.
func NewHeartbeatTimeoutError(details ...interface{}) *TimeoutError {
	return NewTimeoutError(enumspb.TIMEOUT_TYPE_HEARTBEAT, nil, details...)
}

// NewCanceledError creates CanceledError instance.
func NewCanceledError(details ...interface{}) *CanceledError {
	if len(details) == 1 {
		if d, ok := details[0].(*EncodedValues); ok {
			return &CanceledError{details: d}
		}
	}
	return &CanceledError{details: ErrorDetailsValues(details)}
}

// NewServerError create new instance of *ServerError with message.
func NewServerError(message string, nonRetryable bool, cause error) *ServerError {
	return &ServerError{message: message, nonRetryable: nonRetryable, cause: cause}
}

// NewActivityError creates ActivityError instance.
func NewActivityError(
	scheduledEventID int64,
	startedEventID int64,
	identity string,
	activityType *commonpb.ActivityType,
	activityID string,
	retryStatus enumspb.RetryStatus,
	cause error,
) *ActivityError {
	return &ActivityError{
		scheduledEventID: scheduledEventID,
		startedEventID:   startedEventID,
		identity:         identity,
		activityType:     activityType,
		activityID:       activityID,
		retryStatus:      retryStatus,
		cause:            cause,
	}
}

// NewChildWorkflowExecutionError creates ChildWorkflowExecutionError instance.
func NewChildWorkflowExecutionError(
	namespace string,
	workflowID string,
	runID string,
	workflowType string,
	initiatedEventID int64,
	startedEventID int64,
	retryStatus enumspb.RetryStatus,
	cause error,
) *ChildWorkflowExecutionError {
	return &ChildWorkflowExecutionError{
		namespace:        namespace,
		workflowID:       workflowID,
		runID:            runID,
		workflowType:     workflowType,
		initiatedEventID: initiatedEventID,
		startedEventID:   startedEventID,
		retryStatus:      retryStatus,
		cause:            cause,
	}
}

// NewWorkflowExecutionError creates WorkflowExecutionError instance.
func NewWorkflowExecutionError(
	workflowID string,
	runID string,
	workflowType string,
	cause error,
) *WorkflowExecutionError {
	return &WorkflowExecutionError{
		workflowID:   workflowID,
		runID:        runID,
		workflowType: workflowType,
		cause:        cause,
	}
}

func (e *temporalError) setFailure(f *failurepb.Failure) {
	e.originalFailure = f
}

func (e *temporalError) failure() *failurepb.Failure {
	return e.originalFailure
}

// IsCanceledError returns whether error in CanceledError.
func IsCanceledError(err error) bool {
	var canceledErr *CanceledError
	return errors.As(err, &canceledErr)
}

// NewContinueAsNewError creates ContinueAsNewError instance
// If the workflow main function returns this error then the current execution is ended and
// the new execution with same workflow ID is started automatically with options
// provided to this function.
//  ctx - use context to override any options for the new workflow like run timeout, task timeout, task list.
//	  if not mentioned it would use the defaults that the current workflow is using.
//        ctx := WithWorkflowRunTimeout(ctx, 30 * time.Minute)
//        ctx := WithWorkflowTaskTimeout(ctx, 5 * time.Second)
//	  ctx := WithWorkflowTaskList(ctx, "example-group")
//  wfn - workflow function. for new execution it can be different from the currently running.
//  args - arguments for the new workflow.
//
func NewContinueAsNewError(ctx Context, wfn interface{}, args ...interface{}) *ContinueAsNewError {
	// Validate type and its arguments.
	options := getWorkflowEnvOptions(ctx)
	if options == nil {
		panic("context is missing required options for continue as new")
	}
	env := getWorkflowEnvironment(ctx)
	workflowType, input, err := getValidatedWorkflowFunction(wfn, args, options.DataConverter, env.GetRegistry())
	if err != nil {
		panic(err)
	}

	params := &ExecuteWorkflowParams{
		WorkflowOptions: *options,
		WorkflowType:    workflowType,
		Input:           input,
		Header:          getWorkflowHeader(ctx, options.ContextPropagators),
	}
	return &ContinueAsNewError{wfn: wfn, args: args, params: params}
}

// Error from error interface
func (e *ApplicationError) Error() string {
	return e.message
}

// OriginalType returns original error type represented as string.
func (e *ApplicationError) OriginalType() string {
	return e.originalType
}

// HasDetails return if this error has strong typed detail data.
func (e *ApplicationError) HasDetails() bool {
	return e.details != nil && e.details.HasValues()
}

// Details extracts strong typed detail data of this custom error. If there is no details, it will return ErrNoData.
func (e *ApplicationError) Details(d ...interface{}) error {
	if !e.HasDetails() {
		return ErrNoData
	}
	return e.details.Get(d...)
}

// NonRetryable indicated if error is not retryable.
func (e *ApplicationError) NonRetryable() bool {
	return e.nonRetryable
}

func (e *ApplicationError) Unwrap() error {
	return e.cause
}

// Error from error interface
func (e *TimeoutError) Error() string {
	return fmt.Sprintf("TimeoutType: %v, Cause: %v", e.timeoutType, e.cause)
}

func (e *TimeoutError) Unwrap() error {
	return e.cause
}

// TimeoutType return timeout type of this error
func (e *TimeoutError) TimeoutType() enumspb.TimeoutType {
	return e.timeoutType
}

// HasLastHeartbeatDetails return if this error has strong typed detail data.
func (e *TimeoutError) HasLastHeartbeatDetails() bool {
	return e.lastHeartbeatDetails != nil && e.lastHeartbeatDetails.HasValues()
}

// LastHeartbeatDetails extracts strong typed detail data of this error. If there is no details, it will return ErrNoData.
func (e *TimeoutError) LastHeartbeatDetails(d ...interface{}) error {
	if !e.HasLastHeartbeatDetails() {
		return ErrNoData
	}
	return e.lastHeartbeatDetails.Get(d...)
}

// Error from error interface
func (e *CanceledError) Error() string {
	return "Canceled"
}

// HasDetails return if this error has strong typed detail data.
func (e *CanceledError) HasDetails() bool {
	return e.details != nil && e.details.HasValues()
}

// Details extracts strong typed detail data of this error.
func (e *CanceledError) Details(d ...interface{}) error {
	if !e.HasDetails() {
		return ErrNoData
	}
	return e.details.Get(d...)
}

func newPanicError(value interface{}, stackTrace string) *PanicError {
	return &PanicError{value: value, stackTrace: stackTrace}
}

func newWorkflowPanicError(value interface{}, stackTrace string) *workflowPanicError {
	return &workflowPanicError{value: value, stackTrace: stackTrace}
}

// Error from error interface
func (e *PanicError) Error() string {
	return fmt.Sprintf("%v", e.value)
}

// StackTrace return stack trace of the panic
func (e *PanicError) StackTrace() string {
	return e.stackTrace
}

// Error from error interface
func (e *workflowPanicError) Error() string {
	return fmt.Sprintf("%v", e.value)
}

// StackTrace return stack trace of the panic
func (e *workflowPanicError) StackTrace() string {
	return e.stackTrace
}

// Error from error interface
func (e *ContinueAsNewError) Error() string {
	return "ContinueAsNew"
}

// WorkflowType return WorkflowType of the new run
func (e *ContinueAsNewError) WorkflowType() *WorkflowType {
	return e.params.WorkflowType
}

// Args return workflow argument of the new run
func (e *ContinueAsNewError) Args() []interface{} {
	return e.args
}

// newTerminatedError creates NewTerminatedError instance
func newTerminatedError() *TerminatedError {
	return &TerminatedError{}
}

// Error from error interface
func (e *TerminatedError) Error() string {
	return "Terminated"
}

// newUnknownExternalWorkflowExecutionError creates UnknownExternalWorkflowExecutionError instance
func newUnknownExternalWorkflowExecutionError() *UnknownExternalWorkflowExecutionError {
	return &UnknownExternalWorkflowExecutionError{}
}

// Error from error interface
func (e *UnknownExternalWorkflowExecutionError) Error() string {
	return "UnknownExternalWorkflowExecution"
}

// Error from error interface
func (e *ServerError) Error() string {
	return e.message
}

func (e *ServerError) Unwrap() error {
	return e.cause
}

func (e *ActivityError) Error() string {
	return fmt.Sprintf("activity task error (scheduledEventID: %d, startedEventID: %d, identity: %s): %v", e.scheduledEventID, e.startedEventID, e.identity, e.cause)
}

func (e *ActivityError) Unwrap() error {
	return e.cause
}

// Error from error interface
func (e *ChildWorkflowExecutionError) Error() string {
	return fmt.Sprintf("child workflow execution error (workflowID: %s, runID: %s, initiatedEventID: %d, startedEventID: %d, workflowType: %s): %v",
		e.workflowID, e.runID, e.initiatedEventID, e.startedEventID, e.workflowType, e.cause)
}

func (e *ChildWorkflowExecutionError) Unwrap() error {
	return e.cause
}

// Error from error interface
func (e *WorkflowExecutionError) Error() string {
	return fmt.Sprintf("workflow execution error (workflowID: %s, runID: %s, workflowType: %s): %v",
		e.workflowID, e.runID, e.workflowType, e.cause)
}

func (e *WorkflowExecutionError) Unwrap() error {
	return e.cause
}

func convertErrDetailsToPayloads(details Values, dc DataConverter) *commonpb.Payloads {
	switch d := details.(type) {
	case ErrorDetailsValues:
		data, err := encodeArgs(dc, d)
		if err != nil {
			panic(err)
		}
		return data
	case *EncodedValues:
		return d.values
	default:
		panic(fmt.Sprintf("unknown error details type %T", details))
	}
}

// IsRetryable returns if error retryable or not.
func IsRetryable(err error, nonRetryableTypes []string) bool {
	if err == nil {
		return false
	}

	var terminatedErr *TerminatedError
	var canceledErr *CanceledError
	var workflowPanicErr *workflowPanicError
	if errors.As(err, &terminatedErr) || errors.As(err, &canceledErr) || errors.As(err, &workflowPanicErr) {
		return false
	}

	var applicationErr *ApplicationError
	var applicationErrOriginalType string
	if errors.As(err, &applicationErr) {
		if applicationErr.nonRetryable {
			return false
		}
		applicationErrOriginalType = applicationErr.originalType
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		if timeoutErr.timeoutType != enumspb.TIMEOUT_TYPE_START_TO_CLOSE &&
			timeoutErr.timeoutType != enumspb.TIMEOUT_TYPE_HEARTBEAT {
			return false
		}
	}

	var serverErr *ServerError
	if errors.As(err, &serverErr) {
		if serverErr.nonRetryable {
			return false
		}
	}

	for {
		causeErr := errors.Unwrap(err)
		if causeErr == nil {
			break
		}
		err = causeErr
	}
	errType := getErrorType(err)
	for _, nonRetryableType := range nonRetryableTypes {
		if nonRetryableType == errType || nonRetryableType == applicationErrOriginalType {
			return false
		}
	}

	return true
}

func getErrorType(err error) string {
	var t reflect.Type
	for t = reflect.TypeOf(err); t.Kind() == reflect.Ptr; t = t.Elem() {
	}

	return t.Name()
}

// convertErrorToFailure converts error to failure.
func convertErrorToFailure(err error, dc DataConverter) *failurepb.Failure {
	if err == nil {
		return nil
	}

	if fh, ok := err.(failureHolder); ok {
		if fh.failure() != nil {
			return fh.failure()
		}
	}

	failure := &failurepb.Failure{
		Source:  "GoSDK",
		Message: err.Error(),
	}

	switch err := err.(type) {
	case *ApplicationError:
		failureInfo := &failurepb.ApplicationFailureInfo{
			Type:         getErrorType(err),
			NonRetryable: err.nonRetryable,
			Details:      convertErrDetailsToPayloads(err.details, dc),
		}
		failure.FailureInfo = &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: failureInfo}
	case *CanceledError:
		failureInfo := &failurepb.CanceledFailureInfo{
			Details: convertErrDetailsToPayloads(err.details, dc),
		}
		failure.FailureInfo = &failurepb.Failure_CanceledFailureInfo{CanceledFailureInfo: failureInfo}
	case *PanicError:
		failureInfo := &failurepb.ApplicationFailureInfo{
			Type: getErrorType(err),
		}
		failure.FailureInfo = &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: failureInfo}
		failure.StackTrace = err.StackTrace()
	case *workflowPanicError:
		failureInfo := &failurepb.ApplicationFailureInfo{
			Type:         getErrorType(&PanicError{}),
			NonRetryable: true,
		}
		failure.FailureInfo = &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: failureInfo}
		failure.StackTrace = err.StackTrace()
	case *TimeoutError:
		failureInfo := &failurepb.TimeoutFailureInfo{
			TimeoutType:          err.timeoutType,
			LastHeartbeatDetails: convertErrDetailsToPayloads(err.lastHeartbeatDetails, dc),
		}
		failure.FailureInfo = &failurepb.Failure_TimeoutFailureInfo{TimeoutFailureInfo: failureInfo}
	case *TerminatedError:
		failureInfo := &failurepb.TerminatedFailureInfo{}
		failure.FailureInfo = &failurepb.Failure_TerminatedFailureInfo{TerminatedFailureInfo: failureInfo}
	case *ServerError:
		failureInfo := &failurepb.ServerFailureInfo{
			NonRetryable: err.nonRetryable,
		}
		failure.FailureInfo = &failurepb.Failure_ServerFailureInfo{ServerFailureInfo: failureInfo}
	case *ActivityError:
		failureInfo := &failurepb.ActivityFailureInfo{
			ScheduledEventId: err.scheduledEventID,
			StartedEventId:   err.startedEventID,
			Identity:         err.identity,
			ActivityType:     err.activityType,
			ActivityId:       err.activityID,
			RetryStatus:      err.retryStatus,
		}
		failure.FailureInfo = &failurepb.Failure_ActivityFailureInfo{ActivityFailureInfo: failureInfo}
	case *ChildWorkflowExecutionError:
		failureInfo := &failurepb.ChildWorkflowExecutionFailureInfo{
			Namespace: err.namespace,
			WorkflowExecution: &commonpb.WorkflowExecution{
				WorkflowId: err.workflowID,
				RunId:      err.runID,
			},
			WorkflowType:     &commonpb.WorkflowType{Name: err.workflowType},
			InitiatedEventId: err.initiatedEventID,
			StartedEventId:   err.startedEventID,
			RetryStatus:      err.retryStatus,
		}
		failure.FailureInfo = &failurepb.Failure_ChildWorkflowExecutionFailureInfo{ChildWorkflowExecutionFailureInfo: failureInfo}
	default: // All unknown errors are considered to be retryable ApplicationFailureInfo.
		failureInfo := &failurepb.ApplicationFailureInfo{
			Type:         getErrorType(err),
			NonRetryable: false,
		}
		failure.FailureInfo = &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: failureInfo}
	}

	failure.Cause = convertErrorToFailure(errors.Unwrap(err), dc)

	return failure
}

// convertFailureToError converts failure to error.
func convertFailureToError(failure *failurepb.Failure, dc DataConverter) error {
	if failure == nil {
		return nil
	}

	var err error

	if failure.GetApplicationFailureInfo() != nil {
		applicationFailureInfo := failure.GetApplicationFailureInfo()
		details := newEncodedValues(applicationFailureInfo.GetDetails(), dc)
		switch applicationFailureInfo.GetType() {
		case getErrorType(&ApplicationError{}):
			err = NewApplicationError(failure.GetMessage(), applicationFailureInfo.GetNonRetryable(), convertFailureToError(failure.GetCause(), dc), details)
		case getErrorType(&PanicError{}):
			err = newPanicError(failure.GetMessage(), failure.GetStackTrace())
		default:
			applicationErr := NewApplicationError(failure.GetMessage(), false, convertFailureToError(failure.GetCause(), dc))
			applicationErr.originalType = failure.GetApplicationFailureInfo().GetType()
			err = applicationErr
		}
	} else if failure.GetCanceledFailureInfo() != nil {
		details := newEncodedValues(failure.GetCanceledFailureInfo().GetDetails(), dc)
		err = NewCanceledError(details)
	} else if failure.GetTimeoutFailureInfo() != nil {
		timeoutFailureInfo := failure.GetTimeoutFailureInfo()
		lastHeartbeatDetails := newEncodedValues(timeoutFailureInfo.GetLastHeartbeatDetails(), dc)
		err = NewTimeoutError(
			timeoutFailureInfo.GetTimeoutType(),
			convertFailureToError(failure.GetCause(), dc),
			lastHeartbeatDetails)
	} else if failure.GetTerminatedFailureInfo() != nil {
		err = newTerminatedError()
	} else if failure.GetServerFailureInfo() != nil {
		err = NewServerError(failure.GetMessage(), failure.GetServerFailureInfo().GetNonRetryable(), convertFailureToError(failure.GetCause(), dc))
	} else if failure.GetResetWorkflowFailureInfo() != nil {
		err = NewApplicationError(failure.GetMessage(), true, convertFailureToError(failure.GetCause(), dc), failure.GetResetWorkflowFailureInfo().GetLastHeartbeatDetails())
	} else if failure.GetActivityFailureInfo() != nil {
		activityTaskInfoFailure := failure.GetActivityFailureInfo()
		err = NewActivityError(
			activityTaskInfoFailure.GetScheduledEventId(),
			activityTaskInfoFailure.GetStartedEventId(),
			activityTaskInfoFailure.GetIdentity(),
			activityTaskInfoFailure.GetActivityType(),
			activityTaskInfoFailure.GetActivityId(),
			activityTaskInfoFailure.GetRetryStatus(),
			convertFailureToError(failure.GetCause(), dc),
		)
	} else if failure.GetChildWorkflowExecutionFailureInfo() != nil {
		childWorkflowExecutionFailureInfo := failure.GetChildWorkflowExecutionFailureInfo()
		err = NewChildWorkflowExecutionError(
			childWorkflowExecutionFailureInfo.GetNamespace(),
			childWorkflowExecutionFailureInfo.GetWorkflowExecution().GetWorkflowId(),
			childWorkflowExecutionFailureInfo.GetWorkflowExecution().GetRunId(),
			childWorkflowExecutionFailureInfo.GetWorkflowType().GetName(),
			childWorkflowExecutionFailureInfo.GetInitiatedEventId(),
			childWorkflowExecutionFailureInfo.GetStartedEventId(),
			childWorkflowExecutionFailureInfo.GetRetryStatus(),
			convertFailureToError(failure.GetCause(), dc),
		)
	}

	if err == nil {
		// All unknown types are considered to be retryable ApplicationError.
		err = NewApplicationError(failure.GetMessage(), false, convertFailureToError(failure.GetCause(), dc))
	}

	if fh, ok := err.(failureHolder); ok {
		fh.setFailure(failure)
	}

	return err
}
