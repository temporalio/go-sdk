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

package metrics

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
	"go.temporal.io/temporal-proto/serviceerror"
	"go.temporal.io/temporal-proto/workflowservice"
	"go.temporal.io/temporal-proto/workflowservicemock"
	"google.golang.org/grpc"
)

var (
	safeCharacters = []rune{'_'}

	sanitizeOptions = tally.SanitizeOptions{
		NameCharacters: tally.ValidCharacters{
			Ranges:     tally.AlphanumericRange,
			Characters: safeCharacters,
		},
		KeyCharacters: tally.ValidCharacters{
			Ranges:     tally.AlphanumericRange,
			Characters: safeCharacters,
		},
		ValueCharacters: tally.ValidCharacters{
			Ranges:     tally.AlphanumericRange,
			Characters: safeCharacters,
		},
		ReplacementCharacter: tally.DefaultReplacementCharacter,
	}
)

type testCase struct {
	serviceMethod    string
	callArgs         []interface{}
	mockReturns      []interface{}
	expectedCounters []string
}

func Test_Wrapper(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	tests := []testCase{
		// one case for each service call
		{"DeprecateDomain", []interface{}{ctx, &workflowservice.DeprecateDomainRequest{}}, []interface{}{&workflowservice.DeprecateDomainResponse{}, nil}, []string{TemporalRequest}},
		{"DescribeDomain", []interface{}{ctx, &workflowservice.DescribeDomainRequest{}}, []interface{}{&workflowservice.DescribeDomainResponse{}, nil}, []string{TemporalRequest}},
		{"GetWorkflowExecutionHistory", []interface{}{ctx, &workflowservice.GetWorkflowExecutionHistoryRequest{}}, []interface{}{&workflowservice.GetWorkflowExecutionHistoryResponse{}, nil}, []string{TemporalRequest}},
		{"ListClosedWorkflowExecutions", []interface{}{ctx, &workflowservice.ListClosedWorkflowExecutionsRequest{}}, []interface{}{&workflowservice.ListClosedWorkflowExecutionsResponse{}, nil}, []string{TemporalRequest}},
		{"ListOpenWorkflowExecutions", []interface{}{ctx, &workflowservice.ListOpenWorkflowExecutionsRequest{}}, []interface{}{&workflowservice.ListOpenWorkflowExecutionsResponse{}, nil}, []string{TemporalRequest}},
		{"PollForActivityTask", []interface{}{ctx, &workflowservice.PollForActivityTaskRequest{}}, []interface{}{&workflowservice.PollForActivityTaskResponse{}, nil}, []string{TemporalRequest}},
		{"PollForDecisionTask", []interface{}{ctx, &workflowservice.PollForDecisionTaskRequest{}}, []interface{}{&workflowservice.PollForDecisionTaskResponse{}, nil}, []string{TemporalRequest}},
		{"RecordActivityTaskHeartbeat", []interface{}{ctx, &workflowservice.RecordActivityTaskHeartbeatRequest{}}, []interface{}{&workflowservice.RecordActivityTaskHeartbeatResponse{}, nil}, []string{TemporalRequest}},
		{"RegisterDomain", []interface{}{ctx, &workflowservice.RegisterDomainRequest{}}, []interface{}{&workflowservice.RegisterDomainResponse{}, nil}, []string{TemporalRequest}},
		{"RequestCancelWorkflowExecution", []interface{}{ctx, &workflowservice.RequestCancelWorkflowExecutionRequest{}}, []interface{}{&workflowservice.RequestCancelWorkflowExecutionResponse{}, nil}, []string{TemporalRequest}},
		{"RespondActivityTaskCanceled", []interface{}{ctx, &workflowservice.RespondActivityTaskCanceledRequest{}}, []interface{}{&workflowservice.RespondActivityTaskCanceledResponse{}, nil}, []string{TemporalRequest}},
		{"RespondActivityTaskCompleted", []interface{}{ctx, &workflowservice.RespondActivityTaskCompletedRequest{}}, []interface{}{&workflowservice.RespondActivityTaskCompletedResponse{}, nil}, []string{TemporalRequest}},
		{"RespondActivityTaskFailed", []interface{}{ctx, &workflowservice.RespondActivityTaskFailedRequest{}}, []interface{}{&workflowservice.RespondActivityTaskFailedResponse{}, nil}, []string{TemporalRequest}},
		{"RespondActivityTaskCanceledByID", []interface{}{ctx, &workflowservice.RespondActivityTaskCanceledByIDRequest{}}, []interface{}{&workflowservice.RespondActivityTaskCanceledByIDResponse{}, nil}, []string{TemporalRequest}},
		{"RespondActivityTaskCompletedByID", []interface{}{ctx, &workflowservice.RespondActivityTaskCompletedByIDRequest{}}, []interface{}{&workflowservice.RespondActivityTaskCompletedByIDResponse{}, nil}, []string{TemporalRequest}},
		{"RespondActivityTaskFailedByID", []interface{}{ctx, &workflowservice.RespondActivityTaskFailedByIDRequest{}}, []interface{}{&workflowservice.RespondActivityTaskFailedByIDResponse{}, nil}, []string{TemporalRequest}},
		{"RespondDecisionTaskCompleted", []interface{}{ctx, &workflowservice.RespondDecisionTaskCompletedRequest{}}, []interface{}{nil, nil}, []string{TemporalRequest}},
		{"SignalWorkflowExecution", []interface{}{ctx, &workflowservice.SignalWorkflowExecutionRequest{}}, []interface{}{&workflowservice.SignalWorkflowExecutionResponse{}, nil}, []string{TemporalRequest}},
		{"StartWorkflowExecution", []interface{}{ctx, &workflowservice.StartWorkflowExecutionRequest{}}, []interface{}{&workflowservice.StartWorkflowExecutionResponse{}, nil}, []string{TemporalRequest}},
		{"TerminateWorkflowExecution", []interface{}{ctx, &workflowservice.TerminateWorkflowExecutionRequest{}}, []interface{}{&workflowservice.TerminateWorkflowExecutionResponse{}, nil}, []string{TemporalRequest}},
		{"ResetWorkflowExecution", []interface{}{ctx, &workflowservice.ResetWorkflowExecutionRequest{}}, []interface{}{&workflowservice.ResetWorkflowExecutionResponse{}, nil}, []string{TemporalRequest}},
		{"UpdateDomain", []interface{}{ctx, &workflowservice.UpdateDomainRequest{}}, []interface{}{&workflowservice.UpdateDomainResponse{}, nil}, []string{TemporalRequest}},
		// one case of invalid request
		{"PollForActivityTask", []interface{}{ctx, &workflowservice.PollForActivityTaskRequest{}}, []interface{}{nil, serviceerror.NewNotFound("")}, []string{TemporalRequest, TemporalInvalidRequest}},
		// one case of server error
		{"PollForActivityTask", []interface{}{ctx, &workflowservice.PollForActivityTaskRequest{}}, []interface{}{nil, serviceerror.NewInternal("")}, []string{TemporalRequest, TemporalError}},
		{"QueryWorkflow", []interface{}{ctx, &workflowservice.QueryWorkflowRequest{}}, []interface{}{nil, serviceerror.NewInternal("")}, []string{TemporalRequest, TemporalError}},
		{"RespondQueryTaskCompleted", []interface{}{ctx, &workflowservice.RespondQueryTaskCompletedRequest{}}, []interface{}{nil, serviceerror.NewInternal("")}, []string{TemporalRequest, TemporalError}},
	}

	// run each test twice - once with the regular scope, once with a sanitized metrics scope
	for _, test := range tests {
		runTest(t, test, newService, assertMetrics, fmt.Sprintf("%v_normal", test.serviceMethod))
		runTest(t, test, newPromService, assertPromMetrics, fmt.Sprintf("%v_prom_sanitized", test.serviceMethod))
	}
}

