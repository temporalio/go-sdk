package rpc

import (
	"testing"

	"github.com/gogo/status"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/temporal-proto/failure"
	"go.temporal.io/temporal-proto/serviceerror"
	"go.temporal.io/temporal-proto/workflowservicemock"
	"google.golang.org/grpc/codes"
)

func TestErrorWrapper_SimpleError(t *testing.T) {
	require := require.New(t)
	wrapper := NewWorkflowServiceErrorWrapper(workflowservicemock.NewMockWorkflowServiceClient(gomock.NewController(t)))

	st := status.Error(codes.NotFound, "Something not found")

	svcerr := wrapper.(*workflowServiceErrorWrapper).convertError(st)
	require.IsType(&serviceerror.NotFound{}, svcerr)
	require.Equal("Something not found", svcerr.Error())
}

func TestErrorWrapper_ErrorWithFailure(t *testing.T) {
	require := require.New(t)
	wrapper := NewWorkflowServiceErrorWrapper(workflowservicemock.NewMockWorkflowServiceClient(gomock.NewController(t)))

	st, _ := status.New(codes.AlreadyExists, "Something started").WithDetails(&failure.WorkflowExecutionAlreadyStarted{
		StartRequestId: "srId",
		RunId:          "rId",
	})

	svcerr := wrapper.(*workflowServiceErrorWrapper).convertError(st.Err())
	require.IsType(&serviceerror.WorkflowExecutionAlreadyStarted{}, svcerr)
	require.Equal("Something started", svcerr.Error())
	weasErr := svcerr.(*serviceerror.WorkflowExecutionAlreadyStarted)
	require.Equal("rId", weasErr.RunId)
	require.Equal("srId", weasErr.StartRequestId)
}
