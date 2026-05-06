package server

import (
	"context"
	"io"
	"net"
	"testing"

	facerecognition "myproject/face-recognition"
	pb "myproject/face-recognition/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// mockRepo is a test double for FaceRepository.
// Only GetFilesBySubject is configurable; the rest are no-ops.
type mockRepo struct {
	getFilesBySubjectFn func(ctx context.Context, subject string) (*facerecognition.SubjectFilesDTO, error)
}

func (m *mockRepo) InsertFaceSubject(_ context.Context, _ []string) (int64, error) { return 0, nil }
func (m *mockRepo) InsertFilePath(_ context.Context, _ []facerecognition.ImageFile) (int64, error) {
	return 0, nil
}
func (m *mockRepo) InsertFaceAndImage(_ context.Context, _ []*facerecognition.Person) (int64, error) {
	return 0, nil
}
func (m *mockRepo) GetImagesAndProcess(_ context.Context, _ int) ([]facerecognition.ImageFile, error) {
	return nil, nil
}
func (m *mockRepo) UpdateImageStatus(_ context.Context, _ []string, _ string) error { return nil }
func (m *mockRepo) GetFilesBySubject(ctx context.Context, subject string) (*facerecognition.SubjectFilesDTO, error) {
	if m.getFilesBySubjectFn != nil {
		return m.getFilesBySubjectFn(ctx, subject)
	}
	return &facerecognition.SubjectFilesDTO{Subject: subject, FileNames: []string{}}, nil
}

// newTestClient starts an in-process gRPC server via bufconn and returns a client + cleanup func.
func newTestClient(t *testing.T, repo facerecognition.FaceRepository) (pb.FaceRecognitionServiceClient, func()) {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	pb.RegisterFaceRecognitionServiceServer(s, newFaceServer(repo))
	go s.Serve(lis) //nolint:errcheck

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufnet: %v", err)
	}

	return pb.NewFaceRecognitionServiceClient(conn), func() {
		conn.Close()
		s.Stop()
	}
}

// --- RecognizeImage ---

func TestRecognizeImage_EmptyPath_ReturnsInvalidArgument(t *testing.T) {
	client, cleanup := newTestClient(t, &mockRepo{})
	defer cleanup()

	_, err := client.RecognizeImage(context.Background(), &pb.RecognizeRequest{FilePath: ""})

	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

// --- GetSubjectFiles ---

func TestGetSubjectFiles_EmptySubject_ReturnsInvalidArgument(t *testing.T) {
	client, cleanup := newTestClient(t, &mockRepo{})
	defer cleanup()

	_, err := client.GetSubjectFiles(context.Background(), &pb.SubjectRequest{Subject: ""})

	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestGetSubjectFiles_ReturnsFilesFromRepo(t *testing.T) {
	repo := &mockRepo{
		getFilesBySubjectFn: func(_ context.Context, subject string) (*facerecognition.SubjectFilesDTO, error) {
			return &facerecognition.SubjectFilesDTO{
				Subject:   subject,
				FileNames: []string{"a.jpg", "b.jpg"},
			}, nil
		},
	}
	client, cleanup := newTestClient(t, repo)
	defer cleanup()

	resp, err := client.GetSubjectFiles(context.Background(), &pb.SubjectRequest{Subject: "phoebe"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Subject != "phoebe" {
		t.Errorf("expected subject phoebe, got %s", resp.Subject)
	}
	if len(resp.Filenames) != 2 {
		t.Errorf("expected 2 filenames, got %d", len(resp.Filenames))
	}
}

// --- RecognizeBatch ---

// TestRecognizeBatch_EmptyPath verifies the stream stays open and returns an
// empty persons list when the file path is empty (GetFaceFromImage short-circuits).
func TestRecognizeBatch_EmptyPath_ReturnsEmptyPersons(t *testing.T) {
	client, cleanup := newTestClient(t, &mockRepo{})
	defer cleanup()

	stream, err := client.RecognizeBatch(context.Background())
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	if err := stream.Send(&pb.RecognizeRequest{FilePath: ""}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("close send: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil && err != io.EOF {
		t.Fatalf("recv: %v", err)
	}
	if resp != nil && len(resp.Persons) != 0 {
		t.Errorf("expected empty persons, got %v", resp.Persons)
	}
}
