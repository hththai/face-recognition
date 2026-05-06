package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	facerecognition "myproject/face-recognition"
	pb "myproject/face-recognition/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type faceServer struct {
	pb.UnimplementedFaceRecognitionServiceServer
	repo facerecognition.FaceRepository
}

func newFaceServer(repo facerecognition.FaceRepository) *faceServer {
	return &faceServer{repo: repo}
}

// RecognizeImage sends a single image to CompreFace, stores matched persons, and returns results.
func (s *faceServer) RecognizeImage(ctx context.Context, req *pb.RecognizeRequest) (*pb.RecognizeResponse, error) {
	if req.FilePath == "" {
		return nil, status.Error(codes.InvalidArgument, "file_path is required")
	}

	persons := facerecognition.GetFaceFromImage(req.FilePath)
	if len(persons) > 0 {
		if _, err := s.repo.InsertFaceAndImage(ctx, persons); err != nil {
			return nil, status.Errorf(codes.Internal, "store results: %v", err)
		}
	}

	return &pb.RecognizeResponse{Persons: toProtoPersons(persons)}, nil
}

// GetSubjectFiles returns all file names stored for a given subject.
func (s *faceServer) GetSubjectFiles(ctx context.Context, req *pb.SubjectRequest) (*pb.SubjectFilesResponse, error) {
	if req.Subject == "" {
		return nil, status.Error(codes.InvalidArgument, "subject is required")
	}

	dto, err := s.repo.GetFilesBySubject(ctx, req.Subject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get subject files: %v", err)
	}

	return &pb.SubjectFilesResponse{
		Subject:   dto.Subject,
		Filenames: dto.FileNames,
	}, nil
}

// RecognizeBatch is a bidirectional stream: client sends image paths one by one,
// server processes each via CompreFace and streams results back as they complete.
func (s *faceServer) RecognizeBatch(stream pb.FaceRecognitionService_RecognizeBatchServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "recv: %v", err)
		}

		persons := facerecognition.GetFaceFromImage(req.FilePath)
		if len(persons) > 0 {
			if _, err := s.repo.InsertFaceAndImage(stream.Context(), persons); err != nil {
				log.Printf("store %s: %v", req.FilePath, err)
			}
		}

		if err := stream.Send(&pb.RecognizeResponse{Persons: toProtoPersons(persons)}); err != nil {
			return status.Errorf(codes.Internal, "send: %v", err)
		}
	}
}

func toProtoPersons(persons []*facerecognition.Person) []*pb.RecognizedPerson {
	out := make([]*pb.RecognizedPerson, len(persons))
	for i, p := range persons {
		out[i] = &pb.RecognizedPerson{Subject: p.Name, FileName: p.Image.Name}
	}
	return out
}

// Run starts the gRPC server on the given address (e.g. ":50051").
func Run(addr string, repo facerecognition.FaceRepository) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	s := grpc.NewServer()
	pb.RegisterFaceRecognitionServiceServer(s, newFaceServer(repo))

	log.Printf("gRPC server listening on %s", addr)
	return s.Serve(lis)
}