func runTest(
	t *testing.T,
	test testCase,
	serviceFunc func(*testing.T) (*workflowservicemock.MockWorkflowServiceClient, workflowservice.WorkflowServiceClient, io.Closer, *CapturingStatsReporter),
	validationFunc func(*testing.T, *CapturingStatsReporter, string, []string),
	name string,
) {
	t.Run(name, func(t *testing.T) {
		t.Parallel()
		// gomock mutates the returns slice, which leads to different test values between the two runs.
		// copy the slice until gomock fixes it: https://github.com/golang/mock/issues/353
		returns := append(make([]interface{}, 0, len(test.mockReturns)), test.mockReturns...)

		mockService, wrapperService, closer, reporter := serviceFunc(t)
		switch test.serviceMethod {
		case "DeprecateDomain":
			mockService.EXPECT().DeprecateDomain(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "DescribeDomain":
			mockService.EXPECT().DescribeDomain(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "GetWorkflowExecutionHistory":
			mockService.EXPECT().GetWorkflowExecutionHistory(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "ListClosedWorkflowExecutions":
			mockService.EXPECT().ListClosedWorkflowExecutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "ListOpenWorkflowExecutions":
			mockService.EXPECT().ListOpenWorkflowExecutions(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "PollForActivityTask":
			mockService.EXPECT().PollForActivityTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "PollForDecisionTask":
			mockService.EXPECT().PollForDecisionTask(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RecordActivityTaskHeartbeat":
			mockService.EXPECT().RecordActivityTaskHeartbeat(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RecordActivityTaskHeartbeatByID":
			mockService.EXPECT().RecordActivityTaskHeartbeatByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RegisterDomain":
			mockService.EXPECT().RegisterDomain(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RequestCancelWorkflowExecution":
			mockService.EXPECT().RequestCancelWorkflowExecution(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RespondActivityTaskCanceled":
			mockService.EXPECT().RespondActivityTaskCanceled(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RespondActivityTaskCompleted":
			mockService.EXPECT().RespondActivityTaskCompleted(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RespondActivityTaskFailed":
			mockService.EXPECT().RespondActivityTaskFailed(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RespondActivityTaskCanceledByID":
			mockService.EXPECT().RespondActivityTaskCanceledByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RespondActivityTaskCompletedByID":
			mockService.EXPECT().RespondActivityTaskCompletedByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RespondActivityTaskFailedByID":
			mockService.EXPECT().RespondActivityTaskFailedByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RespondDecisionTaskCompleted":
			mockService.EXPECT().RespondDecisionTaskCompleted(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "SignalWorkflowExecution":
			mockService.EXPECT().SignalWorkflowExecution(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "SignaWithStartlWorkflowExecution":
			mockService.EXPECT().SignalWithStartWorkflowExecution(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "StartWorkflowExecution":
			mockService.EXPECT().StartWorkflowExecution(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "TerminateWorkflowExecution":
			mockService.EXPECT().TerminateWorkflowExecution(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "ResetWorkflowExecution":
			mockService.EXPECT().ResetWorkflowExecution(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "UpdateDomain":
			mockService.EXPECT().UpdateDomain(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "QueryWorkflow":
			mockService.EXPECT().QueryWorkflow(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		case "RespondQueryTaskCompleted":
			mockService.EXPECT().RespondQueryTaskCompleted(gomock.Any(), gomock.Any(), gomock.Any()).Return(returns...)
		}

		callOption := grpc.EmptyCallOption{}
		inputs := make([]reflect.Value, len(test.callArgs))
		for i, arg := range test.callArgs {
			inputs[i] = reflect.ValueOf(arg)
		}
		inputs = append(inputs, reflect.ValueOf(callOption))
		method := reflect.ValueOf(wrapperService).MethodByName(test.serviceMethod)
		method.Call(inputs)
		require.NoError(t, closer.Close())
		validationFunc(t, reporter, test.serviceMethod, test.expectedCounters)
	})
}

func assertMetrics(t *testing.T, reporter *CapturingStatsReporter, methodName string, counterNames []string) {
	require.Equal(t, len(counterNames), len(reporter.counts))
	for _, name := range counterNames {
		counterName := TemporalMetricsPrefix + methodName + "." + name
		find := false
		// counters are not in order
		for _, counter := range reporter.counts {
			if counterName == counter.name {
				find = true
				break
			}
		}
		require.True(t, find)
	}
	require.Equal(t, 1, len(reporter.timers))
	require.Equal(t, TemporalMetricsPrefix+methodName+"."+TemporalLatency, reporter.timers[0].name)
}

func assertPromMetrics(t *testing.T, reporter *CapturingStatsReporter, methodName string, counterNames []string) {
	require.Equal(t, len(counterNames), len(reporter.counts))
	for _, name := range counterNames {
		counterName := makePromCompatible(TemporalMetricsPrefix + methodName + "." + name)
		find := false
		// counters are not in order
		for _, counter := range reporter.counts {
			if counterName == counter.name {
				find = true
				break
			}
		}
		require.True(t, find)
	}
	require.Equal(t, 1, len(reporter.timers))
	expected := makePromCompatible(TemporalMetricsPrefix + methodName + "." + TemporalLatency)
	require.Equal(t, expected, reporter.timers[0].name)
}

func makePromCompatible(name string) string {
	name = strings.Replace(name, "-", "_", -1)
	name = strings.Replace(name, ".", "_", -1)
	return name
}

func newService(t *testing.T) (
	mockService *workflowservicemock.MockWorkflowServiceClient,
	wrapperService workflowservice.WorkflowServiceClient,
	closer io.Closer,
	reporter *CapturingStatsReporter,
) {
	mockCtrl := gomock.NewController(t)
	mockService = workflowservicemock.NewMockWorkflowServiceClient(mockCtrl)
	isReplay := false
	scope, closer, reporter := NewMetricsScope(&isReplay)
	wrapperService = NewWorkflowServiceWrapper(mockService, scope)
	return
}

func newPromService(t *testing.T) (
	mockService *workflowservicemock.MockWorkflowServiceClient,
	wrapperService workflowservice.WorkflowServiceClient,
	closer io.Closer,
	reporter *CapturingStatsReporter,
) {
	mockCtrl := gomock.NewController(t)
	mockService = workflowservicemock.NewMockWorkflowServiceClient(mockCtrl)
	isReplay := false
	scope, closer, reporter := newPromScope(&isReplay)
	wrapperService = NewWorkflowServiceWrapper(mockService, scope)
	return
}

func newPromScope(isReplay *bool) (tally.Scope, io.Closer, *CapturingStatsReporter) {
	reporter := &CapturingStatsReporter{}
	opts := tally.ScopeOptions{
		Reporter:        reporter,
		SanitizeOptions: &sanitizeOptions,
	}
	scope, closer := tally.NewRootScope(opts, time.Second)
	return WrapScope(isReplay, scope, &realClock{}), closer, reporter
}
