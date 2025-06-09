package lzcutils

import (
	"context"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
)

func ToGrpcContext(ctx context.Context, uid string) context.Context {
	meta := make(map[string]string)
	meta["X-Hc-User-Id"] = uid
	meta["X_LZCAPI_UID"] = uid
	md := metadata.New(meta)
	return metadata.NewOutgoingContext(ctx, md)
}

func ToGrpcCtxFromGinCtx(ctx *gin.Context) context.Context {
	return ToGrpcContext(context.Background(), ctx.GetHeader("X-Hc-User-Id"))
}
