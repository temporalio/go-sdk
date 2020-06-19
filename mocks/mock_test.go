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

package mocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/temporal-proto/enums/v1"
	historypb "go.temporal.io/temporal-proto/history/v1"

	"go.temporal.io/temporal/client"
)

func Test_MockClient(t *testing.T) {
	testWorkflowID := "test-workflowid"
	testRunID := "test-runid"
	testWorkflowName := "workflow"
	testWorkflowInput := "input"
	mockClient := &Client{}
	var c client.Client = mockClient

	mockWorkflowRun := &WorkflowRun{}
	mockWorkflowRun.On("GetID").Return(testWorkflowID).Times(5)
	mockWorkflowRun.On("GetRunID").Return(testRunID).Times(5)
	mockWorkflowRun.On("Get", mock.Anything, mock.Anything).Return(nil).Times(2)

	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockWorkflowRun, nil).Once()
	wr, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{}, testWorkflowName, testWorkflowInput)
	mockClient.AssertExpectations(t)
	require.NoError(t, err)
	require.Equal(t, testWorkflowID, wr.GetID())
	require.Equal(t, testRunID, wr.GetRunID())

	mockClient.On("SignalWithStartWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockWorkflowRun, nil).Once()
	wr, err = c.SignalWithStartWorkflow(context.Background(), "wid", "signal", "val", client.StartWorkflowOptions{}, testWorkflowName, testWorkflowInput)
	mockClient.AssertExpectations(t)
	require.NoError(t, err)
	require.Equal(t, testWorkflowID, wr.GetID())
	require.Equal(t, testRunID, wr.GetRunID())

	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockWorkflowRun, nil).Once()
	wfRun, err := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{}, testWorkflowName, testWorkflowInput)
	mockClient.AssertExpectations(t)
	require.NoError(t, err)
	require.Equal(t, testWorkflowID, wr.GetID())
	require.Equal(t, testRunID, wr.GetRunID())

	require.Equal(t, testWorkflowID, wfRun.GetID())
	require.Equal(t, testRunID, wfRun.GetRunID())
	require.NoError(t, wfRun.Get(context.Background(), &testWorkflowID))

	mockClient.On("GetWorkflow", mock.Anything, mock.Anything, mock.Anything).
		Return(mockWorkflowRun).Once()
	wfRun = c.GetWorkflow(context.Background(), testWorkflowID, testRunID)
	mockClient.AssertExpectations(t)
	require.Equal(t, testWorkflowID, wfRun.GetID())
	require.Equal(t, testRunID, wfRun.GetRunID())
	require.NoError(t, wfRun.Get(context.Background(), &testWorkflowID))

	mockHistoryIter := &HistoryEventIterator{}
	mockHistoryIter.On("HasNext").Return(true).Once()
	mockHistoryIter.On("Next").Return(&historypb.HistoryEvent{}, nil).Once()
	mockClient.On("GetWorkflowHistory", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockHistoryIter).Once()
	historyIter := c.GetWorkflowHistory(context.Background(), testWorkflowID, testRunID, true, enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT)
	mockClient.AssertExpectations(t)
	mockWorkflowRun.AssertExpectations(t)

	require.NotNil(t, historyIter)
	require.Equal(t, true, historyIter.HasNext())
	next, err := historyIter.Next()
	require.NotNil(t, next)
	require.NoError(t, err)
}
