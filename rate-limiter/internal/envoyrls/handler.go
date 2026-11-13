// handler.go will translate Envoy's descriptors into inputs for your existing limiter.


package envoyrls

import (
	"context"
	"fmt"

	"github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
	"github.com/praneethm/rate-limiter/internal/ratelimit"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

type Handler struct {
	ratelimitv3.UnimplementedRateLimitServiceServer

	limiter ratelimit.Limiter
	domain  string
	policy  ratelimit.Policy
}

func NewHandler(
	limiter ratelimit.Limiter,
	domain string,
	policy ratelimit.Policy,
) (*Handler, error) {
	if limiter == nil {
		return nil, fmt.Errorf("rate limiter must not be nil")
	}

	if domain == "" {
		return nil, fmt.Errorf("rate-limit domain must not be empty")
	}

	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rate-limit policy: %w", err)
	}

	return &Handler{
		limiter: limiter,
		domain:  domain,
		policy:  policy,
	}, nil
}

func (h *Handler) ShouldRateLimit(
	ctx context.Context,
	req *ratelimitv3.RateLimitRequest,
) (*ratelimitv3.RateLimitResponse, error) {

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request must not be nil")
	}

	if req.GetDomain() != h.domain {
		return nil, status.Error(codes.InvalidArgument, "unsupported rate-limit domain")
	}

	descriptors := req.GetDescriptors()

	if len(descriptors) != 1 {
		return nil, status.Error(
			codes.InvalidArgument,
			"exactly one rate-limit descriptor is required",
		)
	}

	clientIP, err := ExtractClientIP(descriptors[0])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid rate-limit descriptor")
	}

	decision, err := h.limiter.Allow(ctx, clientIP, h.policy)
	if err != nil {
		return nil, status.Error(
			codes.Unavailable,
			"rate-limit decision could not be completed",
		)
	}

	response := &ratelimitv3.RateLimitResponse{
		Statuses: []*ratelimitv3.RateLimitResponse_DescriptorStatus{
			{},
		},
	}

	if decision.Allowed {
		response.OverallCode = ratelimitv3.RateLimitResponse_OK
		response.Statuses[0].Code = ratelimitv3.RateLimitResponse_OK
	} else {
		response.OverallCode = ratelimitv3.RateLimitResponse_OVER_LIMIT
		response.Statuses[0].Code = ratelimitv3.RateLimitResponse_OVER_LIMIT

		if decision.RetryAfter > 0 {
			response.Statuses[0].DurationUntilReset =
				durationpb.New(decision.RetryAfter)
		}
	}

	return response, nil
}

var _ ratelimitv3.RateLimitServiceServer = (*Handler)(nil)
