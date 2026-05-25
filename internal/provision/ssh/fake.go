package ssh

import "context"

type FakeTransport struct {
	RunFunc      func(ctx context.Context, request RunRequest) (RunResult, error)
	UploadFunc   func(ctx context.Context, request UploadRequest) error
	DownloadFunc func(ctx context.Context, request DownloadRequest) error
}

func (f FakeTransport) Run(ctx context.Context, request RunRequest) (RunResult, error) {
	if f.RunFunc == nil {
		return RunResult{}, nil
	}
	return f.RunFunc(ctx, request)
}

func (f FakeTransport) Upload(ctx context.Context, request UploadRequest) error {
	if f.UploadFunc == nil {
		return nil
	}
	return f.UploadFunc(ctx, request)
}

func (f FakeTransport) Download(ctx context.Context, request DownloadRequest) error {
	if f.DownloadFunc == nil {
		return nil
	}
	return f.DownloadFunc(ctx, request)
}
